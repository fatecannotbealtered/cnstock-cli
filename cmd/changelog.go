package cmd

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	project "github.com/fatecannotbealtered/cnstock-cli"
	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
	"github.com/spf13/cobra"
)

var changelogSince string

var changelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "Print version changes from CHANGELOG.md",
	Long:  "Print version changes derived from CHANGELOG.md. Use --since to return only entries newer than a known version.",
	Args:  cobra.NoArgs,
	RunE:  runChangelog,
}

func init() {
	changelogCmd.Flags().StringVar(&changelogSince, "since", "", "Only include entries newer than this version")
	rootCmd.AddCommand(changelogCmd)
}

type changelogReport struct {
	CurrentVersion string           `json:"current_version"`
	Since          string           `json:"since,omitempty"`
	Entries        []changelogEntry `json:"entries"`
}

type changelogEntry struct {
	Version string              `json:"version"`
	Date    string              `json:"date,omitempty"`
	Changes map[string][]string `json:"changes"`
}

var (
	changelogVersionRe  = regexp.MustCompile(`^## \[([^\]]+)\](?: - (.+))?$`)
	changelogCategoryRe = regexp.MustCompile(`^### (Added|Changed|Deprecated|Removed|Fixed|Security)$`)
)

func runChangelog(cmd *cobra.Command, args []string) error {
	if outputFormat == "raw" {
		output.Raw(project.ChangelogMarkdown)
		return nil
	}

	entries := parseChangelog(project.ChangelogMarkdown)
	if changelogSince != "" {
		entries = filterChangelogSince(entries, changelogSince)
	}
	report := changelogReport{
		CurrentVersion: version,
		Since:          strings.TrimSpace(changelogSince),
		Entries:        entries,
	}

	if outputFormat != "text" {
		emitJSON(report)
		return nil
	}

	if len(report.Entries) == 0 {
		output.Info("No changelog entries matched.")
		return nil
	}
	for _, entry := range report.Entries {
		title := entry.Version
		if entry.Date != "" {
			title += " - " + entry.Date
		}
		output.Bold("  " + title)
		for _, category := range orderedChangelogCategories(entry.Changes) {
			output.Gray("  " + category)
			for _, item := range entry.Changes[category] {
				fmt.Println("  - " + item)
			}
		}
		fmt.Println()
	}
	return nil
}

func parseChangelog(markdown string) []changelogEntry {
	var entries []changelogEntry
	var current *changelogEntry
	var category string

	flush := func() {
		if current == nil || len(current.Changes) == 0 {
			return
		}
		entries = append(entries, *current)
	}

	for _, rawLine := range strings.Split(markdown, "\n") {
		line := strings.TrimSpace(rawLine)
		if match := changelogVersionRe.FindStringSubmatch(line); match != nil {
			flush()
			current = &changelogEntry{
				Version: strings.TrimSpace(match[1]),
				Date:    strings.TrimSpace(match[2]),
				Changes: map[string][]string{},
			}
			category = ""
			continue
		}
		if current == nil {
			continue
		}
		if match := changelogCategoryRe.FindStringSubmatch(line); match != nil {
			category = strings.ToLower(match[1])
			if _, ok := current.Changes[category]; !ok {
				current.Changes[category] = nil
			}
			continue
		}
		if category == "" || !strings.HasPrefix(line, "- ") {
			continue
		}
		current.Changes[category] = append(current.Changes[category], strings.TrimSpace(strings.TrimPrefix(line, "- ")))
	}
	flush()
	return entries
}

func filterChangelogSince(entries []changelogEntry, since string) []changelogEntry {
	since = strings.TrimSpace(since)
	if since == "" {
		return entries
	}
	filtered := make([]changelogEntry, 0, len(entries))
	for _, entry := range entries {
		if strings.EqualFold(entry.Version, "Unreleased") {
			filtered = append(filtered, entry)
			continue
		}
		cmp, ok := compareVersions(entry.Version, since)
		if ok && cmp > 0 {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func orderedChangelogCategories(changes map[string][]string) []string {
	preferred := []string{"added", "changed", "deprecated", "removed", "fixed", "security"}
	var out []string
	seen := map[string]bool{}
	for _, key := range preferred {
		if len(changes[key]) > 0 {
			out = append(out, key)
			seen[key] = true
		}
	}
	var rest []string
	for key, items := range changes {
		if !seen[key] && len(items) > 0 {
			rest = append(rest, key)
		}
	}
	sort.Strings(rest)
	return append(out, rest...)
}
