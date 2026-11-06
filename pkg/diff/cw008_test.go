package diff

import (
	"reflect"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
)

func TestPauseStateChanged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		before model.Observation
		after  model.Observation
		want   Result
	}{
		{
			name:   "unchanged",
			before: observationWithPause(model.Measure(false)),
			after:  observationWithPause(model.Measure(false)),
			want:   Result{},
		},
		{
			name:   "paused",
			before: observationWithPause(model.Measure(false)),
			after:  observationWithPause(model.Measure(true)),
			want:   Result{Findings: []model.Finding{pauseFinding("false", "true")}},
		},
		{
			name:   "resumed",
			before: observationWithPause(model.Measure(true)),
			after:  observationWithPause(model.Measure(false)),
			want:   Result{Findings: []model.Finding{pauseFinding("true", "false")}},
		},
		{
			name:   "unreadable baseline state",
			before: observationWithPause(model.CannotMeasure[bool]("getter unavailable")),
			after:  observationWithPause(model.Measure(true)),
			want: Result{NotCompared: []model.NotCompared{{
				Subject: tokenPausedSubject,
				Reason:  "before: getter unavailable",
			}}},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := (PauseStateChanged{}).Detect(test.before, test.after)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Detect() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func observationWithPause(paused model.Measured[bool]) model.Observation {
	return model.Observation{Token: &model.TokenSurface{Paused: paused}}
}

func pauseFinding(before string, after string) model.Finding {
	return model.Finding{
		Code:           "CW-008",
		Name:           "PAUSE_STATE_CHANGED",
		Subject:        tokenPausedSubject,
		Before:         before,
		After:          after,
		ReviewPriority: model.PriorityMedium,
		Explanation:    "The token pause state changed.",
		NextStep:       "Review the pause state and confirm the intended operating state.",
		Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
	}
}
