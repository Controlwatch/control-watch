package observe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHorizonFlagsClientReadsAccountFlags(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", request.Method)
		}
		if request.URL.Path != "/accounts/"+testAdminAccount {
			t.Errorf("path = %q, want issuer account path", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
            "last_modified_ledger": 77,
            "flags": {
                "auth_required": true,
                "auth_revocable": false,
                "auth_immutable": true,
                "auth_clawback_enabled": true
            }
        }`))
	}))
	defer server.Close()

	client, err := NewHorizonFlagsClient(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewHorizonFlagsClient returned error: %v", err)
	}
	flags, err := client.ReadClassicFlags(context.Background(), testAdminAccount)
	if err != nil {
		t.Fatalf("ReadClassicFlags returned error: %v", err)
	}
	if !flags.AuthRequired || flags.AuthRevocable || !flags.AuthImmutable || !flags.AuthClawbackEnabled {
		t.Fatalf("flags = %#v", flags)
	}
	if flags.LastModifiedLedger != 77 {
		t.Fatalf("last modified ledger = %d, want 77", flags.LastModifiedLedger)
	}
}

func TestHorizonFlagsClientRejectsInvalidConfigurationAndStatus(t *testing.T) {
	t.Parallel()

	if _, err := NewHorizonFlagsClient("file:///tmp/horizon", http.DefaultClient); err == nil {
		t.Fatal("file-scheme Horizon URL was accepted")
	}
	if _, err := NewHorizonFlagsClient("https://example.com", nil); err == nil {
		t.Fatal("nil HTTP client was accepted")
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "missing", http.StatusNotFound)
	}))
	defer server.Close()
	client, err := NewHorizonFlagsClient(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewHorizonFlagsClient returned error: %v", err)
	}
	_, err = client.ReadClassicFlags(context.Background(), testAdminAccount)
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("error = %v, want HTTP 404", err)
	}
}
