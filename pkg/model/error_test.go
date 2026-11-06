package model

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorWrapping(t *testing.T) {
	t.Parallel()

	cause := errors.New("connection refused")
	err := NewError(CodeRPCUnreachable, cause)

	if got, want := err.Error(), "RPC_UNREACHABLE: connection refused"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, cause) {
		t.Error("coded error does not unwrap to its cause")
	}

	wrapped := fmt.Errorf("inspect token: %w", err)
	code, ok := CodeOf(wrapped)
	if !ok {
		t.Fatal("CodeOf() did not find coded error")
	}
	if code != CodeRPCUnreachable {
		t.Errorf("CodeOf() = %q, want %q", code, CodeRPCUnreachable)
	}
}

func TestNewErrorWithNilCause(t *testing.T) {
	t.Parallel()

	err := NewError(CodeUsage, nil)
	if err.Unwrap() == nil {
		t.Fatal("NewError() left nil cause")
	}
	if got, want := err.Error(), "USAGE: unspecified error"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestCodeOfPlainError(t *testing.T) {
	t.Parallel()

	if code, ok := CodeOf(errors.New("plain")); ok || code != "" {
		t.Errorf("CodeOf() = (%q, %v), want empty false", code, ok)
	}
}
