package cmd

import (
	"fmt"

	"github.com/fatecannotbealtered/cnstock-cli/internal/api"
	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
	"github.com/spf13/cobra"
)

var moneyflowCmd = &cobra.Command{
	Use:   "moneyflow <symbol>",
	Short: "Main-capital / north-bound money flow (主力净流入 / 北向)",
	Args:  cobra.ExactArgs(1),
	RunE:  runMoneyFlow,
}

func init() {
	rootCmd.AddCommand(moneyflowCmd)
}

func runMoneyFlow(cmd *cobra.Command, args []string) error {
	client := api.NewClient()

	if outputFormat == "raw" {
		raw, err := api.FetchMoneyFlowRaw(cmd.Context(), client, args[0])
		if err != nil {
			return handleError(err)
		}
		output.Raw(raw)
		return nil
	}

	mf, err := api.FetchMoneyFlow(cmd.Context(), client, args[0])
	if err != nil {
		return handleError(err)
	}

	if outputFormat != "text" {
		emitJSON(mf)
		return nil
	}

	output.Bold(fmt.Sprintf("  %s (%s) -- %s", mf.Name, mf.Symbol, mf.Market))

	headers := []string{"Field", "Value"}
	rows := [][]string{
		{"Main Inflow", formatLargeNum(mf.MainInflow)},
		{"Main Inflow %", formatPct(mf.MainInflowPct)},
		{"Super-large Inflow", formatLargeNum(mf.SuperInflow)},
		{"Large Inflow", formatLargeNum(mf.LargeInflow)},
		{"Medium Inflow", formatLargeNum(mf.MediumInflow)},
		{"Small Inflow", formatLargeNum(mf.SmallInflow)},
		{"Northbound Flow", formatLargeNum(mf.NorthboundFlow)},
	}
	output.Table(headers, rows)

	for _, w := range mf.Warnings {
		output.Gray("  warning: " + w)
	}
	return nil
}
