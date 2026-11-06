package observe

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/Controlwatch/control-watch/pkg/rpcx"
	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stellar/go-stellar-sdk/xdr"
)

type contractInstance struct {
	executable         xdr.ContractExecutable
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
	key := xdr.LedgerKey{
		Type: xdr.LedgerEntryTypeContractData,
		ContractData: &xdr.LedgerKeyContractData{
			Contract: xdr.ScAddress{
				Type:       xdr.ScAddressTypeScAddressTypeContract,
				ContractId: &contract,
			},
			Key: xdr.ScVal{
				Type: xdr.ScValTypeScvLedgerKeyContractInstance,
			},
			Durability: xdr.ContractDataDurabilityPersistent,
		},
	}
	keyXDR, err := xdr.MarshalBase64(key)
	if err != nil {
		return contractInstance{}, fmt.Errorf("read contract instance: encode ledger key: %w", err)
	}

	response, err := client.GetLedgerEntries(ctx, []string{keyXDR})
	if err != nil {
		return contractInstance{}, fmt.Errorf("read contract instance: get ledger entry: %w", err)
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
	instance, ok := contractData.Val.GetInstance()
	if !ok {
		return contractInstance{}, fmt.Errorf("read contract instance: contract data is not an instance value")
	}

	return contractInstance{
		executable:         instance.Executable,
		latestLedger:       response.LatestLedger,
		lastModifiedLedger: entry.LastModifiedLedger,
	}, nil
}
