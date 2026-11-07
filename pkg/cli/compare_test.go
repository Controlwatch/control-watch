package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/canon"
	"github.com/Controlwatch/control-watch/pkg/evidence"
	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/observe"
	"github.com/Controlwatch/control-watch/pkg/store"
)

func TestCompareRunsDetectorsEvidenceAndRenderers(t *testing.T) {
	t.Parallel()

	baseline := comparisonSnapshot(t, 100, false)
	baselineID := "100-" + baseline.ConfigHash[:8]
	snapshotStore := &comparisonStore{
		baseline: baseline,
		references: []store.SnapshotRef{{
			ID: baselineID, LedgerSeq: baseline.LedgerSeq, ConfigHash: baseline.ConfigHash,
		}},
	}
	evidenceRunner := &fakeComparisonEvidence{window: evidence.RetentionWindow{
		OldestLedger: 90,
		LatestLedger: 120,
	}}
	closed := false
	app := testApp()
	app.newStore = func(string) (store.Store, error) { return snapshotStore, nil }
	app.newComparison = func(Config) (observationRunner, comparisonEvidence, closeFunc, error) {
		return &fakeObservationRunner{result: observe.Result{
			Observation: model.Observation{
				Classification: model.ClassSEP57RWA,
				Token:          &model.TokenSurface{Paused: model.Measure(true)},
			},
			LedgerSeq:    110,
			Completeness: model.CompletenessComplete,
			Warnings:     []string{},
		}}, evidenceRunner, func() error { closed = true; return nil }, nil
	}
	jsonPath := filepath.Join(t.TempDir(), "report.json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := app.Run(context.Background(), []string{
		"compare",
		"--network", "testnet",
		"--contract", cliTestContractID,
		"--baseline", "latest",
		"--save-current",
		"--json", jsonPath,
	}, &stdout, &stderr)
	if exit != 10 {
		t.Fatalf("exit = %d, want 10; stderr = %s", exit, stderr.String())
	}
	if !closed {
		t.Fatal("comparison runtime was not closed")
	}
	if len(snapshotStore.saved) != 1 || snapshotStore.saved[0].LedgerSeq != 110 {
		t.Fatalf("saved snapshots = %#v", snapshotStore.saved)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("CW-008 PAUSE_STATE_CHANGED")) {
		t.Fatalf("human output = %q", stdout.String())
	}
	encoded, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read JSON report: %v", err)
	}
	var comparisonReport model.Report
	if err := json.Unmarshal(encoded, &comparisonReport); err != nil {
		t.Fatalf("decode JSON report: %v", err)
	}
	if comparisonReport.Baseline.SnapshotID != baselineID || len(comparisonReport.Findings) != 1 {
		t.Fatalf("report = %#v", comparisonReport)
	}
	if comparisonReport.Findings[0].Evidence.Status != model.EvidenceNotEmitted {
		t.Fatalf("evidence = %#v", comparisonReport.Findings[0].Evidence)
	}
}

func TestCompareReturnsZeroForEqualConfigHash(t *testing.T) {
	t.Parallel()

	baseline := comparisonSnapshot(t, 100, false)
	baselineID := "100-" + baseline.ConfigHash[:8]
	snapshotStore := &comparisonStore{
		baseline:   baseline,
		references: []store.SnapshotRef{{ID: baselineID, LedgerSeq: 100}},
	}
	evidenceRunner := &fakeComparisonEvidence{window: evidence.RetentionWindow{OldestLedger: 90, LatestLedger: 120}}
	app := testApp()
	app.newStore = func(string) (store.Store, error) { return snapshotStore, nil }
	app.newComparison = func(Config) (observationRunner, comparisonEvidence, closeFunc, error) {
		return &fakeObservationRunner{result: observe.Result{
			Observation:  baseline.Observation,
			LedgerSeq:    110,
			Completeness: model.CompletenessComplete,
			Warnings:     []string{},
		}}, evidenceRunner, nil, nil
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := app.Run(context.Background(), []string{
		"compare", "--network", "testnet", "--contract", cliTestContractID,
	}, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0; stderr = %s", exit, stderr.String())
	}
	if evidenceRunner.attachedCount != 0 {
		t.Fatalf("evidence attached to %d findings, want 0", evidenceRunner.attachedCount)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Findings: 0")) {
		t.Fatalf("human output = %q", stdout.String())
	}
}

func comparisonSnapshot(t *testing.T, ledger uint32, paused bool) model.Snapshot {
	t.Helper()
	snapshot := model.Snapshot{
		SchemaVersion: model.SchemaVersion,
		ToolVersion:   "test-version",
		Network:       "testnet",
		ContractID:    cliTestContractID,
		LedgerSeq:     ledger,
		ObservedAt:    "2026-11-07T00:00:00Z",
		Completeness:  model.CompletenessComplete,
		Warnings:      []string{},
		Observation: model.Observation{
			Classification: model.ClassSEP57RWA,
			Token:          &model.TokenSurface{Paused: model.Measure(paused)},
		},
	}
	if err := canon.PopulateSnapshotHashes(&snapshot); err != nil {
		t.Fatalf("hash snapshot: %v", err)
	}
	return snapshot
}

type comparisonStore struct {
	baseline   model.Snapshot
	references []store.SnapshotRef
	saved      []model.Snapshot
	loadedID   string
}

func (comparison *comparisonStore) Save(_ context.Context, snapshot model.Snapshot) (store.SnapshotRef, error) {
	comparison.saved = append(comparison.saved, snapshot)
	return store.SnapshotRef{LedgerSeq: snapshot.LedgerSeq, ConfigHash: snapshot.ConfigHash}, nil
}

func (comparison *comparisonStore) Load(_ context.Context, _ string, _ string, snapshotID string) (model.Snapshot, error) {
	comparison.loadedID = snapshotID
	if comparison.baseline.SchemaVersion == "" {
		return model.Snapshot{}, model.NewError(model.CodeSnapshotNotFound, errors.New("missing"))
	}
	return comparison.baseline, nil
}

func (comparison *comparisonStore) List(context.Context, string, string) ([]store.SnapshotRef, error) {
	return append([]store.SnapshotRef(nil), comparison.references...), nil
}

type fakeComparisonEvidence struct {
	window        evidence.RetentionWindow
	err           error
	attachedCount int
}

func (fake *fakeComparisonEvidence) Retention(context.Context) (evidence.RetentionWindow, error) {
	return fake.window, fake.err
}

func (fake *fakeComparisonEvidence) Attach(
	_ context.Context,
	_ model.Snapshot,
	_ model.Snapshot,
	findings []model.Finding,
) ([]model.Finding, error) {
	fake.attachedCount = len(findings)
	result := append([]model.Finding(nil), findings...)
	for index := range result {
		result[index].Evidence = model.Evidence{Status: model.EvidenceNotEmitted, Items: []model.EvidenceItem{}}
	}
	return result, fake.err
}
