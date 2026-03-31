package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/client/rpc"
	"github.com/multiformats/go-multiaddr"

	configpkg "github.com/vaultdrift/vaultdrift/internal/config"
)

// IPFSBackend implements storage.Backend for IPFS
type IPFSBackend struct {
	client  *rpc.HttpApi
	cfg     configpkg.IPFSConfig
	gateway string
}

// NewIPFSBackend creates a new IPFS storage backend
func NewIPFSBackend(cfg configpkg.IPFSConfig) (*IPFSBackend, error) {
	// Default to local IPFS node if not specified
	if cfg.APIAddr == "" {
		cfg.APIAddr = "/ip4/127.0.0.1/tcp/5001"
	}

	// Parse the multiaddr
	addr, err := multiaddr.NewMultiaddr(cfg.APIAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid IPFS API address: %w", err)
	}

	// Create IPFS client
	client, err := rpc.NewApi(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create IPFS client: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10)
	defer cancel()

	// Try to get node ID to verify connection
	_, err = client.Key().Self(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to IPFS node: %w", err)
	}

	backend := &IPFSBackend{
		client:  client,
		cfg:     cfg,
		gateway: cfg.Gateway,
	}

	return backend, nil
}

// Get retrieves data from IPFS by CID
func (b *IPFSBackend) Get(ctx context.Context, key string) ([]byte, error) {
	// Parse the CID
	c, err := cid.Decode(key)
	if err != nil {
		return nil, fmt.Errorf("invalid CID: %w", err)
	}

	// Convert CID to path
	p := path.FromCid(c)

	// Get the block from IPFS
	reader, err := b.client.Block().Get(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("failed to get block from IPFS: %w", err)
	}

	return io.ReadAll(reader)
}

// Put stores data on IPFS and returns the CID as the key
func (b *IPFSBackend) Put(ctx context.Context, key string, data []byte) error {
	// Store the block in IPFS
	stat, err := b.client.Block().Put(ctx, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to put block to IPFS: %w", err)
	}

	// The key should match the resulting CID
	resultingCid := stat.Path().RootCid().String()
	if key != "" && key != resultingCid {
		return fmt.Errorf("CID mismatch: expected %s, got %s", key, resultingCid)
	}

	// Pin if configured
	if b.cfg.PinFiles {
		if err := b.client.Pin().Add(ctx, stat.Path()); err != nil {
			return fmt.Errorf("failed to pin block: %w", err)
		}
	}

	return nil
}

// Delete removes a block from IPFS (unpins it)
func (b *IPFSBackend) Delete(ctx context.Context, key string) error {
	// Parse the CID
	c, err := cid.Decode(key)
	if err != nil {
		return fmt.Errorf("invalid CID: %w", err)
	}

	// Unpin the block (garbage collection will remove it)
	p := path.FromCid(c)
	if err := b.client.Pin().Rm(ctx, p); err != nil {
		return fmt.Errorf("failed to unpin block: %w", err)
	}

	return nil
}

// Exists checks if a block exists in IPFS
func (b *IPFSBackend) Exists(ctx context.Context, key string) (bool, error) {
	// Parse the CID
	c, err := cid.Decode(key)
	if err != nil {
		return false, fmt.Errorf("invalid CID: %w", err)
	}

	// Try to stat the block
	p := path.FromCid(c)
	_, err = b.client.Block().Stat(ctx, p)
	if err != nil {
		if err.Error() == "blockservice: key not found" {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat block: %w", err)
	}

	return true, nil
}

// List returns all pinned CIDs with the given prefix
// Note: This is a simplified implementation
func (b *IPFSBackend) List(ctx context.Context, prefix string) ([]string, error) {
	// IPFS pin listing is complex - return empty for now
	// A full implementation would iterate through all pins
	return []string{}, nil
}

// Stats returns storage statistics from IPFS
func (b *IPFSBackend) Stats(ctx context.Context) (*StorageStats, error) {
	// Return basic stats - full implementation would use Repo().Stat() if available
	// Note: Repo() may not be available in all IPFS client configurations
	return &StorageStats{
		TotalBytes:  0, // Unknown for IPFS
		UsedBytes:   0, // Unknown without Repo access
		ChunkCount:  0,
		BackendType: "ipfs",
	}, nil
}

// GetReader returns a reader for streaming data from IPFS
func (b *IPFSBackend) GetReader(ctx context.Context, key string) (io.ReadCloser, error) {
	// Parse the CID
	c, err := cid.Decode(key)
	if err != nil {
		return nil, fmt.Errorf("invalid CID: %w", err)
	}

	// Convert CID to path and get reader
	p := path.FromCid(c)
	reader, err := b.client.Block().Get(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("failed to get block from IPFS: %w", err)
	}

	// The reader from IPFS client is already an io.ReadCloser
	return io.NopCloser(reader), nil
}

// PutReader stores data from a reader to IPFS
func (b *IPFSBackend) PutReader(ctx context.Context, key string, reader io.Reader) error {
	// Store the block in IPFS
	stat, err := b.client.Block().Put(ctx, reader)
	if err != nil {
		return fmt.Errorf("failed to put block to IPFS: %w", err)
	}

	// The key should match the resulting CID
	resultingCid := stat.Path().RootCid().String()
	if key != "" && key != resultingCid {
		return fmt.Errorf("CID mismatch: expected %s, got %s", key, resultingCid)
	}

	// Pin if configured
	if b.cfg.PinFiles {
		if err := b.client.Pin().Add(ctx, stat.Path()); err != nil {
			return fmt.Errorf("failed to pin block: %w", err)
		}
	}

	return nil
}

// GetGatewayURL returns the HTTP gateway URL for a CID
func (b *IPFSBackend) GetGatewayURL(cid string) string {
	if b.gateway == "" {
		return ""
	}
	return fmt.Sprintf("%s/ipfs/%s", b.gateway, cid)
}

// Close closes the IPFS backend connection
func (b *IPFSBackend) Close() error {
	return nil
}

// IPFSInfo returns information about the connected IPFS node
func (b *IPFSBackend) IPFSInfo(ctx context.Context) (map[string]interface{}, error) {
	id, err := b.client.Key().Self(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get node ID: %w", err)
	}

	return map[string]interface{}{
		"id":      id.ID().String(),
		"api":     b.cfg.APIAddr,
		"gateway": b.gateway,
	}, nil
}

// Helper types for compatibility
type bytesReader struct {
	data []byte
	pos  int
}

func (b *bytesReader) Read(p []byte) (n int, err error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}
	n = copy(p, b.data[b.pos:])
	b.pos += n
	return n, nil
}

func (b *bytesReader) Close() error {
	return nil
}
