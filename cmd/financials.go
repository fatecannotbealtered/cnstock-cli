package cmd

import (
	"fmt"

	"github.com/fatecannotbealtered/cnstock-cli/internal/api"
	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
	"github.com/spf13/cobra"
)

var financialsCmd = &cobra.Command{
	Use:   "financials <symbols>",
	Short: "Company fundamentals (market cap, PE, PB, EPS, ROE, revenue/net-profit); supports batch, comma-separated",
	Args:  cobra.ExactArgs(1),
	RunE:  runFinancials,
}

func init() {
	financialsCmd.Flags().Bool("continue-on-error", true, "Keep going after a per-symbol failure (best-effort); set false to stop at the first failure")
	rootCmd.AddCommand(financialsCmd)
}

func runFinancials(cmd *cobra.Command, args []string) error {
	client := api.NewClient()
	continueOnError, _ := cmd.Flags().GetBool("continue-on-error")

	if outputFormat == "raw" {
		raw, err := api.FetchFinancialsRaw(cmd.Context(), client, args[0])
		if err != nil {
			return handleError(err)
		}
		output.Raw(raw)
		return nil
	}

	result, err := api.FetchFinancialsBatch(cmd.Context(), client, args[0], continueOnError)
	if err != nil {
		return handleError(err)
	}

	if outputFormat != "text" {
		emitJSON(result)
		return nil
	}

	printFinancialsBatch(result)
	return nil
}

func printFinancialsBatch(result *api.BatchResult[*api.Financials]) {
	for i, item := range result.Items {
		if i > 0 {
			fmt.Println()
		}
		if !item.OK {
			output.Error(fmt.Sprintf("%s: %s (%s)", item.Target, item.Error.Message, item.Error.Code))
			continue
		}
		f := item.Data
		output.Bold(fmt.Sprintf("  %s (%s) -- %s", f.Name, f.Symbol, f.Market))
		headers := []string{"Field", "Value"}
		rows := [][]string{
			{"Price", formatPrice(f.Price)},
			{"Market Cap", formatLargeNum(f.MarketCap)},
			{"Float Market Cap", formatLargeNum(f.FloatMarketCap)},
			{"PE", formatPrice(f.PeRatio)},
			{"PB", formatPrice(f.Pb)},
			{"Turnover Rate", formatPct(f.TurnoverRate)},
			{"Amount", formatLargeNum(f.Amount)},
		}
		output.Table(headers, rows)
		for _, w := range f.Warnings {
			output.Gray("  warning: " + w)
		}
	}
	output.Gray(fmt.Sprintf("  %d total, %d ok, %d failed", result.Summary.Total, result.Summary.Succeeded, result.Summary.Failed))
}
