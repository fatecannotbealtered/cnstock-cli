package cmd

import (
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/fatecannotbealtered/cnstock-cli/internal/api"
	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
	"github.com/spf13/cobra"
)

// classifyError maps an API error to the appropriate ErrorCode and exit code.
func classifyError(err error) (output.ErrorCode, int, bool) {
	var validationErr *api.ValidationError
	if errors.As(err, &validationErr) {
		return output.ErrValidation, ExitBadArgs, false
	}
	var notFoundErr *api.NotFoundError
	if errors.As(err, &notFoundErr) {
		return output.ErrNotFound, ExitNotFound, false
	}
	var serverErr *api.ServerError
	if errors.As(err, &serverErr) {
		return output.ErrServer, ExitTransient, true
	}
	var networkErr *api.NetworkError
	if errors.As(err, &networkErr) {
		if isTimeoutError(networkErr.Error()) {
			return output.ErrTimeout, ExitTimeout, true
		}
		return output.ErrNetwork, ExitTransient, true
	}
	if isUsageError(err.Error()) {
		return output.ErrValidation, ExitBadArgs, false
	}
	return output.ErrUnknown, ExitGeneric, false
}

// Exit codes for machine-readable error classification.
const (
	ExitOK              = 0
	ExitGeneric         = 1
	ExitBadArgs         = 2
	ExitNotFound        = 3
	ExitAuth            = 4
	ExitForbidden       = 4
	ExitConfirmRequired = 5
	ExitConflict        = 6
	ExitTransient       = 7
	ExitNetwork         = 7
	ExitRateLimit       = 7
	ExitTimeout         = 8
)

// ErrSilent indicates the error has been printed; cobra should not print again.
var ErrSilent = errors.New("")

// version is injected by goreleaser ldflags.
var version = "dev"

const (
	riskTier            = "T0"
	riskTierDescription = "read-only public market-data queries; update is local lifecycle write only; no credentials; no external writes"
)

// Output control (global flags).
var (
	// outputFormat is the resolved output format: "json" (default), "text", or "raw".
	outputFormat = "json"
	// jsonMode is the deprecated --json flag, kept as an alias for --format json.
	jsonMode bool
	// compactMode emits single-line JSON (lower token count).
	compactMode bool
	// fieldsList restricts JSON data to an ordered subset of top-level fields.
	fieldsList []string
	// quietMode suppresses non-result stdout output.
	quietMode bool
	// dryRunMode previews local lifecycle writes; market-data commands reject it.
	dryRunMode bool
	// confirmToken executes a prior lifecycle dry-run; market-data commands reject it.
	confirmToken string
	// commandStartedAt is used for JSON envelope meta.duration_ms.
	commandStartedAt time.Time
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
	rootCmd.PersistentFlags().StringSliceVar(&fieldsList, "fields", nil, "Restrict JSON data to these top-level fields (ordered, comma-separated)")
	rootCmd.PersistentFlags().BoolVar(&jsonMode, "json", false, "Compatibility alias for --format json")
	rootCmd.PersistentFlags().BoolVar(&quietMode, "quiet", false, "Suppress non-result stdout output")
	rootCmd.PersistentFlags().BoolVar(&dryRunMode, "dry-run", false, "Preview local lifecycle writes such as update without applying them")
	rootCmd.PersistentFlags().StringVar(&confirmToken, "confirm", "", "Execute a prior dry-run confirmation token for local lifecycle writes")
	installUpdateNoticeHelp(rootCmd)

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
		if cmd.CommandPath() != "cnstock-cli update" && (flagChanged(cmd, "dry-run") || flagChanged(cmd, "confirm")) {
			return handleError(api.NewValidationError("market-data commands are read-only; --dry-run and --confirm are only supported by update"))
		}
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
	lastExit = ExitOK
	commandStartedAt = time.Now()
	err := rootCmd.Execute()
	if err != nil {
		if errors.Is(err, ErrSilent) {
			return err
		}
		return handleError(err)
	}
	if lastExit != ExitOK {
		return ErrSilent
	}
	return nil
}

// handleError emits an error in the active output format and sets the exit code.
// In text mode it prints a human-readable line to stderr; otherwise it emits a
// machine-readable JSON error envelope to stderr.
func handleError(err error) error {
	msg := err.Error()
	code, exitCode, retryable := classifyError(err)
	if outputFormat == "text" {
		output.Error(msg)
	} else {
		output.PrintErrorEnvelopeWithDuration(msg, code, retryable, nil, compactMode, commandDuration())
	}
	setExitCode(exitCode)
	return ErrSilent
}

func flagChanged(cmd *cobra.Command, name string) bool {
	if cmd == nil {
		return false
	}
	if f := cmd.Flags().Lookup(name); f != nil && f.Changed {
		return true
	}
	if f := cmd.InheritedFlags().Lookup(name); f != nil && f.Changed {
		return true
	}
	if f := cmd.Root().PersistentFlags().Lookup(name); f != nil && f.Changed {
		return true
	}
	return false
}

func commandDuration() time.Duration {
	if commandStartedAt.IsZero() {
		return 0
	}
	return time.Since(commandStartedAt)
}

func isTimeoutError(msg string) bool {
	msg = strings.ToLower(msg)
	return strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded")
}

func isUsageError(msg string) bool {
	msg = strings.ToLower(msg)
	return strings.Contains(msg, "accepts ") ||
		strings.Contains(msg, "requires ") ||
		strings.Contains(msg, "unknown flag") ||
		strings.Contains(msg, "invalid argument")
}
