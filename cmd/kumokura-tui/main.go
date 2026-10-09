package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sekai-labs/kumokura/internal/bootstrap"
	"github.com/sekai-labs/kumokura/internal/presentation/tui"
)

func main() {
	// Silence stdlib logging to prevent terminal corruption during alt-screen execution
	if logPath := os.Getenv("KUMOKURA_LOG_FILE"); logPath != "" {
		f, err := tea.LogToFile(logPath, "kumokura-tui")
		if err == nil {
			defer f.Close()
		} else {
			log.SetOutput(io.Discard)
		}
	} else {
		log.SetOutput(io.Discard)
	}
	ctx := context.Background()
	app, err := bootstrap.Initialize(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing Kumokura: %v\n", err)
		os.Exit(1)
	}
	defer app.Close()

	services := tui.Services{
		AccountService:  app.AccountService,
		TransferService: app.TransferService,
	}

	accounts, err := app.AccountService.ListAccounts(ctx)
	if err == nil && len(accounts) > 0 {
		activeAccountName := accounts[0].Name
		bSvc, bErr := app.CreateBucketService(ctx, activeAccountName)
		if bErr == nil {
			services.BucketService = bSvc
		}
		oSvc, oErr := app.CreateObjectService(ctx, activeAccountName)
		if oErr == nil {
			services.ObjectService = oSvc
		}
	}

	model := tui.NewModel(services)

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running Kumokura TUI: %v\n", err)
		os.Exit(1)
	}
}
