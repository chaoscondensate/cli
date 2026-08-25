package app

import (
	"errors"
	"testing"
)

func TestStableExitCodeMapping(t *testing.T) {
	tests := map[ErrorCode]int{
		CodeUsage: 2, CodeInvalidData: 3, CodeNotFound: 4, CodeConflict: 5,
		CodeVerification: 6, CodeIO: 7, CodeNetwork: 8, CodePending: 9,
		CodeInternal: 1, CodeInterrupted: 130,
	}
	for code, expected := range tests {
		t.Run(string(code), func(t *testing.T) {
			wrapped := errors.Join(errors.New("context"), NewError(code, "safe message", errors.New("cause")))
			if got := ErrorCodeOf(wrapped); got != code {
				t.Fatalf("code = %q, want %q", got, code)
			}
			if got := ExitCodeOf(wrapped); got != expected {
				t.Fatalf("exit = %d, want %d", got, expected)
			}
		})
	}
}

func TestUnknownErrorsMapToInternalWithoutCauseDisclosure(t *testing.T) {
	err := NewError("made_up", "", errors.New("secret cause"))
	if err.Code != CodeInternal || err.Message != "internal error" || ExitCodeOf(errors.New("raw")) != 1 {
		t.Fatalf("unexpected fallback: %#v", err)
	}
	if err.Error() == err.Cause.Error() {
		t.Fatal("public error string disclosed its cause")
	}
}

func TestWithDetailsClonesInput(t *testing.T) {
	details := map[string]any{"question_id": "q-one"}
	err := WithDetails(NewError(CodeNotFound, "question not found", nil), details)
	details["question_id"] = "changed"
	if err.Details["question_id"] != "q-one" {
		t.Fatal("details were not cloned")
	}
}
