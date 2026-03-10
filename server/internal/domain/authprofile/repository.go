package authprofile

import "context"

// Repository defines persistence for auth profiles.
type Repository interface {
	Create(ctx context.Context, profile *Profile) error
	GetByKey(ctx context.Context, key string) (*Profile, error)
	Update(ctx context.Context, currentKey string, profile *Profile) error
	Delete(ctx context.Context, key string) error
	List(ctx context.Context) ([]Profile, error)
	CountFeatureUsages(ctx context.Context, key string) (int64, error)
}

// SecretCipher encrypts and decrypts auth profile secrets.
type SecretCipher interface {
	EncryptMap(payload map[string]string, aad string) (string, error)
	DecryptMap(ciphertext string, aad string) (map[string]string, error)
}
