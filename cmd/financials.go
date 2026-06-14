package cmd

import (
	"fmt"

	"github.com/fatecannotbealtered/cnstock-cli/internal/api"
	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
	"github.com/spf13/cobra"
)

var financialsCmd = &cobra.Command{
	Use:   "financials <symbol>",
	Short: "Company fundamentals (market cap, PE, PB, EPS, ROE, revenue/net-profit)",
	Args:  cobra.ExactArgs(1),
	RunE:  runFinancials,
}

func init() {
	rootCmd.AddCommand(financialsCmd)
}

func runFinancials(cmd *cobra.Command, args []string) error {
	client := api.NewClient()

	if outputFormat == "raw" {
		raw, err := api.FetchFinancialsRaw(cmd.Context(), client, args[0])
		if err != nil {
			return handleError(err)
		}
		output.Raw(raw)
		return nil
	}

	f, err := api.FetchFinancials(cmd.Context(), client, args[0])
	if err != nil {
		return handleError(err)
	}

	if outputFormat != "text" {
		emitJSON(f)
		return nil
	}

	output.Bold(fmt.Sprintf("  %s (%s) -- %s", f.Name, f.Symbol, f.Market))

	headers := []string{"Field", "Value"}
	rows := [][]string{
		{"Price", formatPrice(f.Price)},
		{"Market Cap", formatLargeNum(f.MarketCap)},
		{"Float Market Cap", formatLargeNum(f.FloatMarketCap)},
		{"PE (TTM)", formatPrice(f.PeTTM)},
		{"PE (static)", formatPrice(f.PeStatic)},
		{"PB", formatPrice(f.Pb)},
		{"EPS", formatPrice(f.Eps)},
		{"BVPS", formatPrice(f.Bvps)},
		{"Dividend Yield", formatPct(f.DividendYield)},
		{"ROE", formatPct(f.Roe)},
		{"Revenue", formatLargeNum(f.Revenue)},
		{"Net Profit", formatLargeNum(f.NetProfit)},
		{"Gross Margin", formatPct(f.GrossMargin)},
		{"Total Shares", formatLargeNum(f.TotalShares)},
		{"Float Shares", formatLargeNum(f.FloatShares)},
	}
	output.Table(headers, rows)

	for _, w := range f.Warnings {
		output.Gray("  warning: " + w)
	}
	return nil
}
