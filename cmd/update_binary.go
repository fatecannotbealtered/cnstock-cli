package cmd

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fatecannotbealtered/cnstock-cli/internal/api"
)

// This file gives cnstock-cli a self-contained binary self-update: download the
// platform archive + checksums.txt + Sigstore bundle, verify the signature
// in-process against the release-workflow identity, verify the archive SHA256,
// extract the binary, and replace the running executable. It does not depend on
// npm / go / pip being present on the user's machine.

const (
	updateBinaryName = "cnstock-cli"
	updateGitHubAPI  = "https://api.github.com"
)

// Update stages, in execution order. Every update failure envelope reports the
// stage it failed at so an agent can reason about the post-state (CLI-SPEC §14).
const (
	updateStageDiscover        = "discover"
	updateStageDownload        = "download"
	updateStageVerifySignature = "verify_signature"
	updateStageVerifyChecksum  = "verify_checksum"
	updateStageReplace         = "replace"
	updateStageSkillSync       = "skill_sync"
)

// integrityError marks a non-retryable supply-chain failure (missing/invalid
// signature, or checksum mismatch). Callers map it to E_INTEGRITY, never to a
// retryable network code.
type integrityError struct{ err error }

func (e *integrityError) Error() string { return e.err.Error() }
func (e *integrityError) Unwrap() error { return e.err }

func newIntegrityError(err error) error { return &integrityError{err: err} }

// isIntegrityError reports whether err is a release-integrity failure.
func isIntegrityError(err error) bool {
	var ie *integrityError
	return errors.As(err, &ie)
}

// replaceError marks a local failure during the atomic replace stage (temp dir,
// extract, file write/rename, permission, disk full). These were previously
// misclassified as E_NETWORK; the caller maps permission failures to E_FORBIDDEN
// (exit 4) and all other io/disk failures to E_IO (exit 1). The binary was NOT
// swapped (the atomic rename did not commit), so binary_replaced stays false.
type replaceError struct {
	err        error
	permission bool
}

func (e *replaceError) Error() string { return e.err.Error() }
func (e *replaceError) Unwrap() error { return e.err }

func newReplaceError(err error) error {
	return &replaceError{err: err, permission: errors.Is(err, os.ErrPermission)}
}

// asReplaceError reports whether err is a replace-stage local failure.
func asReplaceError(err error) (*replaceError, bool) {
	var re *replaceError
	if errors.As(err, &re) {
		return re, true
	}
	return nil, false
}

// updateStageOf reports the stage an update error failed at, defaulting to
// discover for an error that carries no stage marker.
func updateStageOf(err error) string {
	var se *stagedError
	if errors.As(err, &se) {
		return se.stage
	}
	return updateStageDiscover
}

// stagedError annotates an update error with the stage it occurred at without
// disturbing the underlying integrity/replace classification (Unwrap chains
// through, so isIntegrityError/asReplaceError still see the cause).
type stagedError struct {
	stage string
	err   error
}

func (e *stagedError) Error() string { return e.err.Error() }
func (e *stagedError) Unwrap() error { return e.err }

func withStage(stage string, err error) error {
	if err == nil {
		return nil
	}
	return &stagedError{stage: stage, err: err}
}

// Testable seams (mirrors the pattern used by sibling tools).
var (
	updateBinaryHTTPClient = &http.Client{Timeout: 2 * time.Minute}
	updateBinaryGitHubAPI  = updateGitHubAPI
	updateBinaryPlatform   = func() (string, string) { return runtime.GOOS, runtime.GOARCH }
	updateBinaryExecutable = os.Executable
	updateBinaryApply      = applyUpdateBinary
)

type updateReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type updateBinaryRelease struct {
	TagName string               `json:"tag_name"`
	HTMLURL string               `json:"html_url"`
	Assets  []updateReleaseAsset `json:"assets"`
}

type updateApplyResult struct {
	Status string
	Path   string
}

