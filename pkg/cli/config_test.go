package cli

import (
	"flag"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/network"
)

func TestResolveConfigPrecedenceAndDefaults(t *testing.T) {
	t.Parallel()

	environment := map[string]string{
		NetworkEnv:                "mainnet",
		network.TestnetRPCEnv:     "https://env-rpc.example",
		network.TestnetHorizonEnv: "https://env-horizon.example",
		StoreDirEnv:               filepath.Join("env", "snapshots"),
		TimeoutEnv:                "30s",
	}
	lookup := mapLookup(environment)
	config, err := resolveConfig(ConfigOptions{
		Network:    "testnet",
		RPCURL:     "https://flag-rpc.example",
		HorizonURL: "https://flag-horizon.example",
		StoreDir:   filepath.Join("flag", "snapshots"),
		Timeout:    "5s",
	}, lookup, filepath.Join("C:", "Users", "tester"))
	if err != nil {
		t.Fatalf("resolveConfig returned error: %v", err)
	}
	if config.Network.Name != network.Testnet || config.Network.RPCURL != "https://flag-rpc.example" || config.Network.HorizonURL != "https://flag-horizon.example" {
		t.Fatalf("network config = %#v", config.Network)
	}
	if config.StoreDir != filepath.Join("flag", "snapshots") || config.Timeout != 5*time.Second {
		t.Fatalf("runtime config = %#v", config)
	}

	defaults, err := resolveConfig(ConfigOptions{Network: "testnet"}, mapLookup(nil), filepath.Join("home", "tester"))
	if err != nil {
		t.Fatalf("resolve default config: %v", err)
	}
	if defaults.StoreDir != filepath.Join("home", "tester", ".controlwatch", "snapshots") {
		t.Fatalf("default store = %q", defaults.StoreDir)
	}
	if defaults.Timeout != 15*time.Second {
		t.Fatalf("default timeout = %s, want 15s", defaults.Timeout)
	}
}

func TestResolveConfigStableErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		options ConfigOptions
		want    model.ErrorCode
	}{
		{name: "missing network", options: ConfigOptions{}, want: model.CodeUsage},
		{name: "mainnet RPC", options: ConfigOptions{Network: "mainnet"}, want: model.CodeConfigMissingRPC},
		{name: "invalid timeout", options: ConfigOptions{Network: "testnet", Timeout: "later"}, want: model.CodeUsage},
		{name: "zero timeout", options: ConfigOptions{Network: "testnet", Timeout: "0s"}, want: model.CodeUsage},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := resolveConfig(test.options, mapLookup(nil), "home")
			code, ok := model.CodeOf(err)
			if !ok || code != test.want {
				t.Fatalf("error code = (%q, %t), want %q; error = %v", code, ok, test.want, err)
			}
		})
	}
}

func TestBindConfigFlagsUsesIsolatedFlagSet(t *testing.T) {
	t.Parallel()

	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var options ConfigOptions
	BindConfigFlags(flags, &options)
	err := flags.Parse([]string{
		"--network", "testnet",
		"--rpc-url", "https://rpc.example",
		"--horizon-url", "https://horizon.example",
		"--store-dir", "snapshots",
		"--timeout", "9s",
	})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	want := ConfigOptions{
		Network:    "testnet",
		RPCURL:     "https://rpc.example",
		HorizonURL: "https://horizon.example",
		StoreDir:   "snapshots",
		Timeout:    "9s",
	}
	if options != want {
		t.Fatalf("options = %#v, want %#v", options, want)
	}
}

func mapLookup(values map[string]string) network.LookupEnv {
	return func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}
}
