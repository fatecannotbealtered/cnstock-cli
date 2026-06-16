package cmd

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/fatecannotbealtered/cnstock-cli/internal/api"
	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
	"github.com/spf13/cobra"
)

const (
	defaultLatestReleaseEndpoint = "https://api.github.com/repos/fatecannotbealtered/cnstock-cli/releases/latest"
	npmPackageName               = "@fateforge/cnstock-cli"
	goInstallTarget              = "github.com/fatecannotbealtered/cnstock-cli/cmd/cnstock-cli"
	latestReleaseURL             = "https://github.com/fatecannotbealtered/cnstock-cli/releases/latest"
	updateSkillRepo              = "fatecannotbealtered/cnstock-cli"
)

var (
	updateMethod        = "auto"
	updateTargetVersion string
	updateCheckOnly     bool
	updateNow           = time.Now
	updateExecCommand   = exec.CommandContext
	updateSkillSync     = runUpdateSkillSync
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update cnstock-cli and sync the bundled Skill",
	Long:  "Check GitHub Releases, dry-run the planned local lifecycle update, then update the package/binary and sync the whole Agent Skill directory after confirmation.",
	Args:  cobra.NoArgs,
	RunE:  runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolVar(&updateCheckOnly, "check", false, "Check for an available update without installing")
	updateCmd.Flags().StringVar(&updateMethod, "method", "auto", "Update method hint: auto|npm|go|github")
	updateCmd.Flags().StringVar(&updateTargetVersion, "target-version", "", "Install a specific version (for example 1.2.3 or v1.2.3)")
}

type latestRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

type updateReport struct {
	CurrentVersion    string         `json:"current_version"`
	LatestVersion     string         `json:"latest_version"`
	TargetVersion     string         `json:"target_version"`
	UpdateAvailable   *bool          `json:"update_available,omitempty"`
	Status            string         `json:"status"`
	InstallMethod     string         `json:"install_method"`
	ReleaseURL        string         `json:"release_url"`
	RecommendedAction string         `json:"recommended_action"`
	Commands          []string       `json:"commands"`
	PostUpdateAction  string         `json:"post_update_action"`
	SignatureStatus   string         `json:"signature_status"`
	SkillSyncCommand  string         `json:"skill_sync_command"`
	SkillSyncStatus   string         `json:"skill_sync_status"`
	ConfirmToken      string         `json:"confirm_token,omitempty"`
	ExpiresAt         string         `json:"expires_at,omitempty"`
	Preview           map[string]any `json:"preview,omitempty"`
	Notes             []string       `json:"notes,omitempty"`
	Notices           []updateNotice `json:"notices,omitempty"`
}

type updateConfirmPayload struct {
	Method           string `json:"method"`
	TargetVersion    string `json:"target_version"`
	Command          string `json:"command"`
	SkillSyncCommand string `json:"skill_sync_command"`
	ExpiresAt        string `json:"expires_at"`
}

