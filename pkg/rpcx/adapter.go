package rpcx

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stellar/go-stellar-sdk/clients/rpcclient"
	protocol "github.com/stellar/go-stellar-sdk/protocols/rpc"
	"github.com/stellar/go-stellar-sdk/xdr"
)

type sdkClient interface {
	GetHealth(context.Context) (protocol.GetHealthResponse, error)
	GetLatestLedger(context.Context) (protocol.GetLatestLedgerResponse, error)
	GetLedgerEntries(context.Context, protocol.GetLedgerEntriesRequest) (protocol.GetLedgerEntriesResponse, error)
	SimulateTransaction(context.Context, protocol.SimulateTransactionRequest) (protocol.SimulateTransactionResponse, error)
	GetEvents(context.Context, protocol.GetEventsRequest) (protocol.GetEventsResponse, error)
	GetTransaction(context.Context, protocol.GetTransactionRequest) (protocol.GetTransactionResponse, error)
	Close() error
}

// Adapter maps the pinned Stellar Go SDK client to ControlWatch domain types.
type Adapter struct {
	client sdkClient
}

var _ Client = (*Adapter)(nil)

// NewAdapter constructs an SDK-backed read-only RPC client.
func NewAdapter(rpcURL string, httpClient *http.Client) *Adapter {
	return &Adapter{client: rpcclient.NewClient(rpcURL, httpClient)}
}

// Close releases the SDK client's network resources.
func (a *Adapter) Close() error {
	return a.client.Close()
}

// GetHealth returns the RPC service's retained ledger range.
func (a *Adapter) GetHealth(ctx context.Context) (Health, error) {
	response, err := a.client.GetHealth(ctx)
	if err != nil {
		return Health{}, fmt.Errorf("get health: %w", err)
	}
	return Health{
		Status:                response.Status,
		LatestLedger:          response.LatestLedger,
		OldestLedger:          response.OldestLedger,
		LedgerRetentionWindow: response.LedgerRetentionWindow,
	}, nil
}

// GetLatestLedger returns the newest ledger observed by the RPC service.
func (a *Adapter) GetLatestLedger(ctx context.Context) (LatestLedger, error) {
	response, err := a.client.GetLatestLedger(ctx)
	if err != nil {
		return LatestLedger{}, fmt.Errorf("get latest ledger: %w", err)
	}
	return LatestLedger{
		Sequence:        response.Sequence,
		ProtocolVersion: response.ProtocolVersion,
		Hash:            response.Hash,
		CloseTime:       response.LedgerCloseTime,
	}, nil
}

// GetLedgerEntries returns base64-XDR ledger entries for the supplied keys.
func (a *Adapter) GetLedgerEntries(ctx context.Context, keys []string) (LedgerEntries, error) {
	response, err := a.client.GetLedgerEntries(ctx, protocol.GetLedgerEntriesRequest{
		Keys:   append([]string(nil), keys...),
		Format: protocol.FormatBase64,
	})
	if err != nil {
		return LedgerEntries{}, fmt.Errorf("get ledger entries: %w", err)
	}

	entries := make([]LedgerEntry, len(response.Entries))
	for index, entry := range response.Entries {
		entries[index] = LedgerEntry{
			Key:                entry.KeyXDR,
			XDR:                entry.DataXDR,
			LastModifiedLedger: entry.LastModifiedLedger,
		}
	}
	return LedgerEntries{LatestLedger: response.LatestLedger, Entries: entries}, nil
}

// SimulateTransaction simulates a transaction without submitting it.
func (a *Adapter) SimulateTransaction(ctx context.Context, transactionXDR string) (Simulation, error) {
	response, err := a.client.SimulateTransaction(ctx, protocol.SimulateTransactionRequest{
		Transaction: transactionXDR,
		Format:      protocol.FormatBase64,
	})
	if err != nil {
		return Simulation{}, fmt.Errorf("simulate transaction: %w", err)
	}

	results := make([]SimulationResult, len(response.Results))
	for index, result := range response.Results {
		if result.AuthXDR != nil {
			results[index].Auth = append([]string(nil), (*result.AuthXDR)...)
		}
		if result.ReturnValueXDR != nil {
			results[index].ReturnXDR = *result.ReturnValueXDR
		}
	}
	return Simulation{
		LatestLedger:     response.LatestLedger,
		Error:            response.Error,
		TransactionData:  response.TransactionDataXDR,
		MinResourceFee:   response.MinResourceFee,
		Results:          results,
		DiagnosticEvents: append([]string(nil), response.EventsXDR...),
	}, nil
}

