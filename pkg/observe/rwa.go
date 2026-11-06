package observe

import (
	"context"
	"fmt"
	"sort"

	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/rpcx"
	"github.com/Controlwatch/control-watch/pkg/surface"
	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stellar/go-stellar-sdk/xdr"
)

// RWASurfaceResult contains the directly observed RWA token surface. Module
// enumeration is performed separately after the compliance address is known.
type RWASurfaceResult struct {
	Token            model.TokenSurface
	Compliance       *model.ContractRef
	IdentityVerifier *model.ContractRef
	LatestLedger     uint32
	Reasons          map[string]string
}

// ReadRWASurface reads the verified OpenZeppelin RWA getters independently so
// that one unavailable value does not discard other measurable state.
func ReadRWASurface(
	ctx context.Context,
	client rpcx.Client,
	sourceAccount string,
	contractID string,
	tokenWasmHash string,
) RWASurfaceResult {
	result := RWASurfaceResult{
		Token: model.TokenSurface{
			WasmHash: measuredWasmHash(tokenWasmHash, "token Wasm hash is unavailable"),
			Roles:    make(map[string]model.Measured[[]string]),
		},
		Reasons: make(map[string]string),
	}
	addMeasuredReason(result.Reasons, "token.wasm_hash", result.Token.WasmHash)

	compliance, ledger := readMeasuredGetter(
		ctx, client, sourceAccount, contractID, surface.FunctionCompliance, decodeContractAddress,
	)
	result.LatestLedger = maxLedger(result.LatestLedger, ledger)
	if compliance.Unmeasurable {
		result.Reasons["compliance.address"] = compliance.Reason
	} else {
		ref, refLedger := readContractRef(ctx, client, compliance.Value)
		result.LatestLedger = maxLedger(result.LatestLedger, refLedger)
		result.Compliance = &ref
		if ref.WasmHash.Unmeasurable {
			result.Reasons["compliance.wasm_hash"] = ref.WasmHash.Reason
		}
	}

	identityVerifier, ledger := readMeasuredGetter(
		ctx, client, sourceAccount, contractID, surface.FunctionIdentityVerifier, decodeContractAddress,
	)
	result.LatestLedger = maxLedger(result.LatestLedger, ledger)
	if identityVerifier.Unmeasurable {
		result.Reasons["identity_verifier.address"] = identityVerifier.Reason
	} else {
		ref, refLedger := readContractRef(ctx, client, identityVerifier.Value)
		result.LatestLedger = maxLedger(result.LatestLedger, refLedger)
		result.IdentityVerifier = &ref
		if ref.WasmHash.Unmeasurable {
			result.Reasons["identity_verifier.wasm_hash"] = ref.WasmHash.Reason
		}
	}

	result.Token.Paused, ledger = readMeasuredGetter(
		ctx, client, sourceAccount, contractID, surface.FunctionPaused, decodeBool,
	)
	result.LatestLedger = maxLedger(result.LatestLedger, ledger)
	addMeasuredReason(result.Reasons, "token.paused", result.Token.Paused)

	result.Token.Admin, ledger = readMeasuredGetter(
		ctx, client, sourceAccount, contractID, surface.FunctionGetAdmin, decodeAddress,
	)
	result.LatestLedger = maxLedger(result.LatestLedger, ledger)
	addMeasuredReason(result.Reasons, "token.admin", result.Token.Admin)

	roles, rolesLedger, rolesReason := readRoles(ctx, client, sourceAccount, contractID)
	result.LatestLedger = maxLedger(result.LatestLedger, rolesLedger)
	result.Token.Roles = roles
	if rolesReason != "" {
		result.Reasons["token.roles"] = rolesReason
	}
	for role, members := range roles {
		if members.Unmeasurable {
			result.Reasons[fmt.Sprintf("token.roles.%s", role)] = members.Reason
		}
	}

	return result
}

func readRoles(
	ctx context.Context,
	client rpcx.Client,
	sourceAccount string,
	contractID string,
) (map[string]model.Measured[[]string], uint32, string) {
	rolesResult, err := callGetter(
		ctx, client, sourceAccount, contractID, surface.FunctionGetExistingRoles,
	)
	if err != nil {
		return map[string]model.Measured[[]string]{}, 0, err.Error()
	}
	roles, err := decodeSymbolVector(rolesResult.value)
	if err != nil {
		return map[string]model.Measured[[]string]{}, rolesResult.latestLedger, err.Error()
	}
	sort.Strings(roles)
	roles = uniqueStrings(roles)

	measured := make(map[string]model.Measured[[]string], len(roles))
	latestLedger := rolesResult.latestLedger
	for _, role := range roles {
		members, ledger, err := readRoleMembers(ctx, client, sourceAccount, contractID, role)
		latestLedger = maxLedger(latestLedger, ledger)
		if err != nil {
			measured[role] = model.CannotMeasure[[]string](err.Error())
			continue
		}
		measured[role] = model.Measure(members)
	}
	return measured, latestLedger, ""
}

