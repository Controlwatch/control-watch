package rpcx

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoSigningOrSubmissionSources(t *testing.T) {
	t.Parallel()

	patterns := map[string]string{
		"transaction submission API": "send" + "Transaction",
		"SDK keypair import":         "go-stellar-sdk/" + "keypair",
		"secret-seed constructor":    "From" + "SecretSeed",
		"signing call":               ".Si" + "gn(",
	}
	err := filepath.WalkDir("../..", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == ".codex-skill-review" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for description, pattern := range patterns {
			if strings.Contains(string(contents), pattern) {
				t.Errorf("%s found in %s", description, path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan source tree: %v", err)
	}
}
