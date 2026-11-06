package network

import (
	"errors"
	"reflect"
	"testing"
)

func TestParseName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    Name
		wantErr bool
	}{
		{name: "testnet", input: "testnet", want: Testnet},
		{name: "mainnet case and space", input: " MAINNET ", want: Mainnet},
		{name: "empty", input: "", wantErr: true},
		{name: "unsupported", input: "futurenet", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ParseName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		network     Name
		rpcFlag     string
		horizonFlag string
		env         map[string]string
		want        Config
		wantErr     error
	}{
		{
			name:    "testnet defaults",
			network: Testnet,
			want: Config{
				Name:       Testnet,
				Passphrase: TestnetPassphrase,
				RPCURL:     DefaultTestnetRPCURL,
				HorizonURL: DefaultTestnetHorizonURL,
			},
		},
		{
			name:    "testnet environment",
			network: Testnet,
			env: map[string]string{
				TestnetRPCEnv:     " https://rpc.example.test ",
				TestnetHorizonEnv: " https://horizon.example.test ",
			},
			want: Config{
				Name:       Testnet,
				Passphrase: TestnetPassphrase,
				RPCURL:     "https://rpc.example.test",
				HorizonURL: "https://horizon.example.test",
			},
		},
		{
			name:        "flags override environment",
			network:     Mainnet,
			rpcFlag:     " https://flag-rpc.example ",
			horizonFlag: " https://flag-horizon.example ",
			env: map[string]string{
				MainnetRPCEnv:     "https://env-rpc.example",
				MainnetHorizonEnv: "https://env-horizon.example",
			},
			want: Config{
				Name:       Mainnet,
				Passphrase: MainnetPassphrase,
				RPCURL:     "https://flag-rpc.example",
				HorizonURL: "https://flag-horizon.example",
			},
		},
		{
			name:    "mainnet Horizon default",
			network: Mainnet,
			env:     map[string]string{MainnetRPCEnv: "https://rpc.example"},
			want: Config{
				Name:       Mainnet,
				Passphrase: MainnetPassphrase,
				RPCURL:     "https://rpc.example",
				HorizonURL: DefaultMainnetHorizonURL,
			},
		},
		{
			name:    "missing mainnet RPC",
			network: Mainnet,
			wantErr: ErrMissingMainnetRPC,
		},
		{
			name:    "unsupported network",
			network: Name("futurenet"),
			wantErr: errors.New("unsupported network"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			lookup := func(key string) (string, bool) {
				value, ok := tt.env[key]
				return value, ok
			}

			got, err := Resolve(tt.network, tt.rpcFlag, tt.horizonFlag, lookup)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Resolve() error = nil, want %v", tt.wantErr)
				}
				if errors.Is(tt.wantErr, ErrMissingMainnetRPC) && !errors.Is(err, tt.wantErr) {
					t.Fatalf("Resolve() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve() unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resolve() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
