// Package observe reads contract state without submitting transactions.
package observe

import (
	"context"
	"fmt"

	"github.com/Controlwatch/control-watch/pkg/rpcx"
	"github.com/Controlwatch/control-watch/pkg/surface"
	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stellar/go-stellar-sdk/txnbuild"
	"github.com/stellar/go-stellar-sdk/xdr"
)

const maximumSymbolLength = 32

type getterResult struct {
	value        xdr.ScVal
	latestLedger uint32
}

// callGetter constructs an unsigned, sequence-zero invocation and evaluates it
// through read-only RPC simulation. It never signs or submits the transaction.
func callGetter(
	ctx context.Context,
	client rpcx.Client,
	sourceAccount string,
	contractID string,
	function surface.FunctionName,
	args ...xdr.ScVal,
) (getterResult, error) {
	if client == nil {
		return getterResult{}, fmt.Errorf("call getter: RPC client is required")
	}
	if _, err := strkey.Decode(strkey.VersionByteAccountID, sourceAccount); err != nil {
		return getterResult{}, fmt.Errorf("call getter: invalid source account: %w", err)
	}
	contractBytes, err := strkey.Decode(strkey.VersionByteContract, contractID)
	if err != nil {
		return getterResult{}, fmt.Errorf("call getter: invalid contract ID: %w", err)
	}
	if len(function) == 0 || len(function) > maximumSymbolLength {
		return getterResult{}, fmt.Errorf("call getter: function name must contain 1 to %d bytes", maximumSymbolLength)
	}

	var contract xdr.ContractId
	copy(contract[:], contractBytes)
	invocationArgs := append(xdr.ScVec(nil), args...)
	operation := &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeContract: &xdr.InvokeContractArgs{
				ContractAddress: xdr.ScAddress{
					Type:       xdr.ScAddressTypeScAddressTypeContract,
					ContractId: &contract,
				},
				FunctionName: xdr.ScSymbol(function),
				Args:         invocationArgs,
			},
		},
	}

	source := txnbuild.NewSimpleAccount(sourceAccount, -1)
	transaction, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount:        &source,
		IncrementSequenceNum: true,
		Operations:           []txnbuild.Operation{operation},
		BaseFee:              txnbuild.MinBaseFee,
		Preconditions: txnbuild.Preconditions{
			TimeBounds: txnbuild.NewInfiniteTimeout(),
		},
	})
	if err != nil {
		return getterResult{}, fmt.Errorf("call getter %s: build unsigned invocation: %w", function, err)
	}
	transactionXDR, err := transaction.Base64()
	if err != nil {
		return getterResult{}, fmt.Errorf("call getter %s: encode unsigned invocation: %w", function, err)
	}

	simulation, err := client.SimulateTransaction(ctx, transactionXDR)
	if err != nil {
		return getterResult{}, fmt.Errorf("call getter %s: simulate invocation: %w", function, err)
	}
	if simulation.Error != "" {
		return getterResult{}, fmt.Errorf("call getter %s: simulation failed: %s", function, simulation.Error)
	}
	if len(simulation.Results) != 1 {
		return getterResult{}, fmt.Errorf("call getter %s: simulation returned %d results, want 1", function, len(simulation.Results))
	}
	if simulation.Results[0].ReturnXDR == "" {
		return getterResult{}, fmt.Errorf("call getter %s: simulation returned an empty value", function)
	}

	var value xdr.ScVal
	if err := xdr.SafeUnmarshalBase64(simulation.Results[0].ReturnXDR, &value); err != nil {
		return getterResult{}, fmt.Errorf("call getter %s: decode return value: %w", function, err)
	}
	return getterResult{value: value, latestLedger: simulation.LatestLedger}, nil
}
