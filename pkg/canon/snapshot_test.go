package canon

import (
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
)

func TestSnapshotGoldenHashes(t *testing.T) {
	t.Parallel()

	snapshot := goldenSnapshot()
	if err := PopulateSnapshotHashes(&snapshot); err != nil {
		t.Fatalf("PopulateSnapshotHashes() error = %v", err)
	}
	const wantConfig = "36084b10df60093b944cc0266fe3db7dd043e7b2e58efc256c233cc3b76a6f82"
	const wantSnapshot = "e9d83ba48e9f5296f6d3c16bd691bcbe6f3b9ac3e976695d01b5237100800f2f"
	if snapshot.ConfigHash != wantConfig {
		t.Fatalf("ConfigHash = %s, want %s", snapshot.ConfigHash, wantConfig)
	}
	if snapshot.SnapshotHash != wantSnapshot {
		t.Fatalf("SnapshotHash = %s, want %s", snapshot.SnapshotHash, wantSnapshot)
	}
}

func TestSnapshotHashExcludesExistingDigestFields(t *testing.T) {
	t.Parallel()

	left := goldenSnapshot()
	left.ConfigHash = "stale config"
	left.SnapshotHash = "stale snapshot"
	right := goldenSnapshot()
	right.ConfigHash = "different stale config"
	right.SnapshotHash = "different stale snapshot"

	leftHashes, err := HashSnapshot(left)
	if err != nil {
		t.Fatalf("HashSnapshot(left) error = %v", err)
	}
	rightHashes, err := HashSnapshot(right)
	if err != nil {
		t.Fatalf("HashSnapshot(right) error = %v", err)
	}
	if leftHashes != rightHashes {
		t.Fatalf("hashes differ: %#v != %#v", leftHashes, rightHashes)
	}
}

func TestWarningsAffectSnapshotButNotConfigHash(t *testing.T) {
	t.Parallel()

	left := goldenSnapshot()
	right := goldenSnapshot()
	right.Warnings = []string{"retention floor advanced"}

	leftHashes, err := HashSnapshot(left)
	if err != nil {
		t.Fatalf("HashSnapshot(left) error = %v", err)
	}
	rightHashes, err := HashSnapshot(right)
	if err != nil {
		t.Fatalf("HashSnapshot(right) error = %v", err)
	}
	if leftHashes.Config != rightHashes.Config {
		t.Fatalf("config hashes differ: %s != %s", leftHashes.Config, rightHashes.Config)
	}
	if leftHashes.Snapshot == rightHashes.Snapshot {
		t.Fatalf("snapshot hashes unexpectedly match: %s", leftHashes.Snapshot)
	}
}

func goldenSnapshot() model.Snapshot {
	return model.Snapshot{
		SchemaVersion: model.SchemaVersion,
		ToolVersion:   "0.1.0-test",
		Network:       "testnet",
		ContractID:    "CTOKEN",
		LedgerSeq:     123456,
		ObservedAt:    "2026-11-06T12:00:00Z",
		Completeness:  model.CompletenessPartial,
		Warnings:      []string{},
		Observation: model.Observation{
			Classification: model.ClassSEP57RWA,
			Token: &model.TokenSurface{
				WasmHash: model.Measure("abcdef"),
				Paused:   model.Measure(false),
				Admin:    model.CannotMeasure[string]("getter failed"),
				Roles: map[string]model.Measured[[]string]{
					"manager": model.Measure([]string{"G1", "G2"}),
				},
			},
			Reasons: map[string]string{"token.admin": "getter failed"},
		},
	}
}
