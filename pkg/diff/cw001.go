package diff

import "github.com/Controlwatch/control-watch/pkg/model"

const complianceAddressSubject = "compliance.address"

// ComplianceContractChanged detects a changed compliance-contract reference.
type ComplianceContractChanged struct{}

// Detect implements Detector.
func (ComplianceContractChanged) Detect(before model.Observation, after model.Observation) Result {
	if reason := comparisonReason(before, after, complianceAddressSubject); reason != "" {
		return Result{NotCompared: []model.NotCompared{{
			Subject: complianceAddressSubject,
			Reason:  reason,
		}}}
	}
	beforeAddress := complianceAddress(before)
	afterAddress := complianceAddress(after)
	if beforeAddress == afterAddress {
		return Result{}
	}
	return Result{Findings: []model.Finding{{
		Code:           "CW-001",
		Name:           "COMPLIANCE_CONTRACT_CHANGED",
		Subject:        complianceAddressSubject,
		Before:         beforeAddress,
		After:          afterAddress,
		ReviewPriority: model.PriorityHigh,
		Explanation:    "The configured compliance contract changed.",
		NextStep:       "Review the referenced contracts and confirm the intended configuration.",
		Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
	}}}
}

func complianceAddress(observation model.Observation) string {
	if observation.Compliance == nil {
		return ""
	}
	return observation.Compliance.Address
}
