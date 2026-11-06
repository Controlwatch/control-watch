package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/network"
	"github.com/Controlwatch/control-watch/pkg/observe"
	"github.com/Controlwatch/control-watch/pkg/store"
)

const cliTestContractID = "CAI26Z4CPI5XQ34NEEN64ISTCHPNU733HD6AQQYQIA6NITB4E2AYYHHK"

func TestInspectWritesAndSavesHashedSnapshot(t *testing.T) {
	t.Parallel()

	outputPath := filepath.Join(t.TempDir(), "inspection.json")
	runner := &fakeObservationRunner{
		result: observe.Result{
			Observation: model.Observation{
				Classification: model.ClassSAC,
				SAC: &model.SACSurface{
					Admin: model.Measure("GD3C2MXS3PT6JJSTOFH4FZGMJ7GRMN52PZQC73FP3JYOEDS4LBAKIFXB"),
				},
			},
			LedgerSeq:    123,
			Completeness: model.CompletenessComplete,
			Warnings:     []string{},
		},
	}
	recordedStore := &recordingStore{}
	closed := false
	app := testApp()
	app.newObserver = func(config Config) (observationRunner, closeFunc, error) {
		if config.Network.Name != network.Testnet {
			t.Fatalf("network = %q, want testnet", config.Network.Name)
		}
		return runner, func() error { closed = true; return nil }, nil
	}
	app.newStore = func(string) (store.Store, error) { return recordedStore, nil }

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := app.Run(context.Background(), []string{
		"inspect",
		"--network", "testnet",
		"--contract", cliTestContractID,
		"--save",
		"--json", outputPath,
	}, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0; stderr = %s", exit, stderr.String())
	}
	if !closed {
		t.Fatal("observer was not closed")
	}
	if runner.contractID != cliTestContractID {
		t.Fatalf("observed contract = %q", runner.contractID)
	}
	if len(recordedStore.saved) != 1 {
		t.Fatalf("saved snapshots = %d, want 1", len(recordedStore.saved))
	}

	var snapshot model.Snapshot
	if err := json.Unmarshal(stdout.Bytes(), &snapshot); err != nil {
		t.Fatalf("decode stdout snapshot: %v", err)
	}
	if snapshot.ConfigHash == "" || snapshot.SnapshotHash == "" {
		t.Fatalf("snapshot hashes are empty: %#v", snapshot)
	}
	if snapshot.ObservedAt != "2026-11-06T12:00:00Z" || snapshot.ToolVersion != "test-version" {
		t.Fatalf("snapshot metadata = %#v", snapshot)
	}
	fileBytes, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read JSON output: %v", err)
	}
	if !bytes.Equal(fileBytes, stdout.Bytes()) {
		t.Fatal("JSON file and stdout differ")
	}
}

func TestInspectRejectsInvalidContractBeforeConstructingObserver(t *testing.T) {
	t.Parallel()

	app := testApp()
	constructed := false
	app.newObserver = func(Config) (observationRunner, closeFunc, error) {
		constructed = true
		return nil, nil, errors.New("must not run")
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := app.Run(context.Background(), []string{
		"inspect", "--network", "testnet", "--contract", "invalid",
	}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	if constructed {
		t.Fatal("observer was constructed for an invalid contract")
	}
	if code := model.CodeInvalidContractID; !bytes.Contains(stderr.Bytes(), []byte(code)) {
		t.Fatalf("stderr = %q, want %s", stderr.String(), code)
	}
}

func TestInspectMapsOperationalErrorToExitThree(t *testing.T) {
	t.Parallel()

	app := testApp()
	app.newObserver = func(Config) (observationRunner, closeFunc, error) {
		return &fakeObservationRunner{
			err: model.NewError(model.CodeContractNotFound, errors.New("missing")),
		}, nil, nil
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := app.Run(context.Background(), []string{
		"inspect", "--network", "testnet", "--contract", cliTestContractID,
	}, &stdout, &stderr)
	if exit != 3 {
		t.Fatalf("exit = %d, want 3", exit)
	}
	if !bytes.Contains(stderr.Bytes(), []byte(model.CodeContractNotFound)) {
		t.Fatalf("stderr = %q, want contract-not-found code", stderr.String())
	}
}

type fakeObservationRunner struct {
	result     observe.Result
	err        error
	contractID string
}

func (runner *fakeObservationRunner) Observe(_ context.Context, contractID string) (observe.Result, error) {
	runner.contractID = contractID
	return runner.result, runner.err
}

type recordingStore struct {
	saved []model.Snapshot
}

func (recording *recordingStore) Save(_ context.Context, snapshot model.Snapshot) (store.SnapshotRef, error) {
	recording.saved = append(recording.saved, snapshot)
	return storepkgReference(snapshot), nil
}

func (recording *recordingStore) Load(context.Context, string, string, string) (model.Snapshot, error) {
	return model.Snapshot{}, errors.New("not implemented in recording store")
}

func (recording *recordingStore) List(context.Context, string, string) ([]store.SnapshotRef, error) {
	return nil, errors.New("not implemented in recording store")
}

func storepkgReference(snapshot model.Snapshot) store.SnapshotRef {
	return store.SnapshotRef{LedgerSeq: snapshot.LedgerSeq, ConfigHash: snapshot.ConfigHash}
}

func testApp() *App {
	return &App{
		lookupEnv:   func(string) (string, bool) { return "", false },
		userHome:    func() (string, error) { return "home", nil },
		now:         func() string { return "2026-11-06T12:00:00Z" },
		toolVersion: "test-version",
		newStore: func(string) (store.Store, error) {
			return &recordingStore{}, nil
		},
	}
}
