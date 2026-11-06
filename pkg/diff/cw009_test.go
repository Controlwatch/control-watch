package diff

import (
	"reflect"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
)

func TestSACAdminChanged(t *testing.T) {
	t.Parallel()

	wantChanged := Result{Findings: []model.Finding{{
		Code:           "CW-009",
		Name:           "SAC_ADMIN_CHANGED",
		Subject:        sacAdminSubject,
		Before:         "admin-a",
		After:          "admin-b",
		ReviewPriority: model.PriorityHigh,
		Explanation:    "The Stellar Asset Contract administrator changed.",
		NextStep:       "Review the administrator accounts and confirm the intended assignment.",
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
			before: observationWithSACAdmin(model.Measure("admin-a")),
			after:  observationWithSACAdmin(model.Measure("admin-a")),
			want:   Result{},
		},
		{
			name:   "changed",
			before: observationWithSACAdmin(model.Measure("admin-a")),
			after:  observationWithSACAdmin(model.Measure("admin-b")),
			want:   wantChanged,
		},
		{
			name:   "unreadable current administrator",
			before: observationWithSACAdmin(model.Measure("admin-a")),
			after:  observationWithSACAdmin(model.CannotMeasure[string]("instance unavailable")),
			want: Result{NotCompared: []model.NotCompared{{
				Subject: sacAdminSubject,
				Reason:  "after: instance unavailable",
			}}},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := (SACAdminChanged{}).Detect(test.before, test.after)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Detect() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func observationWithSACAdmin(admin model.Measured[string]) model.Observation {
	return model.Observation{SAC: &model.SACSurface{Admin: admin}}
}
