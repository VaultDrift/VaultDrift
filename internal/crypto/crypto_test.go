package crypto

import (
	"bytes"
	"crypto/aes"
	"strings"
	"testing"
)

// TestRandomBytes tests random byte generation
func TestRandomBytes(t *testing.T) {
	t.Run("GenerateRandomBytes", func(t *testing.T) {
		b, err := RandomBytes(32)
		if err != nil {
			t.Fatalf("Failed to generate random bytes: %v", err)
		}

		if len(b) != 32 {
			t.Errorf("Expected 32 bytes, got %d", len(b))
		}

		// Should be different each time
		b2, _ := RandomBytes(32)
		if bytes.Equal(b, b2) {
			t.Error("Random bytes should be different each time")
		}

		t.Logf("✅ Random bytes generation working")
	})

	t.Run("GenerateKey", func(t *testing.T) {
		key, err := GenerateKey()
		if err != nil {
			t.Fatalf("Failed to generate key: %v", err)
		}

		if len(key) != KeySize {
			t.Errorf("Expected key size %d, got %d", KeySize, len(key))
		}

		t.Logf("✅ Key generation working")
	})

	t.Run("GenerateNonce", func(t *testing.T) {
		nonce, err := GenerateNonce()
		if err != nil {
			t.Fatalf("Failed to generate nonce: %v", err)
		}

		if len(nonce) != NonceSize {
			t.Errorf("Expected nonce size %d, got %d", NonceSize, len(nonce))
		}

		t.Logf("✅ Nonce generation working")
	})

	t.Run("GenerateSalt", func(t *testing.T) {
		salt, err := GenerateSalt()
		if err != nil {
			t.Fatalf("Failed to generate salt: %v", err)
		}

		if len(salt) != SaltSize {
			t.Errorf("Expected salt size %d, got %d", SaltSize, len(salt))
		}

		t.Logf("✅ Salt generation working")
	})
}

// TestSecureCompare tests constant-time comparison
func TestSecureCompare(t *testing.T) {
	t.Run("EqualSlices", func(t *testing.T) {
		a := []byte("test data")
		b := []byte("test data")

		if !SecureCompare(a, b) {
			t.Error("Should return true for equal slices")
		}
	})

	t.Run("DifferentSlices", func(t *testing.T) {
		a := []byte("test data")
		b := []byte("different")

		if SecureCompare(a, b) {
			t.Error("Should return false for different slices")
		}
	})

	t.Run("DifferentLengths", func(t *testing.T) {
		a := []byte("short")
		b := []byte("longer data")

		if SecureCompare(a, b) {
			t.Error("Should return false for different lengths")
		}
	})

	t.Run("EmptySlices", func(t *testing.T) {
		if !SecureCompare([]byte{}, []byte{}) {
			t.Error("Should return true for two empty slices")
		}
	})
}

// TestDeriveKey tests key derivation
func TestDeriveKey(t *testing.T) {
	t.Run("DeriveKeyPBKDF2", func(t *testing.T) {
		passphrase := "my secret password"
		salt, _ := GenerateSalt()

		key := DeriveKey(passphrase, salt)

		if len(key) != KeySize {
			t.Errorf("Expected key size %d, got %d", KeySize, len(key))
		}

		// Same passphrase and salt should produce same key
		key2 := DeriveKey(passphrase, salt)
		if !bytes.Equal(key, key2) {
			t.Error("Same passphrase and salt should produce same key")
		}

		// Different salt should produce different key
		salt2, _ := GenerateSalt()
		key3 := DeriveKey(passphrase, salt2)
		if bytes.Equal(key, key3) {
			t.Error("Different salt should produce different key")
		}

		t.Logf("✅ PBKDF2 key derivation working")
	})

	t.Run("DeriveKeyArgon2idStyle", func(t *testing.T) {
		passphrase := "my secret password"
		salt, _ := GenerateSalt()

		key := DeriveKeyArgon2idStyle(passphrase, salt, 64, 3)

		if len(key) != KeySize {
			t.Errorf("Expected key size %d, got %d", KeySize, len(key))
		}

		// Same parameters should produce same key
		key2 := DeriveKeyArgon2idStyle(passphrase, salt, 64, 3)
		if !bytes.Equal(key, key2) {
			t.Error("Same parameters should produce same key")
		}

		t.Logf("✅ Argon2id-style key derivation working")
	})
}

