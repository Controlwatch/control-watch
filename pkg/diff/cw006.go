package diff

import "github.com/Controlwatch/control-watch/pkg/model"

const tokenAdminSubject = "token.admin"

// AdminChanged detects a changed RWA token administrator.
type AdminChanged struct{}

// Detect implements Detector.
func (AdminChanged) Detect(before model.Observation, after model.Observation) Result {
	beforeAdmin := tokenAdmin(before)
	afterAdmin := tokenAdmin(after)
	if reason := measuredComparisonReason(before, after, tokenAdminSubject, beforeAdmin, afterAdmin); reason != "" {
		return Result{NotCompared: []model.NotCompared{{Subject: tokenAdminSubject, Reason: reason}}}
	}
	if beforeAdmin.Value == afterAdmin.Value {
		return Result{}
	}
	return Result{Findings: []model.Finding{{
		Code:           "CW-006",
		Name:           "ADMIN_CHANGED",
		Subject:        tokenAdminSubject,
		Before:         beforeAdmin.Value,
		After:          afterAdmin.Value,
		ReviewPriority: model.PriorityHigh,
		Explanation:    "The token administrator changed.",
		NextStep:       "Review the administrator accounts and confirm the intended assignment.",
		Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
	}}}
}

func tokenAdmin(observation model.Observation) model.Measured[string] {
	if observation.Token == nil {
		return model.Measure("")
	}
	return observation.Token.Admin
}
