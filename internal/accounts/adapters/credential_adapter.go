package adapters

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sekai-labs/kumokura/internal/accounts/domain"
	"github.com/sekai-labs/kumokura/internal/security/keyring"
)

const keyringServiceName = "kumokura-s3-credentials"

type KeyringCredentialAdapter struct {
	store keyring.Store
}

func NewKeyringCredentialAdapter(store keyring.Store) *KeyringCredentialAdapter {
	return &KeyringCredentialAdapter{store: store}
}

type serializedCredentials struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	SessionToken    string `json:"session_token,omitempty"`
}

func (a *KeyringCredentialAdapter) Store(ctx context.Context, accountID domain.AccountID, creds domain.Credentials) error {
	payload, err := json.Marshal(serializedCredentials{
		AccessKeyID:     creds.AccessKeyID,
		SecretAccessKey: creds.SecretAccessKey,
		SessionToken:    creds.SessionToken,
	})
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	return a.store.Set(keyringServiceName, string(accountID), string(payload))
}

func (a *KeyringCredentialAdapter) Retrieve(ctx context.Context, accountID domain.AccountID) (domain.Credentials, error) {
	raw, err := a.store.Get(keyringServiceName, string(accountID))
	if err != nil {
		return domain.Credentials{}, fmt.Errorf("retrieve credentials: %w", err)
	}

	var sc serializedCredentials
	if err := json.Unmarshal([]byte(raw), &sc); err != nil {
		return domain.Credentials{}, fmt.Errorf("unmarshal credentials: %w", err)
	}

	return domain.NewCredentials(sc.AccessKeyID, sc.SecretAccessKey, sc.SessionToken)
}

func (a *KeyringCredentialAdapter) Remove(ctx context.Context, accountID domain.AccountID) error {
	return a.store.Delete(keyringServiceName, string(accountID))
}
