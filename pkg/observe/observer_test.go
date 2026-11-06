package observe

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/network"
	"github.com/Controlwatch/control-watch/pkg/rpcx"
	"github.com/Controlwatch/control-watch/pkg/surface"
	sacingest "github.com/stellar/go-stellar-sdk/ingest/sac"
	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stellar/go-stellar-sdk/xdr"
)

const (
	testComplianceContract = "CDJBZS26QTNWPEZPL5NQOQZ3QELLPH47QQN6HIIT56HJMUL2AWGMBATS"
	testIdentityContract   = "CDFDYFIFX3RCOT3YDBM6GDTXLRZNHVDXTAAUBYUBE7GTOELVKMTDQMXK"
	testModuleContract     = "CD47GQOKNAN225WBO7ALLVFCATENZFKRTSDMBFRBF6NWZZGDMLFIPLFH"
	testAdminAccount       = "GD3C2MXS3PT6JJSTOFH4FZGMJ7GRMN52PZQC73FP3JYOEDS4LBAKIFXB"
	testRoleMember         = "GA52JOKWGMWMJWRQQHA5OWEBCLYDH5SGIN77D2OID4LDF7W7BE7BPUJH"
)

func TestObserverClassifiesAndReadsRWASurface(t *testing.T) {
	t.Parallel()

	fixture := newRWAObserverFixture(t)
	observer := mustTestObserver(t, fixture.client(), nil)
	result, err := observer.Observe(context.Background(), testContractID)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if result.Observation.Classification != model.ClassSEP57RWA {
		t.Fatalf("classification = %q, want %q", result.Observation.Classification, model.ClassSEP57RWA)
	}
	if result.Completeness != model.CompletenessComplete {
		t.Fatalf("completeness = %q, want complete; reasons = %#v", result.Completeness, result.Observation.Reasons)
	}
	if result.Observation.Token == nil || result.Observation.Token.Paused.Value {
		t.Fatalf("token surface = %#v, want unpaused token", result.Observation.Token)
	}
	if result.Observation.Token.Admin.Value != testAdminAccount {
		t.Fatalf("admin = %q, want %q", result.Observation.Token.Admin.Value, testAdminAccount)
	}
	members := result.Observation.Token.Roles["manager"]
	if members.Unmeasurable || !reflect.DeepEqual(members.Value, []string{testRoleMember}) {
		t.Fatalf("manager members = %#v", members)
	}
	if result.Observation.Compliance == nil || result.Observation.Compliance.Address != testComplianceContract {
		t.Fatalf("compliance = %#v", result.Observation.Compliance)
	}
	modules := result.Observation.Compliance.Modules
	if modules.Unmeasurable || len(modules.Value) != 1 {
		t.Fatalf("modules = %#v, want one measurable module", modules)
	}
	if got, want := modules.Value[0].Hooks, []string{"Created", "Destroyed", "Transferred"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("module hooks = %#v, want %#v", got, want)
	}
	if result.Observation.IdentityVerifier == nil || result.Observation.IdentityVerifier.Address != testIdentityContract {
		t.Fatalf("identity verifier = %#v", result.Observation.IdentityVerifier)
	}
}

func TestObserverClassifiesReducedWasmSurfaces(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		functions      []surface.FunctionName
		classification model.Classification
	}{
		{name: "SEP-41", functions: sep41Signature(), classification: model.ClassSEP41},
		{name: "unknown", functions: []surface.FunctionName{surface.FunctionPaused}, classification: model.ClassUnknown},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			hash := testHash(9)
			fixture := &observerRPCFixture{
				t: t,
				instances: map[string]xdr.ContractExecutable{
					testContractID: wasmExecutable(hash),
				},
				codes: map[xdr.Hash][]byte{hash: wasmWithFunctions(t, test.functions)},
			}
			observer := mustTestObserver(t, fixture.client(), nil)
			result, err := observer.Observe(context.Background(), testContractID)
			if err != nil {
				t.Fatalf("Observe returned error: %v", err)
			}
			if result.Observation.Classification != test.classification {
				t.Fatalf("classification = %q, want %q", result.Observation.Classification, test.classification)
			}
			if result.Observation.Token == nil || result.Observation.Token.WasmHash.Unmeasurable {
				t.Fatalf("token Wasm hash = %#v", result.Observation.Token)
			}
			if result.Completeness != model.CompletenessPartial {
				t.Fatalf("completeness = %q, want partial reduced surface", result.Completeness)
			}
		})
	}
}

