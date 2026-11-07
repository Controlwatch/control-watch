package observe

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/rpcx"
	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stellar/go-stellar-sdk/xdr"
)

type contractInstance struct {
	executable         xdr.ContractExecutable
	ledgerEntry        xdr.LedgerEntry
	latestLedger       uint32
	lastModifiedLedger uint32
}

func (instance contractInstance) wasmHash() (string, bool) {
	hash, ok := instance.executable.GetWasmHash()
	if !ok {
		return "", false
	}
	return hex.EncodeToString(hash[:]), true
}

func readContractInstance(
	ctx context.Context,
	client rpcx.Client,
	contractID string,
) (contractInstance, error) {
	if client == nil {
		return contractInstance{}, fmt.Errorf("read contract instance: RPC client is required")
	}
	contractBytes, err := strkey.Decode(strkey.VersionByteContract, contractID)
	if err != nil {
		return contractInstance{}, fmt.Errorf("read contract instance: invalid contract ID: %w", err)
	}

	var contract xdr.ContractId
	copy(contract[:], contractBytes)
	keyXDR, err := rpcx.ContractInstanceKey(contractID)
	if err != nil {
		return contractInstance{}, fmt.Errorf("read contract instance: %w", err)
	}

	response, err := client.GetLedgerEntries(ctx, []string{keyXDR})
	if err != nil {
		return contractInstance{}, fmt.Errorf("read contract instance: get ledger entry: %w", err)
	}
	if len(response.Entries) == 0 {
		return contractInstance{}, model.NewError(
			model.CodeContractNotFound,
			fmt.Errorf("read contract instance: RPC returned no entry"),
		)
	}
	if len(response.Entries) != 1 {
		return contractInstance{}, fmt.Errorf(
			"read contract instance: RPC returned %d entries, want 1",
			len(response.Entries),
		)
	}
	entry := response.Entries[0]
	if entry.Key != keyXDR {
		return contractInstance{}, fmt.Errorf("read contract instance: RPC returned an unexpected ledger key")
	}

	var entryData xdr.LedgerEntryData
	if err := xdr.SafeUnmarshalBase64(entry.XDR, &entryData); err != nil {
		return contractInstance{}, fmt.Errorf("read contract instance: decode ledger entry: %w", err)
	}
	contractData, ok := entryData.GetContractData()
	if !ok {
		return contractInstance{}, fmt.Errorf("read contract instance: ledger entry is not contract data")
	}
	entryContract, ok := contractData.Contract.GetContractId()
	if !ok || entryContract != contract {
		return contractInstance{}, fmt.Errorf("read contract instance: entry belongs to an unexpected contract")
	}
	if contractData.Key.Type != xdr.ScValTypeScvLedgerKeyContractInstance {
		return contractInstance{}, fmt.Errorf("read contract instance: entry has an unexpected contract-data key")
	}
	if contractData.Durability != xdr.ContractDataDurabilityPersistent {
		return contractInstance{}, fmt.Errorf("read contract instance: entry is not persistent")
	}
	instance, ok := contractData.Val.GetInstance()
	if !ok {
		return contractInstance{}, fmt.Errorf("read contract instance: contract data is not an instance value")
	}

	return contractInstance{
		executable: instance.Executable,
		ledgerEntry: xdr.LedgerEntry{
			LastModifiedLedgerSeq: xdr.Uint32(entry.LastModifiedLedger),
			Data:                  entryData,
		},
		latestLedger:       response.LatestLedger,
		lastModifiedLedger: entry.LastModifiedLedger,
	}, nil
}
