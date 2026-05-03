package cmd

import (
	"fmt"

	"github.com/fatecannotbealtered/cnstock-cli/internal/api"
	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <keyword>",
	Short: "Search stocks by name (Chinese/pinyin/English)",
	Args:  cobra.ExactArgs(1),
	RunE:  runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	client := api.NewClient()

	results, err := api.FetchSearch(cmd.Context(), client, args[0])
	if err != nil {
		return handleError(err)
	}

	if jsonMode {
		output.PrintJSON(results)
		return nil
	}

	if len(results) == 0 {
		output.Info("No results found.")
		return nil
	}

	headers := []string{"Symbol", "Name", "Market", "Pinyin"}
	var rows [][]string
	for _, r := range results {
		rows = append(rows, []string{r.Symbol, r.Name, r.Market, r.Pinyin})
	}

	output.Table(headers, rows)
	output.Gray(fmt.Sprintf("  %d results", len(results)))
	return nil
}
