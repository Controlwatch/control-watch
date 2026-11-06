package diff

import (
	"reflect"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
)

func TestClassificationChanged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		before model.Classification
		after  model.Classification
		want   Result
	}{
		{
			name:   "unchanged",
			before: model.ClassSEP57RWA,
			after:  model.ClassSEP57RWA,
			want:   Result{},
		},
		{
			name:   "RWA to SAC",
			before: model.ClassSEP57RWA,
			after:  model.ClassSAC,
			want: Result{Findings: []model.Finding{{
				Code:           "CW-010",
				Name:           "CLASSIFICATION_CHANGED",
				Subject:        classificationSubject,
				Before:         string(model.ClassSEP57RWA),
				After:          string(model.ClassSAC),
				ReviewPriority: model.PriorityHigh,
				Explanation:    "The observed contract-surface classification changed.",
				NextStep:       "Review the current contract interface and confirm the intended classification.",
				Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
			}}},
		},
		{
			name:   "unknown to SEP-41",
			before: model.ClassUnknown,
			after:  model.ClassSEP41,
			want: Result{Findings: []model.Finding{{
				Code:           "CW-010",
				Name:           "CLASSIFICATION_CHANGED",
				Subject:        classificationSubject,
				Before:         string(model.ClassUnknown),
				After:          string(model.ClassSEP41),
				ReviewPriority: model.PriorityHigh,
				Explanation:    "The observed contract-surface classification changed.",
				NextStep:       "Review the current contract interface and confirm the intended classification.",
				Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
			}}},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := (ClassificationChanged{}).Detect(
				model.Observation{Classification: test.before},
				model.Observation{Classification: test.after},
			)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Detect() = %#v, want %#v", got, test.want)
			}
		})
	}
}
