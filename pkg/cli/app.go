package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/Controlwatch/control-watch/pkg/evidence"
	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/network"
	"github.com/Controlwatch/control-watch/pkg/observe"
	"github.com/Controlwatch/control-watch/pkg/rpcx"
	"github.com/Controlwatch/control-watch/pkg/store"
	"github.com/Controlwatch/control-watch/pkg/version"
)

const simulationSourceAccount = "GDFENFR7BNSY76YUW7OSTGNOJHS4VBDMJ2Z6D3VX7EXWCZHVDY43SWC7"

type observationRunner interface {
	Observe(ctx context.Context, contractID string) (observe.Result, error)
}

type closeFunc func() error

type observerFactory func(Config) (observationRunner, closeFunc, error)
type storeFactory func(string) (store.Store, error)
type comparisonFactory func(Config) (observationRunner, comparisonEvidence, closeFunc, error)

type comparisonEvidence interface {
	Retention(context.Context) (evidence.RetentionWindow, error)
	Attach(context.Context, model.Snapshot, model.Snapshot, []model.Finding) ([]model.Finding, error)
}

// App owns CLI dependencies and dispatches commands.
type App struct {
	newObserver   observerFactory
	newComparison comparisonFactory
	newStore      storeFactory
	lookupEnv     network.LookupEnv
	userHome      func() (string, error)
	now           func() string
	toolVersion   string
}

// NewApp constructs the production command application.
func NewApp() *App {
	return &App{
		newObserver:   productionObserver,
		newComparison: productionComparison,
		newStore: func(root string) (store.Store, error) {
			return store.NewFileStore(root)
		},
		lookupEnv:   os.LookupEnv,
		userHome:    os.UserHomeDir,
		now:         utcNow,
		toolVersion: version.Current,
	}
}

// Run dispatches one CLI invocation and returns its process exit code.
func (app *App) Run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: controlwatch <inspect|baseline|compare|version>")
		return 2
	}
	switch args[0] {
	case "inspect":
		return app.runInspect(ctx, args[1:], stdout, stderr)
	case "baseline":
		return app.runBaseline(ctx, args[1:], stdout, stderr)
	case "compare":
		return app.runCompare(ctx, args[1:], stdout, stderr)
	case "version":
		return app.runVersion(args[1:], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
}

func productionComparison(config Config) (observationRunner, comparisonEvidence, closeFunc, error) {
	httpClient := &http.Client{Timeout: config.Timeout}
	adapter := rpcx.NewAdapter(config.Network.RPCURL, httpClient)
	retrying := rpcx.NewRetryClient(adapter, config.Timeout)
	flagsReader, err := observe.NewHorizonFlagsClient(config.Network.HorizonURL, httpClient)
	if err != nil {
		_ = adapter.Close()
		return nil, nil, nil, err
	}
	observer, err := observe.NewObserver(retrying, flagsReader, simulationSourceAccount, config.Network.Passphrase)
	if err != nil {
		_ = adapter.Close()
		return nil, nil, nil, err
	}
	evidenceRunner, err := evidence.NewEnricher(retrying)
	if err != nil {
		_ = adapter.Close()
		return nil, nil, nil, err
	}
	return observer, evidenceRunner, adapter.Close, nil
}

func productionObserver(config Config) (observationRunner, closeFunc, error) {
	httpClient := &http.Client{Timeout: config.Timeout}
	adapter := rpcx.NewAdapter(config.Network.RPCURL, httpClient)
	retrying := rpcx.NewRetryClient(adapter, config.Timeout)
	flagsReader, err := observe.NewHorizonFlagsClient(config.Network.HorizonURL, httpClient)
	if err != nil {
		_ = adapter.Close()
		return nil, nil, err
	}
	observer, err := observe.NewObserver(
		retrying,
		flagsReader,
		simulationSourceAccount,
		config.Network.Passphrase,
	)
	if err != nil {
		_ = adapter.Close()
		return nil, nil, err
	}
	return observer, adapter.Close, nil
}