func runUpdate(cmd *cobra.Command, _ []string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	rel, raw, err := fetchLatestRelease(cmd.Context(), client, latestReleaseEndpoint())
	if err != nil {
		return handleError(err)
	}
	if outputFormat == "raw" {
		output.Raw(raw)
		return nil
	}

	latest := normalizeVersion(rel.TagName)
	if latest == "" {
		latest = strings.TrimSpace(rel.TagName)
	}
	target := normalizeVersion(updateTargetVersion)
	if target == "" {
		target = latest
	}
	command := binaryUpdateDescriptor(target)
	skillCommand := updateSkillSyncCommand()
	report := updateReport{
		CurrentVersion:    version,
		LatestVersion:     rel.TagName,
		TargetVersion:     target,
		Status:            "checked",
		InstallMethod:     "github-binary",
		ReleaseURL:        rel.HTMLURL,
		RecommendedAction: command,
		Commands:          []string{command},
		PostUpdateAction:  "After installing, run `cnstock-cli changelog --since " + version + "` and `cnstock-cli reference --compact` before continuing.",
		SignatureStatus:   "not_checked",
		SkillSyncCommand:  skillCommand,
		SkillSyncStatus:   "not_run",
	}
	if report.ReleaseURL == "" {
		report.ReleaseURL = latestReleaseURL
	}

	if cmp, ok := compareVersions(version, rel.TagName); ok {
		available := cmp < 0 || targetVersionRequested(updateTargetVersion, version)
		report.UpdateAvailable = &available
		if available {
			report.Status = "available"
		} else {
			report.Status = "up_to_date"
		}
	} else {
		report.Status = "unknown"
		report.Notes = append(report.Notes, "current version is not a release version; compare manually")
	}

	if updateCheckOnly {
		report.Notices = updateNoticesFromReport(report, "update_check")
		writeUpdateNoticeCache(report.Notices)
		emitUpdateReport(report)
		return nil
	}
	if report.UpdateAvailable != nil && !*report.UpdateAvailable && strings.TrimSpace(updateTargetVersion) == "" {
		emitUpdateReport(report)
		return nil
	}
	if dryRunMode {
		return emitUpdateDryRun(cmd, report, command)
	}
	if strings.TrimSpace(confirmToken) == "" {
		return updateFail("update requires --dry-run followed by --confirm <confirm_token>", output.ErrConfirm, ExitConfirmRequired)
	}
	payload, err := validateUpdateConfirmToken(confirmToken, "github-binary", target, command, skillCommand)
	if err != nil {
		return updateFail(err.Error(), output.ErrConflict, ExitConflict)
	}
	// Single-use enforcement: a confirm token may drive exactly one update. A
	// replay (e.g. an agent retrying an update that timed out) is rejected so the
	// update cannot be duplicated; mark BEFORE performing it so a crash mid-update
	// still consumes the token.
	now := updateNow()
	if isConfirmTokenConsumed(confirmToken, now) {
		return updateFail("confirm token already used; the operation may have completed — re-run --dry-run to see current state", output.ErrConflict, ExitConflict)
	}
	expiresUnix := now.Add(15 * time.Minute).Unix()
	if expires, perr := time.Parse(time.RFC3339, payload.ExpiresAt); perr == nil {
		expiresUnix = expires.Unix()
	}
	markConfirmTokenConsumed(confirmToken, expiresUnix, now)

	// Download + verify (in-process Sigstore) + checksum + replace binary. No
	// dependency on npm/go/pip; an unsigned or unverifiable release is refused.
	status, sigStatus, _, err := performBinaryUpdate(cmd.Context(), target)
	if err != nil {
		if isIntegrityError(err) {
			// Non-retryable: a missing/invalid signature or checksum mismatch is a
			// supply-chain red flag, not a transient blip.
			return updateFail(err.Error(), output.ErrIntegrity, ExitGeneric)
		}
		return updateFail("update failed: "+err.Error(), output.ErrNetwork, ExitNetwork)
	}
	if err := updateSkillSync(cmd.Context(), updateSkillRepo); err != nil {
		return updateFail("syncing skill directory: "+err.Error(), output.ErrNetwork, ExitNetwork)
	}

	report.SignatureStatus = sigStatus
	report.Status = "updated"
	if status == "scheduled" {
		report.Status = "scheduled"
		report.Notes = append(report.Notes, "binary replacement scheduled; restart the command after this process exits")
	}
	report.SkillSyncStatus = "synced"
	report.PostUpdateAction = "Run `cnstock-cli changelog --since " + version + " --compact` and `cnstock-cli reference --compact` before continuing."
	emitUpdateReport(report)
	return nil
}

// binaryUpdateDescriptor is the stable action string bound into the confirm
// token and shown in the report. The dry-run and confirm paths compute it
// identically so the token round-trips.
func binaryUpdateDescriptor(target string) string {
	t := normalizeVersion(target)
	if t == "" {
		t = "latest"
	}
	return "download + verify (sigstore) + replace cnstock-cli binary from GitHub release " + canonicalVersionTag(t)
}

func emitUpdateDryRun(cmd *cobra.Command, report updateReport, command string) error {
	expires := updateNow().UTC().Add(15 * time.Minute).Truncate(time.Second)
	payload := updateConfirmPayload{
		Method:           report.InstallMethod,
		TargetVersion:    report.TargetVersion,
		Command:          command,
		SkillSyncCommand: report.SkillSyncCommand,
		ExpiresAt:        expires.Format(time.RFC3339),
	}
	report.Status = "dry_run"
	report.ConfirmToken = issueUpdateConfirmToken(payload)
	report.ExpiresAt = payload.ExpiresAt
	report.Preview = map[string]any{
		"action": "update cnstock-cli",
		"changes": []map[string]any{
			{"operation": "download, verify signature + checksum, replace binary", "command": command},
			{"operation": "sync skill directory", "command": report.SkillSyncCommand},
		},
		"command_path": cmd.CommandPath(),
	}
	emitUpdateReport(report)
	return nil
}

func latestReleaseEndpoint() string {
	if v := os.Getenv("CNS_UPDATE_ENDPOINT"); v != "" {
		return v
	}
	return defaultLatestReleaseEndpoint
}

func fetchLatestRelease(ctx context.Context, client *http.Client, endpoint string) (latestRelease, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return latestRelease{}, "", api.NewNetworkError("creating update request for %s: %v", api.RedactURL(endpoint), api.RedactText(err.Error()))
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", api.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return latestRelease{}, "", api.NewNetworkError("checking latest release at %s: %v", api.RedactURL(endpoint), api.RedactText(err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return latestRelease{}, "", api.NewNetworkError("reading update response: %v", err)
	}
	raw := string(body)
	if resp.StatusCode >= 500 {
		return latestRelease{}, raw, api.NewNetworkError("checking latest release: HTTP %d %s", resp.StatusCode, resp.Status)
	}
	if resp.StatusCode != http.StatusOK {
		return latestRelease{}, raw, api.NewServerError("checking latest release: HTTP %d %s", resp.StatusCode, resp.Status)
	}

	var rel latestRelease
	if err := json.Unmarshal(body, &rel); err != nil {
		return latestRelease{}, raw, api.NewServerError("update response is not valid JSON: %v", err)
	}
	if strings.TrimSpace(rel.TagName) == "" {
		return latestRelease{}, raw, api.NewServerError("update response missing tag_name")
	}
	return rel, raw, nil
}

