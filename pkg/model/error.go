package model

import (
	"errors"
	"fmt"
)

// ErrorCode is a stable machine-readable ControlWatch error code.
type ErrorCode string

const (
	// CodeUsage indicates invalid command usage.
	CodeUsage ErrorCode = "USAGE"
	// CodeConfigMissingRPC indicates that the selected network requires an RPC URL.
	CodeConfigMissingRPC ErrorCode = "CONFIG_MISSING_RPC"
	// CodeInvalidContractID indicates a malformed contract strkey.
	CodeInvalidContractID ErrorCode = "INVALID_CONTRACT_ID"
	// CodeRPCUnreachable indicates a transport failure reaching Stellar RPC.
	CodeRPCUnreachable ErrorCode = "RPC_UNREACHABLE"
	// CodeRPCRateLimited indicates an RPC rate-limit response.
	CodeRPCRateLimited ErrorCode = "RPC_RATE_LIMITED"
	// CodeContractNotFound indicates that the requested contract does not exist.
	CodeContractNotFound ErrorCode = "CONTRACT_NOT_FOUND"
	// CodeSnapshotNotFound indicates that a requested snapshot does not exist.
	CodeSnapshotNotFound ErrorCode = "SNAPSHOT_NOT_FOUND"
	// CodeSnapshotCorrupt indicates failed snapshot integrity verification.
	CodeSnapshotCorrupt ErrorCode = "SNAPSHOT_CORRUPT"
	// CodeSchemaUnsupported indicates an unsupported persisted schema version.
	CodeSchemaUnsupported ErrorCode = "SCHEMA_UNSUPPORTED"
	// CodeScopeMismatch indicates that snapshots describe different targets.
	CodeScopeMismatch ErrorCode = "SCOPE_MISMATCH"
	// CodeComparisonImpossible indicates that required values cannot be compared.
	CodeComparisonImpossible ErrorCode = "COMPARISON_IMPOSSIBLE"
)

// Error associates a stable code with an underlying error.
type Error struct {
	Code ErrorCode
	Err  error
}

// NewError constructs a coded error. A nil cause is replaced with a generic
// description so Error and Unwrap remain safe to call.
func NewError(code ErrorCode, cause error) *Error {
	if cause == nil {
		cause = errors.New("unspecified error")
	}
	return &Error{Code: code, Err: cause}
}

// Error returns the stable code followed by the underlying error message.
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Err == nil {
		return string(e.Code)
	}
	return fmt.Sprintf("%s: %v", e.Code, e.Err)
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// CodeOf returns the first ControlWatch error code in an error chain.
func CodeOf(err error) (ErrorCode, bool) {
	var coded *Error
	if !errors.As(err, &coded) {
		return "", false
	}
	return coded.Code, true
}
