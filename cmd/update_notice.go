package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	updateNoticeCacheTTL       = 24 * time.Hour
	updateNoticeRefreshTimeout = 2 * time.Second
	updateNoticeEnvOptOut      = "CNSTOCK_CLI_NO_UPDATE_CHECK"
	updateNoticeLegacyOptOut   = "CNS_NO_UPDATE_CHECK"
)

type updateNotice struct {
	Type               string   `json:"type"`
	Severity           string   `json:"severity"`
	Message            string   `json:"message"`
	CurrentVersion     string   `json:"current_version"`
	LatestVersion      string   `json:"latest_version"`
	UpdateAvailable    bool     `json:"update_available"`
	InstallMethod      string   `json:"install_method,omitempty"`
	RecommendedCommand string   `json:"recommended_command"`
	ReleaseURL         string   `json:"release_url,omitempty"`
	CheckedAt          string   `json:"checked_at"`
	Source             string   `json:"source"`
	NextSteps          []string `json:"next_steps"`
}

type updateNoticeCache struct {
	CheckedAt string         `json:"checked_at"`
	Notices   []updateNotice `json:"notices,omitempty"`
}

func installUpdateNoticeHelp(root *cobra.Command) {
	root.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		if cmd.Long != "" {
			fmt.Fprintln(cmd.OutOrStdout(), cmd.Long)
			fmt.Fprintln(cmd.OutOrStdout())
		} else if cmd.Short != "" {
			fmt.Fprintln(cmd.OutOrStdout(), cmd.Short)
			fmt.Fprintln(cmd.OutOrStdout())
		}
		fmt.Fprint(cmd.OutOrStdout(), cmd.UsageString())
		printUpdateNoticeHint(cmd.OutOrStdout(), readCachedUpdateNotices())
	})
}

func refreshUpdateNotices(ctx context.Context, source string) []updateNotice {
	if updateNoticeAutoDisabled() {
		return nil
	}
	refreshCtx, cancel := context.WithTimeout(ctx, updateNoticeRefreshTimeout)
	defer cancel()

	client := &http.Client{Timeout: updateNoticeRefreshTimeout}
	rel, _, err := fetchLatestRelease(refreshCtx, client, latestReleaseEndpoint())
	if err != nil {
		return readCachedUpdateNotices()
	}
	method := detectInstallMethod()
	latest := normalizeVersion(rel.TagName)
	available := false
	if cmp, ok := compareVersions(version, rel.TagName); ok {
		available = cmp < 0
	}
	report := updateReport{
		CurrentVersion:    version,
		LatestVersion:     rel.TagName,
		TargetVersion:     latest,
		UpdateAvailable:   &available,
		InstallMethod:     method,
		ReleaseURL:        rel.HTMLURL,
		RecommendedAction: shellJoin(updateCommandArgs(method, latest)),
	}
	if report.ReleaseURL == "" {
		report.ReleaseURL = latestReleaseURL
	}
	notices := updateNoticesFromReport(report, source)
	writeUpdateNoticeCache(notices)
	return notices
}

func updateNoticesFromReport(report updateReport, source string) []updateNotice {
	if report.UpdateAvailable == nil || !*report.UpdateAvailable {
		return nil
	}
	current := normalizeVersion(report.CurrentVersion)
	latest := normalizeVersion(report.TargetVersion)
	if latest == "" {
		latest = normalizeVersion(report.LatestVersion)
	}
	command := strings.TrimSpace(report.RecommendedAction)
	if command == "" {
		command = "cnstock-cli update --dry-run --compact"
	}
	notice := updateNotice{
		Type:               "update_available",
		Severity:           "info",
		CurrentVersion:     current,
		LatestVersion:      latest,
		UpdateAvailable:    true,
		InstallMethod:      report.InstallMethod,
		RecommendedCommand: command,
		ReleaseURL:         report.ReleaseURL,
		CheckedAt:          time.Now().UTC().Format(time.RFC3339),
		Source:             source,
		NextSteps: []string{
			"run the recommended command",
			"ask the user before confirming the local self-update",
			"after update, run cnstock-cli changelog --since " + current + " --compact",
			"refresh cnstock-cli reference --compact before using new behavior",
		},
	}
	notice.Message = fmt.Sprintf("cnstock-cli %s is available (current %s)", latest, current)
	return []updateNotice{notice}
}

func readCachedUpdateNotices() []updateNotice {
	if updateNoticeAutoDisabled() {
		return nil
	}
	path, err := updateNoticeCachePath()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cache updateNoticeCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil
	}
	checkedAt, err := time.Parse(time.RFC3339, cache.CheckedAt)
	if err != nil || time.Since(checkedAt) > updateNoticeCacheTTL {
		return nil
	}
	notices := make([]updateNotice, 0, len(cache.Notices))
	for _, notice := range cache.Notices {
		if notice.Type != "update_available" || !notice.UpdateAvailable {
			continue
		}
		notice.Source = "cache"
		notices = append(notices, notice)
	}
	return notices
}

func writeUpdateNoticeCache(notices []updateNotice) {
	if updateNoticeAutoDisabled() {
		return
	}
	path, err := updateNoticeCachePath()
	if err != nil {
		return
	}
	if len(notices) == 0 {
		_ = os.Remove(path)
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	checkedAt := time.Now().UTC().Format(time.RFC3339)
	cache := updateNoticeCache{CheckedAt: checkedAt, Notices: notices}
	for i := range cache.Notices {
		cache.Notices[i].CheckedAt = checkedAt
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o600)
}

func updateNoticeCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", err
	}
	return filepath.Join(home, ".cnstock-cli", "update-check.json"), nil
}

func updateNoticeDisabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(updateNoticeEnvOptOut)))
	legacy := strings.ToLower(strings.TrimSpace(os.Getenv(updateNoticeLegacyOptOut)))
	return value == "1" || value == "true" || value == "yes" || legacy == "1" || legacy == "true" || legacy == "yes"
}

func updateNoticeAutoDisabled() bool {
	return updateNoticeDisabled() || strings.HasSuffix(os.Args[0], ".test")
}

func printUpdateNoticeHint(w io.Writer, notices []updateNotice) {
	if len(notices) == 0 {
		return
	}
	notice := notices[0]
	fmt.Fprintf(w, "\nUpdate available: cnstock-cli %s -> %s. Run: %s\n", notice.CurrentVersion, notice.LatestVersion, notice.RecommendedCommand)
}