func updateSkillSyncCommand() string {
	return "npx skills add " + updateSkillRepo + " -y -g"
}

func runUpdateSkillSync(ctx context.Context, repo string) error {
	cmd := updateExecCommand(ctx, "npx", "skills", "add", repo, "-y", "-g")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("%w: %s", err, truncateUpdateMessage(msg, 500))
		}
		return err
	}
	return nil
}

func issueUpdateConfirmToken(payload updateConfirmPayload) string {
	data, _ := json.Marshal(payload)
	sum := confirmDigest32(data)
	return "ct_" + base64.RawURLEncoding.EncodeToString(data) + "." + hex.EncodeToString(sum[:])
}

func validateUpdateConfirmToken(token, method, target, command, skillCommand string) (updateConfirmPayload, error) {
	var empty updateConfirmPayload
	token = strings.TrimSpace(token)
	if !strings.HasPrefix(token, "ct_") {
		return empty, fmt.Errorf("confirmation token is invalid; re-run with --dry-run")
	}
	parts := strings.SplitN(strings.TrimPrefix(token, "ct_"), ".", 2)
	if len(parts) != 2 {
		return empty, fmt.Errorf("confirmation token is invalid; re-run with --dry-run")
	}
	data, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return empty, fmt.Errorf("confirmation token is invalid; re-run with --dry-run")
	}
	sum := confirmDigest32(data)
	if !strings.EqualFold(parts[1], hex.EncodeToString(sum[:])) {
		return empty, fmt.Errorf("confirmation token is invalid; re-run with --dry-run")
	}
	var payload updateConfirmPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return empty, fmt.Errorf("confirmation token is invalid; re-run with --dry-run")
	}
	expires, err := time.Parse(time.RFC3339, payload.ExpiresAt)
	if err != nil || !updateNow().UTC().Before(expires) {
		return empty, fmt.Errorf("confirmation token expired; re-run with --dry-run")
	}
	if payload.Method != method || payload.TargetVersion != target || payload.Command != command || payload.SkillSyncCommand != skillCommand {
		return empty, fmt.Errorf("confirmation token does not match this update; re-run with --dry-run")
	}
	return payload, nil
}

func compareVersions(current, latest string) (int, bool) {
	cur, ok := parseVersion(current)
	if !ok {
		return 0, false
	}
	newest, ok := parseVersion(latest)
	if !ok {
		return 0, false
	}
	for i := range cur {
		if cur[i] < newest[i] {
			return -1, true
		}
		if cur[i] > newest[i] {
			return 1, true
		}
	}
	return 0, true
}

func parseVersion(v string) ([3]int, bool) {
	var out [3]int
	v = normalizeVersion(v)
	if v == "" || v == "dev" || v == "(devel)" || v == "latest" {
		return out, false
	}
	if idx := strings.IndexAny(v, "-+"); idx >= 0 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "refs/tags/")
	v = strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V")
	return v
}

func canonicalVersionTag(v string) string {
	v = normalizeVersion(v)
	if v == "" || v == "latest" {
		return "latest"
	}
	return "v" + v
}

func targetVersionRequested(requested, current string) bool {
	target := normalizeVersion(requested)
	return target != "" && target != normalizeVersion(current)
}

func emitUpdateReport(report updateReport) {
	if outputFormat != "text" {
		emitJSON(report)
		return
	}
	printUpdateReport(report)
}

func printUpdateReport(report updateReport) {
	output.Bold("  cnstock-cli update")
	headers := []string{"Field", "Value"}
	updateStatus := "unknown"
	if report.UpdateAvailable != nil {
		if *report.UpdateAvailable {
			updateStatus = "yes"
		} else {
			updateStatus = "no"
		}
	}
	rows := [][]string{
		{"status", report.Status},
		{"current", report.CurrentVersion},
		{"latest", report.LatestVersion},
		{"target", report.TargetVersion},
		{"update_available", updateStatus},
		{"method", report.InstallMethod},
		{"signature_status", report.SignatureStatus},
		{"release", report.ReleaseURL},
		{"recommended", report.RecommendedAction},
		{"skill_sync", report.SkillSyncCommand},
		{"post_update", report.PostUpdateAction},
	}
	output.Table(headers, rows)
	for _, note := range report.Notes {
		output.Gray("  " + note)
	}
}

func updateFail(msg string, code output.ErrorCode, exitCode int) error {
	if outputFormat == "text" {
		output.Error(msg)
	} else {
		output.PrintErrorEnvelopeWithDuration(msg, code, false, nil, compactMode, commandDuration())
	}
	setExitCode(exitCode)
	return ErrSilent
}

func truncateUpdateMessage(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
