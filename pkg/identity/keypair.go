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
	Private string `json:"private"`
	Public  string `json:"public"`
}

// Save writes the keypair to a JSON file
func (id *Identity) Save(path string) error {
	data := serializedKey{
		Private: base64.RawURLEncoding.EncodeToString(id.PrivateKey),
		Public:  base64.RawURLEncoding.EncodeToString(id.PublicKey),
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
	err = json.NewDecoder(file).Decode(&data)
	if err != nil {
		return nil, err
	}

	priv, err := base64.RawURLEncoding.DecodeString(data.Private)
	if err != nil || len(priv) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid private key")
	}
	pub, err := base64.RawURLEncoding.DecodeString(data.Public)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return nil, errors.New("invalid public key")
	}

	return &Identity{
		PrivateKey: ed25519.PrivateKey(priv),
		PublicKey:  ed25519.PublicKey(pub),
	}, nil
}

// GetOrCreate tries to load an identity from the given path,
// or generates and saves a new one if not found or invalid.
func GetOrCreate(path string) (*Identity, error) {
	// Try to load
	id, err := LoadIdentity(path)
	if err == nil {
		return id, nil
	}

	// Generate new
	id, err = Generate()
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
	return filepath.Join(cwd, "data", "identity.json")
}
