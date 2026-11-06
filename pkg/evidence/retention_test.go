package evidence

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Controlwatch/control-watch/pkg/rpcx"
)

func TestReadRetentionWindow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		health    rpcx.Health
		healthErr error
		want      RetentionWindow
		wantErr   string
	}{
		{
			name:   "valid range",
			health: rpcx.Health{OldestLedger: 100, LatestLedger: 200},
			want:   RetentionWindow{OldestLedger: 100, LatestLedger: 200},
		},
		{
			name:      "health error",
			healthErr: errors.New("unavailable"),
			wantErr:   "unavailable",
		},
		{
			name:    "zero range",
			health:  rpcx.Health{},
			wantErr: "invalid ledger range",
		},
		{
			name:    "reversed range",
			health:  rpcx.Health{OldestLedger: 201, LatestLedger: 200},
			wantErr: "exceeds latest ledger",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			client := &rpcx.FakeClient{GetHealthFunc: func(context.Context) (rpcx.Health, error) {
				return test.health, test.healthErr
			}}
			got, err := ReadRetentionWindow(context.Background(), client)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadRetentionWindow() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("window = %#v, want %#v", got, test.want)
			}
			if calls := client.Calls().GetHealth; calls != 1 {
				t.Fatalf("getHealth calls = %d, want 1", calls)
			}
		})
	}
}

func TestReadRetentionWindowRequiresClient(t *testing.T) {
	t.Parallel()

	if _, err := ReadRetentionWindow(context.Background(), nil); err == nil {
		t.Fatal("ReadRetentionWindow() error = nil")
	}
}
