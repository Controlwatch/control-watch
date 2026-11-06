package observe

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/rpcx"
	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stellar/go-stellar-sdk/xdr"
)

func TestReadContractInstanceReturnsWasmHash(t *testing.T) {
	t.Parallel()

	var wantHash xdr.Hash
	for index := range wantHash {
		wantHash[index] = byte(index)
	}
	wantHashHex := "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

	fake := &rpcx.FakeClient{
		GetLedgerEntriesFunc: func(_ context.Context, keys []string) (rpcx.LedgerEntries, error) {
			if len(keys) != 1 {
				t.Fatalf("key count = %d, want 1", len(keys))
			}
			assertInstanceLedgerKey(t, keys[0], testContractID)
			return rpcx.LedgerEntries{
				LatestLedger: 456,
				Entries: []rpcx.LedgerEntry{{
					Key:                keys[0],
					XDR:                instanceEntryXDR(t, wasmExecutable(wantHash)),
					LastModifiedLedger: 123,
				}},
			}, nil
		},
	}

	instance, err := readContractInstance(context.Background(), fake, testContractID)
	if err != nil {
		t.Fatalf("readContractInstance returned error: %v", err)
	}
	if instance.latestLedger != 456 {
		t.Fatalf("latest ledger = %d, want 456", instance.latestLedger)
	}
	if instance.lastModifiedLedger != 123 {
		t.Fatalf("last modified ledger = %d, want 123", instance.lastModifiedLedger)
	}
	hash, ok := instance.wasmHash()
	if !ok {
		t.Fatal("wasmHash reported a non-Wasm executable")
	}
	if hash != wantHashHex {
		t.Fatalf("Wasm hash = %q, want %q", hash, wantHashHex)
	}
}

func TestContractInstanceDistinguishesStellarAssetExecutable(t *testing.T) {
	t.Parallel()

	instance := contractInstance{
		executable: xdr.ContractExecutable{
			Type: xdr.ContractExecutableTypeContractExecutableStellarAsset,
		},
	}
	if hash, ok := instance.wasmHash(); ok || hash != "" {
		t.Fatalf("wasmHash = (%q, %t), want empty and false", hash, ok)
	}
}

func TestReadContractInstanceRejectsInvalidResponses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		response  func(*testing.T, string) rpcx.LedgerEntries
		clientErr error
		wantError string
	}{
		{name: "transport", clientErr: errors.New("offline"), wantError: "get ledger entry: offline"},
		{name: "missing", response: func(*testing.T, string) rpcx.LedgerEntries { return rpcx.LedgerEntries{} }, wantError: "returned no entry"},
		{name: "multiple", response: func(*testing.T, string) rpcx.LedgerEntries {
			return rpcx.LedgerEntries{Entries: []rpcx.LedgerEntry{{}, {}}}
		}, wantError: "returned 2 entries"},
		{name: "wrong key", response: func(t *testing.T, _ string) rpcx.LedgerEntries {
			return rpcx.LedgerEntries{Entries: []rpcx.LedgerEntry{{Key: "wrong", XDR: instanceEntryXDR(t, stellarAssetExecutable())}}}
		}, wantError: "unexpected ledger key"},
		{name: "malformed entry", response: func(_ *testing.T, key string) rpcx.LedgerEntries {
			return rpcx.LedgerEntries{Entries: []rpcx.LedgerEntry{{Key: key, XDR: "not-xdr"}}}
		}, wantError: "decode ledger entry"},
		{name: "wrong entry type", response: func(t *testing.T, key string) rpcx.LedgerEntries {
			entryXDR, err := xdr.MarshalBase64(xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeTtl,
				Ttl:  &xdr.TtlEntry{LiveUntilLedgerSeq: 1},
			})
			if err != nil {
				t.Fatalf("marshal TTL entry: %v", err)
			}
			return rpcx.LedgerEntries{Entries: []rpcx.LedgerEntry{{Key: key, XDR: entryXDR}}}
		}, wantError: "not contract data"},
		{name: "wrong contract value", response: func(t *testing.T, key string) rpcx.LedgerEntries {
			entryXDR := contractEntryXDR(t, xdr.ScVal{Type: xdr.ScValTypeScvVoid})
			return rpcx.LedgerEntries{Entries: []rpcx.LedgerEntry{{Key: key, XDR: entryXDR}}}
		}, wantError: "not an instance value"},
		{name: "wrong embedded contract", response: func(t *testing.T, key string) rpcx.LedgerEntries {
			entryXDR := mutatedInstanceEntryXDR(t, func(entry *xdr.ContractDataEntry) {
				otherBytes, err := strkey.Decode(strkey.VersionByteContract, "CDJBZS26QTNWPEZPL5NQOQZ3QELLPH47QQN6HIIT56HJMUL2AWGMBATS")
				if err != nil {
					t.Fatalf("decode other contract: %v", err)
				}
				var other xdr.ContractId
				copy(other[:], otherBytes)
				entry.Contract.ContractId = &other
			})
			return rpcx.LedgerEntries{Entries: []rpcx.LedgerEntry{{Key: key, XDR: entryXDR}}}
		}, wantError: "unexpected contract"},
		{name: "wrong embedded key", response: func(t *testing.T, key string) rpcx.LedgerEntries {
			entryXDR := mutatedInstanceEntryXDR(t, func(entry *xdr.ContractDataEntry) {
				entry.Key = xdr.ScVal{Type: xdr.ScValTypeScvVoid}
			})
			return rpcx.LedgerEntries{Entries: []rpcx.LedgerEntry{{Key: key, XDR: entryXDR}}}
		}, wantError: "unexpected contract-data key"},
		{name: "temporary entry", response: func(t *testing.T, key string) rpcx.LedgerEntries {
			entryXDR := mutatedInstanceEntryXDR(t, func(entry *xdr.ContractDataEntry) {
				entry.Durability = xdr.ContractDataDurabilityTemporary
			})
			return rpcx.LedgerEntries{Entries: []rpcx.LedgerEntry{{Key: key, XDR: entryXDR}}}
		}, wantError: "not persistent"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			fake := &rpcx.FakeClient{
				GetLedgerEntriesFunc: func(_ context.Context, keys []string) (rpcx.LedgerEntries, error) {
					if test.clientErr != nil {
						return rpcx.LedgerEntries{}, test.clientErr
					}
					return test.response(t, keys[0]), nil
				},
			}
			_, err := readContractInstance(context.Background(), fake, testContractID)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error = %v, want substring %q", err, test.wantError)
			}
			if test.name == "missing" {
				code, ok := model.CodeOf(err)
				if !ok || code != model.CodeContractNotFound {
					t.Fatalf("error code = (%q, %t), want %q", code, ok, model.CodeContractNotFound)
				}
			}
		})
	}
}

