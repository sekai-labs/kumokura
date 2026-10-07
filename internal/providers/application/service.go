package application

import (
	"context"

	accountdomain "github.com/sekai-labs/kumokura/internal/accounts/domain"
	"github.com/sekai-labs/kumokura/internal/providers/domain"
	"github.com/sekai-labs/kumokura/internal/providers/ports"
)

type ProviderApplicationService struct{}

func NewProviderApplicationService() *ProviderApplicationService {
	return &ProviderApplicationService{}
}

var _ ports.ProviderService = (*ProviderApplicationService)(nil)
var _ ports.ProviderRegistry = (*ProviderApplicationService)(nil)

func (s *ProviderApplicationService) GetCapabilities(ctx context.Context, accType accountdomain.AccountType) (domain.ProviderCapability, error) {
	if !accType.IsValid() {
		return domain.ProviderCapability{}, accountdomain.ErrUnsupportedType
	}
	return domain.CapabilitiesForType(accType), nil
}

func (s *ProviderApplicationService) CheckCapability(ctx context.Context, accType accountdomain.AccountType, feature domain.Feature) (bool, error) {
	caps, err := s.GetCapabilities(ctx, accType)
	if err != nil {
		return false, err
	}
	return caps.SupportsFeature(feature), nil
}

func (s *ProviderApplicationService) ListSupportedTypes(ctx context.Context) []accountdomain.AccountType {
	return []accountdomain.AccountType{
		accountdomain.TypeAWS,
		accountdomain.TypeMinIO,
		accountdomain.TypeCloudflareR2,
		accountdomain.TypeWasabi,
		accountdomain.TypeBackblazeB2,
		accountdomain.TypeDigitalOceanSpaces,
		accountdomain.TypeCeph,
		accountdomain.TypeCustomS3,
	}
}
