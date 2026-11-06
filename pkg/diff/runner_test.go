package diff

import (
	"reflect"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
)

func TestRunner(t *testing.T) {
	t.Parallel()

	base := model.Snapshot{
		SchemaVersion: model.SchemaVersion,
		Network:       "testnet",
		ContractID:    "contract",
		LedgerSeq:     10,
		ObservedAt:    "before",
		Observation:   model.Observation{Classification: model.ClassSEP57RWA},
	}
	tests := []struct {
		name      string
		before    model.Snapshot
		after     model.Snapshot
		detectors []Detector
		want      Result
		wantCode  model.ErrorCode
	}{
		{
			name:   "sorts combined detector output",
			before: base,
			after:  withMetadata(base, 20, "after"),
			detectors: []Detector{
				staticDetector{result: Result{
					Findings: []model.Finding{
						{Code: "CW-002", Subject: "z"},
						{Code: "CW-001", Subject: "b"},
					},
					NotCompared: []model.NotCompared{{Subject: "z"}},
				}},
				staticDetector{result: Result{
					Findings:    []model.Finding{{Code: "CW-001", Subject: "a"}},
					NotCompared: []model.NotCompared{{Subject: "a", Reason: "missing"}},
				}},
			},
			want: Result{
				Findings: []model.Finding{
					{Code: "CW-001", Subject: "a"},
					{Code: "CW-001", Subject: "b"},
					{Code: "CW-002", Subject: "z"},
				},
				NotCompared: []model.NotCompared{
					{Subject: "a", Reason: "missing"},
					{Subject: "z"},
				},
			},
		},
		{
			name:     "rejects scope mismatch",
			before:   base,
			after:    withContract(base, "other"),
			wantCode: model.CodeScopeMismatch,
		},
		{
			name:     "rejects unsupported schema",
			before:   base,
			after:    withSchema(base, "2"),
			wantCode: model.CodeSchemaUnsupported,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewRunner(test.detectors...).Run(test.before, test.after)
			if test.wantCode != "" {
				if code, ok := model.CodeOf(err); !ok || code != test.wantCode {
					t.Fatalf("error = %v, want code %s", err, test.wantCode)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Run() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestRunnerPassesOnlySemanticObservations(t *testing.T) {
	t.Parallel()

	detector := &observationDetector{}
	before := model.Snapshot{
		SchemaVersion: model.SchemaVersion,
		Network:       "testnet",
		ContractID:    "contract",
		Observation:   model.Observation{Classification: model.ClassSAC},
	}
	after := withMetadata(before, 99, "later")
	if _, err := NewRunner(detector).Run(before, after); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if detector.before.Classification != model.ClassSAC || detector.after.Classification != model.ClassSAC {
		t.Fatalf("detector received observations %#v and %#v", detector.before, detector.after)
	}
}

type staticDetector struct {
	result Result
}

func (detector staticDetector) Detect(model.Observation, model.Observation) Result {
	return detector.result
}

type observationDetector struct {
	before model.Observation
	after  model.Observation
}

func (detector *observationDetector) Detect(before model.Observation, after model.Observation) Result {
	detector.before = before
	detector.after = after
	return Result{}
}

func withMetadata(snapshot model.Snapshot, ledger uint32, observedAt string) model.Snapshot {
	snapshot.LedgerSeq = ledger
	snapshot.ObservedAt = observedAt
	snapshot.ToolVersion = "different"
	snapshot.Warnings = []string{"different"}
	return snapshot
}

func withContract(snapshot model.Snapshot, contract string) model.Snapshot {
	snapshot.ContractID = contract
	return snapshot
}

func withSchema(snapshot model.Snapshot, schema string) model.Snapshot {
	snapshot.SchemaVersion = schema
	return snapshot
}
