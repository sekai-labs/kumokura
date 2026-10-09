package bootstrap

import (
	"context"
	"fmt"
	"path/filepath"

	accountsAdapters "github.com/sekai-labs/kumokura/internal/accounts/adapters"
	accountsApp "github.com/sekai-labs/kumokura/internal/accounts/application"
	accountPorts "github.com/sekai-labs/kumokura/internal/accounts/ports"
	bucketsAdapters "github.com/sekai-labs/kumokura/internal/buckets/adapters/s3"
	bucketsApp "github.com/sekai-labs/kumokura/internal/buckets/application"
	objectsAdapters "github.com/sekai-labs/kumokura/internal/objects/adapters/s3"
	objectsApp "github.com/sekai-labs/kumokura/internal/objects/application"
	"github.com/sekai-labs/kumokura/internal/platform/config"
	"github.com/sekai-labs/kumokura/internal/platform/database"
	"github.com/sekai-labs/kumokura/internal/providers/adapters/s3client"
	"github.com/sekai-labs/kumokura/internal/security/keyring"
	syncAdapters "github.com/sekai-labs/kumokura/internal/synchronization/adapters"
	syncApp "github.com/sekai-labs/kumokura/internal/synchronization/application"
	syncPorts "github.com/sekai-labs/kumokura/internal/synchronization/ports"
	transfersAdapters "github.com/sekai-labs/kumokura/internal/transfers/adapters"
	transfersApp "github.com/sekai-labs/kumokura/internal/transfers/application"
	transfersPorts "github.com/sekai-labs/kumokura/internal/transfers/ports"
)

type AppContainer struct {
	Config          *config.Config
	DB              *database.DB
	ClientFactory   *s3client.ClientFactory
	AccountService  accountPorts.AccountService
	TransferService *transfersApp.TransferService
	SyncService     syncPorts.SyncEngine
	SyncRepo        syncPorts.SyncRepository
	TransferRepo    transfersPorts.TransferRepository
}

func Initialize(ctx context.Context) (*AppContainer, error) {
	cfg, err := config.DefaultConfig()
	if err != nil {
		return nil, fmt.Errorf("load default config: %w", err)
	}
	return InitializeWithConfig(ctx, cfg)
}

func InitializeWithConfig(ctx context.Context, cfg *config.Config) (*AppContainer, error) {

	if err := cfg.EnsureDirs(); err != nil {
		return nil, fmt.Errorf("ensure dirs: %w", err)
	}

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Migrate(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	keyringStore := keyring.NewSecureKeyring(cfg.SecretsDir)
	credAdapter := accountsAdapters.NewKeyringCredentialAdapter(keyringStore)
	accountRepo := accountsAdapters.NewSQLiteAccountRepository(db.DB)
	accountSvc := accountsApp.NewAccountApplicationService(accountRepo, credAdapter)

	clientFactory, err := s3client.NewClientFactory(nil)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize client factory: %w", err)
	}

	transferRepo := transfersAdapters.NewSQLiteTransferRepository(db)
	transferPub := transfersAdapters.NewInMemoryEventPublisher()
	bufferPool := transfersAdapters.NewTieredBufferPool()
	transferWorker := transfersAdapters.NewS3TransferWorker(nil, bufferPool)
	concurrency := cfg.MaxUploadConcurrency
	if concurrency <= 0 {
		concurrency = 100
	}
	transferSvc := transfersApp.NewTransferService(transferRepo, transferPub, transferWorker, bufferPool, concurrency)

	syncRepo := syncAdapters.NewSQLiteSyncRepository(db.DB)
	syncSvc := syncApp.NewSyncService(syncRepo)

	return &AppContainer{
		Config:          cfg,
		DB:              db,
		ClientFactory:   clientFactory,
		AccountService:  accountSvc,
		TransferService: transferSvc,
		SyncService:     syncSvc,
		SyncRepo:        syncRepo,
		TransferRepo:    transferRepo,
	}, nil
}

func (c *AppContainer) Close() error {
	if c.DB != nil {
		return c.DB.Close()
	}
	return nil
}