func TestObserverReadsIssuedSACAndIssuerFlags(t *testing.T) {
	t.Parallel()

	asset, err := xdr.NewCreditAsset("USDC", testAdminAccount)
	if err != nil {
		t.Fatalf("construct classic asset: %v", err)
	}
	contractHash, err := asset.ContractID(network.TestnetPassphrase)
	if err != nil {
		t.Fatalf("derive SAC ID: %v", err)
	}
	contractID, err := strkey.Encode(strkey.VersionByteContract, contractHash[:])
	if err != nil {
		t.Fatalf("encode SAC ID: %v", err)
	}
	entryData, err := sacingest.AssetToContractData(false, "USDC", testAdminAccount, contractHash)
	if err != nil {
		t.Fatalf("construct SAC instance: %v", err)
	}
	entryXDR, err := xdr.MarshalBase64(entryData)
	if err != nil {
		t.Fatalf("marshal SAC instance: %v", err)
	}
	fixture := &observerRPCFixture{
		t:              t,
		specialEntries: map[string]string{contractID: entryXDR},
		simulationValues: map[surface.FunctionName]xdr.ScVal{
			surface.FunctionSACAdmin: addressValue(t, testAdminAccount),
		},
	}
	flags := &fakeClassicFlagsReader{
		wantAccount: testAdminAccount,
		flags: ClassicAccountFlags{
			AuthRequired:        true,
			AuthRevocable:       true,
			AuthClawbackEnabled: true,
			LastModifiedLedger:  88,
		},
	}
	observer := mustTestObserver(t, fixture.client(), flags)
	result, err := observer.Observe(context.Background(), contractID)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if result.Observation.Classification != model.ClassSAC || result.Completeness != model.CompletenessComplete {
		t.Fatalf("SAC result = %#v", result)
	}
	if result.Observation.SAC == nil || result.Observation.SAC.Admin.Value != testAdminAccount {
		t.Fatalf("SAC admin = %#v", result.Observation.SAC)
	}
	if result.Observation.ClassicFlags == nil || !result.Observation.ClassicFlags.AuthRequired.Value || !result.Observation.ClassicFlags.AuthClawbackEnabled.Value {
		t.Fatalf("classic flags = %#v", result.Observation.ClassicFlags)
	}
}

func TestObserverPreservesOtherRWAFieldsWhenOneGetterFails(t *testing.T) {
	t.Parallel()

	fixture := newRWAObserverFixture(t)
	fixture.simulationErrors[surface.FunctionPaused] = "HostError"
	observer := mustTestObserver(t, fixture.client(), nil)
	result, err := observer.Observe(context.Background(), testContractID)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if result.Completeness != model.CompletenessPartial {
		t.Fatalf("completeness = %q, want partial", result.Completeness)
	}
	if result.Observation.Token == nil || !result.Observation.Token.Paused.Unmeasurable {
		t.Fatalf("paused = %#v, want unmeasurable", result.Observation.Token)
	}
	if result.Observation.Token.Admin.Unmeasurable || result.Observation.Token.Admin.Value != testAdminAccount {
		t.Fatalf("admin = %#v, want independently measured", result.Observation.Token.Admin)
	}
	if _, ok := result.Observation.Reasons["token.paused"]; !ok {
		t.Fatalf("reasons = %#v, want token.paused", result.Observation.Reasons)
	}
	if !containsWarningPrefix(result.Warnings, "token.paused:") {
		t.Fatalf("warnings = %#v, want paused warning", result.Warnings)
	}
}

func TestObserverRecordsClassificationFallback(t *testing.T) {
	t.Parallel()

	fixture := newRWAObserverFixture(t)
	fixture.codes = nil
	observer := mustTestObserver(t, fixture.client(), nil)
	result, err := observer.Observe(context.Background(), testContractID)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if result.Observation.Classification != model.ClassSEP57RWA {
		t.Fatalf("classification = %q, want RWA fallback", result.Observation.Classification)
	}
	if !containsWarningText(result.Warnings, "getter simulation fallback") {
		t.Fatalf("warnings = %#v, want classification fallback warning", result.Warnings)
	}
}

type observerRPCFixture struct {
	t                *testing.T
	instances        map[string]xdr.ContractExecutable
	specialEntries   map[string]string
	codes            map[xdr.Hash][]byte
	simulationValues map[surface.FunctionName]xdr.ScVal
	simulationErrors map[surface.FunctionName]string
}

