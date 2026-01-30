package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aura-studio/lad/cmd"
	"pgregory.net/rapid"
)

func TestValidateEnv(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		wantErr bool
	}{
		{
			name:    "valid test environment",
			env:     "test",
			wantErr: false,
		},
		{
			name:    "valid prod environment",
			env:     "prod",
			wantErr: false,
		},
		{
			name:    "invalid environment - dev",
			env:     "dev",
			wantErr: true,
		},
		{
			name:    "invalid environment - staging",
			env:     "staging",
			wantErr: true,
		},
		{
			name:    "invalid environment - empty",
			env:     "",
			wantErr: true,
		},
		{
			name:    "invalid environment - uppercase TEST",
			env:     "TEST",
			wantErr: true,
		},
		{
			name:    "invalid environment - uppercase PROD",
			env:     "PROD",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cmd.ValidateEnv(tt.env)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEnv(%q) error = %v, wantErr %v", tt.env, err, tt.wantErr)
			}
		})
	}
}

func TestGetFunctionName(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "lad-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("function flag takes priority", func(t *testing.T) {
		cmd.SetFunction("my-custom-function")
		cmd.SetSamconfigPath(filepath.Join(tmpDir, "nonexistent.toml"))
		defer cmd.SetFunction("")

		result, err := cmd.GetFunctionName("test")
		if err != nil {
			t.Errorf("GetFunctionName() unexpected error: %v", err)
		}
		if result != "my-custom-function" {
			t.Errorf("GetFunctionName() = %q, want %q", result, "my-custom-function")
		}
	})

	t.Run("reads from samconfig when function flag not set", func(t *testing.T) {
		cmd.SetFunction("")
		samconfigPath := filepath.Join(tmpDir, "samconfig.toml")
		cmd.SetSamconfigPath(samconfigPath)

		// Create a valid samconfig.toml
		samconfigContent := `
version = 0.1

[test.deploy.parameters]
stack_name = "my-stack"
profile = "my-profile"
`
		if err := os.WriteFile(samconfigPath, []byte(samconfigContent), 0644); err != nil {
			t.Fatalf("Failed to write samconfig.toml: %v", err)
		}

		result, err := cmd.GetFunctionName("test")
		if err != nil {
			t.Errorf("GetFunctionName() unexpected error: %v", err)
		}
		expected := "my-stack-function-default"
		if result != expected {
			t.Errorf("GetFunctionName() = %q, want %q", result, expected)
		}
	})

	t.Run("error when samconfig not found and no function flag", func(t *testing.T) {
		cmd.SetFunction("")
		cmd.SetSamconfigPath(filepath.Join(tmpDir, "nonexistent.toml"))

		_, err := cmd.GetFunctionName("test")
		if err == nil {
			t.Error("GetFunctionName() expected error, got nil")
		}
	})

	t.Run("error when samconfig has no stack_name", func(t *testing.T) {
		cmd.SetFunction("")
		samconfigPath := filepath.Join(tmpDir, "samconfig_empty.toml")
		cmd.SetSamconfigPath(samconfigPath)

		// Create a samconfig.toml without stack_name
		samconfigContent := `
version = 0.1

[test.deploy.parameters]
profile = "my-profile"
`
		if err := os.WriteFile(samconfigPath, []byte(samconfigContent), 0644); err != nil {
			t.Fatalf("Failed to write samconfig.toml: %v", err)
		}

		_, err := cmd.GetFunctionName("test")
		if err == nil {
			t.Error("GetFunctionName() expected error, got nil")
		}
	})
}

