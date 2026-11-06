package diff

import "github.com/Controlwatch/control-watch/pkg/model"

const sacAdminSubject = "sac.admin"

// SACAdminChanged detects a changed Stellar Asset Contract administrator.
type SACAdminChanged struct{}

// Detect implements Detector.
func (SACAdminChanged) Detect(before model.Observation, after model.Observation) Result {
	beforeAdmin := sacAdmin(before)
	afterAdmin := sacAdmin(after)
	if reason := measuredComparisonReason(before, after, sacAdminSubject, beforeAdmin, afterAdmin); reason != "" {
		return Result{NotCompared: []model.NotCompared{{Subject: sacAdminSubject, Reason: reason}}}
	}
	if beforeAdmin.Value == afterAdmin.Value {
		return Result{}
	}
	return Result{Findings: []model.Finding{{
		Code:           "CW-009",
		Name:           "SAC_ADMIN_CHANGED",
		Subject:        sacAdminSubject,
		Before:         beforeAdmin.Value,
		After:          afterAdmin.Value,
		ReviewPriority: model.PriorityHigh,
		Explanation:    "The Stellar Asset Contract administrator changed.",
		NextStep:       "Review the administrator accounts and confirm the intended assignment.",
		Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
	}}}
}

func sacAdmin(observation model.Observation) model.Measured[string] {
	if observation.SAC == nil {
		return model.Measure("")
	}
	return observation.SAC.Admin
}