// TestEncryptDecrypt tests AES-256-GCM encryption
func TestEncryptDecrypt(t *testing.T) {
	t.Run("EncryptDecryptRoundTrip", func(t *testing.T) {
		key, _ := GenerateKey()
		plaintext := []byte("Hello, World! This is a secret message.")

		// Encrypt
		ciphertext, err := Encrypt(plaintext, key)
		if err != nil {
			t.Fatalf("Failed to encrypt: %v", err)
		}

		// Ciphertext should be longer than plaintext (due to nonce + tag)
		if len(ciphertext) <= len(plaintext) {
			t.Error("Ciphertext should be longer than plaintext")
		}

		// Decrypt
		decrypted, err := Decrypt(ciphertext, key)
		if err != nil {
			t.Fatalf("Failed to decrypt: %v", err)
		}

		if !bytes.Equal(plaintext, decrypted) {
			t.Error("Decrypted text should match original")
		}

		t.Logf("✅ Encrypt/decrypt round trip working")
	})

	t.Run("EncryptEmptyData", func(t *testing.T) {
		key, _ := GenerateKey()
		plaintext := []byte{}

		ciphertext, err := Encrypt(plaintext, key)
		if err != nil {
			t.Fatalf("Failed to encrypt empty data: %v", err)
		}

		decrypted, err := Decrypt(ciphertext, key)
		if err != nil {
			t.Fatalf("Failed to decrypt empty data: %v", err)
		}

		if len(decrypted) != 0 {
			t.Error("Decrypted empty data should be empty")
		}

		t.Logf("✅ Empty data encryption working")
	})

	t.Run("DecryptWithWrongKey", func(t *testing.T) {
		key, _ := GenerateKey()
		wrongKey, _ := GenerateKey()
		plaintext := []byte("secret message")

		ciphertext, _ := Encrypt(plaintext, key)

		_, err := Decrypt(ciphertext, wrongKey)
		if err == nil {
			t.Error("Decryption with wrong key should fail")
		}

		if err != ErrDecryptionFailed {
			t.Errorf("Expected ErrDecryptionFailed, got %v", err)
		}

		t.Logf("✅ Wrong key detection working")
	})

	t.Run("DecryptTamperedCiphertext", func(t *testing.T) {
		key, _ := GenerateKey()
		plaintext := []byte("secret message")

		ciphertext, _ := Encrypt(plaintext, key)

		// Tamper with ciphertext
		ciphertext[len(ciphertext)-1] ^= 0xFF

		_, err := Decrypt(ciphertext, key)
		if err == nil {
			t.Error("Decryption of tampered ciphertext should fail")
		}

		t.Logf("✅ Tamper detection working")
	})

	t.Run("DecryptInvalidCiphertext", func(t *testing.T) {
		key, _ := GenerateKey()

		// Too short
		_, err := Decrypt([]byte("short"), key)
		if err != ErrInvalidCiphertext {
			t.Errorf("Expected ErrInvalidCiphertext for short input, got %v", err)
		}

		t.Logf("✅ Invalid ciphertext detection working")
	})

	t.Run("EncryptWithInvalidKeySize", func(t *testing.T) {
		plaintext := []byte("test")
		shortKey := []byte("too short")

		_, err := Encrypt(plaintext, shortKey)
		if err == nil {
			t.Error("Encryption with invalid key size should fail")
		}

		t.Logf("✅ Invalid key size detection working")
	})
}

