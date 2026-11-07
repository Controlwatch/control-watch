package report

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
)

func TestJSONIsDeterministicAndRoundTrips(t *testing.T) {
	t.Parallel()

	value := model.Report{
		SchemaVersion: model.SchemaVersion,
		GeneratedAt:   "2026-11-07T00:00:00Z",
		Network:       "testnet",
		ContractID:    "CAI26Z4CPI5XQ34NEEN64ISTCHPNU733HD6AQQYQIA6NITB4E2AYYHHK",
		Baseline: model.BaselineReportRef{
			SnapshotID:   "100-12345678",
			LedgerSeq:    100,
			SnapshotHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		Current: model.CurrentReportRef{
			LedgerSeq:    110,
			SnapshotHash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},
		Findings:     []model.Finding{},
		NotCompared:  []model.NotCompared{},
		Completeness: model.CompletenessComplete,
		RPC:          model.ReportRPC{Host: "rpc.example", OldestLedger: 90},
		Disclaimer:   model.ReportDisclaimer,
	}
	first, err := JSON(value)
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}
	second, err := JSON(value)
	if err != nil {
		t.Fatalf("second JSON() error = %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("identical reports rendered differently")
	}
	if len(first) == 0 || first[len(first)-1] != '\n' {
		t.Fatal("JSON report is not newline terminated")
	}
	var decoded model.Report
	if err := json.Unmarshal(first, &decoded); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if !reflect.DeepEqual(decoded, value) {
		t.Fatalf("decoded report = %#v, want %#v", decoded, value)
	}
}
