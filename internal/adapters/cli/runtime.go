package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/chaoscondensate/cli/internal/app"
	urfavecli "github.com/urfave/cli/v3"
	"golang.org/x/term"
)

type Runtime struct {
	NoInput  bool
	Yes      bool
	DryRun   bool
	Timeout  time.Duration
	stdin    io.Reader
	stderr   io.Writer
	inputTTY bool
	errorTTY bool
}

func RuntimeFromCommand(command *urfavecli.Command) Runtime {
	root := command.Root()
	return Runtime{
		NoInput: root.Bool("no-input"), Yes: root.Bool("yes"), DryRun: command.Bool("dry-run"), Timeout: root.Duration("timeout"),
		stdin: root.Reader, stderr: root.ErrWriter, inputTTY: isTerminalReader(root.Reader), errorTTY: terminalWriter(root.ErrWriter),
	}
}

func (r Runtime) Context(parent context.Context) (context.Context, context.CancelFunc) {
	if r.Timeout <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, r.Timeout)
}

// Confirm is for explicit destructive, multi-file, or network boundaries. It
// never prompts in --no-input or non-interactive operation.
func (r Runtime) Confirm(ctx context.Context, prompt string) (bool, error) {
	if err := contextApplicationError(ctx); err != nil {
		return false, err
	}
	if r.Yes {
		return true, nil
	}
	if r.NoInput {
		return false, app.NewError(app.CodeUsage, "confirmation is required; use --yes when --no-input is set", nil)
	}
	if !r.inputTTY || !r.errorTTY {
		return false, app.NewError(app.CodeUsage, "confirmation requires an interactive terminal; use --yes to approve non-interactively", nil)
	}
	if _, err := fmt.Fprintf(r.stderr, "%s [y/N] ", prompt); err != nil {
		return false, app.NewError(app.CodeIO, "confirmation prompt cannot be written", err)
	}
	reader := bufio.NewReader(io.LimitReader(r.stdin, 4096))
	answer, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, app.NewError(app.CodeIO, "confirmation response cannot be read", err)
	}
	if err := contextApplicationError(ctx); err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func contextApplicationError(ctx context.Context) error {
	if ctx == nil || ctx.Err() == nil {
		return nil
	}
	return app.NewError(app.CodeInterrupted, "operation was interrupted", ctx.Err())
}

type descriptorReader interface{ Fd() uintptr }

func isTerminalReader(reader io.Reader) bool {
	file, ok := reader.(descriptorReader)
	return ok && term.IsTerminal(int(file.Fd()))
}

type descriptorWriter interface{ Fd() uintptr }

func terminalWriter(writer io.Writer) bool {
	file, ok := writer.(descriptorWriter)
	return ok && term.IsTerminal(int(file.Fd()))
}
