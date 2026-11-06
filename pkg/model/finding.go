package model

const (
	// PriorityHigh requests the earliest review tier.
	PriorityHigh = "HIGH"
	// PriorityMedium requests the middle review tier.
	PriorityMedium = "MEDIUM"
	// PriorityLow requests the lowest review tier.
	PriorityLow = "LOW"
	// PriorityInfo marks informational output.
	PriorityInfo = "INFO"

	// EvidenceFound indicates that matching ledger evidence was found.
	EvidenceFound = "FOUND"
	// EvidenceUnavailableOutsideRetention indicates that the relevant ledger
	// range predates the RPC server's retained event history.
	EvidenceUnavailableOutsideRetention = "UNAVAILABLE_OUTSIDE_RETENTION"
	// EvidenceNotEmitted indicates that no matching event was emitted in the
	// searched ledger range.
	EvidenceNotEmitted = "NOT_EMITTED"
	// EvidenceUnmeasurable indicates that evidence could not be evaluated.
	EvidenceUnmeasurable = "UNMEASURABLE"
)

// Finding is one semantic difference between two snapshots.
type Finding struct {
	Code           string   `json:"code"`
	Name           string   `json:"name"`
	Subject        string   `json:"subject"`
	Before         string   `json:"before"`
	After          string   `json:"after"`
	ReviewPriority string   `json:"review_priority"`
	Explanation    string   `json:"explanation"`
	NextStep       string   `json:"next_step"`
	Evidence       Evidence `json:"evidence"`
}

// Evidence describes ledger events associated with a finding.
type Evidence struct {
	Status         string         `json:"status"`
	Items          []EvidenceItem `json:"items"`
	ChangedBetween *[2]uint32     `json:"changed_between,omitempty"`
}

// EvidenceItem identifies one matching ledger event.
type EvidenceItem struct {
	Ledger uint32 `json:"ledger"`
	TxHash string `json:"tx_hash"`
	Topic  string `json:"topic"`
}

// NotCompared records one semantic subject omitted from comparison.
type NotCompared struct {
	Subject string `json:"subject"`
	Reason  string `json:"reason"`
}
