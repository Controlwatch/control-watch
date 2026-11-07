package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	t.Parallel()

	app := testApp()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := app.Run(context.Background(), []string{"version"}, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0; stderr = %s", exit, stderr.String())
	}
	for _, expected := range []string{
		"controlwatch test-version",
		"schema 1",
		"go go",
		"stellar-go-sdk v0.8.0",
	} {
		if !strings.Contains(stdout.String(), expected) {
			t.Fatalf("output = %q, want %q", stdout.String(), expected)
		}
	}
}

func TestVersionRejectsArguments(t *testing.T) {
	t.Parallel()

	app := testApp()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if exit := app.Run(context.Background(), []string{"version", "extra"}, &stdout, &stderr); exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
}
