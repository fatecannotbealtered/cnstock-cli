package cmd

import (
	"fmt"

	"github.com/fatecannotbealtered/cnstock-cli/internal/api"
	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
	"github.com/spf13/cobra"
)

var minuteCmd = &cobra.Command{
	Use:   "minute <symbol>",
	Short: "Intraday minute-level data",
	Args:  cobra.ExactArgs(1),
	RunE:  runMinute,
}

func init() {
	rootCmd.AddCommand(minuteCmd)
}

func runMinute(cmd *cobra.Command, args []string) error {
	client := api.NewClient()

	ticks, err := api.FetchMinute(cmd.Context(), client, args[0])
	if err != nil {
		return handleError(err)
	}

	if jsonMode {
		output.PrintJSON(ticks)
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
