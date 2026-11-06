package diff

import (
	"reflect"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
)

func TestIdentityVerifierChanged(t *testing.T) {
	t.Parallel()

	wantChanged := Result{Findings: []model.Finding{{
		Code:           "CW-002",
		Name:           "IDENTITY_VERIFIER_CHANGED",
		Subject:        identityVerifierAddressSubject,
		Before:         "before-verifier",
		After:          "after-verifier",
		ReviewPriority: model.PriorityHigh,
		Explanation:    "The configured identity verifier changed.",
		NextStep:       "Review the referenced contracts and confirm the intended configuration.",
		Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
	}}}
	tests := []struct {
		name   string
		before model.Observation
		after  model.Observation
		want   Result
	}{
		{
			name:   "unchanged",
			before: observationWithIdentityVerifier("same-verifier"),
			after:  observationWithIdentityVerifier("same-verifier"),
			want:   Result{},
		},
		{
			name:   "changed",
			before: observationWithIdentityVerifier("before-verifier"),
			after:  observationWithIdentityVerifier("after-verifier"),
			want:   wantChanged,
		},
		{
			name:   "unreadable after",
			before: observationWithIdentityVerifier("before-verifier"),
			after: model.Observation{
				Reasons: map[string]string{identityVerifierAddressSubject: "getter unavailable"},
			},
			want: Result{NotCompared: []model.NotCompared{{
				Subject: identityVerifierAddressSubject,
				Reason:  "after: getter unavailable",
			}}},
		},
		{
			name: "unreadable both",
			before: model.Observation{
				Reasons: map[string]string{identityVerifierAddressSubject: "old read failed"},
			},
			after: model.Observation{
				Reasons: map[string]string{identityVerifierAddressSubject: "new read failed"},
			},
			want: Result{NotCompared: []model.NotCompared{{
				Subject: identityVerifierAddressSubject,
				Reason:  "before: old read failed; after: new read failed",
			}}},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := (IdentityVerifierChanged{}).Detect(test.before, test.after)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Detect() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func observationWithIdentityVerifier(address string) model.Observation {
	return model.Observation{IdentityVerifier: &model.ContractRef{Address: address}}
}
