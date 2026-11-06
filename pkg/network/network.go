package network

import (
	"errors"
	"fmt"
	"strings"
)

// Name identifies a supported Stellar network.
type Name string

const (
	// Testnet identifies Stellar testnet.
	Testnet Name = "testnet"
	// Mainnet identifies Stellar public network.
	Mainnet Name = "mainnet"

	// TestnetPassphrase is the canonical Stellar testnet passphrase.
	TestnetPassphrase = "Test SDF Network ; September 2015"
	// MainnetPassphrase is the canonical Stellar public-network passphrase.
	MainnetPassphrase = "Public Global Stellar Network ; September 2015"

	// DefaultTestnetRPCURL is the public Stellar testnet RPC endpoint.
	DefaultTestnetRPCURL = "https://soroban-testnet.stellar.org"
	// DefaultTestnetHorizonURL is the public Stellar testnet Horizon endpoint.
	DefaultTestnetHorizonURL = "https://horizon-testnet.stellar.org"
	// DefaultMainnetHorizonURL is the public Stellar mainnet Horizon endpoint.
	DefaultMainnetHorizonURL = "https://horizon.stellar.org"

	// TestnetRPCEnv is the testnet RPC environment variable.
	TestnetRPCEnv = "CONTROLWATCH_TESTNET_RPC"
	// MainnetRPCEnv is the mainnet RPC environment variable.
	MainnetRPCEnv = "CONTROLWATCH_MAINNET_RPC"
	// TestnetHorizonEnv is the testnet Horizon environment variable.
	TestnetHorizonEnv = "CONTROLWATCH_TESTNET_HORIZON"
	// MainnetHorizonEnv is the mainnet Horizon environment variable.
	MainnetHorizonEnv = "CONTROLWATCH_MAINNET_HORIZON"
)

// ErrMissingMainnetRPC indicates that mainnet was selected without an
// explicitly configured RPC endpoint.
var ErrMissingMainnetRPC = errors.New("mainnet RPC URL is required")

// Config contains the resolved, public network settings used by ControlWatch.
type Config struct {
	Name       Name
	Passphrase string
	RPCURL     string
	HorizonURL string
}

// LookupEnv returns an environment value and whether it exists.
type LookupEnv func(string) (string, bool)

// ParseName parses a supported network name.
func ParseName(value string) (Name, error) {
	switch Name(strings.ToLower(strings.TrimSpace(value))) {
	case Testnet:
		return Testnet, nil
	case Mainnet:
		return Mainnet, nil
	default:
		return "", fmt.Errorf("unsupported network %q", value)
	}
}

// Resolve constructs network configuration. Explicit flag values take
// precedence over environment variables. Testnet has a public RPC default;
// mainnet deliberately does not.
func Resolve(name Name, rpcFlag, horizonFlag string, lookup LookupEnv) (Config, error) {
	if lookup == nil {
		lookup = func(string) (string, bool) { return "", false }
	}

	switch name {
	case Testnet:
		return Config{
			Name:       Testnet,
			Passphrase: TestnetPassphrase,
			RPCURL: firstNonEmpty(
				rpcFlag,
				lookupValue(lookup, TestnetRPCEnv),
				DefaultTestnetRPCURL,
			),
			HorizonURL: firstNonEmpty(
				horizonFlag,
				lookupValue(lookup, TestnetHorizonEnv),
				DefaultTestnetHorizonURL,
			),
		}, nil
	case Mainnet:
		rpcURL := firstNonEmpty(rpcFlag, lookupValue(lookup, MainnetRPCEnv))
		if rpcURL == "" {
			return Config{}, ErrMissingMainnetRPC
		}
		return Config{
			Name:       Mainnet,
			Passphrase: MainnetPassphrase,
			RPCURL:     rpcURL,
			HorizonURL: firstNonEmpty(
				horizonFlag,
				lookupValue(lookup, MainnetHorizonEnv),
				DefaultMainnetHorizonURL,
			),
		}, nil
	default:
		return Config{}, fmt.Errorf("unsupported network %q", name)
	}
}

func lookupValue(lookup LookupEnv, name string) string {
	value, ok := lookup(name)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
