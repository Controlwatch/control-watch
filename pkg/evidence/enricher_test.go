package evidence

import (
	"context"
	"reflect"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/rpcx"
	"github.com/Controlwatch/control-watch/pkg/surface"
)

func TestEnricherAttachEvidenceScenarios(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		beforeLedger   uint32
		afterLedger    uint32
		finding        model.Finding
		health         rpcx.Health
		events         []rpcx.Event
		want           model.Evidence
		wantEventCalls int
	}{
		{
			name:         "matching event inside retention",
			beforeLedger: 100,
			afterLedger:  110,
			finding:      model.Finding{Code: "CW-008"},
			health:       rpcx.Health{OldestLedger: 90, LatestLedger: 120},
			events: []rpcx.Event{{
				ID:              "event-1",
				Ledger:          105,
				ContractID:      "contract",
				Topic:           []string{mustEventTopic(t, surface.EventPaused)},
				TransactionHash: "tx-1",
			}},
			want: model.Evidence{
				Status: model.EvidenceFound,
				Items: []model.EvidenceItem{{
					Ledger: 105,
					TxHash: "tx-1",
					Topic:  string(surface.EventPaused),
				}},
			},
			wantEventCalls: 1,
		},
		{
			name:         "comparison interval starts outside retention",
			beforeLedger: 50,
			afterLedger:  110,
			finding:      model.Finding{Code: "CW-008"},
			health:       rpcx.Health{OldestLedger: 90, LatestLedger: 120},
			want: model.Evidence{
				Status:         model.EvidenceUnavailableOutsideRetention,
				Items:          []model.EvidenceItem{},
				ChangedBetween: ledgerInterval(50, 110),
			},
		},
		{
			name:           "mapped event was not emitted",
			beforeLedger:   100,
			afterLedger:    110,
			finding:        model.Finding{Code: "CW-001"},
			health:         rpcx.Health{OldestLedger: 90, LatestLedger: 120},
			want:           model.Evidence{Status: model.EvidenceNotEmitted, Items: []model.EvidenceItem{}},
			wantEventCalls: 1,
		},
		{
			name:         "finding has no contract event family",
			beforeLedger: 100,
			afterLedger:  110,
			finding:      model.Finding{Code: "CW-011"},
			health:       rpcx.Health{OldestLedger: 90, LatestLedger: 120},
			want:         model.Evidence{Status: model.EvidenceNotEmitted, Items: []model.EvidenceItem{}},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var eventRequest rpcx.EventsRequest
			client := &rpcx.FakeClient{
				GetHealthFunc: func(context.Context) (rpcx.Health, error) {
					return test.health, nil
				},
				GetEventsFunc: func(_ context.Context, request rpcx.EventsRequest) (rpcx.EventsPage, error) {
					eventRequest = request
					return rpcx.EventsPage{Events: test.events}, nil
				},
			}
			enricher, err := NewEnricher(client)
			if err != nil {
				t.Fatalf("NewEnricher() error = %v", err)
			}
			input := []model.Finding{test.finding}
			got, err := enricher.Attach(
				context.Background(),
				snapshotAt(test.beforeLedger),
				snapshotAt(test.afterLedger),
				input,
			)
			if err != nil {
				t.Fatalf("Attach() error = %v", err)
			}
			if len(got) != 1 || !reflect.DeepEqual(got[0].Evidence, test.want) {
				t.Fatalf("evidence = %#v, want %#v", got[0].Evidence, test.want)
			}
			if input[0].Evidence.Status != "" {
				t.Fatalf("input finding was mutated: %#v", input[0])
			}
			calls := client.Calls()
			if calls.GetHealth != 1 || len(calls.GetEvents) != test.wantEventCalls {
				t.Fatalf("calls = %#v, want health=1 events=%d", calls, test.wantEventCalls)
			}
			if test.wantEventCalls > 0 && (eventRequest.StartLedger != test.beforeLedger+1 || eventRequest.EndLedger != test.afterLedger) {
				t.Fatalf("event range = %d..%d", eventRequest.StartLedger, eventRequest.EndLedger)
			}
		})
	}
}

func TestEnricherSkipsRPCWhenThereAreNoFindings(t *testing.T) {
	t.Parallel()

	client := &rpcx.FakeClient{}
	enricher, err := NewEnricher(client)
	if err != nil {
		t.Fatalf("NewEnricher() error = %v", err)
	}
	got, err := enricher.Attach(context.Background(), snapshotAt(100), snapshotAt(110), nil)
	if err != nil {
		t.Fatalf("Attach() error = %v", err)
	}
	if got != nil {
		t.Fatalf("findings = %#v, want nil", got)
	}
	if calls := client.Calls(); calls.GetHealth != 0 || len(calls.GetEvents) != 0 {
		t.Fatalf("calls = %#v, want none", calls)
	}
}

func snapshotAt(ledger uint32) model.Snapshot {
	return model.Snapshot{ContractID: "contract", LedgerSeq: ledger}
}

func ledgerInterval(before uint32, after uint32) *[2]uint32 {
	interval := [2]uint32{before, after}
	return &interval
}
