// Package cdp provides Chrome DevTools Protocol connection and management.
package cdp

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ChromeProcess represents a launched Chrome instance.
type ChromeProcess struct {
	Cmd         *exec.Cmd
	Port        string
	UserDataDir string
}

// LaunchChrome starts a new Chrome instance with remote debugging enabled.
func LaunchChrome(port string) (*ChromeProcess, error) {
	chromePath := findChrome()
	if chromePath == "" {
		return nil, errors.New("chrome executable not found")
	}

	// Create a temporary user data directory
	userDataDir, err := os.MkdirTemp("", "browser_tail_chrome_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	args := []string{
		"--remote-debugging-port=" + port,
		"--user-data-dir=" + userDataDir,
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-features=TranslateUI",
		"--disable-background-networking",
		"--disable-sync",
	}

	cmd := exec.Command(chromePath, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		// Clean up temp dir on failure
		_ = os.RemoveAll(userDataDir)
		return nil, fmt.Errorf("failed to start chrome: %w", err)
	}

	return &ChromeProcess{
		Cmd:         cmd,
		Port:        port,
		UserDataDir: userDataDir,
	}, nil
}

// Stop terminates the Chrome process and cleans up.
func (cp *ChromeProcess) Stop() error {
	if cp.Cmd != nil && cp.Cmd.Process != nil {
		if err := cp.Cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill chrome: %w", err)
		}
		// Wait for process to exit
		_ = cp.Cmd.Wait()
	}

	// Clean up user data directory
	if cp.UserDataDir != "" {
		_ = os.RemoveAll(cp.UserDataDir)
	}

	return nil
}

// PID returns the process ID of the Chrome instance.
func (cp *ChromeProcess) PID() int {
	if cp.Cmd != nil && cp.Cmd.Process != nil {
		return cp.Cmd.Process.Pid
	}
	return 0
}

// findChrome locates the Chrome executable on the system.
func findChrome() string {
	var paths []string

	switch runtime.GOOS {
	case "darwin":
		paths = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			filepath.Join(os.Getenv("HOME"), "Applications/Google Chrome.app/Contents/MacOS/Google Chrome"),
		}
	case "linux":
		paths = []string{
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/snap/bin/chromium",
		}
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		programFiles := os.Getenv("PROGRAMFILES")
		programFilesX86 := os.Getenv("PROGRAMFILES(X86)")

		paths = []string{
			filepath.Join(localAppData, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(programFiles, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(programFilesX86, "Google", "Chrome", "Application", "chrome.exe"),
		}
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Try finding in PATH
	if path, err := exec.LookPath("google-chrome"); err == nil {
		return path
	}
	if path, err := exec.LookPath("chrome"); err == nil {
		return path
	}
	if path, err := exec.LookPath("chromium"); err == nil {
		return path
	}

	return ""
}