// TestEncryptDecryptCTR tests AES-256-CTR mode
func TestEncryptDecryptCTR(t *testing.T) {
	t.Run("CTRRoundTrip", func(t *testing.T) {
		key, _ := GenerateKey()
		plaintext := []byte("Hello, CTR mode!")
		iv := make([]byte, aes.BlockSize)
		copy(iv, []byte("1234567890123456"))

		// Encrypt
		ciphertext, err := EncryptCTR(plaintext, key, iv)
		if err != nil {
			t.Fatalf("Failed to encrypt: %v", err)
		}

		// Decrypt
		decrypted, err := DecryptCTR(ciphertext, key, iv)
		if err != nil {
			t.Fatalf("Failed to decrypt: %v", err)
		}

		if !bytes.Equal(plaintext, decrypted) {
			t.Error("Decrypted text should match original")
		}

		t.Logf("✅ CTR mode round trip working")
	})

	t.Run("CTRDifferentIV", func(t *testing.T) {
		key, _ := GenerateKey()
		plaintext := []byte("same plaintext")
		iv1 := make([]byte, aes.BlockSize)
		iv2 := make([]byte, aes.BlockSize)
		iv2[0] = 1 // Different IV

		ciphertext1, _ := EncryptCTR(plaintext, key, iv1)
		ciphertext2, _ := EncryptCTR(plaintext, key, iv2)

		if bytes.Equal(ciphertext1, ciphertext2) {
			t.Error("Same plaintext with different IVs should produce different ciphertexts")
		}

		t.Logf("✅ CTR mode IV randomization working")
	})

	t.Run("CTRInvalidKeySize", func(t *testing.T) {
		plaintext := []byte("test")
		shortKey := []byte("too short")
		iv := make([]byte, aes.BlockSize)

		_, err := EncryptCTR(plaintext, shortKey, iv)
		if err == nil {
			t.Error("CTR encryption with invalid key size should fail")
		}
	})

	t.Run("CTRInvalidIVSize", func(t *testing.T) {
		key, _ := GenerateKey()
		plaintext := []byte("test")
		shortIV := []byte("too short")

		_, err := EncryptCTR(plaintext, key, shortIV)
		if err == nil {
			t.Error("CTR encryption with invalid IV size should fail")
		}
	})
}

// TestHMAC tests HMAC operations
func TestHMAC(t *testing.T) {
	t.Run("HMACComputation", func(t *testing.T) {
		key := []byte("secret key")
		data := []byte("message to authenticate")

		mac := HMAC(key, data)

		if len(mac) == 0 {
			t.Error("HMAC should not be empty")
		}

		// Same key and data should produce same MAC
		mac2 := HMAC(key, data)
		if !bytes.Equal(mac, mac2) {
			t.Error("Same key and data should produce same HMAC")
		}

		t.Logf("✅ HMAC computation working")
	})

	t.Run("VerifyHMACValid", func(t *testing.T) {
		key := []byte("secret key")
		data := []byte("message to authenticate")

		mac := HMAC(key, data)

		if !VerifyHMAC(key, data, mac) {
			t.Error("Valid HMAC should verify")
		}

		t.Logf("✅ HMAC verification working")
	})

	t.Run("VerifyHMACInvalid", func(t *testing.T) {
		key := []byte("secret key")
		data := []byte("message to authenticate")
		wrongData := []byte("different message")

		mac := HMAC(key, data)

		if VerifyHMAC(key, wrongData, mac) {
			t.Error("HMAC for different data should not verify")
		}

		// Tampered MAC
		tamperedMac := make([]byte, len(mac))
		copy(tamperedMac, mac)
		tamperedMac[0] ^= 0xFF

		if VerifyHMAC(key, data, tamperedMac) {
			t.Error("Tampered HMAC should not verify")
		}

		t.Logf("✅ HMAC invalid detection working")
	})

	t.Run("HMACDifferentKeys", func(t *testing.T) {
		key1 := []byte("key one")
		key2 := []byte("key two")
		data := []byte("same message")

		mac1 := HMAC(key1, data)
		mac2 := HMAC(key2, data)

		if bytes.Equal(mac1, mac2) {
			t.Error("Different keys should produce different HMACs")
		}

		t.Logf("✅ HMAC key sensitivity working")
	})
}

// TestHash tests hashing functions
func TestHash(t *testing.T) {
	t.Run("SHA256Hash", func(t *testing.T) {
		data := []byte("test data")

		hash := Hash(data)

		if len(hash) != 32 { // SHA-256 produces 32 bytes
			t.Errorf("Expected 32 bytes, got %d", len(hash))
		}

		// Same data should produce same hash
		hash2 := Hash(data)
		if !bytes.Equal(hash, hash2) {
			t.Error("Same data should produce same hash")
		}

		t.Logf("✅ SHA-256 hashing working")
	})

	t.Run("HashToString", func(t *testing.T) {
		data := []byte("test data")

		hashStr := HashToString(data)

		// Should be 64 hex characters (32 bytes * 2)
		if len(hashStr) != 64 {
			t.Errorf("Expected 64 hex chars, got %d", len(hashStr))
		}

		// Should only contain hex characters
		if !strings.ContainsAny(hashStr, "0123456789abcdef") {
			t.Error("Hash string should be hex encoded")
		}

		t.Logf("✅ Hash to string working")
	})

	t.Run("DifferentDataDifferentHash", func(t *testing.T) {
		data1 := []byte("data one")
		data2 := []byte("data two")

		hash1 := Hash(data1)
		hash2 := Hash(data2)

		if bytes.Equal(hash1, hash2) {
			t.Error("Different data should produce different hashes")
		}

		t.Logf("✅ Hash uniqueness working")
	})
}

