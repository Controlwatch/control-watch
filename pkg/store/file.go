package store

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Controlwatch/control-watch/pkg/canon"
	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/stellar/go-stellar-sdk/strkey"
)

const (
	snapshotExtension = ".json"
	configHashPrefix  = 8
)

// FileStore persists snapshots below a local root directory.
type FileStore struct {
	root string
}

var _ Store = (*FileStore)(nil)

// NewFileStore constructs a filesystem store rooted at an absolute, cleaned
// path. Directories are created lazily by Save.
func NewFileStore(root string) (*FileStore, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("construct file store: root is required")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("construct file store: resolve root: %w", err)
	}
	return &FileStore{root: filepath.Clean(absolute)}, nil
}

// Save hashes a snapshot copy and creates its file with exclusive-create
// semantics. Existing snapshot files are never replaced.
func (store *FileStore) Save(ctx context.Context, snapshot model.Snapshot) (SnapshotRef, error) {
	if err := contextError(ctx); err != nil {
		return SnapshotRef{}, err
	}
	if err := validateScope(snapshot.Network, snapshot.ContractID); err != nil {
		return SnapshotRef{}, err
	}
	if snapshot.SchemaVersion != model.SchemaVersion {
		return SnapshotRef{}, model.NewError(
			model.CodeSchemaUnsupported,
			fmt.Errorf("save snapshot: schema version %q", snapshot.SchemaVersion),
		)
	}
	if err := canon.PopulateSnapshotHashes(&snapshot); err != nil {
		return SnapshotRef{}, fmt.Errorf("save snapshot: %w", err)
	}
	reference := SnapshotRef{
		ID:         fmt.Sprintf("%d-%s", snapshot.LedgerSeq, snapshot.ConfigHash[:configHashPrefix]),
		LedgerSeq:  snapshot.LedgerSeq,
		ConfigHash: snapshot.ConfigHash,
	}
	encoded, err := canon.Marshal(snapshot)
	if err != nil {
		return SnapshotRef{}, fmt.Errorf("save snapshot: encode: %w", err)
	}

	directory := store.scopeDirectory(snapshot.Network, snapshot.ContractID)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return SnapshotRef{}, fmt.Errorf("save snapshot: create directory: %w", err)
	}
	path := filepath.Join(directory, reference.ID+snapshotExtension)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o444)
	if err != nil {
		return SnapshotRef{}, fmt.Errorf("save snapshot: create immutable file: %w", err)
	}
	writeErr := writeAndClose(file, encoded)
	if writeErr != nil {
		if removeErr := os.Remove(path); removeErr != nil {
			return SnapshotRef{}, fmt.Errorf(
				"save snapshot: %v; remove incomplete file: %w",
				writeErr,
				removeErr,
			)
		}
		return SnapshotRef{}, fmt.Errorf("save snapshot: %w", writeErr)
	}
	return reference, nil
}

