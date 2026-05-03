package cmd

import (
	"errors"
	"fmt"
	"runtime/debug"

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

// jsonMode is the global --json flag.
var jsonMode bool

// quietMode is the global --quiet flag.
var quietMode bool

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

	rootCmd.PersistentFlags().BoolVar(&jsonMode, "json", false, "Output result as JSON")
	rootCmd.PersistentFlags().BoolVar(&quietMode, "quiet", false, "Suppress non-JSON stdout output")

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

// handleError handles errors with JSON mode support.
func handleError(err error) error {
	msg := err.Error()
	code, exitCode := classifyError(err)
	if jsonMode {
		output.PrintErrorJSONWithCode(msg, 0, code)
	} else {
		output.Error(msg)
	}
	setExitCode(exitCode)
	return ErrSilent
}
