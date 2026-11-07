package evidence

import (
	"context"
	"fmt"
	"sort"

	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/rpcx"
	"github.com/Controlwatch/control-watch/pkg/surface"
	"github.com/stellar/go-stellar-sdk/xdr"
)

const (
	eventPageSize = 100
	maxEventPages = 1000
)

// EventQuery selects contracts, event names, and an inclusive ledger range.
type EventQuery struct {
	ContractIDs []string
	Names       []surface.EventName
	StartLedger uint32
	EndLedger   uint32
}

// SearchEvents pages through retained contract events and returns exact
// segment-zero topic matches in deterministic order.
func SearchEvents(
	ctx context.Context,
	client rpcx.Client,
	query EventQuery,
) ([]model.EvidenceItem, error) {
	if client == nil {
		return nil, fmt.Errorf("search events: RPC client is required")
	}
	if query.StartLedger == 0 || query.EndLedger < query.StartLedger {
		return nil, fmt.Errorf(
			"search events: invalid ledger range %d..%d",
			query.StartLedger,
			query.EndLedger,
		)
	}
	contracts := sortedUniqueStrings(query.ContractIDs)
	if len(contracts) == 0 {
		return nil, fmt.Errorf("search events: at least one contract ID is required")
	}
	names := sortedUniqueEventNames(query.Names)
	if len(names) == 0 {
		return nil, fmt.Errorf("search events: at least one event name is required")
	}
	topicFilters := make([][]string, 0, len(names))
	allowedNames := make(map[string]struct{}, len(names))
	for _, name := range names {
		encoded, err := encodeEventName(name)
		if err != nil {
			return nil, fmt.Errorf("search events: encode topic %q: %w", name, err)
		}
		topicFilters = append(topicFilters, []string{encoded})
		allowedNames[string(name)] = struct{}{}
	}
	allowedContracts := make(map[string]struct{}, len(contracts))
	for _, contractID := range contracts {
		allowedContracts[contractID] = struct{}{}
	}

	request := rpcx.EventsRequest{
		StartLedger: query.StartLedger,
		EndLedger:   query.EndLedger,
		Limit:       eventPageSize,
		Filters: []rpcx.EventFilter{{
			ContractIDs: contracts,
			Topics:      topicFilters,
		}},
	}
	items := []model.EvidenceItem{}
	for pageNumber := 0; ; pageNumber++ {
		if pageNumber >= maxEventPages {
			return nil, fmt.Errorf("search events: pagination exceeded %d pages", maxEventPages)
		}
		page, err := client.GetEvents(ctx, request)
		if err != nil {
			return nil, fmt.Errorf("search events: %w", err)
		}
		if len(page.Events) == 0 {
			break
		}
		for _, event := range page.Events {
			if _, ok := allowedContracts[event.ContractID]; !ok {
				continue
			}
			name, err := eventName(event)
			if err != nil {
				return nil, fmt.Errorf("search events: event %q: %w", event.ID, err)
			}
			if _, ok := allowedNames[name]; !ok {
				continue
			}
			items = append(items, model.EvidenceItem{
				Ledger: event.Ledger,
				TxHash: event.TransactionHash,
				Topic:  name,
			})
		}
		if page.Cursor == "" {
			break
		}
		if page.Cursor == request.Cursor {
			return nil, fmt.Errorf("search events: paging cursor did not advance")
		}
		request.Cursor = page.Cursor
	}
	sort.SliceStable(items, func(left int, right int) bool {
		if items[left].Ledger != items[right].Ledger {
			return items[left].Ledger < items[right].Ledger
		}
		if items[left].TxHash != items[right].TxHash {
			return items[left].TxHash < items[right].TxHash
		}
		return items[left].Topic < items[right].Topic
	})
	return items, nil
}

func encodeEventName(name surface.EventName) (string, error) {
	value, err := xdr.NewScVal(xdr.ScValTypeScvSymbol, xdr.ScSymbol(name))
	if err != nil {
		return "", err
	}
	return xdr.MarshalBase64(value)
}

func eventName(event rpcx.Event) (string, error) {
	if len(event.Topic) == 0 {
		return "", fmt.Errorf("missing topic segment zero")
	}
	var value xdr.ScVal
	if err := xdr.SafeUnmarshalBase64(event.Topic[0], &value); err != nil {
		return "", fmt.Errorf("decode topic segment zero: %w", err)
	}
	symbol, ok := value.GetSym()
	if !ok {
		return "", fmt.Errorf("topic segment zero is not a symbol")
	}
	return string(symbol), nil
}

func sortedUniqueStrings(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value != "" {
			set[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func sortedUniqueEventNames(values []surface.EventName) []surface.EventName {
	set := make(map[surface.EventName]struct{}, len(values))
	for _, value := range values {
		if value != "" {
			set[value] = struct{}{}
		}
	}
	result := make([]surface.EventName, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Slice(result, func(left int, right int) bool { return result[left] < result[right] })
	return result
}
