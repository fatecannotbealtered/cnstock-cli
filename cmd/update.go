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
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fatecannotbealtered/cnstock-cli/internal/api"
	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
	"github.com/spf13/cobra"
)

const (
	defaultLatestReleaseEndpoint = "https://api.github.com/repos/fatecannotbealtered/cnstock-cli/releases/latest"
	npmPackageName               = "@fatecannotbealtered-/cnstock-cli"
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
	method := strings.ToLower(strings.TrimSpace(updateMethod))
	if _, ok := validUpdateMethods[method]; !ok {
		return handleError(api.NewValidationError("method only supports auto, npm, go, github"))
	}
	if method == "auto" {
		method = detectInstallMethod()
	}

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
	commandArgs := updateCommandArgs(method, target)
	command := shellJoin(commandArgs)
	commands := updateCommands(method)
	skillCommand := updateSkillSyncCommand()
	report := updateReport{
		CurrentVersion:    version,
		LatestVersion:     rel.TagName,
		TargetVersion:     target,
		Status:            "checked",
		InstallMethod:     method,
		ReleaseURL:        rel.HTMLURL,
		RecommendedAction: command,
		Commands:          commands,
		PostUpdateAction:  "After installing, run `cnstock-cli changelog --since " + version + "` and `cnstock-cli reference --compact` before continuing.",
		SignatureStatus:   updateSignatureStatus(method),
		SkillSyncCommand:  skillCommand,
		SkillSyncStatus:   "not_run",
	}
	if report.ReleaseURL == "" {
		report.ReleaseURL = latestReleaseURL
	}
	if command == "" {
		report.RecommendedAction = "Download the latest binary from " + latestReleaseURL
		report.Notes = append(report.Notes, "automatic update is unsupported for method "+method)
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
	payload, err := validateUpdateConfirmToken(confirmToken, method, target, command, skillCommand)
	if err != nil {
		return updateFail(err.Error(), output.ErrConflict, ExitConflict)
	}
	if payload.Command == "" {
		return updateFail("automatic update is unsupported for method "+method, output.ErrValidation, ExitBadArgs)
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
	if err := runUpdateCommand(cmd.Context(), commandArgs); err != nil {
		return updateFail("running update command: "+err.Error(), output.ErrNetwork, ExitNetwork)
	}
	if err := updateSkillSync(cmd.Context(), updateSkillRepo); err != nil {
		return updateFail("syncing skill directory: "+err.Error(), output.ErrNetwork, ExitNetwork)
	}

	report.Status = "updated"
	report.SkillSyncStatus = "synced"
	report.PostUpdateAction = "Run `cnstock-cli changelog --since " + version + " --compact` and `cnstock-cli reference --compact` before continuing."
	emitUpdateReport(report)
	return nil
}

func emitUpdateDryRun(cmd *cobra.Command, report updateReport, command string) error {
	if command == "" {
		return updateFail("automatic update is unsupported for method "+report.InstallMethod, output.ErrValidation, ExitBadArgs)
	}
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
			{"operation": "run package manager update", "command": command},
			{"operation": "sync skill directory", "command": report.SkillSyncCommand},
		},
		"command_path": cmd.CommandPath(),
	}
	emitUpdateReport(report)
	return nil
}

var validUpdateMethods = map[string]struct{}{
	"auto":   {},
	"npm":    {},
	"go":     {},
	"github": {},
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

func detectInstallMethod() string {
	exe, err := os.Executable()
	if err != nil {
		return "npm"
	}
	path := filepath.ToSlash(strings.ToLower(exe))
	switch {
	case strings.Contains(path, "node_modules") && strings.Contains(path, "@fatecannotbealtered-"):
		return "npm"
	case samePath(filepath.Dir(exe), os.Getenv("GOBIN")):
		return "go"
	case strings.Contains(path, "/go/bin/"):
		return "go"
	default:
		for _, gp := range filepath.SplitList(os.Getenv("GOPATH")) {
			if samePath(filepath.Dir(exe), filepath.Join(gp, "bin")) {
				return "go"
			}
		}
		return "npm"
	}
}

func samePath(a, b string) bool {
	if strings.TrimSpace(b) == "" {
		return false
	}
	aa, err := filepath.Abs(a)
	if err != nil {
		aa = a
	}
	bb, err := filepath.Abs(b)
	if err != nil {
		bb = b
	}
	return strings.EqualFold(filepath.Clean(aa), filepath.Clean(bb))
}

func updateCommands(method string) []string {
	switch method {
	case "go":
		return []string{
			shellJoin(updateCommandArgs("go", "latest")),
			shellJoin(updateCommandArgs("npm", "latest")),
			"Download the latest binary from " + latestReleaseURL,
		}
	case "github":
		return []string{
			"Download the latest binary from " + latestReleaseURL,
			shellJoin(updateCommandArgs("npm", "latest")),
			shellJoin(updateCommandArgs("go", "latest")),
		}
	default:
		return []string{
			shellJoin(updateCommandArgs("npm", "latest")),
			shellJoin(updateCommandArgs("go", "latest")),
			"Download the latest binary from " + latestReleaseURL,
		}
	}
}

func updateSignatureStatus(method string) string {
	switch method {
	case "npm":
		return "handled_by_npm_installer"
	case "go":
		return "handled_by_go_module_verification"
	case "github":
		return "manual_release_verification_required"
	default:
		return "not_checked"
	}
}

func updateCommandArgs(method, targetVersion string) []string {
	target := normalizeVersion(targetVersion)
	if target == "" {
		target = "latest"
	}
	switch method {
	case "npm":
		return []string{"npm", "install", "-g", npmPackageName + "@" + target}
	case "go":
		tag := "latest"
		if target != "latest" {
			tag = canonicalVersionTag(target)
		}
		return []string{"go", "install", goInstallTarget + "@" + tag}
	default:
		return nil
	}
}

func runUpdateCommand(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return api.NewValidationError("automatic update command is unavailable")
	}
	cmd := updateExecCommand(ctx, args[0], args[1:]...)
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

func shellJoin(args []string) string {
	if len(args) == 0 {
		return ""
	}
	out := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.ContainsAny(arg, " \t\"'") {
			out = append(out, strconv.Quote(arg))
			continue
		}
		out = append(out, arg)
	}
	return strings.Join(out, " ")
}

func truncateUpdateMessage(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
