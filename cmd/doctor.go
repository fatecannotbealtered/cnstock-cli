package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/fatecannotbealtered/cnstock-cli/internal/api"
	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Probe endpoint connectivity and latency (environment health check)",
	Args:  cobra.NoArgs,
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

// endpointHealth is one connectivity probe result.
type endpointHealth struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	OK        bool   `json:"ok"`
	LatencyMs int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

// doctorCheck is a spec-level health check with an actionable fix.
type doctorCheck struct {
	Check  string `json:"check"`
	Status string `json:"status"`
	Fix    string `json:"fix,omitempty"`
}

// doctorReport is the aggregate health report.
type doctorReport struct {
	OK        bool             `json:"ok"`
	CheckedAt string           `json:"checked_at"`
	RiskTier  string           `json:"risk_tier"`
	Checks    []doctorCheck    `json:"checks"`
	Endpoints []endpointHealth `json:"endpoints"`
	Notices   []updateNotice   `json:"notices,omitempty"`
}

func runDoctor(cmd *cobra.Command, args []string) error {
	client := api.NewClient().WithTimeout(5 * time.Second)

	report := doctorReport{OK: true, CheckedAt: time.Now().UTC().Format(time.RFC3339), RiskTier: riskTier, Notices: refreshUpdateNotices(cmd.Context(), "doctor")}
	versionCheck := doctorCheck{Check: "version", Status: versionStatus(), Fix: versionFix()}
	if versionCheck.Status == "fail" {
		report.OK = false
	}
	report.Checks = append(report.Checks,
		doctorCheck{Check: "credentials", Status: "pass"},
		doctorCheck{Check: "permissions", Status: "pass"},
		versionCheck,
		doctorCheck{Check: "release_readiness", Status: releaseReadinessCheckStatus(), Fix: releaseReadinessCheckFix()},
	)
	networkFailed := false
	for _, t := range api.ProbeTargets() {
		h := endpointHealth{Name: t.Name, URL: api.RedactURL(t.URL)}
		start := time.Now()
		_, err := client.GetString(cmd.Context(), t.URL)
		h.LatencyMs = time.Since(start).Milliseconds()
		if err != nil {
			h.OK = false
			h.Error = err.Error()
			report.OK = false
			networkFailed = true
		} else {
			h.OK = true
		}
		report.Endpoints = append(report.Endpoints, h)
	}
	if !networkFailed {
		report.Checks = append(report.Checks, doctorCheck{Check: "network", Status: "pass"})
	} else {
		report.Checks = append(report.Checks, doctorCheck{Check: "network", Status: "fail", Fix: "check network connectivity, proxy/VPN settings, or overridden CNS_* endpoint URLs"})
	}

	if outputFormat != "text" {
		emitJSON(report)
		if networkFailed {
			setExitCode(ExitNetwork)
		} else if !report.OK {
			setExitCode(ExitAuth)
		}
		return nil
	}

	headers := []string{"Endpoint", "OK", "Latency", "Error"}
	var rows [][]string
	for _, h := range report.Endpoints {
		ok := "✔"
		if !h.OK {
			ok = "✖"
		}
		rows = append(rows, []string{h.Name, ok, fmt.Sprintf("%dms", h.LatencyMs), h.Error})
	}
	output.Table(headers, rows)
	printUpdateNoticeHint(os.Stdout, report.Notices)
	if networkFailed {
		setExitCode(ExitNetwork)
	} else if !report.OK {
		setExitCode(ExitAuth)
	}
	return nil
}

func versionStatus() string {
	if version == "dev" || version == "(devel)" || version == "" {
		return "warn"
	}
	return "pass"
}

func versionFix() string {
	switch versionStatus() {
	case "pass":
		return ""
	case "warn":
		return "development build; confirm the final package version before release"
	default:
		return ""
	}
}
