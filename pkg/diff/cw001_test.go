package diff

import (
	"reflect"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
)

func TestComplianceContractChanged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		before model.Observation
		after  model.Observation
		want   Result
	}{
		{
			name:   "unchanged",
			before: observationWithCompliance("CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD2KM"),
			after:  observationWithCompliance("CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD2KM"),
			want:   Result{},
		},
		{
			name:   "changed",
			before: observationWithCompliance("before-contract"),
			after:  observationWithCompliance("after-contract"),
			want: Result{Findings: []model.Finding{{
				Code:           "CW-001",
				Name:           "COMPLIANCE_CONTRACT_CHANGED",
				Subject:        complianceAddressSubject,
				Before:         "before-contract",
				After:          "after-contract",
				ReviewPriority: model.PriorityHigh,
				Explanation:    "The configured compliance contract changed.",
				NextStep:       "Review the referenced contracts and confirm the intended configuration.",
				Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
			}}},
		},
		{
			name:   "new reference",
			before: model.Observation{},
			after:  observationWithCompliance("after-contract"),
			want: Result{Findings: []model.Finding{{
				Code:           "CW-001",
				Name:           "COMPLIANCE_CONTRACT_CHANGED",
				Subject:        complianceAddressSubject,
				After:          "after-contract",
				ReviewPriority: model.PriorityHigh,
				Explanation:    "The configured compliance contract changed.",
				NextStep:       "Review the referenced contracts and confirm the intended configuration.",
				Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
			}}},
		},
		{
			name: "unreadable before",
			before: model.Observation{
				Reasons: map[string]string{complianceAddressSubject: "getter unavailable"},
			},
			after: observationWithCompliance("after-contract"),
			want: Result{NotCompared: []model.NotCompared{{
				Subject: complianceAddressSubject,
				Reason:  "before: getter unavailable",
			}}},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := (ComplianceContractChanged{}).Detect(test.before, test.after)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Detect() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func observationWithCompliance(address string) model.Observation {
	return model.Observation{Compliance: &model.ComplianceSurface{
		ContractRef: model.ContractRef{Address: address},
	}}
}
