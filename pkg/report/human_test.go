package report

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
)

func TestHumanSortsByPriorityThenCodeWithoutMutation(t *testing.T) {
	t.Parallel()

	value := model.Report{
		Network:      "testnet",
		ContractID:   "contract",
		Completeness: model.CompletenessPartial,
		Baseline:     model.BaselineReportRef{SnapshotID: "100-12345678", LedgerSeq: 100, SnapshotHash: "base"},
		Current:      model.CurrentReportRef{LedgerSeq: 110, SnapshotHash: "current"},
		RPC:          model.ReportRPC{Host: "rpc.example", OldestLedger: 90},
		Disclaimer:   model.ReportDisclaimer,
		Findings: []model.Finding{
			{
				Code: "CW-012", Name: "LOW", Subject: "z", ReviewPriority: model.PriorityLow,
				Evidence: model.Evidence{Status: model.EvidenceFound, Items: []model.EvidenceItem{
					{Ledger: 12, TxHash: "b", Topic: "z"},
					{Ledger: 11, TxHash: "a", Topic: "a"},
				}},
			},
			{Code: "CW-006", Name: "HIGH-B", Subject: "b", ReviewPriority: model.PriorityHigh},
			{Code: "CW-001", Name: "HIGH-A", Subject: "a", ReviewPriority: model.PriorityHigh},
		},
		NotCompared: []model.NotCompared{
			{Subject: "z", Reason: "later"},
			{Subject: "a", Reason: "first"},
		},
	}
	originalFindings := append([]model.Finding(nil), value.Findings...)
	originalItems := append([]model.EvidenceItem(nil), value.Findings[0].Evidence.Items...)
	originalNotCompared := append([]model.NotCompared(nil), value.NotCompared...)

	output := Human(value)
	highA := strings.Index(output, "CW-001")
	highB := strings.Index(output, "CW-006")
	low := strings.Index(output, "CW-012")
	if highA < 0 || highB < 0 || low < 0 || !(highA < highB && highB < low) {
		t.Fatalf("finding order is incorrect:\n%s", output)
	}
	firstEvidence := strings.Index(output, "ledger 11")
	secondEvidence := strings.Index(output, "ledger 12")
	if firstEvidence < 0 || secondEvidence < 0 || firstEvidence > secondEvidence {
		t.Fatalf("evidence order is incorrect:\n%s", output)
	}
	if !reflect.DeepEqual(value.Findings, originalFindings) ||
		!reflect.DeepEqual(value.Findings[0].Evidence.Items, originalItems) ||
		!reflect.DeepEqual(value.NotCompared, originalNotCompared) {
		t.Fatal("Human() mutated its input")
	}
}
