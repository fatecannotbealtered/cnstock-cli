package cmd

import (
	"fmt"

	"github.com/fatecannotbealtered/cnstock-cli/internal/api"
	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
	"github.com/spf13/cobra"
)

var minuteCmd = &cobra.Command{
	Use:   "minute <symbols>",
	Short: "Intraday minute-level data",
	Args:  cobra.ExactArgs(1),
	RunE:  runMinute,
}

func init() {
	rootCmd.AddCommand(minuteCmd)
}

func runMinute(cmd *cobra.Command, args []string) error {
	client := api.NewClient()

	// minute uses the plural --symbols input convention for cross-command
	// consistency, but multi-symbol fetch is deferred until the upstream's
	// multi-code support is confirmed. Reject >1 symbol honestly rather than
	// silently fetching only the first.
	symbol, err := api.SingleSymbol(args[0])
	if err != nil {
		return handleError(err)
	}

	if outputFormat == "raw" {
		raw, err := api.FetchMinuteRaw(cmd.Context(), client, symbol)
		if err != nil {
			return handleError(err)
		}
		output.Raw(raw)
		return nil
	}

	ticks, err := api.FetchMinute(cmd.Context(), client, symbol)
	if err != nil {
		return handleError(err)
	}

	if outputFormat != "text" {
		emitJSON(ticks)
		return nil
	}

	if len(ticks) == 0 {
		output.Info("No minute data returned.")
		return nil
	}

	headers := []string{"Time", "Price", "Volume", "Amount"}
	var rows [][]string
	for _, t := range ticks {
		rows = append(rows, []string{
			t.Time,
			formatPrice(t.Price),
			formatLargeNum(t.Volume),
			formatLargeNum(t.Amount),
		})
	}

	output.Table(headers, rows)
	output.Gray(fmt.Sprintf("  %d ticks", len(ticks)))
	return nil
}
