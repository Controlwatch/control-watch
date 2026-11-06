package observe

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/rpcx"
	"github.com/Controlwatch/control-watch/pkg/surface"
	"github.com/stellar/go-stellar-sdk/xdr"
)

const (
	testSourceAccount = "GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF"
	testContractID    = "CAI26Z4CPI5XQ34NEEN64ISTCHPNU733HD6AQQYQIA6NITB4E2AYYHHK"
)

func TestCallGetterSimulatesUnsignedInvocation(t *testing.T) {
	t.Parallel()

	wantValue := xdr.ScVal{Type: xdr.ScValTypeScvBool, B: boolPointer(true)}
	returnXDR, err := xdr.MarshalBase64(wantValue)
	if err != nil {
		t.Fatalf("marshal return value: %v", err)
	}
	wantArg := xdr.ScVal{Type: xdr.ScValTypeScvU32, U32: uint32Pointer(7)}

	fake := &rpcx.FakeClient{
		SimulateTransactionFunc: func(_ context.Context, transactionXDR string) (rpcx.Simulation, error) {
			var envelope xdr.TransactionEnvelope
			if err := xdr.SafeUnmarshalBase64(transactionXDR, &envelope); err != nil {
				t.Fatalf("decode simulated transaction: %v", err)
			}
			if envelope.Type != xdr.EnvelopeTypeEnvelopeTypeTx || envelope.V1 == nil {
				t.Fatalf("envelope type = %s, want transaction", envelope.Type)
			}
			if len(envelope.V1.Signatures) != 0 {
				t.Fatalf("signature count = %d, want 0", len(envelope.V1.Signatures))
			}
			if envelope.V1.Tx.SeqNum != 0 {
				t.Fatalf("sequence = %d, want 0", envelope.V1.Tx.SeqNum)
			}
			if len(envelope.V1.Tx.Operations) != 1 {
				t.Fatalf("operation count = %d, want 1", len(envelope.V1.Tx.Operations))
			}
			invoke, ok := envelope.V1.Tx.Operations[0].Body.GetInvokeHostFunctionOp()
			if !ok {
				t.Fatal("operation is not an invoke-host-function operation")
			}
			contractCall, ok := invoke.HostFunction.GetInvokeContract()
			if !ok {
				t.Fatal("host function is not a contract invocation")
			}
			if contractCall.FunctionName != xdr.ScSymbol(surface.FunctionGetRoleMember) {
				t.Fatalf("function = %q, want %q", contractCall.FunctionName, surface.FunctionGetRoleMember)
			}
			if len(contractCall.Args) != 1 || !reflect.DeepEqual(contractCall.Args[0], wantArg) {
				t.Fatalf("arguments = %#v, want %#v", contractCall.Args, xdr.ScVec{wantArg})
			}

			return rpcx.Simulation{
				LatestLedger: 1234,
				Results:      []rpcx.SimulationResult{{ReturnXDR: returnXDR}},
			}, nil
		},
	}

	result, err := callGetter(
		context.Background(),
		fake,
		testSourceAccount,
		testContractID,
		surface.FunctionGetRoleMember,
		wantArg,
	)
	if err != nil {
		t.Fatalf("callGetter returned error: %v", err)
	}
	if !reflect.DeepEqual(result.value, wantValue) {
		t.Fatalf("value = %#v, want %#v", result.value, wantValue)
	}
	if result.latestLedger != 1234 {
		t.Fatalf("latest ledger = %d, want 1234", result.latestLedger)
	}
	if calls := fake.Calls(); len(calls.SimulateTransaction) != 1 {
		t.Fatalf("simulation calls = %d, want 1", len(calls.SimulateTransaction))
	}
}

func TestCallGetterRejectsInvalidInputsBeforeSimulation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		client     rpcx.Client
		source     string
		contractID string
		function   surface.FunctionName
		wantError  string
	}{
		{name: "missing client", source: testSourceAccount, contractID: testContractID, function: surface.FunctionPaused, wantError: "RPC client is required"},
		{name: "invalid source", client: &rpcx.FakeClient{}, source: "invalid", contractID: testContractID, function: surface.FunctionPaused, wantError: "invalid source account"},
		{name: "invalid contract", client: &rpcx.FakeClient{}, source: testSourceAccount, contractID: "invalid", function: surface.FunctionPaused, wantError: "invalid contract ID"},
		{name: "empty function", client: &rpcx.FakeClient{}, source: testSourceAccount, contractID: testContractID, wantError: "function name must contain"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := callGetter(context.Background(), test.client, test.source, test.contractID, test.function)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error = %v, want substring %q", err, test.wantError)
			}
			if fake, ok := test.client.(*rpcx.FakeClient); ok {
				if calls := fake.Calls(); len(calls.SimulateTransaction) != 0 {
					t.Fatalf("simulation calls = %d, want 0", len(calls.SimulateTransaction))
				}
			}
		})
	}
}

func TestCallGetterReportsSimulationFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		response  rpcx.Simulation
		clientErr error
		wantError string
	}{
		{name: "transport", clientErr: errors.New("offline"), wantError: "simulate invocation: offline"},
		{name: "contract error", response: rpcx.Simulation{Error: "HostError"}, wantError: "simulation failed: HostError"},
		{name: "no result", response: rpcx.Simulation{}, wantError: "returned 0 results"},
		{name: "multiple results", response: rpcx.Simulation{Results: []rpcx.SimulationResult{{}, {}}}, wantError: "returned 2 results"},
		{name: "empty return", response: rpcx.Simulation{Results: []rpcx.SimulationResult{{}}}, wantError: "returned an empty value"},
		{name: "malformed return", response: rpcx.Simulation{Results: []rpcx.SimulationResult{{ReturnXDR: "not-xdr"}}}, wantError: "decode return value"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			fake := &rpcx.FakeClient{
				SimulateTransactionFunc: func(context.Context, string) (rpcx.Simulation, error) {
					return test.response, test.clientErr
				},
			}
			_, err := callGetter(
				context.Background(),
				fake,
				testSourceAccount,
				testContractID,
				surface.FunctionPaused,
			)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error = %v, want substring %q", err, test.wantError)
			}
		})
	}
}

func boolPointer(value bool) *bool {
	return &value
}

func uint32Pointer(value xdr.Uint32) *xdr.Uint32 {
	return &value
}
