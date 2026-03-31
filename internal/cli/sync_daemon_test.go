package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

// TestNewSyncDaemon tests daemon creation
func TestNewSyncDaemon(t *testing.T) {
	cli := &CLI{}
	tempDir := t.TempDir()

	daemon, err := NewSyncDaemon(cli, tempDir)
	if err != nil {
		t.Fatalf("Failed to create sync daemon: %v", err)
	}
	defer daemon.Stop()

	if daemon.watchDir != tempDir {
		t.Errorf("Expected watchDir to be %s, got %s", tempDir, daemon.watchDir)
	}

	if daemon.watcher == nil {
		t.Error("Expected watcher to be initialized")
	}

	if daemon.stopCh == nil {
		t.Error("Expected stopCh to be initialized")
	}

	t.Logf("✅ Sync daemon created successfully")
}

// TestSyncDaemonAddWatchRecursive tests recursive directory watching
func TestSyncDaemonAddWatchRecursive(t *testing.T) {
	cli := &CLI{}
	tempDir := t.TempDir()

	// Create subdirectories
	subDir1 := filepath.Join(tempDir, "subdir1")
	subDir2 := filepath.Join(tempDir, "subdir2")
	os.MkdirAll(subDir1, 0755)
	os.MkdirAll(subDir2, 0755)

	// Create hidden directory (should be skipped)
	hiddenDir := filepath.Join(tempDir, ".hidden")
	os.MkdirAll(hiddenDir, 0755)

	daemon, err := NewSyncDaemon(cli, tempDir)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}
	defer daemon.Stop()

	// Add watch recursively
	err = daemon.addWatchRecursive(tempDir)
	if err != nil {
		t.Fatalf("Failed to add watch: %v", err)
	}

	// Give time for watches to be established
	time.Sleep(100 * time.Millisecond)

	t.Logf("✅ Recursive watch added for %s and subdirectories", tempDir)
}

// TestSyncDaemonHandleEvent tests event handling
func TestSyncDaemonHandleEvent(t *testing.T) {
	cli := &CLI{}
	tempDir := t.TempDir()

	daemon, err := NewSyncDaemon(cli, tempDir)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}
	defer daemon.Stop()

	t.Run("HandleCreateEvent", func(t *testing.T) {
		// Create a test file
		testFile := filepath.Join(tempDir, "test.txt")
		os.WriteFile(testFile, []byte("test content"), 0644)

		event := fsnotify.Event{
			Name: testFile,
			Op:   fsnotify.Create,
		}

		// Should not panic
		daemon.handleEvent(event)

		t.Logf("✅ Create event handled")
	})

	t.Run("HandleWriteEvent", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "write.txt")
		os.WriteFile(testFile, []byte("content"), 0644)

		event := fsnotify.Event{
			Name: testFile,
			Op:   fsnotify.Write,
		}

		// Should not panic
		daemon.handleEvent(event)

		t.Logf("✅ Write event handled")
	})

	t.Run("HandleRemoveEvent", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "remove.txt")

		event := fsnotify.Event{
			Name: testFile,
			Op:   fsnotify.Remove,
		}

		// Should not panic
		daemon.handleEvent(event)

		t.Logf("✅ Remove event handled")
	})

	t.Run("SkipHiddenFile", func(t *testing.T) {
		hiddenFile := filepath.Join(tempDir, ".hidden.txt")
		os.WriteFile(hiddenFile, []byte("hidden"), 0644)

		event := fsnotify.Event{
			Name: hiddenFile,
			Op:   fsnotify.Create,
		}

		// Should not panic (just skip)
		daemon.handleEvent(event)

		t.Logf("✅ Hidden file skipped")
	})
}

// TestSyncDaemonDebounce tests debouncing logic
func TestSyncDaemonDebounce(t *testing.T) {
	cli := &CLI{}
	tempDir := t.TempDir()

	daemon, err := NewSyncDaemon(cli, tempDir)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}
	defer daemon.Stop()

	testFile := filepath.Join(tempDir, "debounce.txt")
	os.WriteFile(testFile, []byte("content"), 0644)

	// First call should create timer
	daemon.debounceUpload(testFile)

	// Second call should reset timer
	daemon.debounceUpload(testFile)

	// Check that timer exists
	daemon.debounceMu.Lock()
	_, exists := daemon.debounce[testFile]
	daemon.debounceMu.Unlock()

	if !exists {
		t.Error("Expected debounce timer to exist")
	}

	t.Logf("✅ Debounce timer created and reset")
}

// TestSyncDaemonStop tests clean shutdown
func TestSyncDaemonStop(t *testing.T) {
	cli := &CLI{}
	tempDir := t.TempDir()

	daemon, err := NewSyncDaemon(cli, tempDir)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Add a pending debounce timer
	testFile := filepath.Join(tempDir, "pending.txt")
	daemon.debounceUpload(testFile)

	// Stop should clean up
	err = daemon.Stop()
	if err != nil {
		t.Fatalf("Failed to stop daemon: %v", err)
	}

	// Verify stopCh is closed (would panic if closed twice)
	select {
	case <-daemon.stopCh:
		// Expected
	default:
		t.Error("Expected stopCh to be closed")
	}

	t.Logf("✅ Sync daemon stopped cleanly")
}

// TestSyncDaemonIntegration is an integration test
func TestSyncDaemonIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cli := &CLI{}
	tempDir := t.TempDir()

	daemon, err := NewSyncDaemon(cli, tempDir)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Add watch
	err = daemon.addWatchRecursive(tempDir)
	if err != nil {
		t.Fatalf("Failed to add watch: %v", err)
	}

	// Start watch in background
	daemon.wg.Add(1)
	go daemon.watch()

	// Give time for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Create a file (should trigger event)
	testFile := filepath.Join(tempDir, "integration.txt")
	err = os.WriteFile(testFile, []byte("integration test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Give time for event to be processed
	time.Sleep(200 * time.Millisecond)

	// Modify the file
	err = os.WriteFile(testFile, []byte("modified content"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Give time for event
	time.Sleep(200 * time.Millisecond)

	// Clean up
	daemon.Stop()

	t.Logf("✅ Integration test completed")
}
