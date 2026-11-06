package diff

import "github.com/Controlwatch/control-watch/pkg/model"

// ModuleRemoved detects module addresses no longer registered on the
// compliance contract.
type ModuleRemoved struct{}

// Detect implements Detector.
func (ModuleRemoved) Detect(before model.Observation, after model.Observation) Result {
	beforeModules := measuredModules(before)
	afterModules := measuredModules(after)
	if reason := measuredComparisonReason(before, after, modulesSubject, beforeModules, afterModules); reason != "" {
		return Result{NotCompared: []model.NotCompared{{Subject: modulesSubject, Reason: reason}}}
	}
	beforeByAddress := modulesByAddress(beforeModules.Value)
	afterByAddress := modulesByAddress(afterModules.Value)
	result := Result{}
	for _, address := range sortedModuleAddresses(beforeByAddress) {
		if _, remains := afterByAddress[address]; remains {
			continue
		}
		result.Findings = append(result.Findings, model.Finding{
			Code:           "CW-004",
			Name:           "MODULE_REMOVED",
			Subject:        "module:" + address,
			Before:         "present",
			After:          "absent",
			ReviewPriority: model.PriorityMedium,
			Explanation:    "A module was removed from the compliance contract.",
			NextStep:       "Review the removed module and confirm the intended configuration.",
			Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
		})
	}
	return result
}
