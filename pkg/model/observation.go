package model

const (
	// SchemaVersion is the current snapshot schema.
	SchemaVersion = "1"

	CompletenessComplete = "COMPLETE"
	CompletenessPartial  = "PARTIAL"
)

// Classification describes the observed contract surface.
type Classification string

const (
	ClassSEP57RWA Classification = "SEP57_RWA"
	ClassSEP41    Classification = "SEP41"
	ClassSAC      Classification = "SAC"
	ClassUnknown  Classification = "UNKNOWN"
	ClassError    Classification = "ERROR"
)

// Module is one compliance module and the hooks on which it is registered.
type Module struct {
	Address  string           `json:"address"`
	WasmHash Measured[string] `json:"wasm_hash"`
	Hooks    []string         `json:"hooks"`
}

// Observation is the semantic state measured for one contract.
type Observation struct {
	Classification   Classification     `json:"classification"`
	Token            *TokenSurface      `json:"token,omitempty"`
	Compliance       *ComplianceSurface `json:"compliance,omitempty"`
	IdentityVerifier *ContractRef       `json:"identity_verifier,omitempty"`
	SAC              *SACSurface        `json:"sac,omitempty"`
	ClassicFlags     *ClassicFlags      `json:"classic_flags,omitempty"`
	Reasons          map[string]string  `json:"reasons,omitempty"`
}

// TokenSurface is the state observed directly from an RWA token.
type TokenSurface struct {
	WasmHash Measured[string]              `json:"wasm_hash"`
	Paused   Measured[bool]                `json:"paused"`
	Admin    Measured[string]              `json:"admin"`
	Roles    map[string]Measured[[]string] `json:"roles"`
}

// ContractRef identifies a contract and its deployed Wasm.
type ContractRef struct {
	Address  string           `json:"address"`
	WasmHash Measured[string] `json:"wasm_hash"`
}

// ComplianceSurface is a compliance contract and its registered modules.
type ComplianceSurface struct {
	ContractRef
	Modules Measured[[]Module] `json:"modules"`
}

// SACSurface is the state specific to a Stellar Asset Contract.
type SACSurface struct {
	Admin Measured[string] `json:"admin"`
}

// ClassicFlags contains the issuer-account authorization flags for a classic
// asset associated with a Stellar Asset Contract.
type ClassicFlags struct {
	AuthRequired        Measured[bool] `json:"auth_required"`
	AuthRevocable       Measured[bool] `json:"auth_revocable"`
	AuthImmutable       Measured[bool] `json:"auth_immutable"`
	AuthClawbackEnabled Measured[bool] `json:"auth_clawback_enabled"`
}

// Snapshot is an immutable observation envelope.
type Snapshot struct {
	SchemaVersion string      `json:"schema_version"`
	ToolVersion   string      `json:"tool_version"`
	Network       string      `json:"network"`
	ContractID    string      `json:"contract_id"`
	LedgerSeq     uint32      `json:"ledger_sequence"`
	ObservedAt    string      `json:"observed_at"`
	Completeness  string      `json:"completeness"`
	Warnings      []string    `json:"warnings"`
	Observation   Observation `json:"observation"`
	ConfigHash    string      `json:"config_hash"`
	SnapshotHash  string      `json:"snapshot_hash"`
}
