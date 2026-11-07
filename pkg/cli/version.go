package cli

import (
	"fmt"
	"io"
	"runtime"

	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/version"
)

func (app *App) runVersion(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) != 0 {
		_, _ = fmt.Fprintf(stderr, "version: unexpected arguments %q\n", args)
		return 2
	}
	_, err := fmt.Fprintf(
		stdout,
		"controlwatch %s\nschema %s\ngo %s\nstellar-go-sdk %s\n",
		app.toolVersion,
		model.SchemaVersion,
		runtime.Version(),
		version.StellarGoSDK,
	)
	if err != nil {
		writeCLIError(stderr, fmt.Errorf("write version output: %w", err))
		return 3
	}
	return 0
}
