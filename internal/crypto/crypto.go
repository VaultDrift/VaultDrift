// Package crypto provides cryptographic primitives for VaultDrift.
// All implementations use pure Go standard library only.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"io"
)

// KeySize is the standard AES-256 key size in bytes.
const KeySize = 32

// NonceSize is the standard GCM nonce size.
const NonceSize = 12

// SaltSize is the size of salt for key derivation.
const SaltSize = 16

// Encryption overhead: nonce (12) + tag (16).
const Overhead = NonceSize + 16

// ErrInvalidCiphertext is returned when ciphertext is too short or malformed.
var ErrInvalidCiphertext = errors.New("invalid ciphertext")

// ErrDecryptionFailed is returned when decryption authentication fails.
var ErrDecryptionFailed = errors.New("decryption failed: authentication tag mismatch")

// RandomBytes generates n cryptographically secure random bytes.
func RandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to read random bytes: %w", err)
	}
	return b, nil
}

// RandomBytesInto fills the provided slice with cryptographically secure random bytes.
func RandomBytesInto(b []byte) error {
	if _, err := rand.Read(b); err != nil {
		return fmt.Errorf("failed to read random bytes: %w", err)
	}
	return nil
}

// SecureCompare performs constant-time comparison of two byte slices.
// Returns true if they are equal.
func SecureCompare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// GenerateKey generates a random 256-bit key.
func GenerateKey() ([]byte, error) {
	return RandomBytes(KeySize)
}

// GenerateNonce generates a random 96-bit nonce for AES-GCM.
func GenerateNonce() ([]byte, error) {
	return RandomBytes(NonceSize)
}

// GenerateSalt generates a random 128-bit salt for key derivation.
func GenerateSalt() ([]byte, error) {
	return RandomBytes(SaltSize)
}

// pbkdf2 derives a key from password and salt using PBKDF2-HMAC-SHA256.
// This is a pure Go implementation since we're avoiding external dependencies.
func pbkdf2(password, salt []byte, iterations, keyLen int) []byte {
	prf := hmac.New(sha256.New, password)
	dkLen := keyLen
	u := prf.Size()
	n := (dkLen + u - 1) / u

	dk := make([]byte, 0, n*u)
	uBuf := make([]byte, 4)
	t := make([]byte, u)

	for i := 1; i <= n; i++ {
		binary.BigEndian.PutUint32(uBuf, uint32(i))
		prf.Reset()
		prf.Write(salt)
		prf.Write(uBuf)
		dk = prf.Sum(dk)

		copy(t, dk[len(dk)-u:])

		for j := 1; j < iterations; j++ {
			prf.Reset()
			prf.Write(t)
			prf.Sum(t[:0])
			for k := range t {
				dk[len(dk)-u+k] ^= t[k]
			}
		}
	}

	return dk[:dkLen]
}

// DeriveKey derives a 256-bit key from passphrase and salt using PBKDF2-HMAC-SHA256.
// Parameters: 100,000 iterations for good security without being too slow.
func DeriveKey(passphrase string, salt []byte) []byte {
	return pbkdf2([]byte(passphrase), salt, 100000, KeySize)
}

