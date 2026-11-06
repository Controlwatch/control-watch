package diff

import (
	"reflect"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
)

func TestModuleAdded(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		before model.Observation
		after  model.Observation
		want   Result
	}{
		{
			name:   "no modules added",
			before: observationWithModules(measuredModuleList("module-a")),
			after:  observationWithModules(measuredModuleList("module-a")),
			want:   Result{},
		},
		{
			name:   "sorts added modules by address",
			before: observationWithModules(measuredModuleList("module-a")),
			after:  observationWithModules(measuredModuleList("module-c", "module-a", "module-b")),
			want: Result{Findings: []model.Finding{
				moduleAddedFinding("module-b"),
				moduleAddedFinding("module-c"),
			}},
		},
		{
			name:   "unreadable module list",
			before: observationWithModules(model.CannotMeasure[[]model.Module]("getter unavailable")),
			after:  observationWithModules(measuredModuleList("module-a")),
			want: Result{NotCompared: []model.NotCompared{{
				Subject: modulesSubject,
				Reason:  "before: getter unavailable",
			}}},
		},
		{
			name:   "missing compliance surface is an empty set",
			before: model.Observation{},
			after:  observationWithModules(measuredModuleList("module-a")),
			want:   Result{Findings: []model.Finding{moduleAddedFinding("module-a")}},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := (ModuleAdded{}).Detect(test.before, test.after)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Detect() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func observationWithModules(modules model.Measured[[]model.Module]) model.Observation {
	return model.Observation{Compliance: &model.ComplianceSurface{Modules: modules}}
}

func measuredModuleList(addresses ...string) model.Measured[[]model.Module] {
	modules := make([]model.Module, 0, len(addresses))
	for _, address := range addresses {
		modules = append(modules, model.Module{Address: address})
	}
	return model.Measure(modules)
}

func moduleAddedFinding(address string) model.Finding {
	return model.Finding{
		Code:           "CW-003",
		Name:           "MODULE_ADDED",
		Subject:        "module:" + address,
		Before:         "absent",
		After:          "present",
		ReviewPriority: model.PriorityMedium,
		Explanation:    "A module was added to the compliance contract.",
		NextStep:       "Review the module address, deployed code, and registered hooks.",
		Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
	}
}