func TestGetProfile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "lad-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("profile flag takes priority", func(t *testing.T) {
		cmd.SetProfile("my-custom-profile")
		cmd.SetSamconfigPath(filepath.Join(tmpDir, "nonexistent.toml"))
		defer cmd.SetProfile("")

		result := cmd.GetProfile("test")
		if result != "my-custom-profile" {
			t.Errorf("GetProfile() = %q, want %q", result, "my-custom-profile")
		}
	})

	t.Run("reads from samconfig when profile flag not set", func(t *testing.T) {
		cmd.SetProfile("")
		samconfigPath := filepath.Join(tmpDir, "samconfig.toml")
		cmd.SetSamconfigPath(samconfigPath)

		// Create a valid samconfig.toml
		samconfigContent := `
version = 0.1

[test.deploy.parameters]
stack_name = "my-stack"
profile = "samconfig-profile"
`
		if err := os.WriteFile(samconfigPath, []byte(samconfigContent), 0644); err != nil {
			t.Fatalf("Failed to write samconfig.toml: %v", err)
		}

		result := cmd.GetProfile("test")
		if result != "samconfig-profile" {
			t.Errorf("GetProfile() = %q, want %q", result, "samconfig-profile")
		}
	})

	t.Run("returns empty string when samconfig not found", func(t *testing.T) {
		cmd.SetProfile("")
		cmd.SetSamconfigPath(filepath.Join(tmpDir, "nonexistent.toml"))

		result := cmd.GetProfile("test")
		if result != "" {
			t.Errorf("GetProfile() = %q, want empty string", result)
		}
	})

	t.Run("returns empty string when samconfig has no profile", func(t *testing.T) {
		cmd.SetProfile("")
		samconfigPath := filepath.Join(tmpDir, "samconfig_no_profile.toml")
		cmd.SetSamconfigPath(samconfigPath)

		// Create a samconfig.toml without profile
		samconfigContent := `
version = 0.1

[test.deploy.parameters]
stack_name = "my-stack"
`
		if err := os.WriteFile(samconfigPath, []byte(samconfigContent), 0644); err != nil {
			t.Fatalf("Failed to write samconfig.toml: %v", err)
		}

		result := cmd.GetProfile("test")
		if result != "" {
			t.Errorf("GetProfile() = %q, want empty string", result)
		}
	})

	t.Run("reads profile for prod environment", func(t *testing.T) {
		cmd.SetProfile("")
		samconfigPath := filepath.Join(tmpDir, "samconfig_prod.toml")
		cmd.SetSamconfigPath(samconfigPath)

		// Create a samconfig.toml with prod environment
		samconfigContent := `
version = 0.1

[test.deploy.parameters]
stack_name = "my-stack-test"
profile = "test-profile"

[prod.deploy.parameters]
stack_name = "my-stack-prod"
profile = "prod-profile"
`
		if err := os.WriteFile(samconfigPath, []byte(samconfigContent), 0644); err != nil {
			t.Fatalf("Failed to write samconfig.toml: %v", err)
		}

		result := cmd.GetProfile("prod")
		if result != "prod-profile" {
			t.Errorf("GetProfile(prod) = %q, want %q", result, "prod-profile")
		}
	})
}

// =============================================================================
// Property-Based Tests
// =============================================================================