// performBinaryUpdate downloads, verifies, and installs the target release
// binary. It returns the install status, the signature status (always
// "verified" on success), and the installed path. Integrity failures are
// wrapped so the caller can classify them as non-retryable.
func performBinaryUpdate(ctx context.Context, targetVersion string) (status, signatureStatus, installedPath string, err error) {
	exe, err := updateBinaryExecutable()
	if err != nil {
		return "", "", "", withStage(updateStageReplace, newReplaceError(fmt.Errorf("resolving current executable: %w", err)))
	}

	rel, err := fetchBinaryRelease(ctx, targetVersion)
	if err != nil {
		return "", "", "", withStage(updateStageDiscover, err)
	}
	target := normalizeVersion(rel.TagName)
	if target == "" {
		return "", "", "", withStage(updateStageDiscover, errors.New("release is missing tag_name"))
	}
	assetName, err := updateArchiveName(target)
	if err != nil {
		return "", "", "", withStage(updateStageDiscover, err)
	}
	assetURL := findUpdateAssetURL(rel.Assets, assetName)
	if assetURL == "" {
		return "", "", "", withStage(updateStageDiscover, fmt.Errorf("release %s does not include asset %s", rel.TagName, assetName))
	}
	checksumURL := findUpdateAssetURL(rel.Assets, "checksums.txt")
	if checksumURL == "" {
		return "", "", "", withStage(updateStageDiscover, fmt.Errorf("release %s does not include checksums.txt", rel.TagName))
	}
	bundleURL := findUpdateAssetURL(rel.Assets, "checksums.txt.sigstore.json")

	tmpDir, err := os.MkdirTemp("", "cnstock-cli-update-*")
	if err != nil {
		return "", "", "", withStage(updateStageReplace, newReplaceError(fmt.Errorf("creating temp dir: %w", err)))
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	archivePath := filepath.Join(tmpDir, assetName)
	if err := downloadUpdateFile(ctx, assetURL, archivePath); err != nil {
		return "", "", "", withStage(updateStageDownload, fmt.Errorf("downloading archive: %w", err))
	}
	checksumPath := filepath.Join(tmpDir, "checksums.txt")
	if err := downloadUpdateFile(ctx, checksumURL, checksumPath); err != nil {
		return "", "", "", withStage(updateStageDownload, fmt.Errorf("downloading checksums: %w", err))
	}

	// Verify the signature. A missing bundle or a failed verification is a
	// supply-chain integrity failure (E_INTEGRITY, non-retryable). DOWNLOADING the
	// bundle, however, is a network step: a transient fetch failure (or a ctx
	// interrupt) must classify as retryable network/timeout, not as a forged
	// release. verifyUpdateChecksumSignature distinguishes the two via a wrapped
	// download error so the caller assigns the right stage and code.
	signatureStatus, err = verifyUpdateChecksumSignature(ctx, checksumPath, bundleURL, tmpDir)
	if err != nil {
		var de *signatureDownloadError
		if errors.As(err, &de) {
			return "", "", "", withStage(updateStageDownload, de.err)
		}
		return "", "", "", withStage(updateStageVerifySignature, newIntegrityError(fmt.Errorf("verifying release signature: %w", err)))
	}
	if err := verifyUpdateChecksum(archivePath, checksumPath, assetName); err != nil {
		return "", "", "", withStage(updateStageVerifyChecksum, newIntegrityError(fmt.Errorf("verifying archive: %w", err)))
	}

	// From here on, failures are local replace-stage problems (extract, write,
	// rename, permission, disk). The atomic rename has not committed, so the
	// installed binary is untouched.
	binPath, err := extractUpdateArchive(archivePath, assetName, tmpDir)
	if err != nil {
		return "", "", "", withStage(updateStageReplace, newReplaceError(fmt.Errorf("extracting archive: %w", err)))
	}
	applied, err := updateBinaryApply(binPath, exe)
	if err != nil {
		return "", "", "", withStage(updateStageReplace, newReplaceError(fmt.Errorf("installing update: %w", err)))
	}
	return applied.Status, signatureStatus, applied.Path, nil
}

// signatureDownloadError marks a failure to FETCH the signature bundle (network /
// timeout / interrupt), as opposed to a signature that fails to verify. The caller
// classifies the wrapped cause through the normal taxonomy (retryable network/
// timeout) instead of as a non-retryable E_INTEGRITY supply-chain failure.
type signatureDownloadError struct{ err error }

func (e *signatureDownloadError) Error() string { return e.err.Error() }
func (e *signatureDownloadError) Unwrap() error { return e.err }

// verifyUpdateChecksumSignature enforces a mandatory, in-process Sigstore
// signature check on checksums.txt. There is no skip path: a release without a
// signature bundle, or one whose signature does not verify against this repo's
// release-workflow identity, is refused (integrity failure). A failure to fetch
// the bundle is returned wrapped in *signatureDownloadError so the caller can tell
// a transient download blip apart from a forged release.
func verifyUpdateChecksumSignature(ctx context.Context, checksumPath, bundleURL, tmpDir string) (string, error) {
	if strings.TrimSpace(bundleURL) == "" {
		return "missing", errors.New("release does not include checksums.txt.sigstore.json; refusing to install an unsigned release")
	}
	bundlePath := filepath.Join(tmpDir, "checksums.txt.sigstore.json")
	if err := downloadUpdateFile(ctx, bundleURL, bundlePath); err != nil {
		return "download_failed", &signatureDownloadError{err: fmt.Errorf("downloading checksum signature bundle: %w", err)}
	}
	if err := updateVerifySignature(ctx, checksumPath, bundlePath, updateSignerIdentityRegexp()); err != nil {
		if errors.Is(err, errTrustRootUnavailable) {
			// Refreshing the TUF trust metadata is a network step, not a signature
			// verdict: surface as retryable network, not E_INTEGRITY.
			return "trust_root_unavailable", &signatureDownloadError{err: err}
		}
		return "failed", err
	}
	return "verified", nil
}

func fetchBinaryRelease(ctx context.Context, targetVersion string) (*updateBinaryRelease, error) {
	base := strings.TrimRight(updateBinaryGitHubAPI, "/")
	url := base + "/repos/" + updateSkillRepo + "/releases/latest"
	if v := normalizeVersion(targetVersion); v != "" {
		url = base + "/repos/" + updateSkillRepo + "/releases/tags/" + canonicalVersionTag(v)
	}
	req, err := newUpdateRequest(ctx, url, "application/json")
	if err != nil {
		return nil, err
	}
	resp, err := updateBinaryHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	// Classify by status so a 404 (tag/release gone) is non-retryable E_NOT_FOUND,
	// a 408 is E_TIMEOUT, 429 is E_RATE_LIMITED, and 5xx is E_SERVER — instead of
	// every non-2xx collapsing to E_NETWORK at the caller.
	if err := api.ErrorForStatus(resp.StatusCode, "GET %s returned %d: %s", api.RedactURL(url), resp.StatusCode, truncateUpdateMessage(string(data), 200)); err != nil {
		return nil, err
	}
	var rel updateBinaryRelease
	if err := json.Unmarshal(data, &rel); err != nil {
		return nil, fmt.Errorf("parsing release JSON: %w", err)
	}
	return &rel, nil
}

func updateArchiveName(ver string) (string, error) {
	goos, goarch := updateBinaryPlatform()
	platform, ok := map[string]string{"darwin": "darwin", "linux": "linux", "windows": "windows"}[goos]
	if !ok {
		return "", fmt.Errorf("unsupported update platform: %s-%s", goos, goarch)
	}
	arch, ok := map[string]string{"amd64": "amd64", "arm64": "arm64"}[goarch]
	if goos == "windows" && goarch == "arm64" {
		arch, ok = "amd64", true
	}
	if !ok {
		return "", fmt.Errorf("unsupported update platform: %s-%s", goos, goarch)
	}
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("%s-%s-%s-%s%s", updateBinaryName, normalizeVersion(ver), platform, arch, ext), nil
}

func findUpdateAssetURL(assets []updateReleaseAsset, name string) string {
	for _, a := range assets {
		if a.Name == name {
			return a.BrowserDownloadURL
		}
	}
	return ""
}

func newUpdateRequest(ctx context.Context, url, accept string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", updateBinaryName)
	if tok := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	return req, nil
}

func downloadUpdateFile(ctx context.Context, url, dest string) error {
	req, err := newUpdateRequest(ctx, url, "application/octet-stream")
	if err != nil {
		return err
	}
	resp, err := updateBinaryHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return api.ErrorForStatus(resp.StatusCode, "GET %s returned %d: %s", api.RedactURL(url), resp.StatusCode, truncateUpdateMessage(string(data), 200))
	}
	tmp := dest + ".part"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func verifyUpdateChecksum(archivePath, checksumPath, assetName string) error {
	checksumData, err := os.ReadFile(checksumPath)
	if err != nil {
		return fmt.Errorf("reading checksums: %w", err)
	}
	expected := ""
	for _, line := range strings.Split(string(checksumData), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if filepath.Base(fields[len(fields)-1]) == assetName {
			expected = strings.ToLower(fields[0])
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("checksum for %s not found", assetName)
	}
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("reading archive: %w", err)
	}
	defer func() { _ = f.Close() }()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return fmt.Errorf("hashing archive: %w", err)
	}
	if hex.EncodeToString(hash.Sum(nil)) != expected {
		return fmt.Errorf("checksum mismatch for %s", assetName)
	}
	return nil
}

