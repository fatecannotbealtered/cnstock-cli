package cmd

import (
	"fmt"

	"github.com/fatecannotbealtered/cnstock-cli/internal/api"
	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
	"github.com/spf13/cobra"
)

var sectorsCmd = &cobra.Command{
	Use:   "sectors",
	Short: "Sector/industry performance ranking",
	Args:  cobra.NoArgs,
	RunE:  runSectors,
}

func init() {
	sectorsCmd.Flags().String("board", "hy", "Board type: hy=industry, gn=concept, dy=region")
	sectorsCmd.Flags().Int("top", 10, "Number of sectors to return (1-50)")
	sectorsCmd.Flags().String("direction", "up", "Ranking direction: up=top gainers, down=top losers")
	rootCmd.AddCommand(sectorsCmd)
}

func runSectors(cmd *cobra.Command, args []string) error {
	client := api.NewClient()

	board, _ := cmd.Flags().GetString("board")
	top, _ := cmd.Flags().GetInt("top")
	direction, _ := cmd.Flags().GetString("direction")

	sectors, err := api.FetchSectors(cmd.Context(), client, board, direction, top)
	if err != nil {
		return handleError(err)
	}

	if jsonMode {
		output.PrintJSON(sectors)
		return nil
	}

	if len(sectors) == 0 {
		output.Info("No sector data returned.")
		return nil
	}

	headers := []string{"#", "Sector", "Change", "Index", "Leading", "Turnover"}
	var rows [][]string
	for i, s := range sectors {
		leading := "-"
		if s.LeadingStock != nil {
			leading = fmt.Sprintf("%s %s", s.LeadingStock.Name, formatPct(s.LeadingStock.ChangePct))
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			s.Name,
			formatPct(s.ChangePct),
			formatPrice(s.Price),
			leading,
			formatLargeNum(s.Turnover),
		})
	}

	output.Table(headers, rows)
	output.Gray(fmt.Sprintf("  %d sectors", len(sectors)))
	return nil
}