// TestValidateEnvProperty tests the environment parameter validation property.
// **Validates: Requirements 2.1, 2.3**
//
// Property 1: 环境参数验证
// For any 环境参数值，如果值为 "test" 或 "prod"，则应被接受；
// 否则应返回参数错误（退出码 1）并显示有效值列表。
func TestValidateEnvProperty(t *testing.T) {
	// Property 1a: Valid environments ("test" and "prod") should always be accepted
	t.Run("valid_environments_accepted", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate either "test" or "prod"
			validEnvs := []string{"test", "prod"}
			env := rapid.SampledFrom(validEnvs).Draw(t, "env")

			err := cmd.ValidateEnv(env)
			if err != nil {
				t.Fatalf("ValidateEnv(%q) should accept valid environment, got error: %v", env, err)
			}
		})
	})

	// Property 1b: Any string that is not "test" or "prod" should be rejected
	t.Run("invalid_environments_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate arbitrary strings that are NOT "test" or "prod"
			env := rapid.StringMatching(`[a-zA-Z0-9_-]*`).Draw(t, "env")

			// Skip if we accidentally generated a valid environment
			if env == "test" || env == "prod" {
				return
			}

			err := cmd.ValidateEnv(env)
			if err == nil {
				t.Fatalf("ValidateEnv(%q) should reject invalid environment, but got no error", env)
			}

			// Verify error message contains valid values list
			errMsg := err.Error()
			if !strings.Contains(errMsg, "test") || !strings.Contains(errMsg, "prod") {
				t.Fatalf("ValidateEnv(%q) error should display valid values list (test, prod), got: %s", env, errMsg)
			}
		})
	})

	// Property 1c: Empty string should be rejected
	t.Run("empty_string_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			err := cmd.ValidateEnv("")
			if err == nil {
				t.Fatal("ValidateEnv(\"\") should reject empty string, but got no error")
			}

			// Verify error message contains valid values list
			errMsg := err.Error()
			if !strings.Contains(errMsg, "test") || !strings.Contains(errMsg, "prod") {
				t.Fatalf("ValidateEnv(\"\") error should display valid values list (test, prod), got: %s", errMsg)
			}
		})
	})

	// Property 1d: Case sensitivity - uppercase variants should be rejected
	t.Run("case_sensitivity", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate case variations of valid environments
			baseEnvs := []string{"test", "prod"}
			base := rapid.SampledFrom(baseEnvs).Draw(t, "base")

			// Generate a case variation that is NOT the original
			variations := generateCaseVariations(base)
			if len(variations) == 0 {
				return
			}
			variant := rapid.SampledFrom(variations).Draw(t, "variant")

			// Skip if variant equals the original (lowercase)
			if variant == base {
				return
			}

			err := cmd.ValidateEnv(variant)
			if err == nil {
				t.Fatalf("ValidateEnv(%q) should reject case variant of %q, but got no error", variant, base)
			}
		})
	})

	// Property 1e: Strings with whitespace should be rejected
	t.Run("whitespace_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate valid env with leading/trailing whitespace
			validEnvs := []string{"test", "prod"}
			base := rapid.SampledFrom(validEnvs).Draw(t, "base")

			// Add whitespace
			whitespaceTypes := []string{" ", "\t", "\n", "  ", "\t\t"}
			ws := rapid.SampledFrom(whitespaceTypes).Draw(t, "whitespace")

			// Generate variations with whitespace
			variations := []string{
				ws + base,
				base + ws,
				ws + base + ws,
			}
			envWithWs := rapid.SampledFrom(variations).Draw(t, "env_with_whitespace")

			err := cmd.ValidateEnv(envWithWs)
			if err == nil {
				t.Fatalf("ValidateEnv(%q) should reject string with whitespace, but got no error", envWithWs)
			}
		})
	})
}

// generateCaseVariations generates case variations of a string
func generateCaseVariations(s string) []string {
	if len(s) == 0 {
		return nil
	}

	variations := []string{}

	// Uppercase all
	upper := strings.ToUpper(s)
	if upper != s {
		variations = append(variations, upper)
	}

	// Title case
	title := strings.ToUpper(string(s[0])) + s[1:]
	if title != s {
		variations = append(variations, title)
	}

	// Mixed case variations
	if len(s) >= 2 {
		// First char upper, rest lower
		mixed1 := strings.ToUpper(string(s[0])) + strings.ToLower(s[1:])
		if mixed1 != s && mixed1 != upper && mixed1 != title {
			variations = append(variations, mixed1)
		}

		// Alternating case
		var mixed2 strings.Builder
		for i, c := range s {
			if i%2 == 0 {
				mixed2.WriteString(strings.ToUpper(string(c)))
			} else {
				mixed2.WriteString(strings.ToLower(string(c)))
			}
		}
		m2 := mixed2.String()
		if m2 != s {
			variations = append(variations, m2)
		}
	}

	return variations
}