// TestStreamEncryption tests stream encryption
func TestStreamEncryption(t *testing.T) {
	t.Run("StreamEncryptDecryptChunk", func(t *testing.T) {
		key, _ := GenerateKey()
		plaintext := []byte("chunk of data for streaming")

		// Create encryptor
		encryptor, nonce, err := NewStreamEncryptor(key)
		if err != nil {
			t.Fatalf("Failed to create encryptor: %v", err)
		}

		if len(nonce) != NonceSize {
			t.Errorf("Expected nonce size %d, got %d", NonceSize, len(nonce))
		}

		// Encrypt chunk
		ciphertext := encryptor.EncryptChunk(plaintext, 0)

		// Create decryptor
		decryptor, err := NewStreamDecryptor(key, nonce)
		if err != nil {
			t.Fatalf("Failed to create decryptor: %v", err)
		}

		// Decrypt chunk
		decrypted, err := decryptor.DecryptChunk(ciphertext, 0)
		if err != nil {
			t.Fatalf("Failed to decrypt chunk: %v", err)
		}

		if !bytes.Equal(plaintext, decrypted) {
			t.Error("Decrypted chunk should match original")
		}

		t.Logf("✅ Stream encryption round trip working")
	})

	t.Run("StreamDifferentChunks", func(t *testing.T) {
		key, _ := GenerateKey()
		encryptor, nonce, _ := NewStreamEncryptor(key)
		decryptor, _ := NewStreamDecryptor(key, nonce)

		chunk1 := []byte("first chunk")
		chunk2 := []byte("second chunk with different content")

		// Encrypt both chunks
		cipher1 := encryptor.EncryptChunk(chunk1, 0)
		cipher2 := encryptor.EncryptChunk(chunk2, 1)

		// Same plaintext at different chunk indices should produce different ciphertext
		// due to nonce derivation
		cipher1At1 := encryptor.EncryptChunk(chunk1, 1)
		if bytes.Equal(cipher1, cipher1At1) {
			t.Error("Same data at different chunk indices should produce different ciphertext")
		}

		// Decrypt
		decrypted1, _ := decryptor.DecryptChunk(cipher1, 0)
		decrypted2, _ := decryptor.DecryptChunk(cipher2, 1)

		if !bytes.Equal(chunk1, decrypted1) {
			t.Error("First chunk decryption failed")
		}
		if !bytes.Equal(chunk2, decrypted2) {
			t.Error("Second chunk decryption failed")
		}

		t.Logf("✅ Stream multi-chunk encryption working")
	})

	t.Run("StreamDecryptWrongChunkIndex", func(t *testing.T) {
		key, _ := GenerateKey()
		plaintext := []byte("test data")

		encryptor, nonce, _ := NewStreamEncryptor(key)
		ciphertext := encryptor.EncryptChunk(plaintext, 5)

		decryptor, _ := NewStreamDecryptor(key, nonce)

		// Try to decrypt with wrong chunk index
		_, err := decryptor.DecryptChunk(ciphertext, 6)
		if err == nil {
			t.Error("Decrypting with wrong chunk index should fail")
		}

		t.Logf("✅ Stream chunk index verification working")
	})

	t.Run("StreamDecryptorInvalidKey", func(t *testing.T) {
		_, err := NewStreamDecryptor([]byte("short"), make([]byte, NonceSize))
		if err == nil {
			t.Error("Creating decryptor with invalid key should fail")
		}
	})

	t.Run("StreamDecryptorInvalidNonce", func(t *testing.T) {
		key, _ := GenerateKey()
		_, err := NewStreamDecryptor(key, []byte("short"))
		if err == nil {
			t.Error("Creating decryptor with invalid nonce should fail")
		}
	})
}
