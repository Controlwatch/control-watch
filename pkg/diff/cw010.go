package diff

import "github.com/Controlwatch/control-watch/pkg/model"

const classificationSubject = "classification"

// ClassificationChanged detects a changed contract-surface classification.
type ClassificationChanged struct{}

// Detect implements Detector.
func (ClassificationChanged) Detect(before model.Observation, after model.Observation) Result {
	if before.Classification == after.Classification {
		return Result{}
	}
	return Result{Findings: []model.Finding{{
		Code:           "CW-010",
		Name:           "CLASSIFICATION_CHANGED",
		Subject:        classificationSubject,
		Before:         string(before.Classification),
		After:          string(after.Classification),
		ReviewPriority: model.PriorityHigh,
		Explanation:    "The observed contract-surface classification changed.",
		NextStep:       "Review the current contract interface and confirm the intended classification.",
		Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
	}}}
}
