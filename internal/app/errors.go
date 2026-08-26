package app

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	CodeUsage                    ErrorCode = "usage"
	CodeInvalidData              ErrorCode = "invalid_data"
	CodeUnsupportedSchemaVersion ErrorCode = "unsupported_schema_version"
	CodeNotFound                 ErrorCode = "not_found"
	CodeConflict                 ErrorCode = "conflict"
	CodeVerification             ErrorCode = "verification"
	CodeIO                       ErrorCode = "io"
	CodeNetwork                  ErrorCode = "network"
	CodeNetworkDisabled          ErrorCode = "network_disabled"
	CodePending                  ErrorCode = "pending"
	CodeIncomplete               ErrorCode = "incomplete"
	CodeUnavailable              ErrorCode = "unavailable"
	CodeInternal                 ErrorCode = "internal"
	CodeInterrupted              ErrorCode = "interrupted"
)

type Error struct {
	Code    ErrorCode      `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
	Cause   error          `json:"-"`
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Cause }

func NewError(code ErrorCode, message string, cause error) *Error {
	if !validErrorCode(code) {
		code = CodeInternal
	}
	if message == "" {
		message = defaultErrorMessage(code)
	}
	return &Error{Code: code, Message: message, Cause: cause}
}

func WithDetails(err *Error, details map[string]any) *Error {
	if err == nil {
		return nil
	}
	clone := *err
	clone.Details = cloneDetails(details)
	return &clone
}

func ErrorCodeOf(err error) ErrorCode {
	var applicationErr *Error
	if errors.As(err, &applicationErr) && validErrorCode(applicationErr.Code) {
		return applicationErr.Code
	}
	return CodeInternal
}

// ExitCodeOf is the stable CLI mapping. Exit code 1 is reserved for unexpected
// internal failures; 130 follows the shell convention for interruption.
func ExitCodeOf(err error) int {
	switch ErrorCodeOf(err) {
	case CodeUsage:
		return 2
	case CodeInvalidData, CodeUnsupportedSchemaVersion:
		return 3
	case CodeNotFound:
		return 4
	case CodeConflict:
		return 5
	case CodeVerification:
		return 6
	case CodeIO:
		return 7
	case CodeNetwork, CodeNetworkDisabled:
		return 8
	case CodePending, CodeIncomplete:
		return 9
	case CodeUnavailable:
		return 10
	case CodeInterrupted:
		return 130
	default:
		return 1
	}
}

func validErrorCode(code ErrorCode) bool {
	switch code {
	case CodeUsage, CodeInvalidData, CodeUnsupportedSchemaVersion, CodeNotFound, CodeConflict, CodeVerification,
		CodeIO, CodeNetwork, CodeNetworkDisabled, CodePending, CodeIncomplete, CodeUnavailable, CodeInternal, CodeInterrupted:
		return true
	default:
		return false
	}
}

func defaultErrorMessage(code ErrorCode) string {
	switch code {
	case CodeUsage:
		return "command input is not valid"
	case CodeInvalidData:
		return "ledger data is not valid"
	case CodeUnsupportedSchemaVersion:
		return "ledger schema version is not supported"
	case CodeNotFound:
		return "requested item was not found"
	case CodeConflict:
		return "operation conflicts with existing state"
	case CodeVerification:
		return "verification failed"
	case CodeIO:
		return "file operation failed"
	case CodeNetwork:
		return "network operation failed"
	case CodeNetworkDisabled:
		return "network access is disabled"
	case CodePending:
		return "operation is still pending"
	case CodeIncomplete:
		return "required verification is incomplete"
	case CodeUnavailable:
		return "operation is not available in this release"
	case CodeInterrupted:
		return "operation was interrupted"
	default:
		return "internal error"
	}
}

func cloneDetails(details map[string]any) map[string]any {
	if len(details) == 0 {
		return nil
	}
	result := make(map[string]any, len(details))
	for key, value := range details {
		result[key] = value
	}
	return result
}

func Wrapf(code ErrorCode, cause error, format string, arguments ...any) *Error {
	return NewError(code, fmt.Sprintf(format, arguments...), cause)
}
