package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// Cipher encrypts small secret payloads using AES-256-GCM.
type Cipher struct {
	key []byte
}

// NewCipher creates a new AES-256-GCM cipher from a base64-encoded 32-byte key.
func NewCipher(base64Key string) (*Cipher, error) {
	raw, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("decoding AUTH_SECRETS_MASTER_KEY: %w", err)
	}
	if len(raw) != 32 {
		return nil, fmt.Errorf("AUTH_SECRETS_MASTER_KEY must decode to 32 bytes, got %d", len(raw))
	}
	return &Cipher{key: raw}, nil
}

// EncryptMap encrypts a small JSON object and returns an encoded payload.
func (c *Cipher) EncryptMap(payload map[string]string, aad string) (string, error) {
	if payload == nil {
		payload = map[string]string{}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshaling secret payload: %w", err)
	}
	return c.encryptBytes(raw, aad)
}

// DecryptMap decrypts an encoded payload into a string map.
func (c *Cipher) DecryptMap(ciphertext, aad string) (map[string]string, error) {
	if ciphertext == "" {
		return map[string]string{}, nil
	}
	raw, err := c.decryptBytes(ciphertext, aad)
	if err != nil {
		return nil, err
	}
	var payload map[string]string
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("unmarshaling secret payload: %w", err)
	}
	if payload == nil {
		payload = map[string]string{}
	}
	return payload, nil
}

// EncryptString encrypts a plaintext string and returns an encoded payload.
func (c *Cipher) EncryptString(plaintext, aad string) (string, error) {
	return c.encryptBytes([]byte(plaintext), aad)
}

// DecryptString decrypts an encoded payload to a plaintext string.
func (c *Cipher) DecryptString(ciphertext, aad string) (string, error) {
	raw, err := c.decryptBytes(ciphertext, aad)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (c *Cipher) encryptBytes(plaintext []byte, aad string) (string, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("creating AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("creating GCM cipher: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generating nonce: %w", err)
	}

	sealed := gcm.Seal(nil, nonce, plaintext, []byte(aad))
	return base64.StdEncoding.EncodeToString(nonce) + ":" + base64.StdEncoding.EncodeToString(sealed), nil
}

func (c *Cipher) decryptBytes(ciphertext, aad string) ([]byte, error) {
	parts := strings.Split(ciphertext, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid encrypted payload format")
	}

	nonce, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decoding nonce: %w", err)
	}
	sealed, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decoding ciphertext: %w", err)
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, fmt.Errorf("creating AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM cipher: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, sealed, []byte(aad))
	if err != nil {
		return nil, fmt.Errorf("decrypting ciphertext: %w", err)
	}
	return plaintext, nil
}
