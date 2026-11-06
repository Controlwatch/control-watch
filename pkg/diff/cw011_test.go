package diff

import (
	"reflect"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
)

func TestClassicFlagsChanged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		before model.Observation
		after  model.Observation
		want   Result
	}{
		{
			name:   "unchanged",
			before: observationWithClassicFlags(false, false, false, false),
			after:  observationWithClassicFlags(false, false, false, false),
			want:   Result{},
		},
		{
			name:   "multiple flags change in subject order",
			before: observationWithClassicFlags(false, true, false, true),
			after:  observationWithClassicFlags(true, false, false, true),
			want: Result{Findings: []model.Finding{
				classicFlagFinding("classic_flags.auth_required", "false", "true"),
				classicFlagFinding("classic_flags.auth_revocable", "true", "false"),
			}},
		},
		{
			name: "whole flag surface unreadable",
			before: model.Observation{
				Reasons: map[string]string{classicFlagsSubject: "issuer unavailable"},
			},
			after: observationWithClassicFlags(true, false, false, false),
			want: Result{NotCompared: []model.NotCompared{{
				Subject: classicFlagsSubject,
				Reason:  "before: issuer unavailable",
			}}},
		},
		{
			name: "one flag unreadable",
			before: model.Observation{ClassicFlags: &model.ClassicFlags{
				AuthRequired:        model.Measure(false),
				AuthRevocable:       model.CannotMeasure[bool]("flag unavailable"),
				AuthImmutable:       model.Measure(false),
				AuthClawbackEnabled: model.Measure(false),
			}},
			after: observationWithClassicFlags(false, true, false, false),
			want: Result{NotCompared: []model.NotCompared{{
				Subject: "classic_flags.auth_revocable",
				Reason:  "before: flag unavailable",
			}}},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := (ClassicFlagsChanged{}).Detect(test.before, test.after)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Detect() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func observationWithClassicFlags(required bool, revocable bool, immutable bool, clawback bool) model.Observation {
	return model.Observation{ClassicFlags: &model.ClassicFlags{
		AuthRequired:        model.Measure(required),
		AuthRevocable:       model.Measure(revocable),
		AuthImmutable:       model.Measure(immutable),
		AuthClawbackEnabled: model.Measure(clawback),
	}}
}

func classicFlagFinding(subject string, before string, after string) model.Finding {
	return model.Finding{
		Code:           "CW-011",
		Name:           "CLASSIC_FLAGS_CHANGED",
		Subject:        subject,
		Before:         before,
		After:          after,
		ReviewPriority: model.PriorityMedium,
		Explanation:    "A classic issuer authorization flag changed.",
		NextStep:       "Review the issuer account flags and confirm the intended settings.",
		Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
	}
}
