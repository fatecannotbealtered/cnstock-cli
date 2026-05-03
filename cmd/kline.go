package cmd

import (
	"fmt"

	"github.com/fatecannotbealtered/cnstock-cli/internal/api"
	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
	"github.com/spf13/cobra"
)

var klineCmd = &cobra.Command{
	Use:   "kline <symbol>",
	Short: "Historical K-line data",
	Args:  cobra.ExactArgs(1),
	RunE:  runKline,
}

func init() {
	klineCmd.Flags().String("period", "day", "Period: day|week|month")
	klineCmd.Flags().Int("limit", 20, "Number of bars (1-500)")
	klineCmd.Flags().String("adj", "qfq", "Adjustment: qfq=forward, hfq=backward, none=unadjusted")
	rootCmd.AddCommand(klineCmd)
}

func runKline(cmd *cobra.Command, args []string) error {
	client := api.NewClient()

	period, _ := cmd.Flags().GetString("period")
	limit, _ := cmd.Flags().GetInt("limit")
	adj, _ := cmd.Flags().GetString("adj")

	bars, err := api.FetchKline(cmd.Context(), client, args[0], period, limit, adj)
	if err != nil {
		return handleError(err)
	}

	if jsonMode {
		output.PrintJSON(bars)
		return nil
	}

	if len(bars) == 0 {
		output.Info("No K-line data returned.")
		return nil
	}

	headers := []string{"Date", "Open", "Close", "High", "Low", "Volume"}
	var rows [][]string
	for _, b := range bars {
		rows = append(rows, []string{
			b.Date,
			formatPrice(b.Open),
			formatPrice(b.Close),
			formatPrice(b.High),
			formatPrice(b.Low),
			formatLargeNum(b.Volume),
		})
	}

	output.Table(headers, rows)
	output.Gray(fmt.Sprintf("  %d bars", len(bars)))
	return nil
}
