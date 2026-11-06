package rpcx

import (
	"context"
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"time"
)

const (
	defaultAttempts       = 3
	defaultTimeout        = 15 * time.Second
	defaultInitialBackoff = 200 * time.Millisecond
	defaultMaxBackoff     = 2 * time.Second
)

// HTTPStatusError records a non-success HTTP status returned by the RPC
// transport.
type HTTPStatusError struct {
	Code   int
	Status string
}

// Error returns the HTTP status description.
func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("unexpected HTTP status %s", e.Status)
}

// StatusCode returns the numeric HTTP status.
func (e *HTTPStatusError) StatusCode() int {
	return e.Code
}

// RetryClient adds per-attempt timeouts and bounded retries to a Client.
type RetryClient struct {
	client         Client
	timeout        time.Duration
	maxAttempts    int
	initialBackoff time.Duration
	maxBackoff     time.Duration
	sleep          func(context.Context, time.Duration) error
	jitter         func(time.Duration) time.Duration
}

var _ Client = (*RetryClient)(nil)

// NewRetryClient wraps client with three attempts, exponential backoff with
// jitter, and the supplied per-attempt timeout. A non-positive timeout uses
// the 15-second default.
func NewRetryClient(client Client, timeout time.Duration) *RetryClient {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &RetryClient{
		client:         client,
		timeout:        timeout,
		maxAttempts:    defaultAttempts,
		initialBackoff: defaultInitialBackoff,
		maxBackoff:     defaultMaxBackoff,
		sleep:          sleepContext,
		jitter:         jitterDuration,
	}
}

// GetHealth calls GetHealth with retry policy.
func (r *RetryClient) GetHealth(ctx context.Context) (Health, error) {
	return retryCall(r, ctx, r.client.GetHealth)
}

// GetLatestLedger calls GetLatestLedger with retry policy.
func (r *RetryClient) GetLatestLedger(ctx context.Context) (LatestLedger, error) {
	return retryCall(r, ctx, r.client.GetLatestLedger)
}

// GetLedgerEntries calls GetLedgerEntries with retry policy.
func (r *RetryClient) GetLedgerEntries(ctx context.Context, keys []string) (LedgerEntries, error) {
	return retryCall(r, ctx, func(attemptCtx context.Context) (LedgerEntries, error) {
		return r.client.GetLedgerEntries(attemptCtx, keys)
	})
}

// SimulateTransaction calls SimulateTransaction with retry policy.
func (r *RetryClient) SimulateTransaction(ctx context.Context, transactionXDR string) (Simulation, error) {
	return retryCall(r, ctx, func(attemptCtx context.Context) (Simulation, error) {
		return r.client.SimulateTransaction(attemptCtx, transactionXDR)
	})
}

// GetEvents calls GetEvents with retry policy.
func (r *RetryClient) GetEvents(ctx context.Context, request EventsRequest) (EventsPage, error) {
	return retryCall(r, ctx, func(attemptCtx context.Context) (EventsPage, error) {
		return r.client.GetEvents(attemptCtx, request)
	})
}

// GetTransaction calls GetTransaction with retry policy.
func (r *RetryClient) GetTransaction(ctx context.Context, hash string) (Transaction, error) {
	return retryCall(r, ctx, func(attemptCtx context.Context) (Transaction, error) {
		return r.client.GetTransaction(attemptCtx, hash)
	})
}

func retryCall[T any](r *RetryClient, ctx context.Context, call func(context.Context) (T, error)) (T, error) {
	var zero T
	backoff := r.initialBackoff

	for attempt := 1; attempt <= r.maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return zero, err
		}

		attemptCtx, cancel := context.WithTimeout(ctx, r.timeout)
		result, err := call(attemptCtx)
		cancel()
		if err == nil {
			return result, nil
		}
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
		if attempt == r.maxAttempts || !isRetryable(err) {
			return zero, err
		}

		if err := r.sleep(ctx, r.jitter(backoff)); err != nil {
			return zero, err
		}
		backoff = min(backoff*2, r.maxBackoff)
	}

	return zero, errors.New("retry attempts exhausted")
}

func isRetryable(err error) bool {
	var statusError interface{ StatusCode() int }
	if errors.As(err, &statusError) {
		status := statusError.StatusCode()
		return status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
	}

	var networkError net.Error
	return errors.As(err, &networkError)
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func jitterDuration(delay time.Duration) time.Duration {
	quarter := delay / 4
	if quarter <= 0 {
		return delay
	}

	random, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(quarter*2)+1))
	if err != nil {
		return delay
	}
	return delay - quarter + time.Duration(random.Int64())
}

type statusRoundTripper struct {
	next http.RoundTripper
}

func (t statusRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := t.next.RoundTrip(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < http.StatusBadRequest {
		return response, nil
	}

	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	return nil, &HTTPStatusError{Code: response.StatusCode, Status: response.Status}
}

func withStatusErrors(client *http.Client) *http.Client {
	if client == nil {
		client = http.DefaultClient
	}
	clone := *client
	transport := clone.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	clone.Transport = statusRoundTripper{next: transport}
	return &clone
}
