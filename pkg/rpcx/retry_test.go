package rpcx

import (
	"context"
	"errors"
	"net"
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestRetryClientGetHealth(t *testing.T) {
	t.Parallel()

	transportError := &net.DNSError{Err: "connection refused", IsTemporary: true}
	tests := []struct {
		name         string
		errors       []error
		wantAttempts int
		wantDelays   []time.Duration
		wantErr      bool
	}{
		{
			name:         "HTTP 429 retries",
			errors:       []error{httpError(http.StatusTooManyRequests), nil},
			wantAttempts: 2,
			wantDelays:   []time.Duration{defaultInitialBackoff},
		},
		{
			name:         "HTTP 500 retries with exponential backoff",
			errors:       []error{httpError(http.StatusInternalServerError), httpError(http.StatusBadGateway), nil},
			wantAttempts: 3,
			wantDelays:   []time.Duration{defaultInitialBackoff, defaultInitialBackoff * 2},
		},
		{
			name:         "transport error retries",
			errors:       []error{transportError, nil},
			wantAttempts: 2,
			wantDelays:   []time.Duration{defaultInitialBackoff},
		},
		{
			name:         "HTTP 400 does not retry",
			errors:       []error{httpError(http.StatusBadRequest)},
			wantAttempts: 1,
			wantErr:      true,
		},
		{
			name:         "plain error does not retry",
			errors:       []error{errors.New("invalid request")},
			wantAttempts: 1,
			wantErr:      true,
		},
		{
			name: "stops after three retryable failures",
			errors: []error{
				httpError(http.StatusServiceUnavailable),
				httpError(http.StatusServiceUnavailable),
				httpError(http.StatusServiceUnavailable),
			},
			wantAttempts: 3,
			wantDelays:   []time.Duration{defaultInitialBackoff, defaultInitialBackoff * 2},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := &healthSequenceClient{errors: tt.errors}
			retry := NewRetryClient(client, time.Second)
			retry.jitter = func(delay time.Duration) time.Duration { return delay }

			var delays []time.Duration
			retry.sleep = func(_ context.Context, delay time.Duration) error {
				delays = append(delays, delay)
				return nil
			}

			got, err := retry.GetHealth(context.Background())
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetHealth() error = %v, wantErr %v", err, tt.wantErr)
			}
			if client.calls != tt.wantAttempts {
				t.Errorf("attempts = %d, want %d", client.calls, tt.wantAttempts)
			}
			if !reflect.DeepEqual(delays, tt.wantDelays) {
				t.Errorf("delays = %v, want %v", delays, tt.wantDelays)
			}
			if !tt.wantErr && got.Status != "healthy" {
				t.Errorf("GetHealth() status = %q, want healthy", got.Status)
			}
		})
	}
}

func TestRetryClientHonorsParentCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := &healthSequenceClient{errors: []error{httpError(http.StatusServiceUnavailable)}}
	retry := NewRetryClient(client, time.Second)

	_, err := retry.GetHealth(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GetHealth() error = %v, want context canceled", err)
	}
	if client.calls != 0 {
		t.Errorf("attempts = %d, want 0", client.calls)
	}
}

func httpError(status int) error {
	return &HTTPStatusError{Code: status, Status: http.StatusText(status)}
}

type healthSequenceClient struct {
	errors []error
	calls  int
}

func (c *healthSequenceClient) GetHealth(context.Context) (Health, error) {
	index := c.calls
	c.calls++
	if index < len(c.errors) && c.errors[index] != nil {
		return Health{}, c.errors[index]
	}
	return Health{Status: "healthy"}, nil
}

func (*healthSequenceClient) GetLatestLedger(context.Context) (LatestLedger, error) {
	return LatestLedger{}, errors.New("unexpected GetLatestLedger call")
}

func (*healthSequenceClient) GetLedgerEntries(context.Context, []string) (LedgerEntries, error) {
	return LedgerEntries{}, errors.New("unexpected GetLedgerEntries call")
}

func (*healthSequenceClient) SimulateTransaction(context.Context, string) (Simulation, error) {
	return Simulation{}, errors.New("unexpected SimulateTransaction call")
}

func (*healthSequenceClient) GetEvents(context.Context, EventsRequest) (EventsPage, error) {
	return EventsPage{}, errors.New("unexpected GetEvents call")
}

func (*healthSequenceClient) GetTransaction(context.Context, string) (Transaction, error) {
	return Transaction{}, errors.New("unexpected GetTransaction call")
}
