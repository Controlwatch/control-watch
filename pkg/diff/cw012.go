package diff

import "github.com/Controlwatch/control-watch/pkg/model"

// ModuleHooksChanged detects changed hook sets for modules present in both
// snapshots.
type ModuleHooksChanged struct{}

// Detect implements Detector.
func (ModuleHooksChanged) Detect(before model.Observation, after model.Observation) Result {
	beforeModules := measuredModules(before)
	afterModules := measuredModules(after)
	if reason := measuredComparisonReason(before, after, modulesSubject, beforeModules, afterModules); reason != "" {
		return Result{NotCompared: []model.NotCompared{{Subject: modulesSubject, Reason: reason}}}
	}
	beforeByAddress := modulesByAddress(beforeModules.Value)
	afterByAddress := modulesByAddress(afterModules.Value)
	result := Result{}
	for _, address := range sortedModuleAddresses(beforeByAddress) {
		afterModule, remains := afterByAddress[address]
		if !remains {
			continue
		}
		beforeHooks := canonicalStringSet(beforeByAddress[address].Hooks)
		afterHooks := canonicalStringSet(afterModule.Hooks)
		if beforeHooks == afterHooks {
			continue
		}
		result.Findings = append(result.Findings, model.Finding{
			Code:           "CW-012",
			Name:           "MODULE_HOOKS_CHANGED",
			Subject:        "module:" + address,
			Before:         beforeHooks,
			After:          afterHooks,
			ReviewPriority: model.PriorityLow,
			Explanation:    "The registered hook set for a module changed.",
			NextStep:       "Review the module hooks and confirm the intended registration points.",
			Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
		})
	}
	return result
}
