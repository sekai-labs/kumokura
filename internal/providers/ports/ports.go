package ports

import (
	"context"

	accountdomain "github.com/sekai-labs/kumokura/internal/accounts/domain"
	"github.com/sekai-labs/kumokura/internal/providers/domain"
)

type ProviderRegistry interface {
	GetCapabilities(ctx context.Context, accType accountdomain.AccountType) (domain.ProviderCapability, error)
	ListSupportedTypes(ctx context.Context) []accountdomain.AccountType
}

type ProviderService interface {
	GetCapabilities(ctx context.Context, accType accountdomain.AccountType) (domain.ProviderCapability, error)
	CheckCapability(ctx context.Context, accType accountdomain.AccountType, feature domain.Feature) (bool, error)
	ListSupportedTypes(ctx context.Context) []accountdomain.AccountType
}