// GetEvents returns one normalized page of retained contract events.
func (a *Adapter) GetEvents(ctx context.Context, request EventsRequest) (EventsPage, error) {
	sdkRequest, err := toSDKEventsRequest(request)
	if err != nil {
		return EventsPage{}, err
	}
	response, err := a.client.GetEvents(ctx, sdkRequest)
	if err != nil {
		return EventsPage{}, fmt.Errorf("get events: %w", err)
	}

	events := make([]Event, len(response.Events))
	for index, event := range response.Events {
		if event.Ledger < 0 {
			return EventsPage{}, fmt.Errorf("get events: negative ledger %d", event.Ledger)
		}
		events[index] = Event{
			ID:              event.ID,
			Type:            event.EventType,
			Ledger:          uint32(event.Ledger),
			LedgerClosedAt:  event.LedgerClosedAt,
			ContractID:      event.ContractID,
			Topic:           append([]string(nil), event.TopicXDR...),
			ValueXDR:        event.ValueXDR,
			TransactionHash: event.TransactionHash,
		}
	}
	return EventsPage{
		LatestLedger: response.LatestLedger,
		Cursor:       response.Cursor,
		Events:       events,
	}, nil
}

// GetTransaction returns a normalized transaction lookup result.
func (a *Adapter) GetTransaction(ctx context.Context, hash string) (Transaction, error) {
	response, err := a.client.GetTransaction(ctx, protocol.GetTransactionRequest{
		Hash:   hash,
		Format: protocol.FormatBase64,
	})
	if err != nil {
		return Transaction{}, fmt.Errorf("get transaction: %w", err)
	}
	return Transaction{
		Status:        response.Status,
		LatestLedger:  response.LatestLedger,
		Hash:          response.TransactionHash,
		Ledger:        response.Ledger,
		CreatedAt:     response.LedgerCloseTime,
		EnvelopeXDR:   response.EnvelopeXDR,
		ResultXDR:     response.ResultXDR,
		ResultMetaXDR: response.ResultMetaXDR,
	}, nil
}

func toSDKEventsRequest(request EventsRequest) (protocol.GetEventsRequest, error) {
	filters := make([]protocol.EventFilter, len(request.Filters))
	for filterIndex, filter := range request.Filters {
		topics := make([]protocol.TopicFilter, len(filter.Topics))
		for topicIndex, topic := range filter.Topics {
			segments := make(protocol.TopicFilter, len(topic))
			for segmentIndex, segment := range topic {
				converted, err := toSDKSegmentFilter(segment)
				if err != nil {
					return protocol.GetEventsRequest{}, fmt.Errorf(
						"event filter %d topic %d segment %d: %w",
						filterIndex,
						topicIndex,
						segmentIndex,
						err,
					)
				}
				segments[segmentIndex] = converted
			}
			topics[topicIndex] = segments
		}
		filters[filterIndex] = protocol.EventFilter{
			ContractIDs: append([]string(nil), filter.ContractIDs...),
			Topics:      topics,
		}
	}

	pagination := &protocol.PaginationOptions{Limit: uint(request.Limit)}
	if request.Cursor != "" {
		cursor, err := protocol.ParseCursor(request.Cursor)
		if err != nil {
			return protocol.GetEventsRequest{}, fmt.Errorf("parse event cursor: %w", err)
		}
		pagination.Cursor = &cursor
	}
	return protocol.GetEventsRequest{
		StartLedger: request.StartLedger,
		EndLedger:   request.EndLedger,
		Filters:     filters,
		Pagination:  pagination,
		Format:      protocol.FormatBase64,
	}, nil
}

func toSDKSegmentFilter(value string) (protocol.SegmentFilter, error) {
	if value == protocol.WildCardExactOne || value == protocol.WildCardZeroOrMore {
		wildcard := value
		return protocol.SegmentFilter{Wildcard: &wildcard}, nil
	}

	var scVal xdr.ScVal
	if err := xdr.SafeUnmarshalBase64(value, &scVal); err != nil {
		return protocol.SegmentFilter{}, fmt.Errorf("decode topic XDR: %w", err)
	}
	return protocol.SegmentFilter{ScVal: &scVal}, nil
}
