package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// SyncDaemon handles background synchronization
type SyncDaemon struct {
	cli        *CLI
	watchDir   string
	watcher    *fsnotify.Watcher
	stopCh     chan struct{}
	wg         sync.WaitGroup
	debounce   map[string]*time.Timer
	debounceMu sync.Mutex
}

// NewSyncDaemon creates a new sync daemon
func NewSyncDaemon(cli *CLI, watchDir string) (*SyncDaemon, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	return &SyncDaemon{
		cli:      cli,
		watchDir: watchDir,
		watcher:  watcher,
		stopCh:   make(chan struct{}),
		debounce: make(map[string]*time.Timer),
	}, nil
}

// Start starts the sync daemon
func (d *SyncDaemon) Start() error {
	// Add watch directory recursively
	if err := d.addWatchRecursive(d.watchDir); err != nil {
		return err
	}

	d.wg.Add(1)
	go d.watch()

	fmt.Printf("Sync daemon started watching: %s\n", d.watchDir)
	fmt.Println("Press Ctrl+C to stop")

	// Wait for stop signal
	<-d.stopCh
	return d.Stop()
}

// Stop stops the sync daemon
func (d *SyncDaemon) Stop() error {
	close(d.stopCh)
	d.watcher.Close()
	d.wg.Wait()

	// Cancel all pending debounce timers
	d.debounceMu.Lock()
	for _, timer := range d.debounce {
		timer.Stop()
	}
	d.debounceMu.Unlock()

	fmt.Println("Sync daemon stopped")
	return nil
}

// addWatchRecursive adds a directory and all subdirectories to the watcher
func (d *SyncDaemon) addWatchRecursive(path string) error {
	return filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip hidden directories
			if filepath.Base(walkPath)[0] == '.' {
				return filepath.SkipDir
			}
			if err := d.watcher.Add(walkPath); err != nil {
				return fmt.Errorf("failed to watch %s: %w", walkPath, err)
			}
			fmt.Printf("  Watching: %s\n", walkPath)
		}
		return nil
	})
}

// watch handles file system events
func (d *SyncDaemon) watch() {
	defer d.wg.Done()

	for {
		select {
		case event, ok := <-d.watcher.Events:
			if !ok {
				return
			}
			d.handleEvent(event)

		case err, ok := <-d.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("Watch error: %v\n", err)

		case <-d.stopCh:
			return
		}
	}
}

// handleEvent processes a file system event
func (d *SyncDaemon) handleEvent(event fsnotify.Event) {
	// Skip hidden files and directories
	if filepath.Base(event.Name)[0] == '.' {
		return
	}

	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		info, err := os.Stat(event.Name)
		if err != nil {
			return
		}
		if info.IsDir() {
			// New directory - add to watcher
			d.watcher.Add(event.Name)
			fmt.Printf("  New directory: %s\n", event.Name)
		} else {
			// New file - debounce upload
			d.debounceUpload(event.Name)
		}

	case event.Op&fsnotify.Write == fsnotify.Write:
		// File modified - debounce upload
		d.debounceUpload(event.Name)

	case event.Op&fsnotify.Remove == fsnotify.Remove:
		// File removed - delete from server
		d.handleDelete(event.Name)

	case event.Op&fsnotify.Rename == fsnotify.Rename:
		// File renamed - treat as delete
		d.handleDelete(event.Name)
	}
}

// debounceUpload schedules an upload with debouncing
func (d *SyncDaemon) debounceUpload(filePath string) {
	d.debounceMu.Lock()
	defer d.debounceMu.Unlock()

	// Cancel existing timer if any
	if timer, exists := d.debounce[filePath]; exists {
		timer.Stop()
	}

	// Create new timer
	d.debounce[filePath] = time.AfterFunc(2*time.Second, func() {
		d.uploadFile(filePath)
		d.debounceMu.Lock()
		delete(d.debounce, filePath)
		d.debounceMu.Unlock()
	})
}

// uploadFile uploads a file to the server
func (d *SyncDaemon) uploadFile(filePath string) {
	relPath, err := filepath.Rel(d.watchDir, filePath)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}

	// Get parent folder ID (for now, upload to root)
	parentID := ""

	// Upload file
	if err := d.cli.uploadFile(filePath, parentID); err != nil {
		fmt.Printf("  Upload failed %s: %v\n", relPath, err)
	} else {
		fmt.Printf("  Uploaded: %s\n", relPath)
	}
}

// handleDelete handles file deletion
func (d *SyncDaemon) handleDelete(filePath string) {
	relPath, err := filepath.Rel(d.watchDir, filePath)
	if err != nil {
		return
	}
	fmt.Printf("  Deleted: %s (remove from server manually if needed)\n", relPath)
}
