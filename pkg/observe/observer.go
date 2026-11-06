package observe

import (
	"context"
	"fmt"
	"sort"

	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/rpcx"
	"github.com/stellar/go-stellar-sdk/strkey"
)

// Result is one assembled observation and its run-level metadata.
type Result struct {
	Observation  model.Observation
	LedgerSeq    uint32
	Completeness string
	Warnings     []string
}

// Observer coordinates classification and surface readers for one network.
type Observer struct {
	rpcClient         rpcx.Client
	classicFlags      ClassicFlagsReader
	sourceAccount     string
	networkPassphrase string
}

// NewObserver constructs an observation coordinator. The source account is
// used only to build unsigned simulation envelopes.
func NewObserver(
	rpcClient rpcx.Client,
	classicFlags ClassicFlagsReader,
	sourceAccount string,
	networkPassphrase string,
) (*Observer, error) {
	if rpcClient == nil {
		return nil, fmt.Errorf("construct observer: RPC client is required")
	}
	if _, err := strkey.Decode(strkey.VersionByteAccountID, sourceAccount); err != nil {
		return nil, fmt.Errorf("construct observer: invalid source account: %w", err)
	}
	if networkPassphrase == "" {
		return nil, fmt.Errorf("construct observer: network passphrase is required")
	}
	return &Observer{
		rpcClient:         rpcClient,
		classicFlags:      classicFlags,
		sourceAccount:     sourceAccount,
		networkPassphrase: networkPassphrase,
	}, nil
}

// Observe reads one contract's applicable control surface. Operational errors
// prevent an observation; individual field failures produce a partial result.
func (observer *Observer) Observe(ctx context.Context, contractID string) (Result, error) {
	if _, err := strkey.Decode(strkey.VersionByteContract, contractID); err != nil {
		return Result{}, model.NewError(model.CodeInvalidContractID, err)
	}
	latest, err := observer.rpcClient.GetLatestLedger(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("observe contract: get latest ledger: %w", err)
	}
	classification, err := ClassifyContract(
		ctx,
		observer.rpcClient,
		observer.sourceAccount,
		contractID,
	)
	if err != nil {
		return Result{}, fmt.Errorf("observe contract: %w", err)
	}

	result := Result{
		Observation: model.Observation{
			Classification: classification.Classification,
			Reasons:        make(map[string]string),
		},
		LedgerSeq: latest.Sequence,
	}
	result.LedgerSeq = maxLedger(result.LedgerSeq, classification.LatestLedger)
	if classification.Warning != "" {
		result.Warnings = append(result.Warnings, classification.Warning)
	}

	switch classification.Classification {
	case model.ClassSEP57RWA:
		observer.observeRWA(ctx, contractID, classification.WasmHash, &result)
	case model.ClassSEP41:
		result.Observation.Token = reducedTokenSurface(
			classification.WasmHash,
			"field is outside the SEP-41 observation surface",
			result.Observation.Reasons,
		)
	case model.ClassSAC:
		observer.observeSAC(ctx, contractID, &result)
	case model.ClassUnknown:
		result.Observation.Token = reducedTokenSurface(
			classification.WasmHash,
			"field is unavailable for an unknown contract surface",
			result.Observation.Reasons,
		)
	default:
		return Result{}, fmt.Errorf("observe contract: unsupported classification %q", classification.Classification)
	}

	for path, reason := range result.Observation.Reasons {
		result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %s", path, reason))
	}
	result.Warnings = sortedUniqueStrings(result.Warnings)
	if len(result.Observation.Reasons) == 0 {
		result.Observation.Reasons = nil
		result.Completeness = model.CompletenessComplete
	} else {
		result.Completeness = model.CompletenessPartial
	}
	return result, nil
}

func (observer *Observer) observeRWA(
	ctx context.Context,
	contractID string,
	wasmHash string,
	result *Result,
) {
	rwa := ReadRWASurface(
		ctx,
		observer.rpcClient,
		observer.sourceAccount,
		contractID,
		wasmHash,
	)
	result.Observation.Token = &rwa.Token
	result.Observation.IdentityVerifier = rwa.IdentityVerifier
	result.LedgerSeq = maxLedger(result.LedgerSeq, rwa.LatestLedger)
	mergeReasons(result.Observation.Reasons, rwa.Reasons)

	if rwa.Compliance == nil {
		return
	}
	modules := ReadComplianceModules(
		ctx,
		observer.rpcClient,
		observer.sourceAccount,
		rwa.Compliance.Address,
	)
	result.LedgerSeq = maxLedger(result.LedgerSeq, modules.LatestLedger)
	mergeReasons(result.Observation.Reasons, modules.Reasons)
	result.Observation.Compliance = &model.ComplianceSurface{
		ContractRef: *rwa.Compliance,
		Modules:     modules.Modules,
	}
}

func (observer *Observer) observeSAC(ctx context.Context, contractID string, result *Result) {
	sacResult := ReadSACSurface(
		ctx,
		observer.rpcClient,
		observer.classicFlags,
		observer.sourceAccount,
		contractID,
		observer.networkPassphrase,
	)
	result.Observation.SAC = &sacResult.SAC
	result.Observation.ClassicFlags = sacResult.ClassicFlags
	result.LedgerSeq = maxLedger(result.LedgerSeq, sacResult.LatestLedger)
	mergeReasons(result.Observation.Reasons, sacResult.Reasons)
}

func reducedTokenSurface(
	wasmHash string,
	reason string,
	reasons map[string]string,
) *model.TokenSurface {
	surface := &model.TokenSurface{
		WasmHash: measuredWasmHash(wasmHash, "token Wasm hash is unavailable"),
		Paused:   model.CannotMeasure[bool](reason),
		Admin:    model.CannotMeasure[string](reason),
		Roles:    map[string]model.Measured[[]string]{},
	}
	addMeasuredReason(reasons, "token.wasm_hash", surface.WasmHash)
	reasons["token.paused"] = reason
	reasons["token.admin"] = reason
	reasons["token.roles"] = reason
	return surface
}

func mergeReasons(destination map[string]string, source map[string]string) {
	for path, reason := range source {
		destination[path] = reason
	}
}

func sortedUniqueStrings(values []string) []string {
	sort.Strings(values)
	return uniqueStrings(values)
}
