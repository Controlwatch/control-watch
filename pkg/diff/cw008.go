package diff

import (
	"strconv"

	"github.com/Controlwatch/control-watch/pkg/model"
)

const tokenPausedSubject = "token.paused"

// PauseStateChanged detects a changed RWA token pause state.
type PauseStateChanged struct{}

// Detect implements Detector.
func (PauseStateChanged) Detect(before model.Observation, after model.Observation) Result {
	beforePaused := tokenPaused(before)
	afterPaused := tokenPaused(after)
	if reason := measuredComparisonReason(before, after, tokenPausedSubject, beforePaused, afterPaused); reason != "" {
		return Result{NotCompared: []model.NotCompared{{Subject: tokenPausedSubject, Reason: reason}}}
	}
	if beforePaused.Value == afterPaused.Value {
		return Result{}
	}
	return Result{Findings: []model.Finding{{
		Code:           "CW-008",
		Name:           "PAUSE_STATE_CHANGED",
		Subject:        tokenPausedSubject,
		Before:         strconv.FormatBool(beforePaused.Value),
		After:          strconv.FormatBool(afterPaused.Value),
		ReviewPriority: model.PriorityMedium,
		Explanation:    "The token pause state changed.",
		NextStep:       "Review the pause state and confirm the intended operating state.",
		Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
	}}}
}

func tokenPaused(observation model.Observation) model.Measured[bool] {
	if observation.Token == nil {
		return model.Measure(false)
	}
	return observation.Token.Paused
}