// DeriveKeyArgon2idStyle derives a key using a memory-hard approach similar to Argon2id.
// Since we can't use golang.org/x/crypto/argon2, we simulate memory hardness with
// multiple rounds of SHA-512 on expanded memory blocks.
func DeriveKeyArgon2idStyle(passphrase string, salt []byte, memoryKB, iterations int) []byte {
	// Initial key derivation
	key := pbkdf2([]byte(passphrase), salt, 1000, 64) // 512 bits

	// Memory-hard phase: create memory blocks and mix them
	blockSize := 1024 // 1KB blocks
	numBlocks := (memoryKB * 1024) / blockSize
	if numBlocks < 4 {
		numBlocks = 4
	}

	// Initialize memory blocks
	blocks := make([][]byte, numBlocks)
	for i := range blocks {
		blocks[i] = make([]byte, blockSize)
	}

	// Fill first block with initial key
	copy(blocks[0], key)

	// Expand: fill all blocks with derived data
	for i := 1; i < numBlocks; i++ {
		h := sha512.New()
		h.Write(blocks[i-1])
		h.Write([]byte{byte(i), byte(i >> 8), byte(i >> 16), byte(i >> 24)})
		copy(blocks[i], h.Sum(nil))
	}

	// Mix: multiple iterations of memory-hard mixing
	for iter := 0; iter < iterations; iter++ {
		for i := 0; i < numBlocks; i++ {
			// Previous block index (wrapping)
			prevIdx := (i - 1 + numBlocks) % numBlocks

			// Pseudorandom index based on current block
			idxBlock := int(binary.LittleEndian.Uint32(blocks[i][:4])) % numBlocks

			// Mix blocks
			h := sha512.New()
			h.Write(blocks[prevIdx])
			h.Write(blocks[idxBlock])
			result := h.Sum(nil)

			// XOR result into current block
			for j := 0; j < blockSize && j < len(result); j++ {
				blocks[i][j] ^= result[j]
			}
		}
	}

	// Finalize: extract key from last block
	h := sha256.New()
	h.Write(blocks[numBlocks-1])
	h.Write(salt)
	return h.Sum(nil)
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns: nonce (12 bytes) || ciphertext || tag (16 bytes).
func Encrypt(plaintext, key []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("invalid key size: expected %d, got %d", KeySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce, err := GenerateNonce()
	if err != nil {
		return nil, err
	}

	// Seal appends the tag to the ciphertext
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext encrypted with Encrypt.
// Expects: nonce (12 bytes) || ciphertext || tag (16 bytes).
func Decrypt(ciphertext, key []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("invalid key size: expected %d, got %d", KeySize, len(key))
	}

	if len(ciphertext) < NonceSize+16 {
		return nil, ErrInvalidCiphertext
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := ciphertext[:NonceSize]
	ciphertext = ciphertext[NonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// EncryptCTR encrypts using AES-256-CTR for streaming scenarios where
// authentication is handled separately.
func EncryptCTR(plaintext, key, iv []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("invalid key size: expected %d, got %d", KeySize, len(key))
	}

	if len(iv) != aes.BlockSize {
		return nil, fmt.Errorf("invalid IV size: expected %d, got %d", aes.BlockSize, len(iv))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	ciphertext := make([]byte, len(plaintext))
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext, plaintext)

	return ciphertext, nil
}

// DecryptCTR decrypts ciphertext encrypted with EncryptCTR.
func DecryptCTR(ciphertext, key, iv []byte) ([]byte, error) {
	// CTR mode encryption and decryption are identical
	return EncryptCTR(ciphertext, key, iv)
}

// HMAC computes HMAC-SHA256.
func HMAC(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// VerifyHMAC verifies HMAC-SHA256 in constant time.
func VerifyHMAC(key, data, mac []byte) bool {
	expected := HMAC(key, data)
	return SecureCompare(mac, expected)
}

// Hash computes SHA-256 hash.
func Hash(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}

// HashToString computes SHA-256 hash and returns hex string.
func HashToString(data []byte) string {
	return fmt.Sprintf("%x", Hash(data))
}

// NewHashReader wraps a reader and computes a running hash.
type NewHashReader struct {
	r    io.Reader
	hash hash.Hash
}

// NewSHA256Reader creates a reader that computes SHA-256.
func NewSHA256Reader(r io.Reader) *NewHashReader {
	return &NewHashReader{
		r:    r,
		hash: sha256.New(),
	}
}

// Read implements io.Reader.
func (hr *NewHashReader) Read(p []byte) (n int, err error) {
	n, err = hr.r.Read(p)
	if n > 0 {
		hr.hash.Write(p[:n])
	}
	return n, err
}

// Sum returns the hex-encoded hash.
func (hr *NewHashReader) Sum() string {
	return fmt.Sprintf("%x", hr.hash.Sum(nil))
}

// HashFile computes SHA-256 hash of a file.
func HashFile(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// StreamEncryptor encrypts data in a streaming fashion using AES-256-GCM with chunked encryption.
type StreamEncryptor struct {
	gcm    cipher.AEAD
	nonce  []byte
	offset int
}

// NewStreamEncryptor creates a new stream encryptor.
// Returns the encryptor and the nonce that should be stored/transmitted as a header.
func NewStreamEncryptor(key []byte) (*StreamEncryptor, []byte, error) {
	if len(key) != KeySize {
		return nil, nil, fmt.Errorf("invalid key size: expected %d, got %d", KeySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce, err := GenerateNonce()
	if err != nil {
		return nil, nil, err
	}

	return &StreamEncryptor{
		gcm:   gcm,
		nonce: nonce,
	}, nonce, nil
}

// EncryptChunk encrypts a single chunk of data.
// Each chunk uses a unique nonce derived from the main nonce and chunk index.
func (se *StreamEncryptor) EncryptChunk(plaintext []byte, chunkIndex int) []byte {
	// Use a counter-based nonce derived from the main nonce
	chunkNonce := make([]byte, NonceSize)
	copy(chunkNonce, se.nonce)
	binary.BigEndian.PutUint32(chunkNonce[8:], uint32(chunkIndex))

	// Encrypt and return
	return se.gcm.Seal(nil, chunkNonce, plaintext, nil)
}

// StreamDecryptor decrypts data encrypted with StreamEncryptor.
type StreamDecryptor struct {
	gcm   cipher.AEAD
	nonce []byte
}

// NewStreamDecryptor creates a new stream decryptor.
func NewStreamDecryptor(key, nonce []byte) (*StreamDecryptor, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("invalid key size: expected %d, got %d", KeySize, len(key))
	}

	if len(nonce) != NonceSize {
		return nil, fmt.Errorf("invalid nonce size: expected %d, got %d", NonceSize, len(nonce))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &StreamDecryptor{
		gcm:   gcm,
		nonce: nonce,
	}, nil
}

// DecryptChunk decrypts a single chunk.
func (sd *StreamDecryptor) DecryptChunk(ciphertext []byte, chunkIndex int) ([]byte, error) {
	// Reconstruct the nonce for this chunk
	chunkNonce := make([]byte, NonceSize)
	copy(chunkNonce, sd.nonce)
	binary.BigEndian.PutUint32(chunkNonce[8:], uint32(chunkIndex))

	plaintext, err := sd.gcm.Open(nil, chunkNonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}
