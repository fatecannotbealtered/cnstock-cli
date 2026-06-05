package cmd

import "github.com/fatecannotbealtered/cnstock-cli/internal/output"

// emitJSON renders v as JSON to stdout, honoring --fields and --compact.
func emitJSON(v any) {
	output.RenderJSON(v, fieldsList, compactMode)
}
