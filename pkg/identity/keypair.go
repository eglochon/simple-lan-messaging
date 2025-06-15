package identity

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type serializedKey struct {
	SigningPrivate string `json:"signing_private"`
	SigningPublic  string `json:"signing_public"`
	EncryptPrivate string `json:"encrypt_private"`
	EncryptPublic  string `json:"encrypt_public"`
}

// Save writes the keypair to a JSON file
func (id *Identity) Save(path string) error {
	data := serializedKey{
		SigningPrivate: base64.RawURLEncoding.EncodeToString(id.SigningPrivateKey),
		SigningPublic:  base64.RawURLEncoding.EncodeToString(id.SigningPublicKey),
		EncryptPrivate: base64.RawURLEncoding.EncodeToString(id.EncryptPrivateKey[:]),
		EncryptPublic:  base64.RawURLEncoding.EncodeToString(id.EncryptPublicKey[:]),
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(data)
}

// LoadIdentity loads a saved keypair from a file
func LoadIdentity(path string) (*Identity, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data serializedKey
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return nil, err
	}

	// Decode Ed25519 keys
	signingPriv, err := base64.RawURLEncoding.DecodeString(data.SigningPrivate)
	if err != nil || len(signingPriv) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid signing private key")
	}
	signingPub, err := base64.RawURLEncoding.DecodeString(data.SigningPublic)
	if err != nil || len(signingPub) != ed25519.PublicKeySize {
		return nil, errors.New("invalid signing public key")
	}

	// Decode X25519 keys
	encryptPrivBytes, err := base64.RawURLEncoding.DecodeString(data.EncryptPrivate)
	if err != nil || len(encryptPrivBytes) != 32 {
		return nil, errors.New("invalid encryption private key")
	}
	var encryptPriv [32]byte
	copy(encryptPriv[:], encryptPrivBytes)

	encryptPubBytes, err := base64.RawURLEncoding.DecodeString(data.EncryptPublic)
	if err != nil || len(encryptPubBytes) != 32 {
		return nil, errors.New("invalid encryption public key")
	}
	var encryptPub [32]byte
	copy(encryptPub[:], encryptPubBytes)

	return &Identity{
		SigningPrivateKey: ed25519.PrivateKey(signingPriv),
		SigningPublicKey:  ed25519.PublicKey(signingPub),
		EncryptPrivateKey: encryptPriv,
		EncryptPublicKey:  encryptPub,
	}, nil
}

// GetOrCreate tries to load an identity from the given path,
// or generates and saves a new one if not found or invalid.
func GetOrCreateIdentity(path string) (*Identity, error) {
	// Try to load
	id, err := LoadIdentity(path)
	if err == nil {
		return id, nil
	}

	// Generate new
	id, err = NewIdentity()
	if err != nil {
		return nil, err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}

	// Save to path
	if err := id.Save(path); err != nil {
		return nil, err
	}

	return id, nil
}

func DefaultPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "identity.json"
	}
	return filepath.Join(cwd, "identity.json")
}
