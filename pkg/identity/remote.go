package identity

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
)

// RemoteIdentity represents a remote client's identity (public keys)
type RemoteIdentity struct {
	PublicKey ed25519.PublicKey
}

// NewRemoteIdentity creates a new RemoteIdentity from a base64 public key string
func NewRemoteIdentity(idStr string) (*RemoteIdentity, error) {
	pub, err := base64.RawURLEncoding.DecodeString(idStr)
	if err != nil {
		return nil, err
	}
	if len(pub) != ed25519.PublicKeySize {
		return nil, errors.New("invalid public key size")
	}
	key := ed25519.PublicKey(pub)
	return &RemoteIdentity{PublicKey: key}, nil
}

// GetID returns the public key as a URL-safe base64-encoded string
func (id *RemoteIdentity) GetID() string {
	return base64.RawURLEncoding.EncodeToString(id.PublicKey)
}
