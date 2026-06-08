package cmd

import (
	"runtime"
	"sort"

	"github.com/fatecannotbealtered/cnstock-cli/internal/api"
	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Print runtime environment, configuration, and active endpoints",
	Args:  cobra.NoArgs,
	RunE:  runContext,
}

func init() {
	rootCmd.AddCommand(contextCmd)
}

// contextReport describes the runtime environment so an agent can understand
// capabilities and configuration before acting.
type contextReport struct {
	Version        string             `json:"version"`
	GoVersion      string             `json:"go_version"`
	OS             string             `json:"os"`
	Arch           string             `json:"arch"`
	Environment    string             `json:"environment"`
	Account        string             `json:"account"`
	RiskTier       string             `json:"risk_tier"`
	RiskSummary    string             `json:"risk_summary"`
	PermissionTier string             `json:"permission_tier"`
	DefaultFormat  string             `json:"default_format"`
	Formats        []string           `json:"formats"`
	Commands       []string           `json:"commands"`
	Config         contextConfig      `json:"config"`
	Credentials    credentialStatus   `json:"credentials"`
	Endpoints      []api.EndpointInfo `json:"endpoints"`
}

type contextConfig struct {
	EndpointOverrides bool `json:"endpoint_overrides"`
}

type credentialStatus struct {
	Required   bool `json:"required"`
	Configured bool `json:"configured"`
}

func runContext(cmd *cobra.Command, args []string) error {
	report := contextReport{
		Version:        version,
		GoVersion:      runtime.Version(),
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
		Environment:    "public-web-endpoints",
		Account:        "none",
		RiskTier:       riskTier,
		RiskSummary:    riskTierDescription,
		PermissionTier: "read-only",
		DefaultFormat:  "json",
		Formats:        []string{"json", "text", "raw"},
		Commands:       commandNames(),
		Endpoints:      api.Endpoints(),
	}
	report.Config.EndpointOverrides = hasEndpointOverride(report.Endpoints)
	report.Credentials = credentialStatus{Required: false, Configured: false}

	if outputFormat != "text" {
		emitJSON(report)
		return nil
	}

	output.Bold("  cnstock-cli " + report.Version)
	output.Gray("  " + report.GoVersion + " " + report.OS + "/" + report.Arch)
	output.Gray("  " + report.RiskTier + " " + report.PermissionTier)
	headers := []string{"Endpoint", "Env", "Overridden"}
	var rows [][]string
	for _, e := range report.Endpoints {
		ov := "no"
		if e.Overridden {
			ov = "yes"
		}
		rows = append(rows, []string{e.Name, e.Env, ov})
	}
	output.Table(headers, rows)
	return nil
}

func hasEndpointOverride(endpoints []api.EndpointInfo) bool {
	for _, e := range endpoints {
		if e.Overridden {
			return true
		}
	}
	return false
}

// commandNames returns the names of all registered top-level commands.
func commandNames() []string {
	var names []string
	for _, c := range rootCmd.Commands() {
		if c.Hidden || c.Name() == "help" {
			continue
		}
		names = append(names, c.Name())
	}
	sort.Strings(names)
	return names
}
