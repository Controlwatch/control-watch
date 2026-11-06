package diff

import (
	"fmt"
	"sort"

	"github.com/Controlwatch/control-watch/pkg/model"
)

// Result contains deterministic findings and omitted semantic subjects.
type Result struct {
	Findings    []model.Finding     `json:"findings"`
	NotCompared []model.NotCompared `json:"not_compared"`
}

// Detector compares only the semantic observations from two snapshots.
type Detector interface {
	Detect(before model.Observation, after model.Observation) Result
}

// Runner executes a fixed set of detectors.
type Runner struct {
	detectors []Detector
}

// NewRunner constructs a runner with detectors in any order. Run normalizes
// their combined output deterministically.
func NewRunner(detectors ...Detector) *Runner {
	return &Runner{detectors: append([]Detector(nil), detectors...)}
}

// Run validates snapshot scope, compares semantic observations, and sorts
// findings by code then subject.
func (runner *Runner) Run(before model.Snapshot, after model.Snapshot) (Result, error) {
	if before.SchemaVersion != model.SchemaVersion || after.SchemaVersion != model.SchemaVersion {
		return Result{}, model.NewError(
			model.CodeSchemaUnsupported,
			fmt.Errorf(
				"compare snapshots: schema versions %q and %q",
				before.SchemaVersion,
				after.SchemaVersion,
			),
		)
	}
	if before.Network != after.Network || before.ContractID != after.ContractID {
		return Result{}, model.NewError(
			model.CodeScopeMismatch,
			fmt.Errorf("compare snapshots: network or contract differs"),
		)
	}

	result := Result{
		Findings:    []model.Finding{},
		NotCompared: []model.NotCompared{},
	}
	for _, detector := range runner.detectors {
		if detector == nil {
			continue
		}
		detected := detector.Detect(before.Observation, after.Observation)
		result.Findings = append(result.Findings, detected.Findings...)
		result.NotCompared = append(result.NotCompared, detected.NotCompared...)
	}
	sort.SliceStable(result.Findings, func(left int, right int) bool {
		if result.Findings[left].Code == result.Findings[right].Code {
			return result.Findings[left].Subject < result.Findings[right].Subject
		}
		return result.Findings[left].Code < result.Findings[right].Code
	})
	sort.SliceStable(result.NotCompared, func(left int, right int) bool {
		if result.NotCompared[left].Subject == result.NotCompared[right].Subject {
			return result.NotCompared[left].Reason < result.NotCompared[right].Reason
		}
		return result.NotCompared[left].Subject < result.NotCompared[right].Subject
	})
	result.NotCompared = deduplicateNotCompared(result.NotCompared)
	return result, nil
}

func deduplicateNotCompared(values []model.NotCompared) []model.NotCompared {
	if len(values) < 2 {
		return values
	}
	unique := values[:1]
	for _, value := range values[1:] {
		previous := unique[len(unique)-1]
		if value.Subject == previous.Subject && value.Reason == previous.Reason {
			continue
		}
		unique = append(unique, value)
	}
	return unique
}
