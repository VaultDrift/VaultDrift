// Package chunk provides encrypted chunking integration for VaultDrift.
// This extends the base chunking with encryption/decryption capabilities.
package chunk

import (
	"context"
	"fmt"
	"io"

	"github.com/vaultdrift/vaultdrift/internal/crypto"
	"github.com/vaultdrift/vaultdrift/internal/storage"
)

// EncryptedChunkInfo extends ChunkInfo with encryption metadata.
type EncryptedChunkInfo struct {
	ChunkInfo
	EncryptedSize int  // Size after encryption (includes overhead)
	KeyWrapped    bool // Whether the key is wrapped with a master key
}

// ChunkEncryptor handles chunking and encryption of data.
type ChunkEncryptor struct {
	chunker *Chunker
}

// NewChunkEncryptor creates a new chunk encryptor with default chunker settings.
func NewChunkEncryptor() *ChunkEncryptor {
	return &ChunkEncryptor{
		chunker: DefaultChunker(),
	}
}

// NewChunkEncryptorWithParams creates a chunk encryptor with custom chunk sizes.
func NewChunkEncryptorWithParams(min, avg, max int) *ChunkEncryptor {
	return &ChunkEncryptor{
		chunker: NewChunker(min, avg, max),
	}
}

// ChunkEncrypted chunks data and encrypts each chunk.
// Returns chunk metadata and the encrypted chunk data.
// The chunk hash is computed on the PLAINTEXT for deduplication.
func (ce *ChunkEncryptor) ChunkEncrypted(r io.Reader, fileKey []byte) ([]ChunkInfo, [][]byte, error) {
	// First, get chunks and data
	chunkInfos, chunkData, err := ce.chunker.ChunkWithData(r)
	if err != nil {
		return nil, nil, fmt.Errorf("chunking failed: %w", err)
	}

	// Encrypt each chunk
	encryptedData := make([][]byte, len(chunkData))
	for i, data := range chunkData {
		encrypted, err := crypto.Encrypt(data, fileKey)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to encrypt chunk %d: %w", i, err)
		}
		encryptedData[i] = encrypted
	}

	return chunkInfos, encryptedData, nil
}

// EncryptManifest encrypts a file's manifest data.
func EncryptManifest(manifest *Manifest, fileKey []byte) ([]byte, error) {
	data, err := manifest.Serialize()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize manifest: %w", err)
	}

	encrypted, err := crypto.Encrypt(data, fileKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt manifest: %w", err)
	}

	return encrypted, nil
}

// DecryptManifest decrypts an encrypted manifest.
func DecryptManifest(encrypted []byte, fileKey []byte) (*Manifest, error) {
	decrypted, err := crypto.Decrypt(encrypted, fileKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt manifest: %w", err)
	}

	manifest, err := DeserializeManifest(decrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize manifest: %w", err)
	}

	return manifest, nil
}

// EncryptedReassembler handles reassembly of encrypted chunks.
type EncryptedReassembler struct {
	storage storage.Backend
}

// NewEncryptedReassembler creates a new encrypted reassembler.
func NewEncryptedReassembler(store storage.Backend) *EncryptedReassembler {
	return &EncryptedReassembler{
		storage: store,
	}
}

// ReassembleDecrypt fetches encrypted chunks, decrypts them, and writes to output.
func (er *EncryptedReassembler) ReassembleDecrypt(ctx context.Context, manifest *Manifest, fileKey []byte, w io.Writer) error {
	// Validate manifest
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	for i, chunk := range manifest.Chunks {
		// Fetch encrypted chunk
		encryptedData, err := er.storage.Get(ctx, chunk.Hash)
		if err != nil {
			return fmt.Errorf("failed to get chunk %d (%s): %w", i, chunk.Hash, err)
		}

		// Decrypt chunk
		plaintext, err := crypto.Decrypt(encryptedData, fileKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt chunk %d: %w", i, err)
		}

		// Verify size
		if len(plaintext) != chunk.Size {
			return fmt.Errorf("chunk %d size mismatch: expected %d, got %d", i, chunk.Size, len(plaintext))
		}

		// Write to output
		if _, err := w.Write(plaintext); err != nil {
			return fmt.Errorf("failed to write chunk %d: %w", i, err)
		}
	}

	return nil
}

// EncryptAndStore encrypts and stores a file in chunks.
// Returns the manifest and any error.
func EncryptAndStore(ctx context.Context, store storage.Backend, r io.Reader, fileID, deviceID string, fileKey []byte) (*Manifest, error) {
	// Create chunk encryptor
	encryptor := NewChunkEncryptor()

	// Chunk and encrypt
	chunkInfos, encryptedData, err := encryptor.ChunkEncrypted(r, fileKey)
	if err != nil {
		return nil, fmt.Errorf("chunk encryption failed: %w", err)
	}

	// Store encrypted chunks
	for i, info := range chunkInfos {
		if err := store.Put(ctx, info.Hash, encryptedData[i]); err != nil {
			return nil, fmt.Errorf("failed to store chunk %d: %w", i, err)
		}
	}

	// Create manifest
	manifestID, err := generateManifestID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate manifest ID: %w", err)
	}

	manifest := NewManifest(manifestID, fileID, deviceID, chunkInfos)

	return manifest, nil
}

// generateManifestID generates a unique manifest ID.
func generateManifestID() (string, error) {
	// Use crypto package for secure random bytes
	b, err := crypto.RandomBytes(16)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}

// WrappedKeyInfo holds a wrapped (encrypted) file key.
type WrappedKeyInfo struct {
	WrappedKey []byte // The file key encrypted with a master key
	MasterKeyID string // Identifier for the master key used
}

// WrapFileKey wraps a file key with a master key.
func WrapFileKey(fileKey, masterKey []byte) ([]byte, error) {
	wrapper, err := crypto.NewKeyWrapper(masterKey)
	if err != nil {
		return nil, err
	}
	return wrapper.WrapKey(fileKey)
}

// UnwrapFileKey unwraps a file key.
func UnwrapFileKey(wrappedKey, masterKey []byte) ([]byte, error) {
	wrapper, err := crypto.NewKeyWrapper(masterKey)
	if err != nil {
		return nil, err
	}
	return wrapper.UnwrapKey(wrappedKey)
}

// GenerateAndWrapFileKey generates a new file key and wraps it with the master key.
func GenerateAndWrapFileKey(masterKey []byte) (fileKey, wrappedKey []byte, err error) {
	// Generate new file key
	fileKey, err = crypto.GenerateFileKey()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate file key: %w", err)
	}

	// Wrap it
	wrappedKey, err = WrapFileKey(fileKey, masterKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to wrap file key: %w", err)
	}

	return fileKey, wrappedKey, nil
}

// DecryptKeyAndReassemble decrypts a wrapped key and reassembles the file.
func DecryptKeyAndReassemble(ctx context.Context, store storage.Backend, manifest *Manifest, wrappedKey, masterKey []byte, w io.Writer) error {
	// Unwrap the file key
	fileKey, err := UnwrapFileKey(wrappedKey, masterKey)
	if err != nil {
		return fmt.Errorf("failed to unwrap file key: %w", err)
	}

	// Reassemble and decrypt
	reassembler := NewEncryptedReassembler(store)
	return reassembler.ReassembleDecrypt(ctx, manifest, fileKey, w)
}
