package observe

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/rpcx"
	"github.com/Controlwatch/control-watch/pkg/surface"
	"github.com/stellar/go-stellar-sdk/xdr"
)

// ModuleEnumerationResult contains the modules registered on a compliance
// contract and any independently unreadable module hashes.
type ModuleEnumerationResult struct {
	Modules      model.Measured[[]model.Module]
	LatestLedger uint32
	Reasons      map[string]string
}

// ReadComplianceModules queries every verified compliance hook and returns a
// deterministic module list. A failed hook makes the enumeration incomplete;
// a failed module-instance read affects only that module's Wasm hash.
func ReadComplianceModules(
	ctx context.Context,
	client rpcx.Client,
	sourceAccount string,
	complianceID string,
) ModuleEnumerationResult {
	result := ModuleEnumerationResult{Reasons: make(map[string]string)}
	registrations := make(map[string]map[surface.ComplianceHook]struct{})
	var hookFailures []string

	for _, hook := range complianceHooks() {
		getterResult, err := callGetter(
			ctx,
			client,
			sourceAccount,
			complianceID,
			surface.FunctionGetModulesForHook,
			complianceHookValue(hook),
		)
		if err != nil {
			hookFailures = append(hookFailures, fmt.Sprintf("%s: %v", hook, err))
			continue
		}
		result.LatestLedger = maxLedger(result.LatestLedger, getterResult.latestLedger)
		addresses, err := decodeAddressVector(getterResult.value, true)
		if err != nil {
			hookFailures = append(hookFailures, fmt.Sprintf("%s: %v", hook, err))
			continue
		}
		for _, address := range addresses {
			if registrations[address] == nil {
				registrations[address] = make(map[surface.ComplianceHook]struct{})
			}
			registrations[address][hook] = struct{}{}
		}
	}

	if len(hookFailures) > 0 {
		reason := "module hooks unavailable: " + strings.Join(hookFailures, "; ")
		result.Modules = model.CannotMeasure[[]model.Module](reason)
		result.Reasons["compliance.modules"] = reason
		return result
	}

	addresses := make([]string, 0, len(registrations))
	for address := range registrations {
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)
	modules := make([]model.Module, 0, len(addresses))
	for _, address := range addresses {
		ref, ledger := readContractRef(ctx, client, address)
		result.LatestLedger = maxLedger(result.LatestLedger, ledger)
		hooks := make([]string, 0, len(registrations[address]))
		for hook := range registrations[address] {
			hooks = append(hooks, string(hook))
		}
		sort.Strings(hooks)
		module := model.Module{
			Address:  address,
			WasmHash: ref.WasmHash,
			Hooks:    hooks,
		}
		modules = append(modules, module)
		if module.WasmHash.Unmeasurable {
			result.Reasons[fmt.Sprintf("compliance.modules.%s.wasm_hash", address)] = module.WasmHash.Reason
		}
	}
	result.Modules = model.Measure(modules)
	return result
}

func complianceHooks() []surface.ComplianceHook {
	return []surface.ComplianceHook{
		surface.HookTransferred,
		surface.HookCreated,
		surface.HookDestroyed,
	}
}

func complianceHookValue(hook surface.ComplianceHook) xdr.ScVal {
	vector := xdr.ScVec{symbolValue(string(hook))}
	vectorPointer := &vector
	return xdr.ScVal{
		Type: xdr.ScValTypeScvVec,
		Vec:  &vectorPointer,
	}
}

func decodeAddressVector(value xdr.ScVal, contractsOnly bool) ([]string, error) {
	vector, ok := value.GetVec()
	if !ok || vector == nil {
		return nil, fmt.Errorf("value type is %s, want address vector", value.Type)
	}
	result := make([]string, len(*vector))
	for index, item := range *vector {
		var (
			address string
			err     error
		)
		if contractsOnly {
			address, err = decodeContractAddress(item)
		} else {
			address, err = decodeAddress(item)
		}
		if err != nil {
			return nil, fmt.Errorf("vector item %d: %w", index, err)
		}
		result[index] = address
	}
	sort.Strings(result)
	return uniqueStrings(result), nil
}
