package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/network"
	"github.com/stellar/go-stellar-sdk/strkey"
)

type baselineListOptions struct {
	network  string
	contract string
	storeDir string
}

func (app *App) runBaseline(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: controlwatch baseline list --network <network> --contract <contract>")
		return 2
	}
	if args[0] != "list" {
		_, _ = fmt.Fprintf(stderr, "unknown baseline command %q\n", args[0])
		return 2
	}
	return app.runBaselineList(ctx, args[1:], stdout, stderr)
}

func (app *App) runBaselineList(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	options, err := app.parseBaselineList(args, stderr)
	if err != nil {
		writeCLIError(stderr, err)
		return exitCode(err)
	}
	home, err := app.userHome()
	if err != nil {
		writeCLIError(stderr, fmt.Errorf("resolve configuration: user home: %w", err))
		return 3
	}
	storeDir := firstConfigured(
		options.storeDir,
		lookupValue(app.lookupEnv, StoreDirEnv),
		filepath.Join(home, ".controlwatch", "snapshots"),
	)
	snapshotStore, err := app.newStore(storeDir)
	if err != nil {
		writeCLIError(stderr, fmt.Errorf("construct snapshot store: %w", err))
		return 3
	}
	references, err := snapshotStore.List(ctx, options.network, options.contract)
	if err != nil {
		writeCLIError(stderr, err)
		return exitCode(err)
	}
	encoded, err := json.MarshalIndent(references, "", "  ")
	if err != nil {
		writeCLIError(stderr, fmt.Errorf("encode baseline list: %w", err))
		return 3
	}
	if _, err := stdout.Write(append(encoded, '\n')); err != nil {
		writeCLIError(stderr, fmt.Errorf("write baseline list: %w", err))
		return 3
	}
	return 0
}

func (app *App) parseBaselineList(args []string, stderr io.Writer) (baselineListOptions, error) {
	flags := flag.NewFlagSet("baseline list", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var options baselineListOptions
	flags.StringVar(&options.network, "network", "", "Stellar network: testnet or mainnet")
	flags.StringVar(&options.contract, "contract", "", "contract C-address")
	flags.StringVar(&options.storeDir, "store-dir", "", "snapshot store directory")
	if err := flags.Parse(args); err != nil {
		return baselineListOptions{}, model.NewError(model.CodeUsage, err)
	}
	if flags.NArg() != 0 {
		return baselineListOptions{}, model.NewError(
			model.CodeUsage,
			fmt.Errorf("baseline list: unexpected arguments %q", flags.Args()),
		)
	}
	networkName, err := network.ParseName(firstConfigured(
		options.network,
		lookupValue(app.lookupEnv, NetworkEnv),
	))
	if err != nil {
		return baselineListOptions{}, model.NewError(model.CodeUsage, err)
	}
	options.network = string(networkName)
	options.contract = firstConfigured(options.contract, lookupValue(app.lookupEnv, contractEnv))
	if _, err := strkey.Decode(strkey.VersionByteContract, options.contract); err != nil {
		return baselineListOptions{}, model.NewError(model.CodeInvalidContractID, err)
	}
	return options, nil
}
