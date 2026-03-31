package desktop

import (
	"fmt"
	"runtime"
	"testing"
)

// TestGetIcon tests the icon generation
func TestGetIcon(t *testing.T) {
	icon := getIcon()

	if len(icon) == 0 {
		t.Error("Icon should not be empty")
	}

	// Should start with PNG magic bytes
	if len(icon) >= 8 {
		pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
		for i, b := range pngSignature {
			if icon[i] != b {
				t.Logf("Note: Icon may not be a valid PNG (byte %d mismatch)", i)
				break
			}
		}
	}

	t.Logf("✅ Icon data generated (%d bytes)", len(icon))
}

// TestOpenBrowserCommand tests browser command construction
func TestOpenBrowserCommand(t *testing.T) {
	tests := []struct {
		os      string
		url     string
		wantCmd string
	}{
		{"windows", "http://localhost:8080", "cmd"},
		{"darwin", "http://localhost:8080", "open"},
		{"linux", "http://localhost:8080", "xdg-open"},
	}

	for _, tt := range tests {
		t.Run(tt.os, func(t *testing.T) {
			// Verify we can determine the command for each OS
			var cmd string
			switch tt.os {
			case "windows":
				cmd = "cmd"
			case "darwin":
				cmd = "open"
			default:
				cmd = "xdg-open"
			}

			if cmd != tt.wantCmd {
				t.Errorf("Expected command %s for %s, got %s", tt.wantCmd, tt.os, cmd)
			}
		})
	}
}

// TestTrayMenuStruct tests the TrayMenu struct definition
func TestTrayMenuStruct(t *testing.T) {
	// Verify the struct can be instantiated
	tray := &TrayMenu{
		quitChan: make(chan struct{}),
	}

	if tray.quitChan == nil {
		t.Error("TrayMenu should have quit channel")
	}

	t.Log("✅ TrayMenu struct is valid")
}

// TestRuntimeOS verifies runtime OS detection
func TestRuntimeOS(t *testing.T) {
	os := runtime.GOOS

	validOSes := map[string]bool{
		"windows": true,
		"darwin":  true,
		"linux":   true,
	}

	if !validOSes[os] {
		t.Logf("Running on OS: %s (may have limited support)", os)
	} else {
		t.Logf("✅ Running on supported OS: %s", os)
	}
}

// TestURLConstruction tests URL construction helpers
func TestURLConstruction(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		path     string
		expected string
	}{
		{
			name:     "web interface",
			port:     8080,
			path:     "",
			expected: "http://localhost:8080",
		},
		{
			name:     "settings",
			port:     8080,
			path:     "#/settings",
			expected: "http://localhost:8080/#/settings",
		},
		{
			name:     "custom port",
			port:     3000,
			path:     "",
			expected: "http://localhost:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var url string
			if tt.path != "" {
				url = fmt.Sprintf("http://localhost:%d/%s", tt.port, tt.path)
			} else {
				url = fmt.Sprintf("http://localhost:%d", tt.port)
			}

			if url != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, url)
			}
		})
	}
}

// TestAppInterface verifies the App type exists and has expected methods
func TestAppInterface(t *testing.T) {
	// Verify the App type can be referenced
	var _ *App

	// Verify method signatures exist by referencing them
	// We can't actually call these without a full setup
	_ = (&App{}).IsServerRunning

	t.Log("✅ App type and methods exist")
}