func newRWAObserverFixture(t *testing.T) *observerRPCFixture {
	t.Helper()
	tokenHash := testHash(1)
	return &observerRPCFixture{
		t: t,
		instances: map[string]xdr.ContractExecutable{
			testContractID:         wasmExecutable(tokenHash),
			testComplianceContract: wasmExecutable(testHash(2)),
			testIdentityContract:   wasmExecutable(testHash(3)),
			testModuleContract:     wasmExecutable(testHash(4)),
		},
		codes: map[xdr.Hash][]byte{
			tokenHash: wasmWithFunctions(
				t,
				append(append([]surface.FunctionName{}, sep41Signature()...), rwaSignature()...),
			),
		},
		simulationValues: map[surface.FunctionName]xdr.ScVal{
			surface.FunctionCompliance:         addressValue(t, testComplianceContract),
			surface.FunctionIdentityVerifier:   addressValue(t, testIdentityContract),
			surface.FunctionPaused:             boolValue(false),
			surface.FunctionGetAdmin:           addressValue(t, testAdminAccount),
			surface.FunctionGetExistingRoles:   vectorValue(symbolValue("manager")),
			surface.FunctionGetRoleMemberCount: u32Value(1),
			surface.FunctionGetRoleMember:      addressValue(t, testRoleMember),
			surface.FunctionGetModulesForHook:  vectorValue(addressValue(t, testModuleContract)),
		},
		simulationErrors: make(map[surface.FunctionName]string),
	}
}

func (fixture *observerRPCFixture) client() *rpcx.FakeClient {
	return &rpcx.FakeClient{
		GetLatestLedgerFunc: func(context.Context) (rpcx.LatestLedger, error) {
			return rpcx.LatestLedger{Sequence: 100}, nil
		},
		GetLedgerEntriesFunc: func(_ context.Context, keys []string) (rpcx.LedgerEntries, error) {
			if len(keys) != 1 {
				fixture.t.Fatalf("ledger key count = %d, want 1", len(keys))
			}
			return fixture.ledgerEntry(keys[0]), nil
		},
		SimulateTransactionFunc: func(_ context.Context, transactionXDR string) (rpcx.Simulation, error) {
			function := invocationFunction(fixture.t, transactionXDR)
			if detail := fixture.simulationErrors[function]; detail != "" {
				return rpcx.Simulation{LatestLedger: 103, Error: detail}, nil
			}
			value, ok := fixture.simulationValues[function]
			if !ok {
				return rpcx.Simulation{}, errors.New("unexpected getter " + string(function))
			}
			returnXDR, err := xdr.MarshalBase64(value)
			if err != nil {
				fixture.t.Fatalf("marshal getter result: %v", err)
			}
			return rpcx.Simulation{
				LatestLedger: 103,
				Results:      []rpcx.SimulationResult{{ReturnXDR: returnXDR}},
			}, nil
		},
	}
}

func (fixture *observerRPCFixture) ledgerEntry(keyXDR string) rpcx.LedgerEntries {
	var key xdr.LedgerKey
	if err := xdr.SafeUnmarshalBase64(keyXDR, &key); err != nil {
		fixture.t.Fatalf("decode ledger key: %v", err)
	}
	switch key.Type {
	case xdr.LedgerEntryTypeContractData:
		contract := key.ContractData.Contract.MustContractId()
		contractID, err := strkey.Encode(strkey.VersionByteContract, contract[:])
		if err != nil {
			fixture.t.Fatalf("encode requested contract: %v", err)
		}
		entryXDR := fixture.specialEntries[contractID]
		if entryXDR == "" {
			executable, ok := fixture.instances[contractID]
			if !ok {
				return rpcx.LedgerEntries{LatestLedger: 101}
			}
			entryXDR = instanceEntryForContract(fixture.t, contractID, executable)
		}
		return rpcx.LedgerEntries{
			LatestLedger: 101,
			Entries: []rpcx.LedgerEntry{{
				Key:                keyXDR,
				XDR:                entryXDR,
				LastModifiedLedger: 90,
			}},
		}
	case xdr.LedgerEntryTypeContractCode:
		code, ok := fixture.codes[key.ContractCode.Hash]
		if !ok {
			return rpcx.LedgerEntries{LatestLedger: 102}
		}
		entryXDR, err := xdr.MarshalBase64(xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeContractCode,
			ContractCode: &xdr.ContractCodeEntry{
				Hash: key.ContractCode.Hash,
				Code: code,
			},
		})
		if err != nil {
			fixture.t.Fatalf("marshal contract code entry: %v", err)
		}
		return rpcx.LedgerEntries{
			LatestLedger: 102,
			Entries:      []rpcx.LedgerEntry{{Key: keyXDR, XDR: entryXDR}},
		}
	default:
		fixture.t.Fatalf("unexpected ledger key type %s", key.Type)
		return rpcx.LedgerEntries{}
	}
}

type fakeClassicFlagsReader struct {
	wantAccount string
	flags       ClassicAccountFlags
	err         error
}