// Load reads one snapshot and verifies its schema, content hashes, and filename
// metadata before returning it.
func (store *FileStore) Load(
	ctx context.Context,
	network string,
	contractID string,
	snapshotID string,
) (model.Snapshot, error) {
	if err := contextError(ctx); err != nil {
		return model.Snapshot{}, err
	}
	if err := validateScope(network, contractID); err != nil {
		return model.Snapshot{}, err
	}
	reference, err := parseSnapshotID(snapshotID)
	if err != nil {
		return model.Snapshot{}, model.NewError(model.CodeSnapshotNotFound, err)
	}
	path := filepath.Join(store.scopeDirectory(network, contractID), reference.ID+snapshotExtension)
	encoded, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return model.Snapshot{}, model.NewError(model.CodeSnapshotNotFound, err)
		}
		return model.Snapshot{}, fmt.Errorf("load snapshot: read file: %w", err)
	}

	var snapshot model.Snapshot
	if err := json.Unmarshal(encoded, &snapshot); err != nil {
		return model.Snapshot{}, model.NewError(
			model.CodeSnapshotCorrupt,
			fmt.Errorf("decode snapshot: %w", err),
		)
	}
	if snapshot.SchemaVersion != model.SchemaVersion {
		return model.Snapshot{}, model.NewError(
			model.CodeSchemaUnsupported,
			fmt.Errorf("load snapshot: schema version %q", snapshot.SchemaVersion),
		)
	}
	if snapshot.Network != network || snapshot.ContractID != contractID {
		return model.Snapshot{}, model.NewError(
			model.CodeSnapshotCorrupt,
			fmt.Errorf("load snapshot: file scope does not match its directory"),
		)
	}
	hashes, err := canon.HashSnapshot(snapshot)
	if err != nil {
		return model.Snapshot{}, model.NewError(model.CodeSnapshotCorrupt, err)
	}
	if snapshot.ConfigHash != hashes.Config || snapshot.SnapshotHash != hashes.Snapshot {
		return model.Snapshot{}, model.NewError(
			model.CodeSnapshotCorrupt,
			fmt.Errorf("load snapshot: stored hashes do not match content"),
		)
	}
	if snapshot.LedgerSeq != reference.LedgerSeq || !strings.HasPrefix(snapshot.ConfigHash, reference.ConfigHash) {
		return model.Snapshot{}, model.NewError(
			model.CodeSnapshotCorrupt,
			fmt.Errorf("load snapshot: filename does not match content"),
		)
	}
	return snapshot, nil
}

// List returns valid snapshot filename references newest-ledger first. It does
// not open or verify snapshot contents.
func (store *FileStore) List(
	ctx context.Context,
	network string,
	contractID string,
) ([]SnapshotRef, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if err := validateScope(network, contractID); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(store.scopeDirectory(network, contractID))
	if errors.Is(err, fs.ErrNotExist) {
		return []SnapshotRef{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list snapshots: read directory: %w", err)
	}
	references := make([]SnapshotRef, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != snapshotExtension {
			continue
		}
		reference, parseErr := parseSnapshotID(strings.TrimSuffix(entry.Name(), snapshotExtension))
		if parseErr != nil {
			continue
		}
		references = append(references, reference)
	}
	sort.Slice(references, func(left int, right int) bool {
		if references[left].LedgerSeq == references[right].LedgerSeq {
			return references[left].ID < references[right].ID
		}
		return references[left].LedgerSeq > references[right].LedgerSeq
	})
	return references, nil
}

func (store *FileStore) scopeDirectory(network string, contractID string) string {
	return filepath.Join(store.root, network, contractID)
}

func validateScope(network string, contractID string) error {
	if network == "" || filepath.Clean(network) != network || strings.ContainsAny(network, `/\`) || network == "." || network == ".." {
		return fmt.Errorf("snapshot scope: invalid network %q", network)
	}
	if _, err := strkey.Decode(strkey.VersionByteContract, contractID); err != nil {
		return model.NewError(model.CodeInvalidContractID, err)
	}
	return nil
}

func parseSnapshotID(snapshotID string) (SnapshotRef, error) {
	parts := strings.Split(snapshotID, "-")
	if len(parts) != 2 || parts[0] == "" || len(parts[1]) != configHashPrefix {
		return SnapshotRef{}, fmt.Errorf("invalid snapshot ID %q", snapshotID)
	}
	ledger, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return SnapshotRef{}, fmt.Errorf("invalid snapshot ID %q: %w", snapshotID, err)
	}
	if _, err := hex.DecodeString(parts[1]); err != nil {
		return SnapshotRef{}, fmt.Errorf("invalid snapshot ID %q: %w", snapshotID, err)
	}
	return SnapshotRef{
		ID:         snapshotID,
		LedgerSeq:  uint32(ledger),
		ConfigHash: strings.ToLower(parts[1]),
	}, nil
}

func contextError(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func writeAndClose(file *os.File, data []byte) error {
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return nil
}
