package diff

import (
	"reflect"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
)

func TestModuleHooksChanged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		before model.Observation
		after  model.Observation
		want   Result
	}{
		{
			name: "order and duplicates do not matter",
			before: observationWithModuleValues(model.Module{
				Address: "module-a", Hooks: []string{"transfer", "mint", "mint"},
			}),
			after: observationWithModuleValues(model.Module{
				Address: "module-a", Hooks: []string{"mint", "transfer"},
			}),
			want: Result{},
		},
		{
			name: "hooks changed",
			before: observationWithModuleValues(model.Module{
				Address: "module-a", Hooks: []string{"transfer", "mint"},
			}),
			after: observationWithModuleValues(model.Module{
				Address: "module-a", Hooks: []string{"burn", "transfer"},
			}),
			want: Result{Findings: []model.Finding{{
				Code:           "CW-012",
				Name:           "MODULE_HOOKS_CHANGED",
				Subject:        "module:module-a",
				Before:         `["mint","transfer"]`,
				After:          `["burn","transfer"]`,
				ReviewPriority: model.PriorityLow,
				Explanation:    "The registered hook set for a module changed.",
				NextStep:       "Review the module hooks and confirm the intended registration points.",
				Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
			}}},
		},
		{
			name:   "new module is handled by add detector",
			before: model.Observation{},
			after: observationWithModuleValues(model.Module{
				Address: "module-a", Hooks: []string{"transfer"},
			}),
			want: Result{},
		},
		{
			name:   "unreadable module list",
			before: observationWithModules(model.CannotMeasure[[]model.Module]("getter unavailable")),
			after: observationWithModuleValues(model.Module{
				Address: "module-a", Hooks: []string{"transfer"},
			}),
			want: Result{NotCompared: []model.NotCompared{{
				Subject: modulesSubject,
				Reason:  "before: getter unavailable",
			}}},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := (ModuleHooksChanged{}).Detect(test.before, test.after)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Detect() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func observationWithModuleValues(modules ...model.Module) model.Observation {
	return observationWithModules(model.Measure(modules))
}