func (reader *fakeClassicFlagsReader) ReadClassicFlags(
	_ context.Context,
	accountID string,
) (ClassicAccountFlags, error) {
	if accountID != reader.wantAccount {
		return ClassicAccountFlags{}, errors.New("unexpected issuer account")
	}
	return reader.flags, reader.err
}

func mustTestObserver(t *testing.T, client rpcx.Client, flags ClassicFlagsReader) *Observer {
	t.Helper()
	observer, err := NewObserver(client, flags, testSourceAccount, network.TestnetPassphrase)
	if err != nil {
		t.Fatalf("NewObserver returned error: %v", err)
	}
	return observer
}

func instanceEntryForContract(
	t *testing.T,
	contractID string,
	executable xdr.ContractExecutable,
) string {
	t.Helper()
	contractBytes, err := strkey.Decode(strkey.VersionByteContract, contractID)
	if err != nil {
		t.Fatalf("decode fixture contract: %v", err)
	}
	var contract xdr.ContractId
	copy(contract[:], contractBytes)
	entryXDR, err := xdr.MarshalBase64(xdr.LedgerEntryData{
		Type: xdr.LedgerEntryTypeContractData,
		ContractData: &xdr.ContractDataEntry{
			Contract: xdr.ScAddress{
				Type:       xdr.ScAddressTypeScAddressTypeContract,
				ContractId: &contract,
			},
			Key:        xdr.ScVal{Type: xdr.ScValTypeScvLedgerKeyContractInstance},
			Durability: xdr.ContractDataDurabilityPersistent,
			Val: xdr.ScVal{
				Type: xdr.ScValTypeScvContractInstance,
				Instance: &xdr.ScContractInstance{
					Executable: executable,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal fixture instance: %v", err)
	}
	return entryXDR
}

func invocationFunction(t *testing.T, transactionXDR string) surface.FunctionName {
	t.Helper()
	var envelope xdr.TransactionEnvelope
	if err := xdr.SafeUnmarshalBase64(transactionXDR, &envelope); err != nil {
		t.Fatalf("decode simulated invocation: %v", err)
	}
	if envelope.V1 == nil || len(envelope.V1.Tx.Operations) != 1 {
		t.Fatal("simulated invocation does not contain one operation")
	}
	invoke, ok := envelope.V1.Tx.Operations[0].Body.GetInvokeHostFunctionOp()
	if !ok {
		t.Fatal("simulated operation is not invoke-host-function")
	}
	contractCall, ok := invoke.HostFunction.GetInvokeContract()
	if !ok {
		t.Fatal("simulated host function is not invoke-contract")
	}
	return surface.FunctionName(contractCall.FunctionName)
}

func addressValue(t *testing.T, address string) xdr.ScVal {
	t.Helper()
	version := strkey.VersionByteAccountID
	addressType := xdr.ScAddressTypeScAddressTypeAccount
	if strings.HasPrefix(address, "C") {
		version = strkey.VersionByteContract
		addressType = xdr.ScAddressTypeScAddressTypeContract
	}
	decoded, err := strkey.Decode(version, address)
	if err != nil {
		t.Fatalf("decode fixture address: %v", err)
	}
	result := xdr.ScAddress{Type: addressType}
	if addressType == xdr.ScAddressTypeScAddressTypeContract {
		var contract xdr.ContractId
		copy(contract[:], decoded)
		result.ContractId = &contract
	} else {
		var publicKey xdr.Uint256
		copy(publicKey[:], decoded)
		account := xdr.AccountId{
			Type:    xdr.PublicKeyTypePublicKeyTypeEd25519,
			Ed25519: &publicKey,
		}
		result.AccountId = &account
	}
	return xdr.ScVal{Type: xdr.ScValTypeScvAddress, Address: &result}
}

func boolValue(value bool) xdr.ScVal {
	return xdr.ScVal{Type: xdr.ScValTypeScvBool, B: &value}
}

func vectorValue(values ...xdr.ScVal) xdr.ScVal {
	vector := xdr.ScVec(values)
	vectorPointer := &vector
	return xdr.ScVal{Type: xdr.ScValTypeScvVec, Vec: &vectorPointer}
}

func testHash(seed byte) xdr.Hash {
	var hash xdr.Hash
	for index := range hash {
		hash[index] = seed + byte(index)
	}
	return hash
}

func containsWarningPrefix(warnings []string, prefix string) bool {
	for _, warning := range warnings {
		if strings.HasPrefix(warning, prefix) {
			return true
		}
	}
	return false
}

func containsWarningText(warnings []string, text string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, text) {
			return true
		}
	}
	return false
}
