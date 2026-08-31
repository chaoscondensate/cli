// Package cli adapts application services to the command-line interface.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/chaoscondensate/forecast-ledger/internal/app"
	"github.com/chaoscondensate/forecast-ledger/internal/presentation"
	urfavecli "github.com/urfave/cli/v3"
)

func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	command := NewCommand(stdin, stdout, stderr)
	err := command.Run(ctx, args)
	if err == nil {
		return 0
	}
	var frameworkExit urfavecli.ExitCoder
	if errors.As(err, &frameworkExit) && frameworkExit.ExitCode() == 3 {
		// urfave reserves exit 3 for an unknown help topic. The Forecast
		// Ledger contract classifies every unknown command/help lookup as usage.
		err = app.NewError(app.CodeUsage, err.Error(), err)
	}
	code := app.ExitCodeOf(err)
	var presented interface{ AlreadyPresented() bool }
	if errors.As(err, &presented) && presented.AlreadyPresented() {
		return code
	}
	if code == 1 {
		var exitCoder urfavecli.ExitCoder
		if errors.As(err, &exitCoder) {
			code = exitCoder.ExitCode()
		}
	}
	presenter := presentation.New(stdout, stderr, presentation.Options{
		JSON: command.Bool("json"), Plain: command.Bool("plain"), Quiet: command.Bool("quiet"), Verbose: command.Bool("verbose"), NoColor: command.Bool("no-color"),
	})
	if writeErr := presenter.Failure(err); writeErr != nil {
		fmt.Fprintf(stderr, "forecast-ledger: output failed\n")
		return 1
	}
	return code
}

type presentedApplicationError struct{ error }

func (presentedApplicationError) AlreadyPresented() bool { return true }
func (e presentedApplicationError) Unwrap() error        { return e.error }
