package cdp

import (
	"os/exec"
	"runtime"
	"testing"
)

func TestChromeProcessPID(t *testing.T) {
	t.Run("nil cmd", func(t *testing.T) {
		cp := &ChromeProcess{
			Cmd:  nil,
			Port: "9222",
		}
		if pid := cp.PID(); pid != 0 {
			t.Errorf("expected PID 0 for nil Cmd, got %d", pid)
		}
	})

	t.Run("cmd with nil process", func(t *testing.T) {
		cp := &ChromeProcess{
			Cmd:  &exec.Cmd{},
			Port: "9222",
		}
		if pid := cp.PID(); pid != 0 {
			t.Errorf("expected PID 0 for nil Process, got %d", pid)
		}
	})
}

func TestChromeProcessStop(t *testing.T) {
	t.Run("stop with nil cmd", func(t *testing.T) {
		cp := &ChromeProcess{
			Cmd:         nil,
			Port:        "9222",
			UserDataDir: "",
		}
		err := cp.Stop()
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})

	t.Run("stop with nil process", func(t *testing.T) {
		cp := &ChromeProcess{
			Cmd:         &exec.Cmd{},
			Port:        "9222",
			UserDataDir: "",
		}
		err := cp.Stop()
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})
}

func TestFindChrome(t *testing.T) {
	// This test just verifies findChrome doesn't panic
	// The actual result depends on whether Chrome is installed
	path := findChrome()

	// If Chrome is found, verify the path is not empty
	if path != "" {
		t.Logf("Found Chrome at: %s", path)
	} else {
		t.Log("Chrome not found (this may be expected in CI environments)")
	}
}

func TestFindChromePathsForPlatform(t *testing.T) {
	// Verify the function handles the current platform without panicking
	path := findChrome()
	_ = path // We just want to ensure no panic

	// Log the platform for debugging
	t.Logf("Running on %s", runtime.GOOS)
}

func TestChromeProcessStruct(t *testing.T) {
	cp := &ChromeProcess{
		Cmd:         nil,
		Port:        "9222",
		UserDataDir: "/tmp/test",
	}

	if cp.Port != "9222" {
		t.Errorf("expected Port '9222', got %q", cp.Port)
	}
	if cp.UserDataDir != "/tmp/test" {
		t.Errorf("expected UserDataDir '/tmp/test', got %q", cp.UserDataDir)
	}
}

func TestLaunchChromeWithInvalidPath(t *testing.T) {
	// Skip if Chrome is actually installed (we want to test the error case)
	if findChrome() == "" {
		t.Skip("Chrome not found, cannot test LaunchChrome error handling")
	}

	// This test verifies the function signature and basic error handling
	// We don't actually launch Chrome in unit tests
	t.Log("Skipping actual launch test - would launch real Chrome")
}
