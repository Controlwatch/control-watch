package diff

import (
	"strconv"

	"github.com/Controlwatch/control-watch/pkg/model"
)

const classicFlagsSubject = "classic_flags"

// ClassicFlagsChanged detects changes to classic issuer authorization flags.
type ClassicFlagsChanged struct{}

// Detect implements Detector.
func (ClassicFlagsChanged) Detect(before model.Observation, after model.Observation) Result {
	if reason := comparisonReason(before, after, classicFlagsSubject); reason != "" {
		return Result{NotCompared: []model.NotCompared{{Subject: classicFlagsSubject, Reason: reason}}}
	}
	result := Result{}
	for _, flag := range classicFlagComparisons(before, after) {
		if reason := measuredComparisonReason(before, after, flag.subject, flag.before, flag.after); reason != "" {
			result.NotCompared = append(result.NotCompared, model.NotCompared{
				Subject: flag.subject,
				Reason:  reason,
			})
			continue
		}
		if flag.before.Value == flag.after.Value {
			continue
		}
		result.Findings = append(result.Findings, model.Finding{
			Code:           "CW-011",
			Name:           "CLASSIC_FLAGS_CHANGED",
			Subject:        flag.subject,
			Before:         strconv.FormatBool(flag.before.Value),
			After:          strconv.FormatBool(flag.after.Value),
			ReviewPriority: model.PriorityMedium,
			Explanation:    "A classic issuer authorization flag changed.",
			NextStep:       "Review the issuer account flags and confirm the intended settings.",
			Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
		})
	}
	return result
}

type classicFlagComparison struct {
	subject string
	before  model.Measured[bool]
	after   model.Measured[bool]
}

func classicFlagComparisons(before model.Observation, after model.Observation) []classicFlagComparison {
	beforeFlags := classicFlags(before)
	afterFlags := classicFlags(after)
	return []classicFlagComparison{
		{
			subject: "classic_flags.auth_clawback_enabled",
			before:  beforeFlags.AuthClawbackEnabled,
			after:   afterFlags.AuthClawbackEnabled,
		},
		{
			subject: "classic_flags.auth_immutable",
			before:  beforeFlags.AuthImmutable,
			after:   afterFlags.AuthImmutable,
		},
		{
			subject: "classic_flags.auth_required",
			before:  beforeFlags.AuthRequired,
			after:   afterFlags.AuthRequired,
		},
		{
			subject: "classic_flags.auth_revocable",
			before:  beforeFlags.AuthRevocable,
			after:   afterFlags.AuthRevocable,
		},
	}
}

func classicFlags(observation model.Observation) model.ClassicFlags {
	if observation.ClassicFlags == nil {
		return model.ClassicFlags{}
	}
	return *observation.ClassicFlags
}
