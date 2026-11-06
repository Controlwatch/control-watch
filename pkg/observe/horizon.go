package observe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ClassicAccountFlags is the Horizon account flag subset used by ControlWatch.
type ClassicAccountFlags struct {
	AuthRequired        bool
	AuthRevocable       bool
	AuthImmutable       bool
	AuthClawbackEnabled bool
	LastModifiedLedger  uint32
}

// ClassicFlagsReader reads authorization flags for a classic issuer account.
type ClassicFlagsReader interface {
	ReadClassicFlags(ctx context.Context, accountID string) (ClassicAccountFlags, error)
}

// HorizonFlagsClient reads the account flags object from Horizon.
type HorizonFlagsClient struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// NewHorizonFlagsClient validates a Horizon endpoint and constructs a flags
// reader using the supplied HTTP client.
func NewHorizonFlagsClient(baseURL string, httpClient *http.Client) (*HorizonFlagsClient, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse Horizon URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("parse Horizon URL: scheme must be http or https")
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("parse Horizon URL: host is required")
	}
	if httpClient == nil {
		return nil, fmt.Errorf("construct Horizon flags client: HTTP client is required")
	}
	return &HorizonFlagsClient{baseURL: parsed, httpClient: httpClient}, nil
}

// ReadClassicFlags fetches one Horizon account and returns only its flags and
// last-modified ledger.
func (client *HorizonFlagsClient) ReadClassicFlags(
	ctx context.Context,
	accountID string,
) (ClassicAccountFlags, error) {
	endpoint, err := url.JoinPath(client.baseURL.String(), "accounts", accountID)
	if err != nil {
		return ClassicAccountFlags{}, fmt.Errorf("build Horizon account URL: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ClassicAccountFlags{}, fmt.Errorf("build Horizon account request: %w", err)
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		return ClassicAccountFlags{}, fmt.Errorf("read Horizon account: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return ClassicAccountFlags{}, fmt.Errorf(
			"read Horizon account: HTTP %d %s",
			response.StatusCode,
			strings.TrimSpace(response.Status),
		)
	}

	var account struct {
		LastModifiedLedger uint32 `json:"last_modified_ledger"`
		Flags              struct {
			AuthRequired        bool `json:"auth_required"`
			AuthRevocable       bool `json:"auth_revocable"`
			AuthImmutable       bool `json:"auth_immutable"`
			AuthClawbackEnabled bool `json:"auth_clawback_enabled"`
		} `json:"flags"`
	}
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&account); err != nil {
		return ClassicAccountFlags{}, fmt.Errorf("decode Horizon account: %w", err)
	}
	return ClassicAccountFlags{
		AuthRequired:        account.Flags.AuthRequired,
		AuthRevocable:       account.Flags.AuthRevocable,
		AuthImmutable:       account.Flags.AuthImmutable,
		AuthClawbackEnabled: account.Flags.AuthClawbackEnabled,
		LastModifiedLedger:  account.LastModifiedLedger,
	}, nil
}
