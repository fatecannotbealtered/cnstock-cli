package cmd

import (
	"errors"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/fatecannotbealtered/cnstock-cli/internal/api"
	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
	"github.com/spf13/cobra"
)

// classifyError maps an API error to the appropriate ErrorCode and exit code.
func classifyError(err error) (output.ErrorCode, int) {
	var validationErr *api.ValidationError
	if errors.As(err, &validationErr) {
		return output.ErrValidation, ExitBadArgs
	}
	var notFoundErr *api.NotFoundError
	if errors.As(err, &notFoundErr) {
		return output.ErrNotFound, ExitNotFound
	}
	var serverErr *api.ServerError
	if errors.As(err, &serverErr) {
		return output.ErrServer, ExitNetwork
	}
	var networkErr *api.NetworkError
	if errors.As(err, &networkErr) {
		return output.ErrNetwork, ExitNetwork
	}
	return output.ErrUnknown, ExitGeneric
}

// Exit codes for machine-readable error classification.
const (
	ExitOK        = 0
	ExitGeneric   = 1
	ExitBadArgs   = 2
	ExitAuth      = 3
	ExitNotFound  = 4
	ExitForbidden = 5
	ExitRateLimit = 6
	ExitNetwork   = 7
)

// ErrSilent indicates the error has been printed; cobra should not print again.
var ErrSilent = errors.New("")

// version is injected by goreleaser ldflags.
var version = "dev"

// Output control (global flags).
var (
	// outputFormat is the resolved output format: "json" (default), "text", or "raw".
	outputFormat = "json"
	// jsonMode is the deprecated --json flag, kept as an alias for --format json.
	jsonMode bool
	// compactMode emits single-line JSON (lower token count).
	compactMode bool
	// fieldsList restricts JSON output to an ordered subset of top-level fields.
	fieldsList []string
	// quietMode suppresses non-result stdout output.
	quietMode bool
)

// validFormats enumerates accepted --format values.
var validFormats = map[string]struct{}{
	"json": {},
	"text": {},
	"raw":  {},
}

// lastExit tracks the exit code for the current command execution.
var lastExit int

// LastExitCode returns the exit code from the last command execution.
func LastExitCode() int { return lastExit }

// setExitCode sets the exit code (only increases severity, never decreases).
func setExitCode(code int) {
	if code > lastExit {
		lastExit = code
	}
}

var rootCmd = &cobra.Command{
	Use:           "cnstock-cli",
	Short:         "Real-time quotes, K-line, intraday minutes, and stock search via Tencent Finance",
	Version:       version,
	SilenceErrors: true,
	SilenceUsage:  true,
	Long: fmt.Sprintf("\n  %s\n  %s",
		output.FormatCyanBold("cnstock-cli"),
		output.FormatGray("Query financial data through Tencent Finance web endpoints")),
}

func init() {
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
			version = info.Main.Version
		}
	}
	rootCmd.Version = version
	api.UserAgent = fmt.Sprintf("cnstock-cli/%s (+https://github.com/fatecannotbealtered/cnstock-cli)", version)
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.PersistentFlags().StringVar(&outputFormat, "format", "json", "Output format: json|text|raw")
	rootCmd.PersistentFlags().BoolVar(&compactMode, "compact", false, "Emit single-line JSON (lower token count)")
	rootCmd.PersistentFlags().StringSliceVar(&fieldsList, "fields", nil, "Restrict JSON output to these top-level fields (ordered, comma-separated)")
	rootCmd.PersistentFlags().BoolVar(&jsonMode, "json", false, "Deprecated alias for --format json")
	rootCmd.PersistentFlags().BoolVar(&quietMode, "quiet", false, "Suppress non-result stdout output")
	_ = rootCmd.PersistentFlags().MarkDeprecated("json", "use --format json (json is the default)")

	// Resolve and validate output flags before any command runs.
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// --json is a compatibility alias that forces JSON output.
		if cmd.Flags().Changed("json") {
			outputFormat = "json"
		}
		outputFormat = strings.ToLower(strings.TrimSpace(outputFormat))
		if _, ok := validFormats[outputFormat]; !ok {
			return handleError(api.NewValidationError("format only supports json, text, raw"))
		}
		output.Quiet = quietMode
		return nil
	}

	cobra.OnInitialize(func() {
		output.Quiet = quietMode
	})

	// Intercept Cobra flag/arg errors so they get proper JSON output in --json mode.
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return handleError(err)
	})
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// handleError emits an error in the active output format and sets the exit code.
// In text mode it prints a human-readable line to stderr; otherwise it emits a
// machine-readable JSON error envelope to stderr.
func handleError(err error) error {
	msg := err.Error()
	code, exitCode := classifyError(err)
	if outputFormat == "text" {
		output.Error(msg)
	} else {
		output.PrintErrorJSONWithCode(msg, 0, code)
	}
	setExitCode(exitCode)
	return ErrSilent
}