func extractUpdateArchive(archivePath, assetName, tmpDir string) (string, error) {
	if strings.HasSuffix(assetName, ".zip") {
		return extractUpdateZip(archivePath, tmpDir)
	}
	if strings.HasSuffix(assetName, ".tar.gz") {
		return extractUpdateTarGz(archivePath, tmpDir)
	}
	return "", fmt.Errorf("unsupported archive type: %s", assetName)
}

func extractUpdateZip(archivePath, tmpDir string) (string, error) {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = zr.Close() }()
	want := updateArchiveBinaryName()
	for _, f := range zr.File {
		if filepath.Base(f.Name) != want {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		defer func() { _ = rc.Close() }()
		return writeExtractedUpdateBinary(tmpDir, want, rc)
	}
	return "", fmt.Errorf("%s not found in archive", want)
}

func extractUpdateTarGz(archivePath, tmpDir string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer func() { _ = gz.Close() }()
	tr := tar.NewReader(gz)
	want := updateArchiveBinaryName()
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		if hdr.Typeflag != tar.TypeReg || filepath.Base(hdr.Name) != want {
			continue
		}
		return writeExtractedUpdateBinary(tmpDir, want, tr)
	}
	return "", fmt.Errorf("%s not found in archive", want)
}

func updateArchiveBinaryName() string {
	goos, _ := updateBinaryPlatform()
	if goos == "windows" {
		return updateBinaryName + ".exe"
	}
	return updateBinaryName
}

