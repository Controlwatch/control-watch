package rpcx

// Health describes the connected RPC service and its retained ledger range.
type Health struct {
	Status                string
	LatestLedger          uint32
	OldestLedger          uint32
	LedgerRetentionWindow uint32
}

// LatestLedger describes the newest ledger known to the RPC service.
type LatestLedger struct {
	Sequence        uint32
	ProtocolVersion uint32
	Hash            string
	CloseTime       int64
}

// LedgerEntries is a batch ledger-entry response.
type LedgerEntries struct {
	LatestLedger uint32
	Entries      []LedgerEntry
}

// LedgerEntry is one base64-XDR ledger entry and its metadata.
type LedgerEntry struct {
	Key                string
	XDR                string
	LastModifiedLedger uint32
}

// Simulation is a read-only transaction simulation response.
type Simulation struct {
	LatestLedger     uint32
	Error            string
	TransactionData  string
	MinResourceFee   int64
	Results          []SimulationResult
	DiagnosticEvents []string
}

// SimulationResult contains one simulated host-function result.
type SimulationResult struct {
	Auth      []string
	ReturnXDR string
}

// EventsRequest selects a retained ledger range and optional event filters.
type EventsRequest struct {
	StartLedger uint32
	EndLedger   uint32
	Cursor      string
	Limit       uint32
	Filters     []EventFilter
}

// EventFilter limits events to contract IDs and topic patterns.
type EventFilter struct {
	ContractIDs []string
	Topics      [][]string
}

// EventsPage is one page of contract events.
type EventsPage struct {
	LatestLedger uint32
	Cursor       string
	Events       []Event
}

// Event is one normalized Stellar contract event.
type Event struct {
	ID                       string
	Type                     string
	Ledger                   uint32
	LedgerClosedAt           string
	ContractID               string
	Topic                    []string
	ValueXDR                 string
	TransactionHash          string
	PagingToken              string
	InSuccessfulContractCall bool
}

// Transaction is a normalized getTransaction response.
type Transaction struct {
	Status        string
	LatestLedger  uint32
	Hash          string
	Ledger        uint32
	CreatedAt     string
	EnvelopeXDR   string
	ResultXDR     string
	ResultMetaXDR string
}