func (c *AppContainer) CreateBucketService(ctx context.Context, accountName string) (*bucketsApp.BucketService, error) {
	accounts, err := c.AccountService.ListAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}

	for _, acc := range accounts {
		if accountName == "" || acc.Name == accountName || string(acc.ID) == accountName {
			creds, err := c.AccountService.GetCredentials(ctx, acc.ID)
			if err != nil {
				return nil, fmt.Errorf("retrieve credentials for account %s: %w", acc.Name, err)
			}
			s3Cli, err := c.ClientFactory.Build(ctx, acc, creds)
			if err != nil {
				return nil, fmt.Errorf("build s3 client: %w", err)
			}
			storage := bucketsAdapters.NewS3BucketStorage(s3Cli)
			return bucketsApp.NewBucketService(storage), nil
		}
	}

	return nil, fmt.Errorf("account %q not found", accountName)
}

func (c *AppContainer) CreateObjectService(ctx context.Context, accountName string) (*objectsApp.ObjectService, error) {
	accounts, err := c.AccountService.ListAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}

	for _, acc := range accounts {
		if accountName == "" || acc.Name == accountName || string(acc.ID) == accountName {
			creds, err := c.AccountService.GetCredentials(ctx, acc.ID)
			if err != nil {
				return nil, fmt.Errorf("retrieve credentials for account %s: %w", acc.Name, err)
			}
			s3Cli, err := c.ClientFactory.Build(ctx, acc, creds)
			if err != nil {
				return nil, fmt.Errorf("build s3 client: %w", err)
			}
			storage := objectsAdapters.NewS3ObjectStorage(s3Cli, nil)
			return objectsApp.NewObjectService(storage), nil
		}
	}

	return nil, fmt.Errorf("account %q not found", accountName)
}
func (c *AppContainer) CreateTransferService(ctx context.Context, accountName string) (*transfersApp.TransferService, error) {
	accounts, err := c.AccountService.ListAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	for _, acc := range accounts {
		if accountName == "" || acc.Name == accountName || string(acc.ID) == accountName {
			creds, err := c.AccountService.GetCredentials(ctx, acc.ID)
			if err != nil {
				return nil, fmt.Errorf("retrieve credentials for account %s: %w", acc.Name, err)
			}
			s3Cli, err := c.ClientFactory.Build(ctx, acc, creds)
			if err != nil {
				return nil, fmt.Errorf("build s3 client: %w", err)
			}
			bufferPool := transfersAdapters.NewTieredBufferPool()
			worker := transfersAdapters.NewS3TransferWorker(s3Cli, bufferPool)
			concurrency := c.Config.MaxUploadConcurrency
			if concurrency <= 0 {
				concurrency = 100
			}
			return transfersApp.NewTransferService(c.TransferRepo, transfersAdapters.NewInMemoryEventPublisher(), worker, bufferPool, concurrency), nil
		}
	}
	return nil, fmt.Errorf("account %q not found", accountName)
}

func (c *AppContainer) CreateSyncScanners(ctx context.Context, accountName, source, dest string) (syncPorts.SyncScanner, syncPorts.SyncScanner, error) {
	var srcScanner, dstScanner syncPorts.SyncScanner

	if filepath.IsAbs(source) || source == "." || source[:2] == "./" {
		srcScanner = syncAdapters.NewLocalScanner(true)
	}
	if filepath.IsAbs(dest) || dest == "." || dest[:2] == "./" {
		dstScanner = syncAdapters.NewLocalScanner(true)
	}

	if srcScanner != nil && dstScanner != nil {
		return srcScanner, dstScanner, nil
	}

	accounts, err := c.AccountService.ListAccounts(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("list accounts: %w", err)
	}

	for _, acc := range accounts {
		if accountName == "" || acc.Name == accountName || string(acc.ID) == accountName {
			creds, err := c.AccountService.GetCredentials(ctx, acc.ID)
			if err != nil {
				return nil, nil, err
			}
			s3Cli, err := c.ClientFactory.Build(ctx, acc, creds)
			if err != nil {
				return nil, nil, err
			}

			if srcScanner == nil {
				srcScanner = syncAdapters.NewS3Scanner(s3Cli, source)
			}
			if dstScanner == nil {
				dstScanner = syncAdapters.NewS3Scanner(s3Cli, dest)
			}
			return srcScanner, dstScanner, nil
		}
	}

	return nil, nil, fmt.Errorf("account not found for remote scanning")
}
