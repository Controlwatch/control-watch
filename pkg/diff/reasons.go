package diff

import (
	"fmt"
	"strings"

	"github.com/Controlwatch/control-watch/pkg/model"
)

func comparisonReason(before model.Observation, after model.Observation, path string) string {
	beforeReason := strings.TrimSpace(before.Reasons[path])
	afterReason := strings.TrimSpace(after.Reasons[path])
	switch {
	case beforeReason != "" && afterReason != "":
		return fmt.Sprintf("before: %s; after: %s", beforeReason, afterReason)
	case beforeReason != "":
		return "before: " + beforeReason
	case afterReason != "":
		return "after: " + afterReason
	default:
		return ""
	}
}

func measuredComparisonReason[T any](
	beforeObservation model.Observation,
	afterObservation model.Observation,
	path string,
	before model.Measured[T],
	after model.Measured[T],
) string {
	beforeReason := ""
	if before.Unmeasurable {
		beforeReason = firstConfiguredReason(before.Reason, beforeObservation.Reasons[path])
	}
	afterReason := ""
	if after.Unmeasurable {
		afterReason = firstConfiguredReason(after.Reason, afterObservation.Reasons[path])
	}
	switch {
	case beforeReason != "" && afterReason != "":
		return fmt.Sprintf("before: %s; after: %s", beforeReason, afterReason)
	case beforeReason != "":
		return "before: " + beforeReason
	case afterReason != "":
		return "after: " + afterReason
	default:
		return ""
	}
}

func firstConfiguredReason(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return "value could not be measured"
}
