// Package store persists immutable ControlWatch snapshots.
package store

import (
	"context"

	"github.com/Controlwatch/control-watch/pkg/model"
)

// SnapshotRef is the stable identifier and filename metadata for one snapshot.
type SnapshotRef struct {
	ID         string
	LedgerSeq  uint32
	ConfigHash string
}

// Store saves, loads, and lists immutable snapshots.
type Store interface {
	Save(ctx context.Context, snapshot model.Snapshot) (SnapshotRef, error)
	Load(ctx context.Context, network string, contractID string, snapshotID string) (model.Snapshot, error)
	List(ctx context.Context, network string, contractID string) ([]SnapshotRef, error)
}
