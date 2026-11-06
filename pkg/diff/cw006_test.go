package diff

import (
	"reflect"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
)

func TestAdminChanged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		before model.Observation
		after  model.Observation
		want   Result
	}{
		{
			name:   "unchanged",
			before: observationWithTokenAdmin(model.Measure("admin-a")),
			after:  observationWithTokenAdmin(model.Measure("admin-a")),
			want:   Result{},
		},
		{
			name:   "changed",
			before: observationWithTokenAdmin(model.Measure("admin-a")),
			after:  observationWithTokenAdmin(model.Measure("admin-b")),
			want: Result{Findings: []model.Finding{{
				Code:           "CW-006",
				Name:           "ADMIN_CHANGED",
				Subject:        tokenAdminSubject,
				Before:         "admin-a",
				After:          "admin-b",
				ReviewPriority: model.PriorityHigh,
				Explanation:    "The token administrator changed.",
				NextStep:       "Review the administrator accounts and confirm the intended assignment.",
				Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
			}}},
		},
		{
			name:   "unreadable current administrator",
			before: observationWithTokenAdmin(model.Measure("admin-a")),
			after:  observationWithTokenAdmin(model.CannotMeasure[string]("getter unavailable")),
			want: Result{NotCompared: []model.NotCompared{{
				Subject: tokenAdminSubject,
				Reason:  "after: getter unavailable",
			}}},
		},
		{
			name:   "missing token surface compares as absent",
			before: model.Observation{},
			after:  observationWithTokenAdmin(model.Measure("admin-b")),
			want: Result{Findings: []model.Finding{{
				Code:           "CW-006",
				Name:           "ADMIN_CHANGED",
				Subject:        tokenAdminSubject,
				After:          "admin-b",
				ReviewPriority: model.PriorityHigh,
				Explanation:    "The token administrator changed.",
				NextStep:       "Review the administrator accounts and confirm the intended assignment.",
				Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
			}}},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := (AdminChanged{}).Detect(test.before, test.after)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Detect() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func observationWithTokenAdmin(admin model.Measured[string]) model.Observation {
	return model.Observation{Token: &model.TokenSurface{Admin: admin}}
}
