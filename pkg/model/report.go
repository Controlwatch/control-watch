package model

// ReportDisclaimer is the fixed scope statement included in every report.
const ReportDisclaimer = "Review priority indicates review order only. This report describes observed configuration differences and available ledger evidence; it does not determine legal or compliance status."

// Report is the stable comparison-report envelope.
type Report struct {
	SchemaVersion string            `json:"schema_version"`
	GeneratedAt   string            `json:"generated_at"`
	Network       string            `json:"network"`
	ContractID    string            `json:"contract_id"`
	Baseline      BaselineReportRef `json:"baseline"`
	Current       CurrentReportRef  `json:"current"`
	Findings      []Finding         `json:"findings"`
	NotCompared   []NotCompared     `json:"not_compared"`
	Completeness  string            `json:"completeness"`
	RPC           ReportRPC         `json:"rpc"`
	Disclaimer    string            `json:"disclaimer"`
}

// BaselineReportRef identifies the saved snapshot used as the baseline.
type BaselineReportRef struct {
	SnapshotID   string `json:"snapshot_id"`
	LedgerSeq    uint32 `json:"ledger_sequence"`
	SnapshotHash string `json:"snapshot_hash"`
}

// CurrentReportRef identifies the newly observed snapshot.
type CurrentReportRef struct {
	LedgerSeq    uint32 `json:"ledger_sequence"`
	SnapshotHash string `json:"snapshot_hash"`
}

// ReportRPC contains non-secret RPC retention metadata.
type ReportRPC struct {
	Host         string `json:"host"`
	OldestLedger uint32 `json:"oldest_ledger"`
}
