package evidence

import (
	"context"
	"fmt"

	"github.com/Controlwatch/control-watch/pkg/rpcx"
)

// RetentionWindow is the inclusive ledger range retained by an RPC server.
type RetentionWindow struct {
	OldestLedger uint32
	LatestLedger uint32
}

// ReadRetentionWindow reads and validates the RPC server's retained ledger
// range using getHealth.
func ReadRetentionWindow(ctx context.Context, client rpcx.Client) (RetentionWindow, error) {
	if client == nil {
		return RetentionWindow{}, fmt.Errorf("read event retention: RPC client is required")
	}
	health, err := client.GetHealth(ctx)
	if err != nil {
		return RetentionWindow{}, fmt.Errorf("read event retention: %w", err)
	}
	if health.OldestLedger == 0 || health.LatestLedger == 0 {
		return RetentionWindow{}, fmt.Errorf(
			"read event retention: invalid ledger range %d..%d",
			health.OldestLedger,
			health.LatestLedger,
		)
	}
	if health.OldestLedger > health.LatestLedger {
		return RetentionWindow{}, fmt.Errorf(
			"read event retention: oldest ledger %d exceeds latest ledger %d",
			health.OldestLedger,
			health.LatestLedger,
		)
	}
	return RetentionWindow{
		OldestLedger: health.OldestLedger,
		LatestLedger: health.LatestLedger,
	}, nil
}
