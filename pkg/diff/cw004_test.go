package diff

import (
	"reflect"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
)

func TestModuleRemoved(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		before model.Observation
		after  model.Observation
		want   Result
	}{
		{
			name:   "no modules removed",
			before: observationWithModules(measuredModuleList("module-a")),
			after:  observationWithModules(measuredModuleList("module-a", "module-b")),
			want:   Result{},
		},
		{
			name:   "sorts removed modules by address",
			before: observationWithModules(measuredModuleList("module-c", "module-a", "module-b")),
			after:  observationWithModules(measuredModuleList("module-b")),
			want: Result{Findings: []model.Finding{
				moduleRemovedFinding("module-a"),
				moduleRemovedFinding("module-c"),
			}},
		},
		{
			name:   "unreadable current module list",
			before: observationWithModules(measuredModuleList("module-a")),
			after:  observationWithModules(model.CannotMeasure[[]model.Module]("getter unavailable")),
			want: Result{NotCompared: []model.NotCompared{{
				Subject: modulesSubject,
				Reason:  "after: getter unavailable",
			}}},
		},
		{
			name:   "missing current compliance surface is an empty set",
			before: observationWithModules(measuredModuleList("module-a")),
			after:  model.Observation{},
			want:   Result{Findings: []model.Finding{moduleRemovedFinding("module-a")}},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := (ModuleRemoved{}).Detect(test.before, test.after)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Detect() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func moduleRemovedFinding(address string) model.Finding {
	return model.Finding{
		Code:           "CW-004",
		Name:           "MODULE_REMOVED",
		Subject:        "module:" + address,
		Before:         "present",
		After:          "absent",
		ReviewPriority: model.PriorityMedium,
		Explanation:    "A module was removed from the compliance contract.",
		NextStep:       "Review the removed module and confirm the intended configuration.",
		Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
	}
}
