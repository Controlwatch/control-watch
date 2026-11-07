package evidence

import (
	"context"
	"fmt"
	"strings"

	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/rpcx"
)

func readLastModifiedLedgers(
	ctx context.Context,
	client rpcx.Client,
	contractIDs []string,
) (map[string]uint32, error) {
	contracts := sortedUniqueStrings(contractIDs)
	keys := make([]string, 0, len(contracts))
	contractByKey := make(map[string]string, len(contracts))
	for _, contractID := range contracts {
		key, err := rpcx.ContractInstanceKey(contractID)
		if err != nil {
			return nil, fmt.Errorf("read evidence ledger bounds for %s: %w", contractID, err)
		}
		keys = append(keys, key)
		contractByKey[key] = contractID
	}
	response, err := client.GetLedgerEntries(ctx, keys)
	if err != nil {
		return nil, fmt.Errorf("read evidence ledger bounds: %w", err)
	}
	result := make(map[string]uint32, len(contracts))
	for _, entry := range response.Entries {
		contractID, expected := contractByKey[entry.Key]
		if !expected {
			return nil, fmt.Errorf("read evidence ledger bounds: RPC returned an unexpected ledger key")
		}
		if _, duplicate := result[contractID]; duplicate {
			return nil, fmt.Errorf("read evidence ledger bounds: RPC returned duplicate entry for %s", contractID)
		}
		if entry.LastModifiedLedger == 0 {
			return nil, fmt.Errorf("read evidence ledger bounds: zero last-modified ledger for %s", contractID)
		}
		result[contractID] = entry.LastModifiedLedger
	}
	for _, contractID := range contracts {
		if _, found := result[contractID]; !found {
			return nil, fmt.Errorf("read evidence ledger bounds: no instance entry for %s", contractID)
		}
	}
	return result, nil
}

func evidenceLedgerContract(
	finding model.Finding,
	before model.Snapshot,
	after model.Snapshot,
) string {
	switch finding.Code {
	case "CW-003":
		return complianceContract(after.Observation)
	case "CW-004":
		return complianceContract(before.Observation)
	case "CW-005":
		switch {
		case finding.Subject == "compliance":
			return complianceContract(after.Observation)
		case finding.Subject == "identity_verifier":
			return identityVerifierContract(after.Observation)
		case strings.HasPrefix(finding.Subject, "module:"):
			return strings.TrimPrefix(finding.Subject, "module:")
		default:
			return after.ContractID
		}
	case "CW-012":
		return complianceContract(after.Observation)
	default:
		return after.ContractID
	}
}

func identityVerifierContract(observation model.Observation) string {
	if observation.IdentityVerifier == nil {
		return ""
	}
	return strings.TrimSpace(observation.IdentityVerifier.Address)
}

func sortedLedgerContracts(
	findings []model.Finding,
	before model.Snapshot,
	after model.Snapshot,
) ([]string, error) {
	contracts := make([]string, 0, len(findings))
	for _, finding := range findings {
		contractID := evidenceLedgerContract(finding, before, after)
		if contractID == "" {
			return nil, fmt.Errorf("select evidence ledger bound for %s: contract is unavailable", finding.Code)
		}
		contracts = append(contracts, contractID)
	}
	contracts = sortedUniqueStrings(contracts)
	return contracts, nil
}
