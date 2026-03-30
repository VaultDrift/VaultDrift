// Package crypto provides key management and wrapping functions.
package crypto

import (
	"crypto/ecdh"
	"crypto/rand"
	"fmt"
)

// KeyWrapper handles wrapping and unwrapping of file keys.
type KeyWrapper struct {
	masterKey []byte
}

// NewKeyWrapper creates a new key wrapper with the given master key.
func NewKeyWrapper(masterKey []byte) (*KeyWrapper, error) {
	if len(masterKey) != KeySize {
		return nil, fmt.Errorf("invalid master key size: expected %d, got %d", KeySize, len(masterKey))
	}

	return &KeyWrapper{
		masterKey: append([]byte(nil), masterKey...),
	}, nil
}

// WrapKey wraps a file key with the master key using AES-256-GCM.
// Returns the encrypted key (nonce || ciphertext || tag).
func (kw *KeyWrapper) WrapKey(fileKey []byte) ([]byte, error) {
	return Encrypt(fileKey, kw.masterKey)
}

// UnwrapKey unwraps a file key that was wrapped with WrapKey.
func (kw *KeyWrapper) UnwrapKey(wrappedKey []byte) ([]byte, error) {
	return Decrypt(wrappedKey, kw.masterKey)
}

// GenerateFileKey generates a new random 256-bit file encryption key.
func GenerateFileKey() ([]byte, error) {
	return GenerateKey()
}

// ECDHKeyPair represents an ECDH key pair for secure key exchange.
type ECDHKeyPair struct {
	privateKey *ecdh.PrivateKey
	publicKey  *ecdh.PublicKey
}

// GenerateECDHKeyPair generates a new X25519 key pair for ECDH.
func GenerateECDHKeyPair() (*ECDHKeyPair, error) {
	curve := ecdh.X25519()

	privateKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	return &ECDHKeyPair{
		privateKey: privateKey,
		publicKey:  privateKey.PublicKey(),
	}, nil
}

// PublicKeyBytes returns the raw bytes of the public key.
func (kp *ECDHKeyPair) PublicKeyBytes() []byte {
	return kp.publicKey.Bytes()
}

// PrivateKeyBytes returns the raw bytes of the private key.
func (kp *ECDHKeyPair) PrivateKeyBytes() []byte {
	return kp.privateKey.Bytes()
}

// ECDHSharedSecret computes a shared secret using ECDH.
// This derives an AES-256 key from the shared secret.
func ECDHSharedSecret(privateKey *ecdh.PrivateKey, publicKey *ecdh.PublicKey) ([]byte, error) {
	sharedSecret, err := privateKey.ECDH(publicKey)
	if err != nil {
		return nil, fmt.Errorf("ECDH failed: %w", err)
	}

	// Derive an AES-256 key from the shared secret using SHA-256
	key := Hash(sharedSecret)
	return key, nil
}

// ShareFileKey encrypts a file key for sharing with another user.
// Uses ECDH to establish a shared secret, then wraps the file key.
func ShareFileKey(fileKey []byte, senderPrivate, recipientPublic *ecdh.PrivateKey) ([]byte, []byte, error) {
	// Get sender's public key
	senderPublic := senderPrivate.PublicKey()

	// Compute shared secret
	sharedKey, err := ECDHSharedSecret(senderPrivate, recipientPublic.PublicKey())
	if err != nil {
		return nil, nil, err
	}

	// Wrap the file key with the shared secret
	wrappedKey, err := Encrypt(fileKey, sharedKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encrypt file key: %w", err)
	}

	// Return sender's public key (for recipient to verify) and wrapped key
	return senderPublic.Bytes(), wrappedKey, nil
}

// ReceiveFileKey decrypts a file key that was shared using ShareFileKey.
func ReceiveFileKey(wrappedKey, senderPublicBytes []byte, recipientPrivate *ecdh.PrivateKey) ([]byte, error) {
	// Parse sender's public key
	curve := ecdh.X25519()
	senderPublic, err := curve.NewPublicKey(senderPublicBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid sender public key: %w", err)
	}

	// Compute shared secret
	sharedKey, err := ECDHSharedSecret(recipientPrivate, senderPublic)
	if err != nil {
		return nil, err
	}

	// Unwrap the file key
	fileKey, err := Decrypt(wrappedKey, sharedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt file key: %w", err)
	}

	return fileKey, nil
}

// MasterKeyStore securely stores and manages master keys.
// This is a simple in-memory implementation. Production should use hardware security.
type MasterKeyStore struct {
	keys map[string][]byte
}

// NewMasterKeyStore creates a new master key store.
func NewMasterKeyStore() *MasterKeyStore {
	return &MasterKeyStore{
		keys: make(map[string][]byte),
	}
}

// Store stores a master key for a user.
func (mks *MasterKeyStore) Store(userID string, key []byte) {
	// Make a copy to prevent external modification
	mks.keys[userID] = append([]byte(nil), key...)
}

// Get retrieves a master key for a user.
func (mks *MasterKeyStore) Get(userID string) ([]byte, bool) {
	key, ok := mks.keys[userID]
	if !ok {
		return nil, false
	}
	// Return a copy
	return append([]byte(nil), key...), true
}

// Delete removes a master key.
func (mks *MasterKeyStore) Delete(userID string) {
	delete(mks.keys, userID)
}

// Clear wipes all keys from memory.
func (mks *MasterKeyStore) Clear() {
	for k := range mks.keys {
		// Zero out the key
		for i := range mks.keys[k] {
			mks.keys[k][i] = 0
		}
		delete(mks.keys, k)
	}
}
