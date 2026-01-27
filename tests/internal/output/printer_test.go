package output_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/aura-studio/lambda-alias-deployment/internal/output"
)

// captureOutput captures stdout and stderr during function execution
func captureOutput(fn func()) (stdout, stderr string) {
	// Save original stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Create pipes
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	// Replace stdout and stderr
	os.Stdout = wOut
	os.Stderr = wErr

	// Execute the function
	fn()

	// Close writers and restore
	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// Read captured output
	var bufOut, bufErr bytes.Buffer
	bufOut.ReadFrom(rOut)
	bufErr.ReadFrom(rErr)

	return bufOut.String(), bufErr.String()
}

func TestInfo(t *testing.T) {
	stdout, stderr := captureOutput(func() {
		output.Info("test message")
	})

	if !strings.Contains(stdout, "test message") {
		t.Errorf("Info should output to stdout, got: %q", stdout)
	}
	if stderr != "" {
		t.Errorf("Info should not output to stderr, got: %q", stderr)
	}
}

func TestInfoWithFormat(t *testing.T) {
	stdout, _ := captureOutput(func() {
		output.Info("value: %d", 42)
	})

	if !strings.Contains(stdout, "value: 42") {
		t.Errorf("Info should format message correctly, got: %q", stdout)
	}
}

func TestError(t *testing.T) {
	stdout, stderr := captureOutput(func() {
		output.Error("test error")
	})

	if stdout != "" {
		t.Errorf("Error should not output to stdout, got: %q", stdout)
	}
	if !strings.Contains(stderr, "错误: test error") {
		t.Errorf("Error should output to stderr with '错误: ' prefix, got: %q", stderr)
	}
}

func TestErrorWithFormat(t *testing.T) {
	_, stderr := captureOutput(func() {
		output.Error("code: %d", 500)
	})

	if !strings.Contains(stderr, "错误: code: 500") {
		t.Errorf("Error should format message correctly, got: %q", stderr)
	}
}

func TestSuccess(t *testing.T) {
	stdout, stderr := captureOutput(func() {
		output.Success("operation completed")
	})

	if !strings.Contains(stdout, "✓ operation completed") {
		t.Errorf("Success should output with '✓ ' prefix, got: %q", stdout)
	}
	if stderr != "" {
		t.Errorf("Success should not output to stderr, got: %q", stderr)
	}
}

func TestSuccessWithFormat(t *testing.T) {
	stdout, _ := captureOutput(func() {
		output.Success("created %d items", 5)
	})

	if !strings.Contains(stdout, "✓ created 5 items") {
		t.Errorf("Success should format message correctly, got: %q", stdout)
	}
}

func TestWarning(t *testing.T) {
	stdout, stderr := captureOutput(func() {
		output.Warning("be careful")
	})

	if !strings.Contains(stdout, "⚠ be careful") {
		t.Errorf("Warning should output with '⚠ ' prefix, got: %q", stdout)
	}
	if stderr != "" {
		t.Errorf("Warning should not output to stderr, got: %q", stderr)
	}
}

func TestWarningWithFormat(t *testing.T) {
	stdout, _ := captureOutput(func() {
		output.Warning("limit: %d%%", 80)
	})

	if !strings.Contains(stdout, "⚠ limit: 80%") {
		t.Errorf("Warning should format message correctly, got: %q", stdout)
	}
}

func TestSeparator(t *testing.T) {
	stdout, stderr := captureOutput(func() {
		output.Separator()
	})

	expected := "=========================================="
	if !strings.Contains(stdout, expected) {
		t.Errorf("Separator should output separator line, got: %q", stdout)
	}
	if stderr != "" {
		t.Errorf("Separator should not output to stderr, got: %q", stderr)
	}
}
