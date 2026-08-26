// Package cli adapts application services to the command-line interface.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/presentation"
	urfavecli "github.com/urfave/cli/v3"
)

func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	command := NewCommand(stdin, stdout, stderr)
	err := command.Run(ctx, args)
	if err == nil {
		return 0
	}
	code := app.ExitCodeOf(err)
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
