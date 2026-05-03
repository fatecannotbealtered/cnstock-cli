package cmd

import (
	"fmt"

	"github.com/fatecannotbealtered/cnstock-cli/internal/api"
	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
	"github.com/spf13/cobra"
)

var quoteCmd = &cobra.Command{
	Use:   "quote <symbols>",
	Short: "Real-time quotes (supports batch, comma-separated)",
	Args:  cobra.ExactArgs(1),
	RunE:  runQuote,
}

func init() {
	rootCmd.AddCommand(quoteCmd)
}

func runQuote(cmd *cobra.Command, args []string) error {
	client := api.NewClient()
	quotes, err := api.FetchQuote(cmd.Context(), client, args[0])
	if err != nil {
		return handleError(err)
	}

	if jsonMode {
		output.PrintJSON(quotes)
		return nil
	}

	if len(quotes) == 0 {
		output.Info("No data returned.")
		return nil
	}

	for i, q := range quotes {
		if i > 0 {
			fmt.Println()
		}
		printQuoteTable(q)
	}
	return nil
}

func printQuoteTable(q api.Quote) {
	if q.Error != "" {
		output.Error(fmt.Sprintf("%s: %s", q.Symbol, q.Error))
		return
	}

	output.Bold(fmt.Sprintf("  %s (%s) -- %s", q.Name, q.Symbol, q.Market))

	headers := []string{"Field", "Value"}
	var rows [][]string

	rows = appendRow(rows, "Price", formatPrice(q.Price))
	if q.Change != nil && q.ChangePct != nil {
		rows = appendRow(rows, "Change", fmt.Sprintf("%s (%.2f%%)", output.ChangeColor(*q.Change), *q.ChangePct))
	}
	rows = appendRow(rows, "Open", formatPrice(q.Open))
	rows = appendRow(rows, "High", formatPrice(q.High))
	rows = appendRow(rows, "Low", formatPrice(q.Low))
	rows = appendRow(rows, "Prev Close", formatPrice(q.PrevClose))
	rows = appendRow(rows, "Volume", formatLargeNum(q.Volume))
	rows = appendRow(rows, "Amount", formatLargeNum(q.Amount))
	if q.Turnover != nil {
		rows = appendRow(rows, "Turnover", fmt.Sprintf("%.2f%%", *q.Turnover))
	}
	if q.PeRatio != nil {
		rows = appendRow(rows, "PE Ratio", fmt.Sprintf("%.2f", *q.PeRatio))
	}
	if q.Currency != "" {
		rows = appendRow(rows, "Currency", q.Currency)
	}
	if q.Time != "" {
		rows = appendRow(rows, "Time", q.Time)
	}

	output.Table(headers, rows)

	if len(q.Bid) > 0 {
		fmt.Println()
		output.Gray("  Bid (5 levels):")
		printDepthTable(q.Bid)
	}
	if len(q.Ask) > 0 {
		fmt.Println()
		output.Gray("  Ask (5 levels):")
		printDepthTable(q.Ask)
	}

	if len(q.Warnings) > 0 {
		for _, w := range q.Warnings {
			output.Warn(w)
		}
	}
}

func printDepthTable(levels []api.DepthLevel) {
	headers := []string{"Price", "Volume"}
	var rows [][]string
	for _, l := range levels {
		rows = append(rows, []string{formatPrice(l.Price), formatLargeNum(l.Vol)})
	}
	output.Table(headers, rows)
}

func appendRow(rows [][]string, key, value string) [][]string {
	return append(rows, []string{key, value})
}
