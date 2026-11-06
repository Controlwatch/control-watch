package diff

import "github.com/Controlwatch/control-watch/pkg/model"

const identityVerifierAddressSubject = "identity_verifier.address"

// IdentityVerifierChanged detects a changed identity-verifier reference.
type IdentityVerifierChanged struct{}

// Detect implements Detector.
func (IdentityVerifierChanged) Detect(before model.Observation, after model.Observation) Result {
	if reason := comparisonReason(before, after, identityVerifierAddressSubject); reason != "" {
		return Result{NotCompared: []model.NotCompared{{
			Subject: identityVerifierAddressSubject,
			Reason:  reason,
		}}}
	}
	beforeAddress := identityVerifierAddress(before)
	afterAddress := identityVerifierAddress(after)
	if beforeAddress == afterAddress {
		return Result{}
	}
	return Result{Findings: []model.Finding{{
		Code:           "CW-002",
		Name:           "IDENTITY_VERIFIER_CHANGED",
		Subject:        identityVerifierAddressSubject,
		Before:         beforeAddress,
		After:          afterAddress,
		ReviewPriority: model.PriorityHigh,
		Explanation:    "The configured identity verifier changed.",
		NextStep:       "Review the referenced contracts and confirm the intended configuration.",
		Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
	}}}
}

func identityVerifierAddress(observation model.Observation) string {
	if observation.IdentityVerifier == nil {
		return ""
	}
	return observation.IdentityVerifier.Address
}
