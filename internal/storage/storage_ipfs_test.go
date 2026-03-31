package storage

import (
	"bytes"
	"context"
	"testing"

	"github.com/vaultdrift/vaultdrift/internal/config"
)

func TestNewIPFSBackend(t *testing.T) {
	// Note: This test requires a running IPFS node
	// Skip if IPFS is not available

	cfg := config.IPFSConfig{
		APIAddr:  "/ip4/127.0.0.1/tcp/5001",
		Gateway:  "http://localhost:8080",
		PinFiles: true,
	}

	// Try to create backend - will fail if IPFS is not running
	_, err := NewIPFSBackend(cfg)
	if err != nil {
		t.Skipf("IPFS node not available: %v", err)
	}
}

func TestIPFSBackend_Methods(t *testing.T) {
	// Skip if no IPFS node available
	cfg := config.IPFSConfig{
		APIAddr:  "/ip4/127.0.0.1/tcp/5001",
		PinFiles: false,
	}

	backend, err := NewIPFSBackend(cfg)
	if err != nil {
		t.Skipf("IPFS node not available: %v", err)
	}
	defer backend.Close()

	ctx := context.Background()

	t.Run("PutAndGet", func(t *testing.T) {
		data := []byte("hello ipfs world")

		// Put data
		stat, err := backend.client.Block().Put(ctx, bytes.NewReader(data))
		if err != nil {
			t.Fatalf("Failed to put block: %v", err)
		}

		cid := stat.Path().RootCid().String()

		// Get data back
		retrieved, err := backend.Get(ctx, cid)
		if err != nil {
			t.Fatalf("Failed to get block: %v", err)
		}

		if !bytes.Equal(data, retrieved) {
			t.Errorf("Data mismatch: got %s, want %s", string(retrieved), string(data))
		}
	})

	t.Run("Exists", func(t *testing.T) {
		data := []byte("test exists")

		stat, err := backend.client.Block().Put(ctx, bytes.NewReader(data))
		if err != nil {
			t.Fatalf("Failed to put block: %v", err)
		}

		cid := stat.Path().RootCid().String()

		exists, err := backend.Exists(ctx, cid)
		if err != nil {
			t.Fatalf("Failed to check existence: %v", err)
		}

		if !exists {
			t.Error("Expected block to exist")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		data := []byte("test delete")

		stat, err := backend.client.Block().Put(ctx, bytes.NewReader(data))
		if err != nil {
			t.Fatalf("Failed to put block: %v", err)
		}

		cid := stat.Path().RootCid().String()

		// Delete (unpin) the block
		err = backend.Delete(ctx, cid)
		if err != nil {
			t.Logf("Delete (unpin) returned: %v", err)
		}
	})

	t.Run("List", func(t *testing.T) {
		items, err := backend.List(ctx, "")
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}

		// List may return empty for now
		t.Logf("List returned %d items", len(items))
	})

	t.Run("Stats", func(t *testing.T) {
		stats, err := backend.Stats(ctx)
		if err != nil {
			t.Fatalf("Stats failed: %v", err)
		}

		if stats.BackendType != "ipfs" {
			t.Errorf("Expected backend type 'ipfs', got %s", stats.BackendType)
		}
	})

	t.Run("GetReader", func(t *testing.T) {
		data := []byte("test streaming")

		stat, err := backend.client.Block().Put(ctx, bytes.NewReader(data))
		if err != nil {
			t.Fatalf("Failed to put block: %v", err)
		}

		cid := stat.Path().RootCid().String()

		reader, err := backend.GetReader(ctx, cid)
		if err != nil {
			t.Fatalf("Failed to get reader: %v", err)
		}
		defer reader.Close()

		retrieved := make([]byte, len(data))
		n, err := reader.Read(retrieved)
		if err != nil && err.Error() != "EOF" {
			t.Fatalf("Failed to read: %v", err)
		}

		if n != len(data) {
			t.Errorf("Read %d bytes, expected %d", n, len(data))
		}
	})

	t.Run("PutReader", func(t *testing.T) {
		data := []byte("test put reader")
		reader := bytes.NewReader(data)

		err := backend.PutReader(ctx, "", reader)
		if err != nil {
			t.Fatalf("Failed to put reader: %v", err)
		}
	})

	t.Run("GetGatewayURL", func(t *testing.T) {
		url := backend.GetGatewayURL("QmTest")
		expected := "http://localhost:8080/ipfs/QmTest"
		if url != expected {
			t.Errorf("Expected URL %s, got %s", expected, url)
		}
	})

	t.Run("IPFSInfo", func(t *testing.T) {
		info, err := backend.IPFSInfo(ctx)
		if err != nil {
			t.Fatalf("Failed to get IPFS info: %v", err)
		}

		if _, ok := info["id"]; !ok {
			t.Error("Expected 'id' in info")
		}
	})
}

func TestIPFSBackend_InvalidCID(t *testing.T) {
	cfg := config.IPFSConfig{
		APIAddr: "/ip4/127.0.0.1/tcp/5001",
	}

	backend, err := NewIPFSBackend(cfg)
	if err != nil {
		t.Skipf("IPFS node not available: %v", err)
	}
	defer backend.Close()

	ctx := context.Background()

	t.Run("GetInvalidCID", func(t *testing.T) {
		_, err := backend.Get(ctx, "invalid-cid")
		if err == nil {
			t.Error("Expected error for invalid CID")
		}
	})

	t.Run("DeleteInvalidCID", func(t *testing.T) {
		err := backend.Delete(ctx, "invalid-cid")
		if err == nil {
			t.Error("Expected error for invalid CID")
		}
	})

	t.Run("ExistsInvalidCID", func(t *testing.T) {
		_, err := backend.Exists(ctx, "invalid-cid")
		if err == nil {
			t.Error("Expected error for invalid CID")
		}
	})
}
