package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
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
	goInstallTarget              = "github.com/fatecannotbealtered/cnstock-cli/cmd/cnstock-cli@latest"
	latestReleaseURL             = "https://github.com/fatecannotbealtered/cnstock-cli/releases/latest"
)

var updateMethod = "auto"

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for newer cnstock-cli releases and print update instructions",
	Long:  "Check GitHub Releases for the latest cnstock-cli version and print safe update instructions. This command does not modify files.",
	Args:  cobra.NoArgs,
	RunE:  runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().StringVar(&updateMethod, "method", "auto", "Update method hint: auto|npm|go|github")
}

type latestRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

type updateReport struct {
	CurrentVersion    string   `json:"current_version"`
	LatestVersion     string   `json:"latest_version"`
	UpdateAvailable   *bool    `json:"update_available,omitempty"`
	InstallMethod     string   `json:"install_method"`
	ReleaseURL        string   `json:"release_url"`
	RecommendedAction string   `json:"recommended_action"`
	Commands          []string `json:"commands"`
	PostUpdateAction  string   `json:"post_update_action"`
	Notes             []string `json:"notes,omitempty"`
}

func runUpdate(cmd *cobra.Command, args []string) error {
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

	commands := updateCommands(method)
	report := updateReport{
		CurrentVersion:    version,
		LatestVersion:     rel.TagName,
		InstallMethod:     method,
		ReleaseURL:        rel.HTMLURL,
		RecommendedAction: commands[0],
		Commands:          commands,
		PostUpdateAction:  "After installing, run `cnstock-cli changelog --since " + version + "` before continuing.",
	}
	if report.ReleaseURL == "" {
		report.ReleaseURL = latestReleaseURL
	}

	if cmp, ok := compareVersions(version, rel.TagName); ok {
		available := cmp < 0
		report.UpdateAvailable = &available
	} else {
		report.Notes = append(report.Notes, "current version is not a release version; compare manually")
	}

	if outputFormat != "text" {
		emitJSON(report)
		return nil
	}

	printUpdateReport(report)
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
			"go install " + goInstallTarget,
			"npm install -g " + npmPackageName + "@latest",
			"Download the latest binary from " + latestReleaseURL,
		}
	case "github":
		return []string{
			"Download the latest binary from " + latestReleaseURL,
			"npm install -g " + npmPackageName + "@latest",
			"go install " + goInstallTarget,
		}
	default:
		return []string{
			"npm install -g " + npmPackageName + "@latest",
			"go install " + goInstallTarget,
			"Download the latest binary from " + latestReleaseURL,
		}
	}
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
	v = strings.TrimSpace(strings.TrimPrefix(v, "v"))
	if v == "" || v == "dev" || v == "(devel)" {
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
		{"current", report.CurrentVersion},
		{"latest", report.LatestVersion},
		{"update_available", updateStatus},
		{"method", report.InstallMethod},
		{"release", report.ReleaseURL},
		{"recommended", report.RecommendedAction},
		{"post_update", report.PostUpdateAction},
	}
	output.Table(headers, rows)
	for _, note := range report.Notes {
		output.Gray("  " + note)
	}
}
