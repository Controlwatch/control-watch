package rpcx

import (
	"fmt"

	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stellar/go-stellar-sdk/xdr"
)

// ContractInstanceKey returns the base64-XDR persistent instance key for a
// contract C-address.
func ContractInstanceKey(contractID string) (string, error) {
	contractBytes, err := strkey.Decode(strkey.VersionByteContract, contractID)
	if err != nil {
		return "", fmt.Errorf("contract instance key: invalid contract ID: %w", err)
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
	encoded, err := xdr.MarshalBase64(key)
	if err != nil {
		return "", fmt.Errorf("contract instance key: encode: %w", err)
	}
	return encoded, nil
}
