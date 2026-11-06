package evidence

import (
	"context"
	"reflect"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/rpcx"
	"github.com/Controlwatch/control-watch/pkg/surface"
)

func TestSearchEventsPagesAndMatchesExactTopics(t *testing.T) {
	t.Parallel()

	pausedTopic := mustEventTopic(t, surface.EventPaused)
	unpausedTopic := mustEventTopic(t, surface.EventUnpaused)
	requests := []rpcx.EventsRequest{}
	client := &rpcx.FakeClient{GetEventsFunc: func(_ context.Context, request rpcx.EventsRequest) (rpcx.EventsPage, error) {
		requests = append(requests, request)
		if request.Cursor == "" {
			return rpcx.EventsPage{
				Cursor: "next",
				Events: []rpcx.Event{
					{ID: "2", Ledger: 102, ContractID: "contract-a", Topic: []string{unpausedTopic}, TransactionHash: "tx-b"},
					{ID: "ignored-name", Ledger: 101, ContractID: "contract-a", Topic: []string{pausedTopic}, TransactionHash: "tx-a"},
					{ID: "ignored-contract", Ledger: 100, ContractID: "contract-b", Topic: []string{unpausedTopic}, TransactionHash: "tx-c"},
				},
			}, nil
		}
		return rpcx.EventsPage{}, nil
	}}
	items, err := SearchEvents(context.Background(), client, EventQuery{
		ContractIDs: []string{"contract-a", "contract-a"},
		Names:       []surface.EventName{surface.EventUnpaused, surface.EventUnpaused},
		StartLedger: 90,
		EndLedger:   110,
	})
	if err != nil {
		t.Fatalf("SearchEvents() error = %v", err)
	}
	want := []model.EvidenceItem{{Ledger: 102, TxHash: "tx-b", Topic: string(surface.EventUnpaused)}}
	if !reflect.DeepEqual(items, want) {
		t.Fatalf("items = %#v, want %#v", items, want)
	}
	if len(requests) != 2 || requests[1].Cursor != "next" {
		t.Fatalf("requests = %#v, want second page cursor", requests)
	}
	if requests[0].Limit != eventPageSize || len(requests[0].Filters) != 1 {
		t.Fatalf("first request = %#v", requests[0])
	}
	if got := requests[0].Filters[0].ContractIDs; !reflect.DeepEqual(got, []string{"contract-a"}) {
		t.Fatalf("contract filters = %#v", got)
	}
}

func TestSearchEventsSortsMatches(t *testing.T) {
	t.Parallel()

	topic := mustEventTopic(t, surface.EventComplianceSet)
	client := &rpcx.FakeClient{GetEventsFunc: func(context.Context, rpcx.EventsRequest) (rpcx.EventsPage, error) {
		return rpcx.EventsPage{Events: []rpcx.Event{
			{ID: "b", Ledger: 12, ContractID: "contract", Topic: []string{topic}, TransactionHash: "tx-b"},
			{ID: "a", Ledger: 11, ContractID: "contract", Topic: []string{topic}, TransactionHash: "tx-a"},
		}}, nil
	}}
	items, err := SearchEvents(context.Background(), client, EventQuery{
		ContractIDs: []string{"contract"},
		Names:       []surface.EventName{surface.EventComplianceSet},
		StartLedger: 10,
		EndLedger:   20,
	})
	if err != nil {
		t.Fatalf("SearchEvents() error = %v", err)
	}
	if len(items) != 2 || items[0].Ledger != 11 || items[1].Ledger != 12 {
		t.Fatalf("items = %#v, want ledger order", items)
	}
}

func TestSearchEventsRejectsMalformedTopic(t *testing.T) {
	t.Parallel()

	client := &rpcx.FakeClient{GetEventsFunc: func(context.Context, rpcx.EventsRequest) (rpcx.EventsPage, error) {
		return rpcx.EventsPage{Events: []rpcx.Event{{
			ID: "bad", Ledger: 10, ContractID: "contract", Topic: []string{"not-xdr"},
		}}}, nil
	}}
	_, err := SearchEvents(context.Background(), client, EventQuery{
		ContractIDs: []string{"contract"},
		Names:       []surface.EventName{surface.EventPaused},
		StartLedger: 10,
		EndLedger:   20,
	})
	if err == nil {
		t.Fatal("SearchEvents() error = nil")
	}
}

func mustEventTopic(t *testing.T, name surface.EventName) string {
	t.Helper()
	encoded, err := encodeEventName(name)
	if err != nil {
		t.Fatalf("encode event topic: %v", err)
	}
	return encoded
}
