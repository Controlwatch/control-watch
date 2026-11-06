package rpcx

import (
	"context"
	"strings"
	"testing"
)

func TestFakeClientRecordsDefensiveCallSnapshots(t *testing.T) {
	t.Parallel()

	fake := &FakeClient{
		GetLedgerEntriesFunc: func(_ context.Context, keys []string) (LedgerEntries, error) {
			keys[0] = "callback mutation"
			return LedgerEntries{}, nil
		},
		GetEventsFunc: func(_ context.Context, request EventsRequest) (EventsPage, error) {
			request.Filters[0].ContractIDs[0] = "callback mutation"
			request.Filters[0].Topics[0][0] = "callback mutation"
			return EventsPage{}, nil
		},
	}

	keys := []string{"ledger-key"}
	request := EventsRequest{Filters: []EventFilter{{
		ContractIDs: []string{"contract-id"},
		Topics:      [][]string{{"topic"}},
	}}}
	if _, err := fake.GetLedgerEntries(context.Background(), keys); err != nil {
		t.Fatalf("GetLedgerEntries() error = %v", err)
	}
	if _, err := fake.GetEvents(context.Background(), request); err != nil {
		t.Fatalf("GetEvents() error = %v", err)
	}

	keys[0] = "caller mutation"
	request.Filters[0].ContractIDs[0] = "caller mutation"
	request.Filters[0].Topics[0][0] = "caller mutation"

	calls := fake.Calls()
	if got := calls.GetLedgerEntries[0][0]; got != "ledger-key" {
		t.Fatalf("recorded ledger key = %q, want ledger-key", got)
	}
	if got := calls.GetEvents[0].Filters[0].ContractIDs[0]; got != "contract-id" {
		t.Fatalf("recorded contract ID = %q, want contract-id", got)
	}
	if got := calls.GetEvents[0].Filters[0].Topics[0][0]; got != "topic" {
		t.Fatalf("recorded topic = %q, want topic", got)
	}

	calls.GetLedgerEntries[0][0] = "snapshot mutation"
	calls.GetEvents[0].Filters[0].ContractIDs[0] = "snapshot mutation"
	calls.GetEvents[0].Filters[0].Topics[0][0] = "snapshot mutation"

	secondSnapshot := fake.Calls()
	if got := secondSnapshot.GetLedgerEntries[0][0]; got != "ledger-key" {
		t.Fatalf("ledger key after snapshot mutation = %q, want ledger-key", got)
	}
	if got := secondSnapshot.GetEvents[0].Filters[0].ContractIDs[0]; got != "contract-id" {
		t.Fatalf("contract ID after snapshot mutation = %q, want contract-id", got)
	}
	if got := secondSnapshot.GetEvents[0].Filters[0].Topics[0][0]; got != "topic" {
		t.Fatalf("topic after snapshot mutation = %q, want topic", got)
	}
}

func TestFakeClientReportsUnsetMethods(t *testing.T) {
	t.Parallel()

	_, err := (&FakeClient{}).GetHealth(context.Background())
	if err == nil {
		t.Fatal("GetHealth() error = nil, want an unset-method error")
	}
	if !strings.Contains(err.Error(), "GetHealth") {
		t.Fatalf("GetHealth() error = %q, want method name", err)
	}
}
