package rpcx

import (
	"context"
	"fmt"
	"sync"
)

// FakeCalls is a snapshot of calls received by FakeClient.
type FakeCalls struct {
	GetHealth           int
	GetLatestLedger     int
	GetLedgerEntries    [][]string
	SimulateTransaction []string
	GetEvents           []EventsRequest
	GetTransaction      []string
}

// FakeClient is a configurable, concurrency-safe Client for offline tests.
// Set the relevant function fields before invoking a method.
type FakeClient struct {
	GetHealthFunc           func(context.Context) (Health, error)
	GetLatestLedgerFunc     func(context.Context) (LatestLedger, error)
	GetLedgerEntriesFunc    func(context.Context, []string) (LedgerEntries, error)
	SimulateTransactionFunc func(context.Context, string) (Simulation, error)
	GetEventsFunc           func(context.Context, EventsRequest) (EventsPage, error)
	GetTransactionFunc      func(context.Context, string) (Transaction, error)

	mu    sync.Mutex
	calls FakeCalls
}

var _ Client = (*FakeClient)(nil)

// Calls returns a defensive copy of the recorded calls.
func (f *FakeClient) Calls() FakeCalls {
	f.mu.Lock()
	defer f.mu.Unlock()

	calls := f.calls
	calls.GetLedgerEntries = make([][]string, len(f.calls.GetLedgerEntries))
	for index, keys := range f.calls.GetLedgerEntries {
		calls.GetLedgerEntries[index] = append([]string(nil), keys...)
	}
	calls.SimulateTransaction = append([]string(nil), f.calls.SimulateTransaction...)
	calls.GetEvents = append([]EventsRequest(nil), f.calls.GetEvents...)
	calls.GetTransaction = append([]string(nil), f.calls.GetTransaction...)
	return calls
}

// GetHealth records and handles a health call.
func (f *FakeClient) GetHealth(ctx context.Context) (Health, error) {
	f.mu.Lock()
	f.calls.GetHealth++
	callback := f.GetHealthFunc
	f.mu.Unlock()
	if callback == nil {
		return Health{}, unsetFakeMethod("GetHealth")
	}
	return callback(ctx)
}

// GetLatestLedger records and handles a latest-ledger call.
func (f *FakeClient) GetLatestLedger(ctx context.Context) (LatestLedger, error) {
	f.mu.Lock()
	f.calls.GetLatestLedger++
	callback := f.GetLatestLedgerFunc
	f.mu.Unlock()
	if callback == nil {
		return LatestLedger{}, unsetFakeMethod("GetLatestLedger")
	}
	return callback(ctx)
}

// GetLedgerEntries records and handles a ledger-entry call.
func (f *FakeClient) GetLedgerEntries(ctx context.Context, keys []string) (LedgerEntries, error) {
	keys = append([]string(nil), keys...)
	f.mu.Lock()
	f.calls.GetLedgerEntries = append(f.calls.GetLedgerEntries, keys)
	callback := f.GetLedgerEntriesFunc
	f.mu.Unlock()
	if callback == nil {
		return LedgerEntries{}, unsetFakeMethod("GetLedgerEntries")
	}
	return callback(ctx, append([]string(nil), keys...))
}

// SimulateTransaction records and handles a simulation call.
func (f *FakeClient) SimulateTransaction(ctx context.Context, transactionXDR string) (Simulation, error) {
	f.mu.Lock()
	f.calls.SimulateTransaction = append(f.calls.SimulateTransaction, transactionXDR)
	callback := f.SimulateTransactionFunc
	f.mu.Unlock()
	if callback == nil {
		return Simulation{}, unsetFakeMethod("SimulateTransaction")
	}
	return callback(ctx, transactionXDR)
}

// GetEvents records and handles an events call.
func (f *FakeClient) GetEvents(ctx context.Context, request EventsRequest) (EventsPage, error) {
	request = cloneEventsRequest(request)
	f.mu.Lock()
	f.calls.GetEvents = append(f.calls.GetEvents, request)
	callback := f.GetEventsFunc
	f.mu.Unlock()
	if callback == nil {
		return EventsPage{}, unsetFakeMethod("GetEvents")
	}
	return callback(ctx, cloneEventsRequest(request))
}

// GetTransaction records and handles a transaction call.
func (f *FakeClient) GetTransaction(ctx context.Context, hash string) (Transaction, error) {
	f.mu.Lock()
	f.calls.GetTransaction = append(f.calls.GetTransaction, hash)
	callback := f.GetTransactionFunc
	f.mu.Unlock()
	if callback == nil {
		return Transaction{}, unsetFakeMethod("GetTransaction")
	}
	return callback(ctx, hash)
}

func unsetFakeMethod(name string) error {
	return fmt.Errorf("rpcx fake method %s is not configured", name)
}

func cloneEventsRequest(request EventsRequest) EventsRequest {
	clone := request
	clone.Filters = make([]EventFilter, len(request.Filters))
	for filterIndex, filter := range request.Filters {
		clone.Filters[filterIndex].ContractIDs = append([]string(nil), filter.ContractIDs...)
		clone.Filters[filterIndex].Topics = make([][]string, len(filter.Topics))
		for topicIndex, topic := range filter.Topics {
			clone.Filters[filterIndex].Topics[topicIndex] = append([]string(nil), topic...)
		}
	}
	return clone
}
