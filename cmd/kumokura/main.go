package main

import (
	"context"
	"os"

	"github.com/sekai-labs/kumokura/internal/bootstrap"
	"github.com/sekai-labs/kumokura/internal/presentation/cli"
)

func main() {
	ctx := context.Background()
	app, err := bootstrap.Initialize(ctx)
	if err != nil {
		os.Stderr.WriteString("Initialization error: " + err.Error() + "\n")
		os.Exit(1)
	}
	defer app.Close()

	rootCmd := cli.NewRootCmd(app, os.Stdout, os.Stderr)
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
