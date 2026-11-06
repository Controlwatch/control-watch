package diff

import "github.com/Controlwatch/control-watch/pkg/model"

// ModuleAdded detects module addresses newly registered on the compliance
// contract.
type ModuleAdded struct{}

// Detect implements Detector.
func (ModuleAdded) Detect(before model.Observation, after model.Observation) Result {
	beforeModules := measuredModules(before)
	afterModules := measuredModules(after)
	if reason := measuredComparisonReason(before, after, modulesSubject, beforeModules, afterModules); reason != "" {
		return Result{NotCompared: []model.NotCompared{{Subject: modulesSubject, Reason: reason}}}
	}
	beforeByAddress := modulesByAddress(beforeModules.Value)
	afterByAddress := modulesByAddress(afterModules.Value)
	result := Result{}
	for _, address := range sortedModuleAddresses(afterByAddress) {
		if _, existed := beforeByAddress[address]; existed {
			continue
		}
		result.Findings = append(result.Findings, model.Finding{
			Code:           "CW-003",
			Name:           "MODULE_ADDED",
			Subject:        "module:" + address,
			Before:         "absent",
			After:          "present",
			ReviewPriority: model.PriorityMedium,
			Explanation:    "A module was added to the compliance contract.",
			NextStep:       "Review the module address, deployed code, and registered hooks.",
			Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
		})
	}
	return result
}
