package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"
)

// Identity represents a client with signing and encryption key pairs
type Identity struct {
	SigningPrivateKey ed25519.PrivateKey
	SigningPublicKey  ed25519.PublicKey
	EncryptPrivateKey [32]byte // X25519
	EncryptPublicKey  [32]byte // X25519
}

// NewIdentity generates a new identity with Ed25519 and X25519 key pairs
func NewIdentity() (*Identity, error) {
	// Generate Ed25519 keys
	pubSign, privSign, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	// Generate X25519 keys
	var privEnc [32]byte
	if _, err := rand.Read(privEnc[:]); err != nil {
		return nil, err
	}
	var pubEnc [32]byte
	curve25519.ScalarBaseMult(&pubEnc, &privEnc)

	return &Identity{
		SigningPrivateKey: privSign,
		SigningPublicKey:  pubSign,
		EncryptPrivateKey: privEnc,
		EncryptPublicKey:  pubEnc,
	}, nil
}

// GetID returns a base64-encoded version of the signing public key
func (id *Identity) GetID() string {
	return base64.RawURLEncoding.EncodeToString(id.SigningPublicKey)
}

// SignMessage signs a message using Ed25519
func (id *Identity) SignMessage(message []byte) []byte {
	return ed25519.Sign(id.SigningPrivateKey, message)
}

// VerifySignature verifies the signature for a given message using the sender's public key
func VerifySignature(pub ed25519.PublicKey, message, signature []byte) bool {
	return ed25519.Verify(pub, message, signature)
}

// Encrypt encrypts a message using the recipient's public key
func (id *Identity) Encrypt(recipientPublicKey *[32]byte, message []byte) ([]byte, error) {
	var nonce [24]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, err
	}
	encrypted := box.Seal(nonce[:], message, &nonce, recipientPublicKey, &id.EncryptPrivateKey)
	return encrypted, nil
}

// Decrypt decrypts a message using the sender's public key
func (id *Identity) Decrypt(senderPublicKey *[32]byte, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < 24 {
		return nil, errors.New("ciphertext too short")
	}
	var nonce [24]byte
	copy(nonce[:], ciphertext[:24])
	plaintext, ok := box.Open(nil, ciphertext[24:], &nonce, senderPublicKey, &id.EncryptPrivateKey)
	if !ok {
		return nil, errors.New("decryption failed")
	}
	return plaintext, nil
}
