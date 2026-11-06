package diff

import (
	"reflect"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
)

func TestRoleMembersChanged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		before model.Observation
		after  model.Observation
		want   Result
	}{
		{
			name:   "order and duplicates do not matter",
			before: observationWithRoles(map[string]model.Measured[[]string]{"admin": model.Measure([]string{"b", "a", "a"})}),
			after:  observationWithRoles(map[string]model.Measured[[]string]{"admin": model.Measure([]string{"a", "b"})}),
			want:   Result{},
		},
		{
			name:   "member set changed",
			before: observationWithRoles(map[string]model.Measured[[]string]{"admin": model.Measure([]string{"b", "a"})}),
			after:  observationWithRoles(map[string]model.Measured[[]string]{"admin": model.Measure([]string{"c", "b"})}),
			want: Result{Findings: []model.Finding{{
				Code:           "CW-007",
				Name:           "ROLE_MEMBERS_CHANGED",
				Subject:        "token.roles.admin",
				Before:         `["a","b"]`,
				After:          `["b","c"]`,
				ReviewPriority: model.PriorityMedium,
				Explanation:    "The member set for a token role changed.",
				NextStep:       "Review the role members and confirm the intended assignments.",
				Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
			}}},
		},
		{
			name:   "new role compares against empty set",
			before: model.Observation{},
			after:  observationWithRoles(map[string]model.Measured[[]string]{"operator": model.Measure([]string{"account-a"})}),
			want: Result{Findings: []model.Finding{{
				Code:           "CW-007",
				Name:           "ROLE_MEMBERS_CHANGED",
				Subject:        "token.roles.operator",
				Before:         `[]`,
				After:          `["account-a"]`,
				ReviewPriority: model.PriorityMedium,
				Explanation:    "The member set for a token role changed.",
				NextStep:       "Review the role members and confirm the intended assignments.",
				Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
			}}},
		},
		{
			name: "one role unreadable",
			before: observationWithRoles(map[string]model.Measured[[]string]{
				"admin": model.CannotMeasure[[]string]("getter unavailable"),
			}),
			after: observationWithRoles(map[string]model.Measured[[]string]{
				"admin": model.Measure([]string{"account-a"}),
			}),
			want: Result{NotCompared: []model.NotCompared{{
				Subject: "token.roles.admin",
				Reason:  "before: getter unavailable",
			}}},
		},
		{
			name: "whole role map unreadable",
			before: model.Observation{
				Reasons: map[string]string{tokenRolesSubject: "roles getter unavailable"},
			},
			after: observationWithRoles(map[string]model.Measured[[]string]{
				"admin": model.Measure([]string{"account-a"}),
			}),
			want: Result{NotCompared: []model.NotCompared{{
				Subject: tokenRolesSubject,
				Reason:  "before: roles getter unavailable",
			}}},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := (RoleMembersChanged{}).Detect(test.before, test.after)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Detect() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func observationWithRoles(roles map[string]model.Measured[[]string]) model.Observation {
	return model.Observation{Token: &model.TokenSurface{Roles: roles}}
}
