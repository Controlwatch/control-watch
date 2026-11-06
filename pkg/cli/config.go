// Package cli contains command parsing and orchestration for ControlWatch.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/network"
)

const (
	// NetworkEnv is the environment equivalent of --network.
	NetworkEnv = "CONTROLWATCH_NETWORK"
	// StoreDirEnv is the environment equivalent of --store-dir.
	StoreDirEnv = "CONTROLWATCH_STORE_DIR"
	// TimeoutEnv is the environment equivalent of --timeout.
	TimeoutEnv = "CONTROLWATCH_TIMEOUT"

	defaultTimeout = 15 * time.Second
)

// ConfigOptions are raw flag values before environment resolution.
type ConfigOptions struct {
	Network    string
	RPCURL     string
	HorizonURL string
	StoreDir   string
	Timeout    string
}

// Config is the resolved runtime configuration shared by commands.
type Config struct {
	Network  network.Config
	StoreDir string
	Timeout  time.Duration
}

// BindConfigFlags registers shared flags on an isolated FlagSet.
func BindConfigFlags(flags *flag.FlagSet, options *ConfigOptions) {
	flags.StringVar(&options.Network, "network", "", "Stellar network: testnet or mainnet")
	flags.StringVar(&options.RPCURL, "rpc-url", "", "Stellar RPC endpoint")
	flags.StringVar(&options.HorizonURL, "horizon-url", "", "Horizon endpoint")
	flags.StringVar(&options.StoreDir, "store-dir", "", "snapshot store directory")
	flags.StringVar(&options.Timeout, "timeout", "", "timeout per network call")
}

// ResolveConfig resolves flags over environment values and applies documented
// defaults. It reads the current process environment and user home directory.
func ResolveConfig(options ConfigOptions) (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("resolve configuration: user home: %w", err)
	}
	return resolveConfig(options, os.LookupEnv, home)
}

func resolveConfig(options ConfigOptions, lookup network.LookupEnv, userHome string) (Config, error) {
	networkValue := firstConfigured(options.Network, lookupValue(lookup, NetworkEnv))
	networkName, err := network.ParseName(networkValue)
	if err != nil {
		return Config{}, model.NewError(model.CodeUsage, err)
	}
	networkConfig, err := network.Resolve(
		networkName,
		strings.TrimSpace(options.RPCURL),
		strings.TrimSpace(options.HorizonURL),
		lookup,
	)
	if err != nil {
		if errors.Is(err, network.ErrMissingMainnetRPC) {
			return Config{}, model.NewError(model.CodeConfigMissingRPC, err)
		}
		return Config{}, fmt.Errorf("resolve configuration: network: %w", err)
	}

	storeDirectory := firstConfigured(
		options.StoreDir,
		lookupValue(lookup, StoreDirEnv),
		filepath.Join(userHome, ".controlwatch", "snapshots"),
	)
	timeoutValue := firstConfigured(options.Timeout, lookupValue(lookup, TimeoutEnv))
	timeout := defaultTimeout
	if timeoutValue != "" {
		timeout, err = time.ParseDuration(timeoutValue)
		if err != nil {
			return Config{}, model.NewError(
				model.CodeUsage,
				fmt.Errorf("invalid timeout %q: %w", timeoutValue, err),
			)
		}
	}
	if timeout <= 0 {
		return Config{}, model.NewError(
			model.CodeUsage,
			fmt.Errorf("timeout must be greater than zero"),
		)
	}
	return Config{
		Network:  networkConfig,
		StoreDir: storeDirectory,
		Timeout:  timeout,
	}, nil
}

func lookupValue(lookup network.LookupEnv, name string) string {
	if lookup == nil {
		return ""
	}
	value, ok := lookup(name)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func firstConfigured(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
