package rpcx

import "context"

// Client provides the read-only Stellar RPC operations used by ControlWatch.
// Implementations must not submit transactions.
type Client interface {
	GetHealth(ctx context.Context) (Health, error)
	GetLatestLedger(ctx context.Context) (LatestLedger, error)
	GetLedgerEntries(ctx context.Context, keys []string) (LedgerEntries, error)
	SimulateTransaction(ctx context.Context, transactionXDR string) (Simulation, error)
	GetEvents(ctx context.Context, request EventsRequest) (EventsPage, error)
	GetTransaction(ctx context.Context, hash string) (Transaction, error)
}
