package main

import (
	"context"
	"log"

	"github.com/sekai-labs/kumokura/internal/bootstrap"
	"github.com/sekai-labs/kumokura/internal/presentation/desktop"
)

func main() {
	ctx := context.Background()
	appContainer, err := bootstrap.Initialize(ctx)
	if err != nil {
		log.Fatalf("initialize kumokura core: %v", err)
	}
	defer appContainer.Close()

	app := desktop.NewDesktopApp(appContainer)
	app.Run()
}
