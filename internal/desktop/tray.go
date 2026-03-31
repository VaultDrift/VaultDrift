package desktop

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/getlantern/systray"
)

// TrayMenu represents the system tray menu
type TrayMenu struct {
	app      *App
	menu     *systray.MenuItem
	quitChan chan struct{}
}

// NewTrayMenu creates a new system tray menu
func NewTrayMenu(app *App) *TrayMenu {
	return &TrayMenu{
		app:      app,
		quitChan: make(chan struct{}),
	}
}

// Run starts the system tray
func (t *TrayMenu) Run() error {
	// Run systray in a goroutine
	go systray.Run(t.onReady, t.onExit)

	// Wait for quit signal
	<-t.quitChan
	return nil
}

// onReady is called when the tray is ready
func (t *TrayMenu) onReady() {
	// Set icon and tooltip
	systray.SetIcon(getIcon())
	systray.SetTooltip("VaultDrift - Secure Cloud Storage")

	// Set title (macOS only)
	systray.SetTitle("VaultDrift")

	// Add menu items
	mOpen := systray.AddMenuItem("Open VaultDrift", "Open the web interface")
	mSync := systray.AddMenuItem("Sync Now", "Trigger sync")
	systray.AddSeparator()
	mStatus := systray.AddMenuItem("Status: Running", "Server status")
	mStatus.Disable()
	systray.AddSeparator()
	mSettings := systray.AddMenuItem("Settings", "Open settings")
	mAbout := systray.AddMenuItem("About", "About VaultDrift")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit VaultDrift")

	// Handle menu clicks
	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				t.openWebInterface()
			case <-mSync.ClickedCh:
				t.triggerSync()
			case <-mSettings.ClickedCh:
				t.openSettings()
			case <-mAbout.ClickedCh:
				t.showAbout()
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

// onExit is called when the tray exits
func (t *TrayMenu) onExit() {
	close(t.quitChan)
}

// openWebInterface opens the web interface in the default browser
func (t *TrayMenu) openWebInterface() {
	url := fmt.Sprintf("http://localhost:%d", t.app.config.Server.Port)
	openBrowser(url)
}

// triggerSync triggers a sync operation
func (t *TrayMenu) triggerSync() {
	// TODO: Trigger sync with local sync folder
	// For now, just log
	fmt.Println("Sync triggered from tray")
}

// openSettings opens the settings
func (t *TrayMenu) openSettings() {
	url := fmt.Sprintf("http://localhost:%d/#/settings", t.app.config.Server.Port)
	openBrowser(url)
}

// showAbout shows the about dialog
func (t *TrayMenu) showAbout() {
	// TODO: Show native about dialog
	// For now, just open the web interface
	url := fmt.Sprintf("http://localhost:%d", t.app.config.Server.Port)
	openBrowser(url)
}

// openBrowser opens a URL in the default browser
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}

// getIcon returns the tray icon data
func getIcon() []byte {
	// Return a simple 16x16 icon (blue square for now)
	// In production, this would be a proper embedded icon
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
	}
}
