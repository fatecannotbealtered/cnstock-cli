package cmd

import (
	"fmt"
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

// doctorReport is the aggregate health report.
type doctorReport struct {
	OK        bool             `json:"ok"`
	CheckedAt string           `json:"checked_at"`
	Endpoints []endpointHealth `json:"endpoints"`
}

func runDoctor(cmd *cobra.Command, args []string) error {
	client := api.NewClient().WithTimeout(5 * time.Second)

	report := doctorReport{OK: true, CheckedAt: time.Now().Format(time.RFC3339)}
	for _, t := range api.ProbeTargets() {
		h := endpointHealth{Name: t.Name, URL: t.URL}
		start := time.Now()
		_, err := client.GetString(cmd.Context(), t.URL)
		h.LatencyMs = time.Since(start).Milliseconds()
		if err != nil {
			h.OK = false
			h.Error = err.Error()
			report.OK = false
		} else {
			h.OK = true
		}
		report.Endpoints = append(report.Endpoints, h)
	}

	if outputFormat != "text" {
		emitJSON(report)
		if !report.OK {
			setExitCode(ExitNetwork)
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
	if !report.OK {
		setExitCode(ExitNetwork)
	}
	return nil
}