func writeExtractedUpdateBinary(tmpDir, name string, r io.Reader) (string, error) {
	outDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(outDir, 0o700); err != nil {
		return "", err
	}
	outPath := filepath.Join(outDir, name)
	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o700)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return outPath, nil
}

// applyUpdateBinary installs the freshly extracted binary over the running
// executable using a cross-platform rename trick (identical on Windows and
// Unix): write `.<base>.new`, move the in-use binary aside to `.<base>.old`,
// rename `.new` into place, and on failure roll back from `.old`. On Windows a
// running executable can still be renamed (just not overwritten or deleted), so
// this commits atomically in-process — no .cmd helper, no restart deferral. The
// leftover `.old` is best-effort removed; if Windows still has it locked, the
// removal is ignored and it is reaped on a later run.
func applyUpdateBinary(src, dst string) (updateApplyResult, error) {
	target := dst
	if resolved, err := filepath.EvalSymlinks(dst); err == nil {
		target = resolved
	}
	mode := os.FileMode(0o755)
	if st, err := os.Stat(target); err == nil {
		mode = st.Mode().Perm()
		if mode&0o111 == 0 {
			mode |= 0o755
		}
	}
	dir := filepath.Dir(target)
	base := filepath.Base(target)
	newPath := filepath.Join(dir, "."+base+".new")
	backupPath := filepath.Join(dir, "."+base+".old")

	_ = os.Remove(newPath)
	if err := updateCopyFile(src, newPath, mode); err != nil {
		return updateApplyResult{}, err
	}

	_ = os.Remove(backupPath)
	if err := os.Rename(target, backupPath); err != nil {
		_ = os.Remove(newPath)
		return updateApplyResult{}, fmt.Errorf("preparing to replace %s: %w", target, err)
	}
	if err := os.Rename(newPath, target); err != nil {
		_ = os.Rename(backupPath, target)
		return updateApplyResult{}, fmt.Errorf("replacing %s: %w; original restored", target, err)
	}
	// Best-effort cleanup; on Windows the old binary may still be locked by the
	// running process and refuse deletion — that is fine, it is reaped later.
	_ = os.Remove(backupPath)
	return updateApplyResult{Status: "installed", Path: target}, nil
}

func updateCopyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(dst, mode)
}
