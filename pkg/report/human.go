package report

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Controlwatch/control-watch/pkg/model"
)

// Human returns a deterministic plain-text comparison report.
func Human(value model.Report) string {
	findings := cloneAndSortFindings(value.Findings)
	notCompared := append([]model.NotCompared(nil), value.NotCompared...)
	sort.SliceStable(notCompared, func(left int, right int) bool {
		if notCompared[left].Subject == notCompared[right].Subject {
			return notCompared[left].Reason < notCompared[right].Reason
		}
		return notCompared[left].Subject < notCompared[right].Subject
	})

	var output strings.Builder
	_, _ = fmt.Fprintln(&output, "ControlWatch comparison")
	_, _ = fmt.Fprintf(&output, "Network: %s\n", value.Network)
	_, _ = fmt.Fprintf(&output, "Contract: %s\n", value.ContractID)
	_, _ = fmt.Fprintf(
		&output,
		"Baseline: %s (ledger %d, hash %s)\n",
		value.Baseline.SnapshotID,
		value.Baseline.LedgerSeq,
		value.Baseline.SnapshotHash,
	)
	_, _ = fmt.Fprintf(
		&output,
		"Current: ledger %d (hash %s)\n",
		value.Current.LedgerSeq,
		value.Current.SnapshotHash,
	)
	_, _ = fmt.Fprintf(&output, "Completeness: %s\n", value.Completeness)
	_, _ = fmt.Fprintf(&output, "RPC: %s (oldest ledger %d)\n", value.RPC.Host, value.RPC.OldestLedger)
	_, _ = fmt.Fprintf(&output, "Findings: %d\n", len(findings))
	for _, finding := range findings {
		_, _ = fmt.Fprintf(
			&output,
			"\n[%s] %s %s - %s\n",
			finding.ReviewPriority,
			finding.Code,
			finding.Name,
			finding.Subject,
		)
		_, _ = fmt.Fprintf(&output, "  Before: %s\n", finding.Before)
		_, _ = fmt.Fprintf(&output, "  After: %s\n", finding.After)
		_, _ = fmt.Fprintf(&output, "  Explanation: %s\n", finding.Explanation)
		_, _ = fmt.Fprintf(&output, "  Evidence: %s\n", finding.Evidence.Status)
		for _, item := range finding.Evidence.Items {
			_, _ = fmt.Fprintf(
				&output,
				"    - ledger %d, tx %s, topic %s\n",
				item.Ledger,
				item.TxHash,
				item.Topic,
			)
		}
		if finding.Evidence.ChangedBetween != nil {
			_, _ = fmt.Fprintf(
				&output,
				"  Changed between ledgers: %d..%d\n",
				finding.Evidence.ChangedBetween[0],
				finding.Evidence.ChangedBetween[1],
			)
		}
		_, _ = fmt.Fprintf(&output, "  Next step: %s\n", finding.NextStep)
	}

	_, _ = fmt.Fprintf(&output, "\nNot compared: %d\n", len(notCompared))
	for _, omitted := range notCompared {
		_, _ = fmt.Fprintf(&output, "  - %s: %s\n", omitted.Subject, omitted.Reason)
	}
	_, _ = fmt.Fprintf(&output, "\nDisclaimer: %s\n", value.Disclaimer)
	return output.String()
}

func cloneAndSortFindings(values []model.Finding) []model.Finding {
	cloned := append([]model.Finding(nil), values...)
	for index := range cloned {
		cloned[index].Evidence.Items = append([]model.EvidenceItem(nil), cloned[index].Evidence.Items...)
		sort.SliceStable(cloned[index].Evidence.Items, func(left int, right int) bool {
			items := cloned[index].Evidence.Items
			if items[left].Ledger != items[right].Ledger {
				return items[left].Ledger < items[right].Ledger
			}
			if items[left].TxHash != items[right].TxHash {
				return items[left].TxHash < items[right].TxHash
			}
			return items[left].Topic < items[right].Topic
		})
	}
	sort.SliceStable(cloned, func(left int, right int) bool {
		leftRank := priorityRank(cloned[left].ReviewPriority)
		rightRank := priorityRank(cloned[right].ReviewPriority)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		if cloned[left].Code != cloned[right].Code {
			return cloned[left].Code < cloned[right].Code
		}
		return cloned[left].Subject < cloned[right].Subject
	})
	return cloned
}

func priorityRank(priority string) int {
	switch priority {
	case model.PriorityHigh:
		return 0
	case model.PriorityMedium:
		return 1
	case model.PriorityLow:
		return 2
	case model.PriorityInfo:
		return 3
	default:
		return 4
	}
}
