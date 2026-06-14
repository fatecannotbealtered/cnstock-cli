package cmd

import (
	"fmt"

	"github.com/fatecannotbealtered/cnstock-cli/internal/api"
	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
	"github.com/spf13/cobra"
)

var constituentsCmd = &cobra.Command{
	Use:   "constituents <index>",
	Short: "List constituent stocks of an index or board (code/name/weight/change_pct)",
	Args:  cobra.ExactArgs(1),
	RunE:  runConstituents,
}

func init() {
	rootCmd.AddCommand(constituentsCmd)
}

func runConstituents(cmd *cobra.Command, args []string) error {
	client := api.NewClient()

	if outputFormat == "raw" {
		raw, err := api.FetchConstituentsRaw(cmd.Context(), client, args[0])
		if err != nil {
			return handleError(err)
		}
		output.Raw(raw)
		return nil
	}

	members, err := api.FetchConstituents(cmd.Context(), client, args[0])
	if err != nil {
		return handleError(err)
	}

	if outputFormat != "text" {
		emitJSON(members)
		return nil
	}

	if len(members) == 0 {
		output.Info("No constituents returned.")
		return nil
	}

	headers := []string{"#", "Code", "Name", "Price", "Change", "Weight"}
	var rows [][]string
	for i, c := range members {
		weight := "-"
		if c.Weight != nil {
			weight = fmt.Sprintf("%.2f%%", *c.Weight)
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			c.Code,
			c.Name,
			formatPrice(c.Price),
			formatPct(c.ChangePct),
			weight,
		})
	}
	output.Table(headers, rows)
	output.Gray(fmt.Sprintf("  %d constituents", len(members)))
	return nil
}
