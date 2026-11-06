package diff

import (
	"reflect"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
)

func TestWasmHashChanged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		before model.Observation
		after  model.Observation
		want   Result
	}{
		{
			name:   "unchanged token",
			before: observationWithTokenHash(model.Measure("hash-a")),
			after:  observationWithTokenHash(model.Measure("hash-a")),
			want:   Result{},
		},
		{
			name:   "token hash changed",
			before: observationWithTokenHash(model.Measure("hash-a")),
			after:  observationWithTokenHash(model.Measure("hash-b")),
			want:   Result{Findings: []model.Finding{wasmHashFinding("token", "hash-a", "hash-b", model.PriorityHigh)}},
		},
		{
			name:   "compliance hash changed",
			before: observationWithComplianceHash(model.Measure("hash-a")),
			after:  observationWithComplianceHash(model.Measure("hash-b")),
			want:   Result{Findings: []model.Finding{wasmHashFinding("compliance", "hash-a", "hash-b", model.PriorityHigh)}},
		},
		{
			name:   "identity verifier hash changed",
			before: observationWithIdentityHash(model.Measure("hash-a")),
			after:  observationWithIdentityHash(model.Measure("hash-b")),
			want:   Result{Findings: []model.Finding{wasmHashFinding("identity_verifier", "hash-a", "hash-b", model.PriorityHigh)}},
		},
		{
			name: "module hash changed at medium priority",
			before: observationWithModules(model.Measure([]model.Module{{
				Address: "module-a", WasmHash: model.Measure("hash-a"),
			}})),
			after: observationWithModules(model.Measure([]model.Module{{
				Address: "module-a", WasmHash: model.Measure("hash-b"),
			}})),
			want: Result{Findings: []model.Finding{
				wasmHashFinding("module:module-a", "hash-a", "hash-b", model.PriorityMedium),
			}},
		},
		{
			name:   "unreadable token hash",
			before: observationWithTokenHash(model.CannotMeasure[string]("instance missing")),
			after:  observationWithTokenHash(model.Measure("hash-b")),
			want: Result{NotCompared: []model.NotCompared{{
				Subject: "token.wasm_hash",
				Reason:  "before: instance missing",
			}}},
		},
		{
			name: "new and removed modules are handled by other detectors",
			before: observationWithModules(model.Measure([]model.Module{{
				Address: "module-a", WasmHash: model.Measure("hash-a"),
			}})),
			after: observationWithModules(model.Measure([]model.Module{{
				Address: "module-b", WasmHash: model.Measure("hash-b"),
			}})),
			want: Result{},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := (WasmHashChanged{}).Detect(test.before, test.after)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Detect() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func observationWithTokenHash(hash model.Measured[string]) model.Observation {
	return model.Observation{Token: &model.TokenSurface{WasmHash: hash}}
}

func observationWithComplianceHash(hash model.Measured[string]) model.Observation {
	return model.Observation{Compliance: &model.ComplianceSurface{
		ContractRef: model.ContractRef{WasmHash: hash},
	}}
}

func observationWithIdentityHash(hash model.Measured[string]) model.Observation {
	return model.Observation{IdentityVerifier: &model.ContractRef{WasmHash: hash}}
}

func wasmHashFinding(subject string, before string, after string, priority string) model.Finding {
	return model.Finding{
		Code:           "CW-005",
		Name:           "WASM_HASH_CHANGED",
		Subject:        subject,
		Before:         before,
		After:          after,
		ReviewPriority: priority,
		Explanation:    "A tracked contract's deployed Wasm hash changed.",
		NextStep:       "Review the deployed Wasm and confirm the intended implementation.",
		Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
	}
}
