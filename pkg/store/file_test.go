package store

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/canon"
	"github.com/Controlwatch/control-watch/pkg/model"
)

const storeTestContractID = "CAI26Z4CPI5XQ34NEEN64ISTCHPNU733HD6AQQYQIA6NITB4E2AYYHHK"

func TestFileStoreSaveLoadAndList(t *testing.T) {
	t.Parallel()

	store := mustFileStore(t)
	older := testSnapshot(41)
	newer := testSnapshot(42)
	olderRef, err := store.Save(context.Background(), older)
	if err != nil {
		t.Fatalf("Save older returned error: %v", err)
	}
	newerRef, err := store.Save(context.Background(), newer)
	if err != nil {
		t.Fatalf("Save newer returned error: %v", err)
	}
	if olderRef.ID == newerRef.ID {
		t.Fatalf("different ledgers produced the same ID %q", olderRef.ID)
	}

	loaded, err := store.Load(context.Background(), newer.Network, newer.ContractID, newerRef.ID)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.ConfigHash == "" || loaded.SnapshotHash == "" {
		t.Fatalf("loaded hashes are empty: %#v", loaded)
	}
	wantConfigHash, err := canon.Hash(newer.Observation)
	if err != nil {
		t.Fatalf("hash observation: %v", err)
	}
	if loaded.ConfigHash != wantConfigHash {
		t.Fatalf("config hash = %q, want %q", loaded.ConfigHash, wantConfigHash)
	}

	references, err := store.List(context.Background(), newer.Network, newer.ContractID)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	wantIDs := []string{newerRef.ID, olderRef.ID}
	gotIDs := []string{references[0].ID, references[1].ID}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("list IDs = %#v, want %#v", gotIDs, wantIDs)
	}

	path := store.snapshotPath(newer, newerRef)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat snapshot: %v", err)
	}
	if info.Mode().Perm()&0o222 != 0 {
		t.Fatalf("snapshot permissions = %o, want no write bits", info.Mode().Perm())
	}
}

func TestFileStoreExclusiveCreateAllowsOnlyOneWriter(t *testing.T) {
	t.Parallel()

	store := mustFileStore(t)
	snapshot := testSnapshot(51)
	start := make(chan struct{})
	errorsByWriter := make([]error, 2)
	var wait sync.WaitGroup
	for index := range errorsByWriter {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			_, errorsByWriter[index] = store.Save(context.Background(), snapshot)
		}(index)
	}
	close(start)
	wait.Wait()

	successes := 0
	failures := 0
	for _, err := range errorsByWriter {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}
	if successes != 1 || failures != 1 {
		t.Fatalf("save outcomes = %d successes, %d failures; want one each", successes, failures)
	}
	references, err := store.List(context.Background(), snapshot.Network, snapshot.ContractID)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(references) != 1 {
		t.Fatalf("snapshot count = %d, want 1", len(references))
	}
}

func TestFileStoreDetectsCorruption(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{
			name: "tampered semantic value",
			mutate: func(data []byte) []byte {
				return bytes.Replace(data, []byte(`"admin":"G`), []byte(`"admin":"C`), 1)
			},
		},
		{name: "malformed JSON", mutate: func([]byte) []byte { return []byte("{") }},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			store := mustFileStore(t)
			snapshot := testSnapshot(61)
			reference, err := store.Save(context.Background(), snapshot)
			if err != nil {
				t.Fatalf("Save returned error: %v", err)
			}
			path := store.snapshotPath(snapshot, reference)
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read snapshot: %v", err)
			}
			if err := os.Chmod(path, 0o644); err != nil {
				t.Fatalf("make test snapshot writable: %v", err)
			}
			if err := os.WriteFile(path, test.mutate(original), 0o644); err != nil {
				t.Fatalf("tamper snapshot: %v", err)
			}

			_, err = store.Load(context.Background(), snapshot.Network, snapshot.ContractID, reference.ID)
			assertErrorCode(t, err, model.CodeSnapshotCorrupt)
		})
	}
}

func TestFileStoreReturnsStableErrors(t *testing.T) {
	t.Parallel()

	store := mustFileStore(t)
	snapshot := testSnapshot(71)
	_, err := store.Load(context.Background(), snapshot.Network, snapshot.ContractID, "71-deadbeef")
	assertErrorCode(t, err, model.CodeSnapshotNotFound)

	snapshot.SchemaVersion = "2"
	_, err = store.Save(context.Background(), snapshot)
	assertErrorCode(t, err, model.CodeSchemaUnsupported)

	_, err = store.Load(context.Background(), snapshot.Network, "invalid", "71-deadbeef")
	assertErrorCode(t, err, model.CodeInvalidContractID)
}

func TestFileStoreHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	store := mustFileStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := store.Save(ctx, testSnapshot(81))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Save error = %v, want context canceled", err)
	}
	if _, statErr := os.Stat(store.root); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("store root exists after canceled save: %v", statErr)
	}
}

func mustFileStore(t *testing.T) *FileStore {
	t.Helper()
	store, err := NewFileStore(filepath.Join(t.TempDir(), "snapshots"))
	if err != nil {
		t.Fatalf("NewFileStore returned error: %v", err)
	}
	return store
}

func testSnapshot(ledger uint32) model.Snapshot {
	return model.Snapshot{
		SchemaVersion: model.SchemaVersion,
		ToolVersion:   "test",
		Network:       "testnet",
		ContractID:    storeTestContractID,
		LedgerSeq:     ledger,
		ObservedAt:    "2026-11-06T12:00:00Z",
		Completeness:  model.CompletenessComplete,
		Warnings:      []string{},
		Observation: model.Observation{
			Classification: model.ClassSAC,
			SAC: &model.SACSurface{
				Admin: model.Measure("GD3C2MXS3PT6JJSTOFH4FZGMJ7GRMN52PZQC73FP3JYOEDS4LBAKIFXB"),
			},
		},
	}
}

func (store *FileStore) snapshotPath(snapshot model.Snapshot, reference SnapshotRef) string {
	return filepath.Join(
		store.root,
		snapshot.Network,
		snapshot.ContractID,
		reference.ID+snapshotExtension,
	)
}

func assertErrorCode(t *testing.T, err error, want model.ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("error is nil, want code %q", want)
	}
	code, ok := model.CodeOf(err)
	if !ok || code != want {
		t.Fatalf("error code = (%q, %t), want %q; error = %v", code, ok, want, err)
	}
}
