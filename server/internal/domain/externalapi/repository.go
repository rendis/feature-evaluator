package externalapi

import "context"

// Repository defines persistence for reusable external APIs.
type Repository interface {
	Create(ctx context.Context, api *ExternalAPI) error
	GetByKey(ctx context.Context, key string) (*ExternalAPI, error)
	Update(ctx context.Context, currentKey string, api *ExternalAPI) error
	Delete(ctx context.Context, key string) error
	List(ctx context.Context) ([]ExternalAPI, error)
	CountRuleUsages(ctx context.Context, key string) (int64, error)
}

// SecretCipher encrypts and decrypts external API secrets.
type SecretCipher interface {
	EncryptMap(payload map[string]string, aad string) (string, error)
	DecryptMap(ciphertext string, aad string) (map[string]string, error)
}
