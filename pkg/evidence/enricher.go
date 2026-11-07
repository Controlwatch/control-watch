package evidence

import (
	"context"
	"fmt"
	"strings"

	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/rpcx"
	"github.com/Controlwatch/control-watch/pkg/surface"
)

// Enricher associates retained events with comparison findings.
type Enricher struct {
	client rpcx.Client
}

// NewEnricher constructs an evidence enricher.
func NewEnricher(client rpcx.Client) (*Enricher, error) {
	if client == nil {
		return nil, fmt.Errorf("construct evidence enricher: RPC client is required")
	}
	return &Enricher{client: client}, nil
}

// Attach returns a copy of findings with evidence for the interval after the
// baseline snapshot through the current snapshot.
func (enricher *Enricher) Attach(
	ctx context.Context,
	before model.Snapshot,
	after model.Snapshot,
	findings []model.Finding,
) ([]model.Finding, error) {
	result := append([]model.Finding(nil), findings...)
	if len(result) == 0 {
		return result, nil
	}
	if after.LedgerSeq <= before.LedgerSeq {
		return nil, fmt.Errorf(
			"attach evidence: current ledger %d must be after baseline ledger %d",
			after.LedgerSeq,
			before.LedgerSeq,
		)
	}
	window, err := ReadRetentionWindow(ctx, enricher.client)
	if err != nil {
		return nil, err
	}
	startLedger := before.LedgerSeq + 1
	if after.LedgerSeq > window.LatestLedger {
		return nil, fmt.Errorf(
			"attach evidence: current ledger %d exceeds RPC latest ledger %d",
			after.LedgerSeq,
			window.LatestLedger,
		)
	}
	if startLedger < window.OldestLedger {
		changedBetween := [2]uint32{before.LedgerSeq, after.LedgerSeq}
		for index := range result {
			result[index].Evidence = model.Evidence{
				Status:         model.EvidenceUnavailableOutsideRetention,
				Items:          []model.EvidenceItem{},
				ChangedBetween: &changedBetween,
			}
		}
		return result, nil
	}

	for index := range result {
		contracts, names := evidenceTargets(result[index], before, after)
		if len(contracts) == 0 || len(names) == 0 {
			result[index].Evidence = model.Evidence{
				Status: model.EvidenceNotEmitted,
				Items:  []model.EvidenceItem{},
			}
			continue
		}
		items, err := SearchEvents(ctx, enricher.client, EventQuery{
			ContractIDs: contracts,
			Names:       names,
			StartLedger: startLedger,
			EndLedger:   after.LedgerSeq,
		})
		if err != nil {
			return nil, fmt.Errorf("attach evidence for %s: %w", result[index].Code, err)
		}
		status := model.EvidenceNotEmitted
		if len(items) > 0 {
			status = model.EvidenceFound
		}
		result[index].Evidence = model.Evidence{Status: status, Items: items}
	}
	return result, nil
}

func evidenceTargets(
	finding model.Finding,
	before model.Snapshot,
	after model.Snapshot,
) ([]string, []surface.EventName) {
	switch finding.Code {
	case "CW-001":
		return []string{after.ContractID}, []surface.EventName{surface.EventComplianceSet}
	case "CW-002":
		return []string{after.ContractID}, []surface.EventName{surface.EventIdentityVerifierSet}
	case "CW-003":
		return []string{complianceContract(after.Observation)}, []surface.EventName{surface.EventModuleAdded}
	case "CW-004":
		return []string{complianceContract(before.Observation)}, []surface.EventName{surface.EventModuleRemoved}
	case "CW-006":
		return []string{after.ContractID}, []surface.EventName{
			surface.EventAdminRenounced,
			surface.EventAdminTransferCompleted,
		}
	case "CW-007":
		return []string{after.ContractID}, []surface.EventName{
			surface.EventRoleGranted,
			surface.EventRoleRevoked,
		}
	case "CW-008":
		return []string{after.ContractID}, []surface.EventName{
			surface.EventPaused,
			surface.EventUnpaused,
		}
	case "CW-012":
		return []string{
				complianceContract(before.Observation),
				complianceContract(after.Observation),
			}, []surface.EventName{
				surface.EventModuleAdded,
				surface.EventModuleRemoved,
			}
	default:
		return nil, nil
	}
}

func complianceContract(observation model.Observation) string {
	if observation.Compliance == nil {
		return ""
	}
	return strings.TrimSpace(observation.Compliance.Address)
}
