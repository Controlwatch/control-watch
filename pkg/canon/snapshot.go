package canon

import (
	"fmt"

	"github.com/Controlwatch/control-watch/pkg/model"
)

// SnapshotHashes contains the two content digests stored in a snapshot.
type SnapshotHashes struct {
	Config   string
	Snapshot string
}

// HashSnapshot computes the observation hash and then the snapshot-envelope
// hash. The envelope includes the freshly computed config hash and excludes the
// snapshot_hash field itself.
func HashSnapshot(snapshot model.Snapshot) (SnapshotHashes, error) {
	configHash, err := Hash(snapshot.Observation)
	if err != nil {
		return SnapshotHashes{}, fmt.Errorf("hash observation: %w", err)
	}

	envelope := snapshotEnvelope{
		SchemaVersion: snapshot.SchemaVersion,
		ToolVersion:   snapshot.ToolVersion,
		Network:       snapshot.Network,
		ContractID:    snapshot.ContractID,
		LedgerSeq:     snapshot.LedgerSeq,
		ObservedAt:    snapshot.ObservedAt,
		Completeness:  snapshot.Completeness,
		Warnings:      snapshot.Warnings,
		Observation:   snapshot.Observation,
		ConfigHash:    configHash,
	}
	snapshotHash, err := Hash(envelope)
	if err != nil {
		return SnapshotHashes{}, fmt.Errorf("hash snapshot envelope: %w", err)
	}
	return SnapshotHashes{Config: configHash, Snapshot: snapshotHash}, nil
}

// PopulateSnapshotHashes computes and stores both snapshot digests.
func PopulateSnapshotHashes(snapshot *model.Snapshot) error {
	if snapshot == nil {
		return fmt.Errorf("populate snapshot hashes: nil snapshot")
	}
	hashes, err := HashSnapshot(*snapshot)
	if err != nil {
		return err
	}
	snapshot.ConfigHash = hashes.Config
	snapshot.SnapshotHash = hashes.Snapshot
	return nil
}

type snapshotEnvelope struct {
	SchemaVersion string            `json:"schema_version"`
	ToolVersion   string            `json:"tool_version"`
	Network       string            `json:"network"`
	ContractID    string            `json:"contract_id"`
	LedgerSeq     uint32            `json:"ledger_sequence"`
	ObservedAt    string            `json:"observed_at"`
	Completeness  string            `json:"completeness"`
	Warnings      []string          `json:"warnings"`
	Observation   model.Observation `json:"observation"`
	ConfigHash    string            `json:"config_hash"`
}
