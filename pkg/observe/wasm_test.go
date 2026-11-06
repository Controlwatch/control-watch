package observe

import (
	"bytes"
	"context"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/rpcx"
	"github.com/Controlwatch/control-watch/pkg/surface"
	"github.com/stellar/go-stellar-sdk/xdr"
)

func TestExportedFunctionsReadsContractSpec(t *testing.T) {
	t.Parallel()

	wasm := wasmWithFunctions(t, []surface.FunctionName{
		surface.FunctionTransfer,
		surface.FunctionPaused,
	})
	functions, err := exportedFunctions(wasm)
	if err != nil {
		t.Fatalf("exportedFunctions returned error: %v", err)
	}
	if len(functions) != 2 {
		t.Fatalf("function count = %d, want 2", len(functions))
	}
	for _, function := range []surface.FunctionName{surface.FunctionTransfer, surface.FunctionPaused} {
		if _, ok := functions[function]; !ok {
			t.Errorf("function set does not contain %q", function)
		}
	}
}

func TestExportedFunctionsRejectsMalformedWasm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		wasm      []byte
		wantError string
	}{
		{name: "invalid header", wasm: []byte("not-wasm"), wantError: "invalid header"},
		{name: "truncated section size", wasm: append(append([]byte{}, []byte(wasmHeader)...), 0, 0x80), wantError: "unexpected EOF"},
		{name: "oversized section", wasm: append(append([]byte{}, []byte(wasmHeader)...), 0, 10, 1), wantError: "exceeds remaining bytes"},
		{name: "missing spec", wasm: append(append([]byte{}, []byte(wasmHeader)...), 1, 0), wantError: "section is missing"},
		{name: "malformed spec", wasm: wasmWithContractSpec([]byte{0xff}), wantError: "decode contractspecv0 entry"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := exportedFunctions(test.wasm)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error = %v, want substring %q", err, test.wantError)
			}
		})
	}
}

func TestReadContractCodeUsesHashLedgerKey(t *testing.T) {
	t.Parallel()

	var hash xdr.Hash
	for index := range hash {
		hash[index] = byte(31 - index)
	}
	wantCode := wasmWithFunctions(t, []surface.FunctionName{surface.FunctionBalance})
	fake := &rpcx.FakeClient{
		GetLedgerEntriesFunc: func(_ context.Context, keys []string) (rpcx.LedgerEntries, error) {
			if len(keys) != 1 {
				t.Fatalf("key count = %d, want 1", len(keys))
			}
			var key xdr.LedgerKey
			if err := xdr.SafeUnmarshalBase64(keys[0], &key); err != nil {
				t.Fatalf("decode code key: %v", err)
			}
			contractCode, ok := key.GetContractCode()
			if !ok || contractCode.Hash != hash {
				t.Fatalf("ledger key = %#v, want contract-code hash", key)
			}
			entryXDR, err := xdr.MarshalBase64(xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeContractCode,
				ContractCode: &xdr.ContractCodeEntry{
					Hash: hash,
					Code: wantCode,
				},
			})
			if err != nil {
				t.Fatalf("marshal code entry: %v", err)
			}
			return rpcx.LedgerEntries{
				LatestLedger: 789,
				Entries: []rpcx.LedgerEntry{{
					Key: keys[0],
					XDR: entryXDR,
				}},
			}, nil
		},
	}

	code, ledger, err := readContractCode(context.Background(), fake, hash)
	if err != nil {
		t.Fatalf("readContractCode returned error: %v", err)
	}
	if ledger != 789 {
		t.Fatalf("latest ledger = %d, want 789", ledger)
	}
	if !bytes.Equal(code, wantCode) {
		t.Fatal("returned Wasm does not match the ledger entry")
	}
}

func wasmWithFunctions(t *testing.T, functions []surface.FunctionName) []byte {
	t.Helper()

	var spec bytes.Buffer
	for _, function := range functions {
		entry := xdr.ScSpecEntry{
			Kind: xdr.ScSpecEntryKindScSpecEntryFunctionV0,
			FunctionV0: &xdr.ScSpecFunctionV0{
				Name: xdr.ScSymbol(function),
			},
		}
		if _, err := xdr.Marshal(&spec, entry); err != nil {
			t.Fatalf("marshal function spec: %v", err)
		}
	}
	return wasmWithContractSpec(spec.Bytes())
}

func wasmWithContractSpec(spec []byte) []byte {
	name := []byte(contractSpecSection)
	payload := append(encodeWasmUint32(uint32(len(name))), name...)
	payload = append(payload, spec...)
	wasm := append([]byte{}, []byte(wasmHeader)...)
	wasm = append(wasm, 0)
	wasm = append(wasm, encodeWasmUint32(uint32(len(payload)))...)
	return append(wasm, payload...)
}

func encodeWasmUint32(value uint32) []byte {
	var encoded [binary.MaxVarintLen32]byte
	length := binary.PutUvarint(encoded[:], uint64(value))
	return append([]byte(nil), encoded[:length]...)
}
