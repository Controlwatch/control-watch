package observe

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/Controlwatch/control-watch/pkg/rpcx"
	"github.com/Controlwatch/control-watch/pkg/surface"
	"github.com/stellar/go-stellar-sdk/xdr"
)

const (
	contractSpecSection = "contractspecv0"
	wasmHeader          = "\x00asm\x01\x00\x00\x00"
)

func readContractCode(ctx context.Context, client rpcx.Client, hash xdr.Hash) ([]byte, uint32, error) {
	key := xdr.LedgerKey{
		Type: xdr.LedgerEntryTypeContractCode,
		ContractCode: &xdr.LedgerKeyContractCode{
			Hash: hash,
		},
	}
	keyXDR, err := xdr.MarshalBase64(key)
	if err != nil {
		return nil, 0, fmt.Errorf("read contract code: encode ledger key: %w", err)
	}
	response, err := client.GetLedgerEntries(ctx, []string{keyXDR})
	if err != nil {
		return nil, 0, fmt.Errorf("read contract code: get ledger entry: %w", err)
	}
	if len(response.Entries) != 1 {
		return nil, response.LatestLedger, fmt.Errorf(
			"read contract code: RPC returned %d entries, want 1",
			len(response.Entries),
		)
	}
	entry := response.Entries[0]
	if entry.Key != keyXDR {
		return nil, response.LatestLedger, fmt.Errorf("read contract code: RPC returned an unexpected ledger key")
	}

	var entryData xdr.LedgerEntryData
	if err := xdr.SafeUnmarshalBase64(entry.XDR, &entryData); err != nil {
		return nil, response.LatestLedger, fmt.Errorf("read contract code: decode ledger entry: %w", err)
	}
	codeEntry, ok := entryData.GetContractCode()
	if !ok {
		return nil, response.LatestLedger, fmt.Errorf("read contract code: ledger entry is not contract code")
	}
	if codeEntry.Hash != hash {
		return nil, response.LatestLedger, fmt.Errorf("read contract code: ledger entry has an unexpected Wasm hash")
	}
	return append([]byte(nil), codeEntry.Code...), response.LatestLedger, nil
}

func exportedFunctions(wasm []byte) (map[surface.FunctionName]struct{}, error) {
	if len(wasm) < len(wasmHeader) || string(wasm[:len(wasmHeader)]) != wasmHeader {
		return nil, fmt.Errorf("parse contract Wasm: invalid header")
	}

	remaining := wasm[len(wasmHeader):]
	for len(remaining) > 0 {
		sectionID := remaining[0]
		remaining = remaining[1:]
		sectionSize, consumed, err := decodeWasmUint32(remaining)
		if err != nil {
			return nil, fmt.Errorf("parse contract Wasm: section size: %w", err)
		}
		remaining = remaining[consumed:]
		if uint64(sectionSize) > uint64(len(remaining)) {
			return nil, fmt.Errorf("parse contract Wasm: section length exceeds remaining bytes")
		}
		section := remaining[:sectionSize]
		remaining = remaining[sectionSize:]
		if sectionID != 0 {
			continue
		}

		nameLength, nameBytes, err := decodeWasmUint32(section)
		if err != nil {
			return nil, fmt.Errorf("parse contract Wasm: custom section name: %w", err)
		}
		if uint64(nameLength) > uint64(len(section)-nameBytes) {
			return nil, fmt.Errorf("parse contract Wasm: custom section name exceeds section length")
		}
		nameEnd := nameBytes + int(nameLength)
		if string(section[nameBytes:nameEnd]) != contractSpecSection {
			continue
		}
		return decodeContractSpec(section[nameEnd:])
	}
	return nil, fmt.Errorf("parse contract Wasm: %s section is missing", contractSpecSection)
}

func decodeContractSpec(spec []byte) (map[surface.FunctionName]struct{}, error) {
	reader := bytes.NewReader(spec)
	functions := make(map[surface.FunctionName]struct{})
	for reader.Len() > 0 {
		var entry xdr.ScSpecEntry
		if _, err := xdr.Unmarshal(reader, &entry); err != nil {
			return nil, fmt.Errorf("decode %s entry: %w", contractSpecSection, err)
		}
		function, ok := entry.GetFunctionV0()
		if !ok {
			continue
		}
		name := surface.FunctionName(function.Name)
		if name == "" {
			return nil, fmt.Errorf("decode %s entry: empty function name", contractSpecSection)
		}
		functions[name] = struct{}{}
	}
	return functions, nil
}

func decodeWasmUint32(data []byte) (uint32, int, error) {
	var encoded [binary.MaxVarintLen32]byte
	limit := len(encoded)
	if len(data) < limit {
		limit = len(data)
	}
	copy(encoded[:], data[:limit])
	value, consumed := binary.Uvarint(encoded[:limit])
	if consumed == 0 {
		return 0, 0, io.ErrUnexpectedEOF
	}
	if consumed < 0 || value > uint64(^uint32(0)) {
		return 0, 0, fmt.Errorf("unsigned LEB128 value overflows uint32")
	}
	return uint32(value), consumed, nil
}
