package rpcx

import (
	"testing"

	"github.com/stellar/go-stellar-sdk/xdr"
)

const ledgerKeyTestContract = "CAI26Z4CPI5XQ34NEEN64ISTCHPNU733HD6AQQYQIA6NITB4E2AYYHHK"

func TestContractInstanceKey(t *testing.T) {
	t.Parallel()

	encoded, err := ContractInstanceKey(ledgerKeyTestContract)
	if err != nil {
		t.Fatalf("ContractInstanceKey() error = %v", err)
	}
	var key xdr.LedgerKey
	if err := xdr.SafeUnmarshalBase64(encoded, &key); err != nil {
		t.Fatalf("decode key: %v", err)
	}
	contractData, ok := key.GetContractData()
	if !ok {
		t.Fatalf("key type = %s, want contract data", key.Type)
	}
	if contractData.Key.Type != xdr.ScValTypeScvLedgerKeyContractInstance {
		t.Fatalf("data key type = %s", contractData.Key.Type)
	}
	if contractData.Durability != xdr.ContractDataDurabilityPersistent {
		t.Fatalf("durability = %s", contractData.Durability)
	}
}

func TestContractInstanceKeyRejectsInvalidContract(t *testing.T) {
	t.Parallel()

	if _, err := ContractInstanceKey("invalid"); err == nil {
		t.Fatal("ContractInstanceKey() error = nil")
	}
}
