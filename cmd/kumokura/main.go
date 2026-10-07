package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sekai-labs/kumokura/internal/bootstrap"
	"github.com/sekai-labs/kumokura/internal/presentation/cli"
)

func main() {
	ctx := context.Background()
	app, err := bootstrap.Initialize(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Initialization error: %v\n", err)
		os.Exit(1)
	}
	defer app.Close()

	rootCmd := cli.NewRootCmd(app, os.Stdout, os.Stderr)
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
