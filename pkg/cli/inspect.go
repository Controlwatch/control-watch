package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Controlwatch/control-watch/pkg/canon"
	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/observe"
	"github.com/stellar/go-stellar-sdk/strkey"
)

const (
	contractEnv = "CONTROLWATCH_CONTRACT"
	jsonPathEnv = "CONTROLWATCH_JSON"
	saveEnv     = "CONTROLWATCH_SAVE"
)

type inspectOptions struct {
	config   ConfigOptions
	contract string
	jsonPath string
	save     bool
}

func (app *App) runInspect(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	options, err := app.parseInspect(args, stderr)
	if err != nil {
		writeCLIError(stderr, err)
		return exitCode(err)
	}
	home, err := app.userHome()
	if err != nil {
		writeCLIError(stderr, fmt.Errorf("resolve configuration: user home: %w", err))
		return 3
	}
	config, err := resolveConfig(options.config, app.lookupEnv, home)
	if err != nil {
		writeCLIError(stderr, err)
		return exitCode(err)
	}

	observer, closeObserver, err := app.newObserver(config)
	if err != nil {
		writeCLIError(stderr, fmt.Errorf("construct observer: %w", err))
		return 3
	}
	if closeObserver != nil {
		defer func() {
			if closeErr := closeObserver(); closeErr != nil {
				_, _ = fmt.Fprintf(stderr, "close observer: %v\n", closeErr)
			}
		}()
	}
	observed, err := observer.Observe(ctx, options.contract)
	if err != nil {
		writeCLIError(stderr, err)
		return exitCode(err)
	}
	snapshot := app.snapshot(config, options.contract, observed)
	if err := canon.PopulateSnapshotHashes(&snapshot); err != nil {
		writeCLIError(stderr, err)
		return 3
	}
	if options.save {
		snapshotStore, err := app.newStore(config.StoreDir)
		if err != nil {
			writeCLIError(stderr, err)
			return 3
		}
		if _, err := snapshotStore.Save(ctx, snapshot); err != nil {
			writeCLIError(stderr, err)
			return exitCode(err)
		}
	}
	encoded, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		writeCLIError(stderr, fmt.Errorf("encode inspection: %w", err))
		return 3
	}
	encoded = append(encoded, '\n')
	if options.jsonPath != "" {
		if err := os.WriteFile(options.jsonPath, encoded, 0o644); err != nil {
			writeCLIError(stderr, fmt.Errorf("write inspection JSON: %w", err))
			return 3
		}
	}
	if _, err := stdout.Write(encoded); err != nil {
		writeCLIError(stderr, fmt.Errorf("write inspection output: %w", err))
		return 3
	}
	return 0
}

func (app *App) snapshot(config Config, contractID string, observed observe.Result) model.Snapshot {
	return model.Snapshot{
		SchemaVersion: model.SchemaVersion,
		ToolVersion:   app.toolVersion,
		Network:       string(config.Network.Name),
		ContractID:    contractID,
		LedgerSeq:     observed.LedgerSeq,
		ObservedAt:    app.now(),
		Completeness:  observed.Completeness,
		Warnings:      observed.Warnings,
		Observation:   observed.Observation,
	}
}

func (app *App) parseInspect(args []string, stderr io.Writer) (inspectOptions, error) {
	flags := flag.NewFlagSet("inspect", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var options inspectOptions
	BindConfigFlags(flags, &options.config)
	flags.StringVar(&options.contract, "contract", "", "contract C-address")
	flags.StringVar(&options.jsonPath, "json", "", "write snapshot JSON to path")
	flags.BoolVar(&options.save, "save", false, "save immutable snapshot")
	if err := flags.Parse(args); err != nil {
		return inspectOptions{}, model.NewError(model.CodeUsage, err)
	}
	if flags.NArg() != 0 {
		return inspectOptions{}, model.NewError(
			model.CodeUsage,
			fmt.Errorf("inspect: unexpected arguments %q", flags.Args()),
		)
	}
	options.contract = firstConfigured(options.contract, lookupValue(app.lookupEnv, contractEnv))
	options.jsonPath = firstConfigured(options.jsonPath, lookupValue(app.lookupEnv, jsonPathEnv))
	if !flagWasSet(flags, "save") {
		value := lookupValue(app.lookupEnv, saveEnv)
		if value != "" {
			parsed, err := parseBool(value)
			if err != nil {
				return inspectOptions{}, model.NewError(model.CodeUsage, fmt.Errorf("invalid %s: %w", saveEnv, err))
			}
			options.save = parsed
		}
	}
	if _, err := strkey.Decode(strkey.VersionByteContract, options.contract); err != nil {
		return inspectOptions{}, model.NewError(model.CodeInvalidContractID, err)
	}
	return options, nil
}

func exitCode(err error) int {
	code, ok := model.CodeOf(err)
	if !ok {
		return 3
	}
	switch code {
	case model.CodeUsage, model.CodeConfigMissingRPC, model.CodeInvalidContractID:
		return 2
	case model.CodeComparisonImpossible:
		return 4
	default:
		return 3
	}
}

func writeCLIError(destination io.Writer, err error) {
	_, _ = fmt.Fprintln(destination, err)
}

func flagWasSet(flags *flag.FlagSet, name string) bool {
	set := false
	flags.Visit(func(value *flag.Flag) {
		if value.Name == name {
			set = true
		}
	})
	return set
}

func parseBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("expected a boolean, got %q", value)
	}
}

func utcNow() string {
	return time.Now().UTC().Format(time.RFC3339)
}
