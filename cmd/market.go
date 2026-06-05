package cmd

import (
	"github.com/fatecannotbealtered/cnstock-cli/internal/api"
	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
	"github.com/spf13/cobra"
)

var marketCmd = &cobra.Command{
	Use:   "market",
	Short: "Whole-market statistics (advance/decline, limit-up/down, turnover)",
	Args:  cobra.NoArgs,
	RunE:  runMarket,
}

func init() {
	rootCmd.AddCommand(marketCmd)
}

func runMarket(cmd *cobra.Command, args []string) error {
	client := api.NewClient()

	stats, err := api.FetchMarketStats(cmd.Context(), client)
	if err != nil {
		return handleError(err)
	}

	if jsonMode {
		output.PrintJSON(stats)
		return nil
	}

	headers := []string{"Metric", "Value"}
	rows := [][]string{
		{"Advancing", formatInt(stats.Advancing)},
		{"Declining", formatInt(stats.Declining)},
		{"Flat", formatInt(stats.Flat)},
		{"Limit-up", formatInt(stats.LimitUp)},
		{"Limit-down", formatInt(stats.LimitDown)},
		{"Total Amount", formatLargeNum(stats.Amount)},
	}
	output.Table(headers, rows)

	for _, w := range stats.Warnings {
		output.Gray("  warning: " + w)
	}
	return nil
}
