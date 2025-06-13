package identity

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
)

// Identity represents a client's identity (public + private keys)
type Identity struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

// Generate creates a new identity
func Generate() (*Identity, error) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}
	return &Identity{PrivateKey: priv, PublicKey: pub}, nil
}

// GetID returns the public key as a URL-safe base64-encoded string
func (id *Identity) GetID() string {
	return base64.RawURLEncoding.EncodeToString(id.PublicKey)
}

// ParseID decodes a base64 public key string into a byte slice
func ParseID(idStr string) (ed25519.PublicKey, error) {
	pub, err := base64.RawURLEncoding.DecodeString(idStr)
	if err != nil {
		return nil, err
	}
	if len(pub) != ed25519.PublicKeySize {
		return nil, errors.New("invalid public key size")
	}
	return ed25519.PublicKey(pub), nil
}
