package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/chaoscondensate/cli/internal/app"
)

func TestRuntimeConfirmationRules(t *testing.T) {
	var stderr bytes.Buffer
	approved := Runtime{Yes: true, NoInput: true}
	if confirmed, err := approved.Confirm(context.Background(), "Continue?"); err != nil || !confirmed {
		t.Fatalf("explicit approval confirmation=%v err=%v", confirmed, err)
	}
	runtime := Runtime{NoInput: true, stdin: strings.NewReader("yes\n"), stderr: &stderr, inputTTY: true, errorTTY: true}
	if _, err := runtime.Confirm(context.Background(), "Continue?"); app.ErrorCodeOf(err) != app.CodeUsage {
		t.Fatalf("--no-input confirmation: %v", err)
	}
	runtime.NoInput = false
	runtime.inputTTY = false
	if _, err := runtime.Confirm(context.Background(), "Continue?"); app.ErrorCodeOf(err) != app.CodeUsage {
		t.Fatalf("non-TTY confirmation: %v", err)
	}
	runtime.inputTTY = true
	confirmed, err := runtime.Confirm(context.Background(), "Continue?")
	if err != nil || !confirmed || stderr.String() != "Continue? [y/N] " {
		t.Fatalf("interactive confirmation=%v stderr=%q err=%v", confirmed, stderr.String(), err)
	}
}

func TestRuntimeTimeoutAndCancellation(t *testing.T) {
	runtime := Runtime{Timeout: 10 * time.Millisecond}
	ctx, cancel := runtime.Context(context.Background())
	defer cancel()
	select {
	case <-ctx.Done():
		if ctx.Err() != context.DeadlineExceeded {
			t.Fatalf("context error = %v", ctx.Err())
		}
	case <-time.After(time.Second):
		t.Fatal("runtime timeout did not cancel context")
	}
	canceled, cancelNow := context.WithCancel(context.Background())
	cancelNow()
	if _, err := runtime.Confirm(canceled, "Continue?"); app.ErrorCodeOf(err) != app.CodeInterrupted {
		t.Fatalf("canceled confirmation: %v", err)
	}
}

func TestPresentedApplicationErrorPreservesExitCategory(t *testing.T) {
	for _, code := range []app.ErrorCode{app.CodePending, app.CodeIncomplete, app.CodeVerification} {
		err := presentedApplicationError{app.NewError(code, "report already presented", nil)}
		if got, want := app.ExitCodeOf(err), app.ExitCodeOf(app.NewError(code, "", nil)); got != want {
			t.Fatalf("code %s mapped to %d, want %d", code, got, want)
		}
	}
}
