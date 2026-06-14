package cmd

import (
	"fmt"

	"github.com/fatecannotbealtered/cnstock-cli/internal/api"
	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
	"github.com/spf13/cobra"
)

var klineCmd = &cobra.Command{
	Use:   "kline <symbols>",
	Short: "Historical K-line data (supports batch, comma-separated)",
	Args:  cobra.ExactArgs(1),
	RunE:  runKline,
}

func init() {
	klineCmd.Flags().String("period", "day", "Period: day|week|month")
	klineCmd.Flags().Int("limit", 20, "Number of bars (1-500)")
	klineCmd.Flags().String("adj", "qfq", "Adjustment: qfq=forward, hfq=backward, none=unadjusted")
	klineCmd.Flags().String("from", "", "Start date YYYY-MM-DD (date-bounded range; limit still caps the count)")
	klineCmd.Flags().String("to", "", "End date YYYY-MM-DD (date-bounded range)")
	klineCmd.Flags().Bool("continue-on-error", true, "Keep going after a per-symbol failure (best-effort); set false to stop at the first failure")
	rootCmd.AddCommand(klineCmd)
}

func runKline(cmd *cobra.Command, args []string) error {
	client := api.NewClient()

	period, _ := cmd.Flags().GetString("period")
	limit, _ := cmd.Flags().GetInt("limit")
	adj, _ := cmd.Flags().GetString("adj")
	from, _ := cmd.Flags().GetString("from")
	to, _ := cmd.Flags().GetString("to")
	continueOnError, _ := cmd.Flags().GetBool("continue-on-error")

	if outputFormat == "raw" {
		raw, err := api.FetchKlineRangeRaw(cmd.Context(), client, args[0], period, limit, adj, from, to)
		if err != nil {
			return handleError(err)
		}
		output.Raw(raw)
		return nil
	}

	result, err := api.FetchKlineBatch(cmd.Context(), client, args[0], period, limit, adj, from, to, continueOnError)
	if err != nil {
		return handleError(err)
	}

	if outputFormat != "text" {
		emitJSON(result)
		return nil
	}

	printKlineBatch(result)
	return nil
}

func printKlineBatch(result *api.BatchResult[[]api.KlineBar]) {
	headers := []string{"Date", "Open", "Close", "High", "Low", "Volume"}
	for i, item := range result.Items {
		if i > 0 {
			fmt.Println()
		}
		if !item.OK {
			output.Error(fmt.Sprintf("%s: %s (%s)", item.Target, item.Error.Message, item.Error.Code))
			continue
		}
		output.Bold("  " + item.Target)
		var rows [][]string
		for _, b := range item.Data {
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
		output.Gray(fmt.Sprintf("  %d bars", len(item.Data)))
	}
	output.Gray(fmt.Sprintf("  %d total, %d ok, %d failed", result.Summary.Total, result.Summary.Succeeded, result.Summary.Failed))
}
