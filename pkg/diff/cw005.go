package diff

import "github.com/Controlwatch/control-watch/pkg/model"

// WasmHashChanged detects changed deployed Wasm hashes for tracked contracts.
type WasmHashChanged struct{}

// Detect implements Detector.
func (WasmHashChanged) Detect(before model.Observation, after model.Observation) Result {
	result := Result{}
	compareWasmHash(
		&result, before, after, "token.wasm_hash", "token",
		tokenWasmHash(before), tokenWasmHash(after), model.PriorityHigh,
	)
	compareWasmHash(
		&result, before, after, "compliance.wasm_hash", "compliance",
		complianceWasmHash(before), complianceWasmHash(after), model.PriorityHigh,
	)
	compareWasmHash(
		&result, before, after, "identity_verifier.wasm_hash", "identity_verifier",
		identityVerifierWasmHash(before), identityVerifierWasmHash(after), model.PriorityHigh,
	)

	beforeModules := measuredModules(before)
	afterModules := measuredModules(after)
	if reason := measuredComparisonReason(before, after, modulesSubject, beforeModules, afterModules); reason != "" {
		result.NotCompared = append(result.NotCompared, model.NotCompared{
			Subject: modulesSubject,
			Reason:  reason,
		})
		return result
	}
	beforeByAddress := modulesByAddress(beforeModules.Value)
	afterByAddress := modulesByAddress(afterModules.Value)
	for _, address := range sortedModuleAddresses(beforeByAddress) {
		afterModule, remains := afterByAddress[address]
		if !remains {
			continue
		}
		beforeModule := beforeByAddress[address]
		path := "compliance.modules." + address + ".wasm_hash"
		compareWasmHash(
			&result, before, after, path, "module:"+address,
			beforeModule.WasmHash, afterModule.WasmHash, model.PriorityMedium,
		)
	}
	return result
}

func compareWasmHash(
	result *Result,
	beforeObservation model.Observation,
	afterObservation model.Observation,
	path string,
	subject string,
	before model.Measured[string],
	after model.Measured[string],
	priority string,
) {
	if reason := measuredComparisonReason(beforeObservation, afterObservation, path, before, after); reason != "" {
		result.NotCompared = append(result.NotCompared, model.NotCompared{Subject: path, Reason: reason})
		return
	}
	if before.Value == after.Value {
		return
	}
	result.Findings = append(result.Findings, model.Finding{
		Code:           "CW-005",
		Name:           "WASM_HASH_CHANGED",
		Subject:        subject,
		Before:         before.Value,
		After:          after.Value,
		ReviewPriority: priority,
		Explanation:    "A tracked contract's deployed Wasm hash changed.",
		NextStep:       "Review the deployed Wasm and confirm the intended implementation.",
		Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
	})
}

func tokenWasmHash(observation model.Observation) model.Measured[string] {
	if observation.Token == nil {
		return model.Measure("")
	}
	return observation.Token.WasmHash
}

func complianceWasmHash(observation model.Observation) model.Measured[string] {
	if observation.Compliance == nil {
		return model.Measure("")
	}
	return observation.Compliance.WasmHash
}

func identityVerifierWasmHash(observation model.Observation) model.Measured[string] {
	if observation.IdentityVerifier == nil {
		return model.Measure("")
	}
	return observation.IdentityVerifier.WasmHash
}
