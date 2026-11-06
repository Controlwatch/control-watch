package main

import (
	"context"
	"os"

	"github.com/Controlwatch/control-watch/pkg/cli"
)

func main() {
	os.Exit(cli.NewApp().Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}
