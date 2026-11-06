package observe

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/rpcx"
	"github.com/Controlwatch/control-watch/pkg/surface"
	"github.com/stellar/go-stellar-sdk/ingest/sac"
	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stellar/go-stellar-sdk/xdr"
)

// SACSurfaceResult contains the SAC admin and optional classic issuer flags.
type SACSurfaceResult struct {
	SAC          model.SACSurface
	ClassicFlags *model.ClassicFlags
	LatestLedger uint32
	Reasons      map[string]string
}

// ReadSACSurface reads the SAC admin and, for an issued classic asset, the
// original issuer's Horizon authorization flags.
func ReadSACSurface(
	ctx context.Context,
	rpcClient rpcx.Client,
	flagsReader ClassicFlagsReader,
	sourceAccount string,
	contractID string,
	networkPassphrase string,
) SACSurfaceResult {
	result := SACSurfaceResult{Reasons: make(map[string]string)}
	adminResult, err := callGetter(
		ctx, rpcClient, sourceAccount, contractID, surface.FunctionSACAdmin,
	)
	if err != nil {
		var simulationFailure *simulationError
		if errors.As(err, &simulationFailure) && strings.Contains(simulationFailure.detail, "MissingValue") {
			result.SAC.Admin = model.Measure("")
			result.LatestLedger = simulationFailure.latestLedger
		} else {
			result.SAC.Admin = model.CannotMeasure[string](err.Error())
			result.Reasons["sac.admin"] = err.Error()
		}
	} else {
		result.LatestLedger = adminResult.latestLedger
		admin, decodeErr := decodeAddress(adminResult.value)
		if decodeErr != nil {
			reason := fmt.Sprintf("decode %s: %v", surface.FunctionSACAdmin, decodeErr)
			result.SAC.Admin = model.CannotMeasure[string](reason)
			result.Reasons["sac.admin"] = reason
		} else {
			result.SAC.Admin = model.Measure(admin)
		}
	}

	instance, instanceErr := readContractInstance(ctx, rpcClient, contractID)
	if instanceErr != nil {
		result.ClassicFlags = unmeasurableClassicFlags(instanceErr.Error())
		result.Reasons["classic_flags"] = instanceErr.Error()
		return result
	}
	result.LatestLedger = maxLedger(result.LatestLedger, instance.latestLedger)
	if isNativeSAC(contractID, networkPassphrase) {
		return result
	}
	asset, ok := sac.AssetFromContractData(instance.ledgerEntry, networkPassphrase)
	if !ok {
		reason := "SAC instance does not contain verifiable classic asset identity"
		result.ClassicFlags = unmeasurableClassicFlags(reason)
		result.Reasons["classic_flags"] = reason
		return result
	}
	if flagsReader == nil {
		reason := "classic flags reader is required"
		result.ClassicFlags = unmeasurableClassicFlags(reason)
		result.Reasons["classic_flags"] = reason
		return result
	}
	flags, flagsErr := flagsReader.ReadClassicFlags(ctx, asset.GetIssuer())
	if flagsErr != nil {
		reason := flagsErr.Error()
		result.ClassicFlags = unmeasurableClassicFlags(reason)
		result.Reasons["classic_flags"] = reason
		return result
	}
	result.LatestLedger = maxLedger(result.LatestLedger, flags.LastModifiedLedger)
	result.ClassicFlags = &model.ClassicFlags{
		AuthRequired:        model.Measure(flags.AuthRequired),
		AuthRevocable:       model.Measure(flags.AuthRevocable),
		AuthImmutable:       model.Measure(flags.AuthImmutable),
		AuthClawbackEnabled: model.Measure(flags.AuthClawbackEnabled),
	}
	return result
}

func isNativeSAC(contractID string, networkPassphrase string) bool {
	contractBytes, err := strkey.Decode(strkey.VersionByteContract, contractID)
	if err != nil {
		return false
	}
	var actual xdr.ContractId
	copy(actual[:], contractBytes)
	native, err := xdr.MustNewNativeAsset().ContractID(networkPassphrase)
	return err == nil && actual == native
}

func unmeasurableClassicFlags(reason string) *model.ClassicFlags {
	return &model.ClassicFlags{
		AuthRequired:        model.CannotMeasure[bool](reason),
		AuthRevocable:       model.CannotMeasure[bool](reason),
		AuthImmutable:       model.CannotMeasure[bool](reason),
		AuthClawbackEnabled: model.CannotMeasure[bool](reason),
	}
}
