package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// handleUpload handles the upload command
func (cli *CLI) handleUpload(args []string) error {
	if err := cli.ensureLoggedIn(); err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: upload <file-path>")
	}

	filePath := args[0]
	parentID := ""

	// Check for --to flag
	for i, arg := range args {
		if arg == "--to" && i+1 < len(args) {
			// Find folder by name
			files, err := cli.client.ListFiles("", 100, 0)
			if err != nil {
				return err
			}
			for _, f := range files {
				if f.Name == args[i+1] && f.Type == "folder" {
					parentID = f.ID
					break
				}
			}
			if parentID == "" {
				return fmt.Errorf("folder not found: %s", args[i+1])
			}
			break
		}
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if stat.IsDir() {
		return fmt.Errorf("cannot upload directories (yet)")
	}

	fileName := filepath.Base(filePath)
	mimeType := detectMimeType(fileName)

	// Get upload URL
	uploadInfo, err := cli.client.GetUploadURL(parentID, fileName, mimeType, stat.Size())
	if err != nil {
		return fmt.Errorf("failed to get upload URL: %w", err)
	}

	fmt.Printf("Uploading %s...\n", fileName)

	// Upload file directly
	req, err := http.NewRequest("PUT", uploadInfo.UploadURL, file)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mimeType)

	resp, err := cli.client.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed: %s", string(body))
	}

	fmt.Printf("Uploaded: %s (ID: %s)\n", fileName, uploadInfo.FileID)
	return nil
}

// handleDownload handles the download command
func (cli *CLI) handleDownload(args []string) error {
	if err := cli.ensureLoggedIn(); err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: download <file-name> [--output <path>]")
	}

	fileName := args[0]
	outputPath := fileName

	// Check for --output flag
	for i, arg := range args {
		if arg == "--output" || arg == "-o" {
			if i+1 < len(args) {
				outputPath = args[i+1]
			}
			break
		}
	}

	// Find file by name
	files, err := cli.client.ListFiles("", 100, 0)
	if err != nil {
		return err
	}

	var target *File
	for _, f := range files {
		if f.Name == fileName && f.Type == "file" {
			target = &f
			break
		}
	}

	if target == nil {
		return fmt.Errorf("file not found: %s", fileName)
	}

	// Get download URL
	downloadInfo, err := cli.client.GetDownloadURL(target.ID)
	if err != nil {
		return fmt.Errorf("failed to get download URL: %w", err)
	}

	fmt.Printf("Downloading %s...\n", fileName)

	// Download file
	resp, err := cli.client.HTTPClient.Get(downloadInfo.DownloadURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("download failed: %s", string(body))
	}

	// Create output file
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	// Copy content
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	fmt.Printf("Downloaded: %s\n", outputPath)
	return nil
}

// handleSync handles the sync command
func (cli *CLI) handleSync(args []string) error {
	if err := cli.ensureLoggedIn(); err != nil {
		return err
	}

	syncDir := cli.config.DefaultDir
	if len(args) > 0 {
		syncDir = args[0]
	}

	if syncDir == "" {
		return fmt.Errorf("no sync directory specified. Use 'config dir <path>' or provide path as argument")
	}

	// Ensure directory exists
	if _, err := os.Stat(syncDir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", syncDir)
	}

	fmt.Printf("Syncing %s with server...\n", syncDir)

	// List local files
	localFiles := make(map[string]os.FileInfo)
	err := filepath.Walk(syncDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(syncDir, path)
			localFiles[relPath] = info
		}
		return nil
	})
	if err != nil {
		return err
	}

	// List remote files
	remoteFiles, err := cli.client.ListFiles("", 1000, 0)
	if err != nil {
		return err
	}

	// Simple sync: upload files not on server
	uploaded := 0
	skipped := 0

	for relPath, localInfo := range localFiles {
		found := false
		for _, rf := range remoteFiles {
			if rf.Name == localInfo.Name() && rf.Type == "file" {
				found = true
				break
			}
		}

		if !found {
			// Upload file
			localPath := filepath.Join(syncDir, relPath)
			if err := cli.uploadFile(localPath, ""); err != nil {
				fmt.Printf("  Failed to upload %s: %v\n", relPath, err)
			} else {
				fmt.Printf("  Uploaded: %s\n", relPath)
				uploaded++
			}
		} else {
			skipped++
		}
	}

	fmt.Printf("\nSync complete: %d uploaded, %d skipped\n", uploaded, skipped)
	return nil
}

// uploadFile uploads a single file
func (cli *CLI) uploadFile(filePath, parentID string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	fileName := filepath.Base(filePath)
	mimeType := detectMimeType(fileName)

	uploadInfo, err := cli.client.GetUploadURL(parentID, fileName, mimeType, stat.Size())
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", uploadInfo.UploadURL, file)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mimeType)

	resp, err := cli.client.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s", string(body))
	}

	return nil
}

// detectMimeType detects MIME type from filename
func detectMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".txt":
		return "text/plain"
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	case ".zip":
		return "application/zip"
	case ".mp4":
		return "video/mp4"
	case ".mp3":
		return "audio/mpeg"
	default:
		return "application/octet-stream"
	}
}

// handleDaemon handles the daemon command
func (cli *CLI) handleDaemon(args []string) error {
	if err := cli.ensureLoggedIn(); err != nil {
		return err
	}

	watchDir := cli.config.DefaultDir
	if len(args) > 0 {
		watchDir = args[0]
	}

	if watchDir == "" {
		return fmt.Errorf("no watch directory specified. Use 'config dir <path>' or provide path as argument")
	}

	// Ensure directory exists
	if _, err := os.Stat(watchDir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", watchDir)
	}

	// Create and start daemon
	daemon, err := NewSyncDaemon(cli, watchDir)
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}

	return daemon.Start()
}
