package cli

import (
	"fmt"
	"strings"
	"time"
)

// handleList handles the ls/list command
func (cli *CLI) handleList(args []string) error {
	if err := cli.ensureLoggedIn(); err != nil {
		return err
	}

	parentID := ""
	if len(args) > 0 {
		// Resolve folder name to ID
		files, err := cli.client.ListFiles("", 100, 0)
		if err != nil {
			return err
		}
		for _, f := range files {
			if f.Name == args[0] && f.Type == "folder" {
				parentID = f.ID
				break
			}
		}
		if parentID == "" {
			return fmt.Errorf("folder not found: %s", args[0])
		}
	}

	files, err := cli.client.ListFiles(parentID, 100, 0)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		fmt.Println("(empty)")
		return nil
	}

	// Print header
	fmt.Printf("%-30s %10s %20s\n", "NAME", "SIZE", "MODIFIED")
	fmt.Println(strings.Repeat("-", 70))

	for _, f := range files {
		sizeStr := "-"
		if f.Type == "file" {
			sizeStr = formatBytes(f.Size)
		}

		name := f.Name
		if f.Type == "folder" {
			name = name + "/"
		}

		modified := time.Unix(f.UpdatedAt, 0).Format("Jan 02 15:04")

		fmt.Printf("%-30s %10s %20s\n", name, sizeStr, modified)
	}

	return nil
}

// handleCD handles the cd command
func (cli *CLI) handleCD(args []string) error {
	if err := cli.ensureLoggedIn(); err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: cd <folder>")
	}

	// For now, just store the folder ID in memory
	// In a full implementation, we'd maintain a current directory stack
	fmt.Printf("Changed to: %s\n", args[0])
	return nil
}

// handlePWD handles the pwd command
func (cli *CLI) handlePWD() error {
	fmt.Println("/")
	return nil
}

// handleMkdir handles the mkdir command
func (cli *CLI) handleMkdir(args []string) error {
	if err := cli.ensureLoggedIn(); err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: mkdir <name>")
	}

	folder, err := cli.client.CreateFolder(args[0], "")
	if err != nil {
		return err
	}

	fmt.Printf("Created folder: %s (ID: %s)\n", folder.Name, folder.ID)
	return nil
}

// handleDelete handles the rm/delete command
func (cli *CLI) handleDelete(args []string) error {
	if err := cli.ensureLoggedIn(); err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: rm <file-or-folder>")
	}

	name := args[0]

	// Find the file by name
	files, err := cli.client.ListFiles("", 100, 0)
	if err != nil {
		return err
	}

	var target *File
	for _, f := range files {
		if f.Name == name {
			target = &f
			break
		}
	}

	if target == nil {
		return fmt.Errorf("not found: %s", name)
	}

	if target.Type == "folder" {
		err = cli.client.DeleteFolder(target.ID)
	} else {
		err = cli.client.DeleteFile(target.ID)
	}

	if err != nil {
		return err
	}

	fmt.Printf("Deleted: %s\n", name)
	return nil
}

// handleMove handles the mv/move command
func (cli *CLI) handleMove(args []string) error {
	if err := cli.ensureLoggedIn(); err != nil {
		return err
	}

	if len(args) < 2 {
		return fmt.Errorf("usage: mv <source> <destination-folder>")
	}

	srcName := args[0]
	dstName := args[1]

	// Find source
	files, err := cli.client.ListFiles("", 100, 0)
	if err != nil {
		return err
	}

	var srcFile *File
	var dstFolder *File
	for _, f := range files {
		if f.Name == srcName {
			srcFile = &f
		}
		if f.Name == dstName && f.Type == "folder" {
			dstFolder = &f
		}
	}

	if srcFile == nil {
		return fmt.Errorf("source not found: %s", srcName)
	}
	if dstFolder == nil {
		return fmt.Errorf("destination folder not found: %s", dstName)
	}

	if err := cli.client.MoveFile(srcFile.ID, dstFolder.ID); err != nil {
		return err
	}

	fmt.Printf("Moved %s to %s/\n", srcName, dstName)
	return nil
}

// handleRename handles the rename command
func (cli *CLI) handleRename(args []string) error {
	if err := cli.ensureLoggedIn(); err != nil {
		return err
	}

	if len(args) < 2 {
		return fmt.Errorf("usage: rename <file> <new-name>")
	}

	fileName := args[0]
	newName := args[1]

	// Find file
	files, err := cli.client.ListFiles("", 100, 0)
	if err != nil {
		return err
	}

	var file *File
	for _, f := range files {
		if f.Name == fileName {
			file = &f
			break
		}
	}

	if file == nil {
		return fmt.Errorf("file not found: %s", fileName)
	}

	if err := cli.client.RenameFile(file.ID, newName); err != nil {
		return err
	}

	fmt.Printf("Renamed %s to %s\n", fileName, newName)
	return nil
}

// handleSearch handles the search command
func (cli *CLI) handleSearch(args []string) error {
	if err := cli.ensureLoggedIn(); err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: search <query>")
	}

	query := strings.Join(args, " ")
	files, err := cli.client.SearchFiles(query, 50)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		fmt.Println("No results found")
		return nil
	}

	fmt.Printf("Found %d results:\n\n", len(files))
	fmt.Printf("%-30s %10s %20s\n", "NAME", "SIZE", "TYPE")
	fmt.Println(strings.Repeat("-", 70))

	for _, f := range files {
		sizeStr := "-"
		if f.Type == "file" {
			sizeStr = formatBytes(f.Size)
		}
		fmt.Printf("%-30s %10s %20s\n", f.Name, sizeStr, f.Type)
	}

	return nil
}

// formatBytes formats bytes to human readable string
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
