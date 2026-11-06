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
