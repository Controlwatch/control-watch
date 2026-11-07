package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/Controlwatch/control-watch/pkg/canon"
	"github.com/Controlwatch/control-watch/pkg/diff"
	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/report"
	"github.com/Controlwatch/control-watch/pkg/store"
	"github.com/stellar/go-stellar-sdk/strkey"
)

const (
	baselineEnv    = "CONTROLWATCH_BASELINE"
	saveCurrentEnv = "CONTROLWATCH_SAVE_CURRENT"
)

type compareOptions struct {
	config      ConfigOptions
	contract    string
	baseline    string
	jsonPath    string
	saveCurrent bool
}

func (app *App) runCompare(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	options, err := app.parseCompare(args, stderr)
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
	snapshotStore, err := app.newStore(config.StoreDir)
	if err != nil {
		writeCLIError(stderr, fmt.Errorf("construct snapshot store: %w", err))
		return 3
	}
	baselineRef, baseline, err := loadBaseline(ctx, snapshotStore, string(config.Network.Name), options.contract, options.baseline)
	if err != nil {
		writeCLIError(stderr, err)
		return exitCode(err)
	}

	observer, evidenceRunner, closeObserver, err := app.newComparison(config)
	if err != nil {
		writeCLIError(stderr, fmt.Errorf("construct comparison runtime: %w", err))
		return 3
	}
	if closeObserver != nil {
		defer func() {
			if closeErr := closeObserver(); closeErr != nil {
				_, _ = fmt.Fprintf(stderr, "close comparison runtime: %v\n", closeErr)
			}
		}()
	}
	observed, err := observer.Observe(ctx, options.contract)
	if err != nil {
		writeCLIError(stderr, err)
		return exitCode(err)
	}
	current := app.snapshot(config, options.contract, observed)
	if err := canon.PopulateSnapshotHashes(&current); err != nil {
		writeCLIError(stderr, err)
		return 3
	}
	if options.saveCurrent {
		if _, err := snapshotStore.Save(ctx, current); err != nil {
			writeCLIError(stderr, err)
			return exitCode(err)
		}
	}

	difference := diff.Result{Findings: []model.Finding{}, NotCompared: []model.NotCompared{}}
	if baseline.ConfigHash != current.ConfigHash {
		difference, err = diff.NewDefaultRunner().Run(baseline, current)
		if err != nil {
			writeCLIError(stderr, err)
			return exitCode(err)
		}
	}
	window, err := evidenceRunner.Retention(ctx)
	if err != nil {
		writeCLIError(stderr, err)
		return 3
	}
	difference.Findings, err = evidenceRunner.Attach(ctx, baseline, current, difference.Findings)
	if err != nil {
		writeCLIError(stderr, err)
		return 3
	}
	rpcHost, err := publicRPCHost(config.Network.RPCURL)
	if err != nil {
		writeCLIError(stderr, err)
		return 3
	}
	comparisonReport := buildReport(app, config, options.contract, baselineRef, baseline, current, difference, window.OldestLedger, rpcHost)
	jsonReport, err := report.JSON(comparisonReport)
	if err != nil {
		writeCLIError(stderr, err)
		return 3
	}
	if options.jsonPath != "" {
		if err := os.WriteFile(options.jsonPath, jsonReport, 0o644); err != nil {
			writeCLIError(stderr, fmt.Errorf("write comparison JSON: %w", err))
			return 3
		}
	}
	if _, err := io.WriteString(stdout, report.Human(comparisonReport)); err != nil {
		writeCLIError(stderr, fmt.Errorf("write comparison output: %w", err))
		return 3
	}
	if len(difference.NotCompared) > 0 {
		return 4
	}
	if len(difference.Findings) > 0 {
		return 10
	}
	return 0
}

