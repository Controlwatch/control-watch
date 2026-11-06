package observe

import (
	"context"
	"fmt"

	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/rpcx"
	"github.com/Controlwatch/control-watch/pkg/surface"
	"github.com/stellar/go-stellar-sdk/xdr"
)

// ClassificationResult describes a contract's detected interface and the
// ledger metadata needed by the observation orchestrator.
type ClassificationResult struct {
	Classification model.Classification
	WasmHash       string
	LatestLedger   uint32
	Warning        string
}

// ClassifyContract reads a contract instance and classifies its callable
// surface. If Wasm specification data is unavailable, it uses read-only getter
// simulations and records that fallback in Warning.
func ClassifyContract(
	ctx context.Context,
	client rpcx.Client,
	sourceAccount string,
	contractID string,
) (ClassificationResult, error) {
	instance, err := readContractInstance(ctx, client, contractID)
	if err != nil {
		return ClassificationResult{Classification: model.ClassError}, err
	}
	if instance.executable.Type == xdr.ContractExecutableTypeContractExecutableStellarAsset {
		return ClassificationResult{
			Classification: model.ClassSAC,
			LatestLedger:   instance.latestLedger,
		}, nil
	}
	hash, ok := instance.executable.GetWasmHash()
	if !ok {
		return ClassificationResult{Classification: model.ClassError}, fmt.Errorf(
			"classify contract: unsupported executable type %s",
			instance.executable.Type,
		)
	}
	hashHex, _ := instance.wasmHash()
	result := ClassificationResult{
		Classification: model.ClassUnknown,
		WasmHash:       hashHex,
		LatestLedger:   instance.latestLedger,
	}

	wasm, ledger, readErr := readContractCode(ctx, client, hash)
	if ledger > result.LatestLedger {
		result.LatestLedger = ledger
	}
	if readErr == nil {
		functions, parseErr := exportedFunctions(wasm)
		if parseErr == nil {
			result.Classification = classifyFunctions(functions)
			return result, nil
		}
		readErr = parseErr
	}

	result.Classification = classifyByGetterFallback(ctx, client, sourceAccount, contractID)
	result.Warning = fmt.Sprintf("contract spec unavailable; classification used getter simulation fallback: %v", readErr)
	return result, nil
}

func classifyFunctions(functions map[surface.FunctionName]struct{}) model.Classification {
	if containsAllFunctions(functions, sep41Signature()) && containsAllFunctions(functions, rwaSignature()) {
		return model.ClassSEP57RWA
	}
	if containsAllFunctions(functions, sep41Signature()) {
		return model.ClassSEP41
	}
	return model.ClassUnknown
}

func classifyByGetterFallback(
	ctx context.Context,
	client rpcx.Client,
	sourceAccount string,
	contractID string,
) model.Classification {
	if allGettersSucceed(ctx, client, sourceAccount, contractID, rwaFallbackGetters()) {
		return model.ClassSEP57RWA
	}
	if allGettersSucceed(ctx, client, sourceAccount, contractID, sep41FallbackGetters()) {
		return model.ClassSEP41
	}
	return model.ClassUnknown
}

func allGettersSucceed(
	ctx context.Context,
	client rpcx.Client,
	sourceAccount string,
	contractID string,
	getters []surface.FunctionName,
) bool {
	for _, getter := range getters {
		if _, err := callGetter(ctx, client, sourceAccount, contractID, getter); err != nil {
			return false
		}
	}
	return true
}

func containsAllFunctions(
	functions map[surface.FunctionName]struct{},
	required []surface.FunctionName,
) bool {
	for _, function := range required {
		if _, ok := functions[function]; !ok {
			return false
		}
	}
	return true
}

func sep41Signature() []surface.FunctionName {
	return []surface.FunctionName{
		surface.FunctionAllowance,
		surface.FunctionApprove,
		surface.FunctionBalance,
		surface.FunctionBurn,
		surface.FunctionBurnFrom,
		surface.FunctionDecimals,
		surface.FunctionTokenName,
		surface.FunctionSymbol,
		surface.FunctionTransfer,
		surface.FunctionTransferFrom,
	}
}

func rwaSignature() []surface.FunctionName {
	return []surface.FunctionName{
		surface.FunctionSetCompliance,
		surface.FunctionSetIdentityVerifier,
		surface.FunctionForcedTransfer,
		surface.FunctionRecoverBalance,
		surface.FunctionFreezePartialTokens,
		surface.FunctionUnfreezePartialTokens,
		surface.FunctionGetFrozenTokens,
		surface.FunctionSetAddressFrozen,
		surface.FunctionIsFrozen,
	}
}

func rwaFallbackGetters() []surface.FunctionName {
	return []surface.FunctionName{
		surface.FunctionCompliance,
		surface.FunctionIdentityVerifier,
		surface.FunctionPaused,
		surface.FunctionGetAdmin,
	}
}

func sep41FallbackGetters() []surface.FunctionName {
	return []surface.FunctionName{
		surface.FunctionDecimals,
		surface.FunctionTokenName,
		surface.FunctionSymbol,
	}
}
