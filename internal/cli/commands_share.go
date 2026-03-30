package cli

import (
	"fmt"
	"strconv"
	"time"
)

// handleShare handles the share command
func (cli *CLI) handleShare(args []string) error {
	if err := cli.ensureLoggedIn(); err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: share <file-name> [--expires <days>] [--password <pass>] [--max-downloads <n>]")
	}

	fileName := args[0]
	expiresDays := 0
	password := ""
	maxDownloads := 0

	// Parse flags
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--expires", "-e":
			if i+1 < len(args) {
				expiresDays, _ = strconv.Atoi(args[i+1])
				i++
			}
		case "--password", "-p":
			if i+1 < len(args) {
				password = args[i+1]
				i++
			}
		case "--max-downloads", "-m":
			if i+1 < len(args) {
				maxDownloads, _ = strconv.Atoi(args[i+1])
				i++
			}
		}
	}

	// Find file
	files, err := cli.client.ListFiles("", 100, 0)
	if err != nil {
		return err
	}

	var target *File
	for _, f := range files {
		if f.Name == fileName {
			target = &f
			break
		}
	}

	if target == nil {
		return fmt.Errorf("file not found: %s", fileName)
	}

	// Create share request
	req := &CreateShareRequest{
		ShareType:   "link",
		Permission:  "read",
		AllowUpload: false,
		PreviewOnly: false,
	}

	if expiresDays > 0 {
		req.ExpiresDays = &expiresDays
	}
	if password != "" {
		req.Password = &password
	}
	if maxDownloads > 0 {
		req.MaxDownloads = &maxDownloads
	}

	share, shareURL, err := cli.client.CreateShare(target.ID, req)
	if err != nil {
		return err
	}

	fmt.Printf("Share created:\n")
	fmt.Printf("  File: %s\n", fileName)
	fmt.Printf("  Share ID: %s\n", share.ID)
	if shareURL != "" {
		fmt.Printf("  Share URL: %s\n", shareURL)
	}
	if share.ExpiresAt != nil {
		fmt.Printf("  Expires: %s\n", time.Unix(*share.ExpiresAt, 0).Format("2006-01-02 15:04"))
	}

	return nil
}

// handleListShares handles the shares command
func (cli *CLI) handleListShares(args []string) error {
	if err := cli.ensureLoggedIn(); err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: shares <file-name>")
	}

	fileName := args[0]

	// Find file
	files, err := cli.client.ListFiles("", 100, 0)
	if err != nil {
		return err
	}

	var target *File
	for _, f := range files {
		if f.Name == fileName {
			target = &f
			break
		}
	}

	if target == nil {
		return fmt.Errorf("file not found: %s", fileName)
	}

	shares, err := cli.client.ListShares(target.ID)
	if err != nil {
		return err
	}

	if len(shares) == 0 {
		fmt.Println("No shares for this file")
		return nil
	}

	fmt.Printf("Shares for %s:\n\n", fileName)
	for _, share := range shares {
		status := "active"
		if !share.IsActive {
			status = "revoked"
		}

		expires := "never"
		if share.ExpiresAt != nil {
			expires = time.Unix(*share.ExpiresAt, 0).Format("2006-01-02")
		}

		fmt.Printf("  %s [%s] expires: %s\n", share.ID, status, expires)
	}

	return nil
}

// handleUnshare handles the unshare command
func (cli *CLI) handleUnshare(args []string) error {
	if err := cli.ensureLoggedIn(); err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: unshare <share-id>")
	}

	shareID := args[0]

	if err := cli.client.RevokeShare(shareID); err != nil {
		return err
	}

	fmt.Printf("Share revoked: %s\n", shareID)
	return nil
}