func (app *App) parseCompare(args []string, stderr io.Writer) (compareOptions, error) {
	flags := flag.NewFlagSet("compare", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var options compareOptions
	BindConfigFlags(flags, &options.config)
	flags.StringVar(&options.contract, "contract", "", "contract C-address")
	flags.StringVar(&options.baseline, "baseline", "", "snapshot ID or latest")
	flags.StringVar(&options.jsonPath, "json", "", "write report JSON to path")
	flags.BoolVar(&options.saveCurrent, "save-current", false, "save the current immutable snapshot")
	if err := flags.Parse(args); err != nil {
		return compareOptions{}, model.NewError(model.CodeUsage, err)
	}
	if flags.NArg() != 0 {
		return compareOptions{}, model.NewError(model.CodeUsage, fmt.Errorf("compare: unexpected arguments %q", flags.Args()))
	}
	options.contract = firstConfigured(options.contract, lookupValue(app.lookupEnv, contractEnv))
	options.baseline = firstConfigured(options.baseline, lookupValue(app.lookupEnv, baselineEnv), "latest")
	options.jsonPath = firstConfigured(options.jsonPath, lookupValue(app.lookupEnv, jsonPathEnv))
	if !flagWasSet(flags, "save-current") {
		value := lookupValue(app.lookupEnv, saveCurrentEnv)
		if value != "" {
			parsed, err := parseBool(value)
			if err != nil {
				return compareOptions{}, model.NewError(model.CodeUsage, fmt.Errorf("invalid %s: %w", saveCurrentEnv, err))
			}
			options.saveCurrent = parsed
		}
	}
	if _, err := strkey.Decode(strkey.VersionByteContract, options.contract); err != nil {
		return compareOptions{}, model.NewError(model.CodeInvalidContractID, err)
	}
	return options, nil
}

func loadBaseline(
	ctx context.Context,
	snapshotStore store.Store,
	network string,
	contractID string,
	selected string,
) (store.SnapshotRef, model.Snapshot, error) {
	if selected != "latest" {
		snapshot, err := snapshotStore.Load(ctx, network, contractID, selected)
		return store.SnapshotRef{ID: selected, LedgerSeq: snapshot.LedgerSeq, ConfigHash: snapshot.ConfigHash}, snapshot, err
	}
	references, err := snapshotStore.List(ctx, network, contractID)
	if err != nil {
		return store.SnapshotRef{}, model.Snapshot{}, err
	}
	if len(references) == 0 {
		return store.SnapshotRef{}, model.Snapshot{}, model.NewError(
			model.CodeSnapshotNotFound,
			fmt.Errorf("no saved snapshots for %s on %s", contractID, network),
		)
	}
	sort.SliceStable(references, func(left int, right int) bool {
		if references[left].LedgerSeq == references[right].LedgerSeq {
			return references[left].ID > references[right].ID
		}
		return references[left].LedgerSeq > references[right].LedgerSeq
	})
	selectedRef := references[0]
	snapshot, err := snapshotStore.Load(ctx, network, contractID, selectedRef.ID)
	return selectedRef, snapshot, err
}

func buildReport(
	app *App,
	config Config,
	contractID string,
	baselineRef store.SnapshotRef,
	baseline model.Snapshot,
	current model.Snapshot,
	difference diff.Result,
	oldestLedger uint32,
	rpcHost string,
) model.Report {
	completeness := model.CompletenessComplete
	if baseline.Completeness == model.CompletenessPartial || current.Completeness == model.CompletenessPartial || len(difference.NotCompared) > 0 {
		completeness = model.CompletenessPartial
	}
	return model.Report{
		SchemaVersion: model.SchemaVersion,
		GeneratedAt:   app.now(),
		Network:       string(config.Network.Name),
		ContractID:    contractID,
		Baseline: model.BaselineReportRef{
			SnapshotID:   baselineRef.ID,
			LedgerSeq:    baseline.LedgerSeq,
			SnapshotHash: baseline.SnapshotHash,
		},
		Current: model.CurrentReportRef{
			LedgerSeq:    current.LedgerSeq,
			SnapshotHash: current.SnapshotHash,
		},
		Findings:     difference.Findings,
		NotCompared:  difference.NotCompared,
		Completeness: completeness,
		RPC:          model.ReportRPC{Host: rpcHost, OldestLedger: oldestLedger},
		Disclaimer:   model.ReportDisclaimer,
	}
}

func publicRPCHost(rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("resolve RPC host: invalid RPC URL")
	}
	return parsed.Host, nil
}
