package report

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

func TestJSONValidatesAgainstPublishedSchema(t *testing.T) {
	t.Parallel()

	schema := compileReportSchema(t)
	encoded, err := JSON(validReport())
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}
	var document any
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if err := schema.Validate(document); err != nil {
		t.Fatalf("report does not validate: %v", err)
	}
}

func TestPublishedSchemaRejectsUnexpectedFields(t *testing.T) {
	t.Parallel()

	schema := compileReportSchema(t)
	encoded, err := JSON(validReport())
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	document["unexpected"] = true
	if err := schema.Validate(document); err == nil {
		t.Fatal("schema accepted an unexpected field")
	}
}

func compileReportSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	encoded, err := os.ReadFile("../../docs/report.schema.json")
	if err != nil {
		t.Fatalf("read report schema: %v", err)
	}
	var document any
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatalf("decode report schema: %v", err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	if err := compiler.AddResource("report.schema.json", document); err != nil {
		t.Fatalf("add report schema: %v", err)
	}
	schema, err := compiler.Compile("report.schema.json")
	if err != nil {
		t.Fatalf("compile report schema: %v", err)
	}
	return schema
}

func validReport() model.Report {
	return model.Report{
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
		Findings: []model.Finding{{
			Code:           "CW-008",
			Name:           "PAUSE_STATE_CHANGED",
			Subject:        "token.paused",
			Before:         "false",
			After:          "true",
			ReviewPriority: model.PriorityMedium,
			Explanation:    "The token pause state changed.",
			NextStep:       "Review the pause state and confirm the intended operating state.",
			Evidence: model.Evidence{
				Status: model.EvidenceFound,
				Items: []model.EvidenceItem{{
					Ledger: 105,
					TxHash: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
					Topic:  "paused",
				}},
			},
		}},
		NotCompared:  []model.NotCompared{},
		Completeness: model.CompletenessComplete,
		RPC:          model.ReportRPC{Host: "rpc.example", OldestLedger: 90},
		Disclaimer:   model.ReportDisclaimer,
	}
}
