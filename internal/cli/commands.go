package cli

import (
	"fmt"
	"net/http"
)

// CLI represents the command-line interface
type CLI struct {
	config  *Config
	client  *Client
	configMgr *ConfigManager
}

// NewCLI creates a new CLI instance
func NewCLI() (*CLI, error) {
	configMgr, err := NewConfigManager()
	if err != nil {
		return nil, err
	}

	config, err := configMgr.Load()
	if err != nil {
		return nil, err
	}

	client := NewClient(config.ServerURL, config.Token)

	return &CLI{
		config:    config,
		client:    client,
		configMgr: configMgr,
	}, nil
}

// Run executes the CLI with the given arguments
func (cli *CLI) Run(args []string) error {
	if len(args) < 1 {
		cli.printHelp()
		return nil
	}

	command := args[0]

	switch command {
	case "help", "--help", "-h":
		cli.printHelp()
	case "version", "--version", "-v":
		cli.printVersion()
	case "config":
		return cli.handleConfig(args[1:])
	case "login":
		return cli.handleLogin()
	case "logout":
		return cli.handleLogout()
	case "ls", "list":
		return cli.handleList(args[1:])
	case "cd":
		return cli.handleCD(args[1:])
	case "pwd":
		return cli.handlePWD()
	case "mkdir":
		return cli.handleMkdir(args[1:])
	case "rm", "delete":
		return cli.handleDelete(args[1:])
	case "mv", "move":
		return cli.handleMove(args[1:])
	case "rename":
		return cli.handleRename(args[1:])
	case "upload", "up":
		return cli.handleUpload(args[1:])
	case "download", "dl":
		return cli.handleDownload(args[1:])
	case "search":
		return cli.handleSearch(args[1:])
	case "share":
		return cli.handleShare(args[1:])
	case "shares":
		return cli.handleListShares(args[1:])
	case "unshare":
		return cli.handleUnshare(args[1:])
	case "sync":
		return cli.handleSync(args[1:])
	case "status":
		return cli.handleStatus()
	default:
		return fmt.Errorf("unknown command: %s", command)
	}

	return nil
}

// printHelp prints the help message
func (cli *CLI) printHelp() {
	fmt.Print(`VaultDrift CLI - Command Line Interface for VaultDrift

USAGE:
  vaultdrift-cli <command> [options]

COMMANDS:
  help, -h, --help       Show this help message
  version, -v            Show version information
  config                 Manage configuration
  login                  Authenticate with the server
  logout                 Sign out and remove credentials
  ls, list [folder]      List files in a folder
  cd <folder>            Change current directory
  pwd                    Show current directory
  mkdir <name>           Create a new folder
  rm, delete <file>      Delete a file or folder
  mv, move <src> <dst>   Move a file or folder
  rename <file> <name>   Rename a file or folder
  upload, up <file>      Upload a file
  download, dl <file>    Download a file
  search <query>         Search for files
  share <file>           Create a share link
  shares <file>          List shares for a file
  unshare <share-id>     Revoke a share
  sync [folder]          Sync local folder with server
  status                 Show connection and auth status

EXAMPLES:
  vaultdrift-cli login
  vaultdrift-cli ls
  vaultdrift-cli mkdir "My Documents"
  vaultdrift-cli upload ./document.pdf
  vaultdrift-cli share myfile.txt --expires 7
`)
}

// printVersion prints version information
func (cli *CLI) printVersion() {
	fmt.Println("VaultDrift CLI v0.1.0")
}

// ensureLoggedIn checks if the user is logged in
func (cli *CLI) ensureLoggedIn() error {
	if cli.config.Token == "" {
		return fmt.Errorf("not logged in. Run 'vaultdrift-cli login' first")
	}
	return nil
}

// saveConfig saves the current configuration
func (cli *CLI) saveConfig() error {
	return cli.configMgr.Save(cli.config)
}

// handleConfig handles configuration commands
func (cli *CLI) handleConfig(args []string) error {
	if len(args) == 0 {
		// Show current config
		fmt.Printf("Server URL: %s\n", cli.config.ServerURL)
		fmt.Printf("Username: %s\n", cli.config.Username)
		fmt.Printf("Default Directory: %s\n", cli.config.DefaultDir)
		fmt.Printf("Logged In: %v\n", cli.config.Token != "")
		return nil
	}

	switch args[0] {
	case "server":
		if len(args) < 2 {
			return fmt.Errorf("usage: config server <url>")
		}
		cli.config.ServerURL = args[1]
		cli.client.BaseURL = args[1]
		return cli.saveConfig()
	case "dir":
		if len(args) < 2 {
			return fmt.Errorf("usage: config dir <path>")
		}
		cli.config.DefaultDir = args[1]
		return cli.saveConfig()
	default:
		return fmt.Errorf("unknown config option: %s", args[0])
	}
}

// handleLogin handles the login command
func (cli *CLI) handleLogin() error {
	username := PromptInput("Username: ")
	password := PromptPassword("Password: ")

	resp, err := cli.client.Login(username, password)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	cli.config.Token = resp.Token
	cli.config.Username = resp.Username
	cli.client.SetToken(resp.Token)

	if err := cli.saveConfig(); err != nil {
		return err
	}

	fmt.Printf("Logged in as %s\n", resp.Username)
	return nil
}

// handleLogout handles the logout command
func (cli *CLI) handleLogout() error {
	if cli.config.Token == "" {
		fmt.Println("Not logged in")
		return nil
	}

	// Try to logout on server (ignore errors)
	_ = cli.client.Logout()

	cli.config.Token = ""
	cli.config.Username = ""
	cli.client.SetToken("")

	if err := cli.saveConfig(); err != nil {
		return err
	}

	fmt.Println("Logged out successfully")
	return nil
}

// handleStatus shows connection status
func (cli *CLI) handleStatus() error {
	fmt.Printf("Server URL: %s\n", cli.config.ServerURL)
	fmt.Printf("Username: %s\n", cli.config.Username)
	fmt.Printf("Logged In: %v\n", cli.config.Token != "")

	// Test connection
	resp, err := http.Get(cli.config.ServerURL + "/health")
	if err != nil {
		fmt.Printf("Server Status: Offline (%v)\n", err)
	} else {
		resp.Body.Close()
		fmt.Printf("Server Status: Online (%d)\n", resp.StatusCode)
	}

	return nil
}