func readRoleMembers(
	ctx context.Context,
	client rpcx.Client,
	sourceAccount string,
	contractID string,
	role string,
) ([]string, uint32, error) {
	roleArg := symbolValue(role)
	countResult, err := callGetter(
		ctx,
		client,
		sourceAccount,
		contractID,
		surface.FunctionGetRoleMemberCount,
		roleArg,
	)
	if err != nil {
		return nil, 0, err
	}
	count, err := decodeU32(countResult.value)
	if err != nil {
		return nil, countResult.latestLedger, err
	}

	members := make([]string, 0, count)
	latestLedger := countResult.latestLedger
	for index := uint32(0); index < count; index++ {
		memberResult, err := callGetter(
			ctx,
			client,
			sourceAccount,
			contractID,
			surface.FunctionGetRoleMember,
			roleArg,
			u32Value(index),
		)
		if err != nil {
			return nil, latestLedger, err
		}
		latestLedger = maxLedger(latestLedger, memberResult.latestLedger)
		member, err := decodeAddress(memberResult.value)
		if err != nil {
			return nil, latestLedger, err
		}
		members = append(members, member)
	}
	sort.Strings(members)
	return uniqueStrings(members), latestLedger, nil
}

func readContractRef(ctx context.Context, client rpcx.Client, address string) (model.ContractRef, uint32) {
	ref := model.ContractRef{Address: address}
	instance, err := readContractInstance(ctx, client, address)
	if err != nil {
		ref.WasmHash = model.CannotMeasure[string](err.Error())
		return ref, 0
	}
	hash, ok := instance.wasmHash()
	if !ok {
		ref.WasmHash = model.CannotMeasure[string]("contract executable is not Wasm")
		return ref, instance.latestLedger
	}
	ref.WasmHash = model.Measure(hash)
	return ref, instance.latestLedger
}

func readMeasuredGetter[T any](
	ctx context.Context,
	client rpcx.Client,
	sourceAccount string,
	contractID string,
	function surface.FunctionName,
	decode func(xdr.ScVal) (T, error),
) (model.Measured[T], uint32) {
	result, err := callGetter(ctx, client, sourceAccount, contractID, function)
	if err != nil {
		return model.CannotMeasure[T](err.Error()), 0
	}
	value, err := decode(result.value)
	if err != nil {
		return model.CannotMeasure[T](fmt.Sprintf("decode %s: %v", function, err)), result.latestLedger
	}
	return model.Measure(value), result.latestLedger
}

func decodeBool(value xdr.ScVal) (bool, error) {
	result, ok := value.GetB()
	if !ok {
		return false, fmt.Errorf("value type is %s, want bool", value.Type)
	}
	return result, nil
}

func decodeU32(value xdr.ScVal) (uint32, error) {
	result, ok := value.GetU32()
	if !ok {
		return 0, fmt.Errorf("value type is %s, want u32", value.Type)
	}
	return uint32(result), nil
}

func decodeAddress(value xdr.ScVal) (string, error) {
	address, ok := value.GetAddress()
	if !ok {
		return "", fmt.Errorf("value type is %s, want address", value.Type)
	}
	switch address.Type {
	case xdr.ScAddressTypeScAddressTypeAccount:
		account, ok := address.GetAccountId()
		if !ok {
			return "", fmt.Errorf("account address has no account ID")
		}
		return strkey.Encode(strkey.VersionByteAccountID, account.Ed25519[:])
	case xdr.ScAddressTypeScAddressTypeContract:
		contract, ok := address.GetContractId()
		if !ok {
			return "", fmt.Errorf("contract address has no contract ID")
		}
		return strkey.Encode(strkey.VersionByteContract, contract[:])
	default:
		return "", fmt.Errorf("unsupported address type %s", address.Type)
	}
}

func decodeContractAddress(value xdr.ScVal) (string, error) {
	address, ok := value.GetAddress()
	if !ok || address.Type != xdr.ScAddressTypeScAddressTypeContract {
		return "", fmt.Errorf("value type is %s, want contract address", value.Type)
	}
	return decodeAddress(value)
}

func decodeSymbolVector(value xdr.ScVal) ([]string, error) {
	vector, ok := value.GetVec()
	if !ok || vector == nil {
		return nil, fmt.Errorf("value type is %s, want symbol vector", value.Type)
	}
	result := make([]string, len(*vector))
	for index, item := range *vector {
		symbol, ok := item.GetSym()
		if !ok {
			return nil, fmt.Errorf("vector item %d has type %s, want symbol", index, item.Type)
		}
		result[index] = string(symbol)
	}
	return result, nil
}

func symbolValue(value string) xdr.ScVal {
	symbol := xdr.ScSymbol(value)
	return xdr.ScVal{Type: xdr.ScValTypeScvSymbol, Sym: &symbol}
}

func u32Value(value uint32) xdr.ScVal {
	u32 := xdr.Uint32(value)
	return xdr.ScVal{Type: xdr.ScValTypeScvU32, U32: &u32}
}

func measuredWasmHash(value string, reason string) model.Measured[string] {
	if value == "" {
		return model.CannotMeasure[string](reason)
	}
	return model.Measure(value)
}

func addMeasuredReason[T any](reasons map[string]string, path string, measured model.Measured[T]) {
	if measured.Unmeasurable {
		reasons[path] = measured.Reason
	}
}

func maxLedger(left uint32, right uint32) uint32 {
	if right > left {
		return right
	}
	return left
}

func uniqueStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	write := 1
	for read := 1; read < len(values); read++ {
		if values[read] == values[write-1] {
			continue
		}
		values[write] = values[read]
		write++
	}
	return values[:write]
}
