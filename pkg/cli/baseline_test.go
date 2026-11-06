package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/store"
)

func TestBaselineListReadsLocalMainnetStoreWithoutRPC(t *testing.T) {
	t.Parallel()

	want := []store.SnapshotRef{
		{ID: "200-abcdef12", LedgerSeq: 200, ConfigHash: "abcdef12"},
		{ID: "100-12345678", LedgerSeq: 100, ConfigHash: "12345678"},
	}
	lister := &listingStore{references: want}
	app := testApp()
	var gotRoot string
	app.newStore = func(root string) (store.Store, error) {
		gotRoot = root
		return lister, nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := app.Run(context.Background(), []string{
		"baseline", "list",
		"--network", "mainnet",
		"--contract", cliTestContractID,
		"--store-dir", "baselines",
	}, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0; stderr = %s", exit, stderr.String())
	}
	if gotRoot != "baselines" {
		t.Fatalf("store root = %q, want baselines", gotRoot)
	}
	if lister.network != "mainnet" || lister.contractID != cliTestContractID {
		t.Fatalf("list scope = %q/%q", lister.network, lister.contractID)
	}
	var got []store.SnapshotRef
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("output = %#v, want %#v", got, want)
	}
}

func TestBaselineListRejectsInvalidContractBeforeConstructingStore(t *testing.T) {
	t.Parallel()

	app := testApp()
	constructed := false
	app.newStore = func(string) (store.Store, error) {
		constructed = true
		return nil, errors.New("must not run")
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := app.Run(context.Background(), []string{
		"baseline", "list", "--network", "testnet", "--contract", "invalid",
	}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	if constructed {
		t.Fatal("store was constructed for an invalid contract")
	}
	if !bytes.Contains(stderr.Bytes(), []byte(model.CodeInvalidContractID)) {
		t.Fatalf("stderr = %q, want invalid-contract code", stderr.String())
	}
}

func TestBaselineRequiresListSubcommand(t *testing.T) {
	t.Parallel()

	app := testApp()
	for _, args := range [][]string{{"baseline"}, {"baseline", "delete"}} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		if exit := app.Run(context.Background(), args, &stdout, &stderr); exit != 2 {
			t.Fatalf("args %v: exit = %d, want 2", args, exit)
		}
	}
}

type listingStore struct {
	references []store.SnapshotRef
	err        error
	network    string
	contractID string
}

func (*listingStore) Save(context.Context, model.Snapshot) (store.SnapshotRef, error) {
	return store.SnapshotRef{}, errors.New("not implemented")
}

func (*listingStore) Load(context.Context, string, string, string) (model.Snapshot, error) {
	return model.Snapshot{}, errors.New("not implemented")
}

func (listing *listingStore) List(_ context.Context, network string, contractID string) ([]store.SnapshotRef, error) {
	listing.network = network
	listing.contractID = contractID
	return listing.references, listing.err
}