func TestReadContractInstanceRejectsInvalidInputBeforeRPC(t *testing.T) {
	t.Parallel()

	if _, err := readContractInstance(context.Background(), nil, testContractID); err == nil || !strings.Contains(err.Error(), "RPC client is required") {
		t.Fatalf("nil-client error = %v", err)
	}
	fake := &rpcx.FakeClient{}
	if _, err := readContractInstance(context.Background(), fake, "invalid"); err == nil || !strings.Contains(err.Error(), "invalid contract ID") {
		t.Fatalf("invalid-contract error = %v", err)
	}
	if calls := fake.Calls(); len(calls.GetLedgerEntries) != 0 {
		t.Fatalf("ledger-entry calls = %d, want 0", len(calls.GetLedgerEntries))
	}
}

func assertInstanceLedgerKey(t *testing.T, keyXDR string, contractID string) {
	t.Helper()

	var key xdr.LedgerKey
	if err := xdr.SafeUnmarshalBase64(keyXDR, &key); err != nil {
		t.Fatalf("decode ledger key: %v", err)
	}
	contractData, ok := key.GetContractData()
	if !ok {
		t.Fatal("ledger key is not contract data")
	}
	if contractData.Durability != xdr.ContractDataDurabilityPersistent {
		t.Fatalf("durability = %s, want persistent", contractData.Durability)
	}
	if contractData.Key.Type != xdr.ScValTypeScvLedgerKeyContractInstance {
		t.Fatalf("key type = %s, want contract instance", contractData.Key.Type)
	}
	contract, ok := contractData.Contract.GetContractId()
	if !ok {
		t.Fatal("contract address is not a contract ID")
	}
	wantBytes, err := strkey.Decode(strkey.VersionByteContract, contractID)
	if err != nil {
		t.Fatalf("decode expected contract ID: %v", err)
	}
	if string(contract[:]) != string(wantBytes) {
		t.Fatalf("contract bytes do not match %s", contractID)
	}
}

func instanceEntryXDR(t *testing.T, executable xdr.ContractExecutable) string {
	t.Helper()

	return contractEntryXDR(t, xdr.ScVal{
		Type: xdr.ScValTypeScvContractInstance,
		Instance: &xdr.ScContractInstance{
			Executable: executable,
		},
	})
}

func contractEntryXDR(t *testing.T, value xdr.ScVal) string {
	t.Helper()

	contractBytes, err := strkey.Decode(strkey.VersionByteContract, testContractID)
	if err != nil {
		t.Fatalf("decode fixture contract ID: %v", err)
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
			Val:        value,
		},
	})
	if err != nil {
		t.Fatalf("marshal instance entry: %v", err)
	}
	return entryXDR
}

func wasmExecutable(hash xdr.Hash) xdr.ContractExecutable {
	return xdr.ContractExecutable{
		Type:     xdr.ContractExecutableTypeContractExecutableWasm,
		WasmHash: &hash,
	}
}

func stellarAssetExecutable() xdr.ContractExecutable {
	return xdr.ContractExecutable{
		Type: xdr.ContractExecutableTypeContractExecutableStellarAsset,
	}
}

func mutatedInstanceEntryXDR(t *testing.T, mutate func(*xdr.ContractDataEntry)) string {
	t.Helper()

	var entry xdr.LedgerEntryData
	if err := xdr.SafeUnmarshalBase64(instanceEntryXDR(t, stellarAssetExecutable()), &entry); err != nil {
		t.Fatalf("decode instance fixture: %v", err)
	}
	mutate(entry.ContractData)
	encoded, err := xdr.MarshalBase64(entry)
	if err != nil {
		t.Fatalf("marshal mutated instance fixture: %v", err)
	}
	return encoded
}
