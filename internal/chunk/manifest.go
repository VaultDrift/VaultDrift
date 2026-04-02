package chunk

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Manifest represents a file version manifest.
// It contains an ordered list of chunk hashes that reconstruct the file.
type Manifest struct {
	ID        string      `json:"id"`
	FileID    string      `json:"file_id"`
	Version   int         `json:"version"`
	Size      int64       `json:"size"`
	Chunks    []ChunkInfo `json:"chunks"`
	Checksum  string      `json:"checksum"` // SHA-256 of complete file
	DeviceID  string      `json:"device_id"`
	CreatedAt time.Time   `json:"created_at"`
}

// NewManifest creates a new manifest from chunk information.
func NewManifest(id, fileID, deviceID string, chunks []ChunkInfo) *Manifest {
	var totalSize int64
	for _, c := range chunks {
		totalSize += int64(c.Size)
	}

	return &Manifest{
		ID:        id,
		FileID:    fileID,
		Version:   1, // Will be updated by caller if needed
		Size:      totalSize,
		Chunks:    chunks,
		DeviceID:  deviceID,
		CreatedAt: time.Now().UTC(),
	}
}

// GetChunkHashes returns the ordered list of chunk hashes.
func (m *Manifest) GetChunkHashes() []string {
	hashes := make([]string, len(m.Chunks))
	for i, c := range m.Chunks {
		hashes[i] = c.Hash
	}
	return hashes
}

// Serialize converts the manifest to JSON.
func (m *Manifest) Serialize() ([]byte, error) {
	return json.Marshal(m)
}

// DeserializeManifest parses a manifest from JSON.
func DeserializeManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to deserialize manifest: %w", err)
	}
	return &m, nil
}

// CalculateChecksum calculates the checksum of the complete file.
// This should be called after the file is assembled.
func (m *Manifest) CalculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// ChunkCount returns the number of chunks in the manifest.
func (m *Manifest) ChunkCount() int {
	return len(m.Chunks)
}

// Validate checks if the manifest is valid.
func (m *Manifest) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("manifest ID is required")
	}
	if m.FileID == "" {
		return fmt.Errorf("file ID is required")
	}
	if len(m.Chunks) == 0 {
		return fmt.Errorf("manifest must have at least one chunk")
	}
	if m.Size < 0 {
		return fmt.Errorf("size cannot be negative")
	}

	// Validate chunk hashes
	for i, c := range m.Chunks {
		if c.Hash == "" {
			return fmt.Errorf("chunk %d has no hash", i)
		}
		if c.Size <= 0 {
			return fmt.Errorf("chunk %d has invalid size", i)
		}
	}

	return nil
}
