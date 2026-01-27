package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"pgregory.net/rapid"
)

// **Feature: lambda-alias-deployment, Property 2: SAMConfig 解析**
// **Validates: Requirements 3.1, 3.2, 3.5, 13.2**

// TestProperty2_SAMConfigParsing tests that for any valid samconfig.toml file,
// parsing correctly extracts stack_name and profile, and generates correct function names.
func TestProperty2_SAMConfigParsing(t *testing.T) {
	tmpDir := t.TempDir()

	rapid.Check(t, func(rt *rapid.T) {
		// Generate random but valid configuration values
		stackName := rapid.StringMatching(`[a-z][a-z0-9-]{2,20}`).Draw(rt, "stackName")
		profile := rapid.StringMatching(`[a-z][a-z0-9-]{2,15}`).Draw(rt, "profile")
		env := rapid.SampledFrom([]string{"test", "prod"}).Draw(rt, "env")

		// Create a temporary samconfig.toml file
		configPath := filepath.Join(tmpDir, fmt.Sprintf("samconfig_%s.toml", stackName))

		content := fmt.Sprintf(`version = 0.1
[%s]
[%s.deploy]
[%s.deploy.parameters]
stack_name = "%s"
profile = "%s"
`, env, env, env, stackName, profile)

		err := os.WriteFile(configPath, []byte(content), 0644)
		if err != nil {
			rt.Fatalf("Failed to write config file: %v", err)
		}

		// Load and parse the config
		config, err := LoadSAMConfig(configPath)
		if err != nil {
			rt.Fatalf("LoadSAMConfig failed: %v", err)
		}

		// Property: stack_name should be correctly extracted
		gotStackName := config.GetStackName(env)
		if gotStackName != stackName {
			rt.Fatalf("GetStackName(%q) = %q, want %q", env, gotStackName, stackName)
		}

		// Property: profile should be correctly extracted
		gotProfile := config.GetProfile(env)
		if gotProfile != profile {
			rt.Fatalf("GetProfile(%q) = %q, want %q", env, gotProfile, profile)
		}

		// Property: function name should follow the format {stack_name}-function-default
		expectedFunctionName := fmt.Sprintf("%s-function-default", stackName)
		gotFunctionName := config.GetFunctionName(env)
		if gotFunctionName != expectedFunctionName {
			rt.Fatalf("GetFunctionName(%q) = %q, want %q", env, gotFunctionName, expectedFunctionName)
		}
	})
}

// TestProperty2_SAMConfigParsing_MissingEnv tests that missing environment returns empty values
func TestProperty2_SAMConfigParsing_MissingEnv(t *testing.T) {
	tmpDir := t.TempDir()

	rapid.Check(t, func(rt *rapid.T) {
		stackName := rapid.StringMatching(`[a-z][a-z0-9-]{2,20}`).Draw(rt, "stackName")
		configuredEnv := rapid.SampledFrom([]string{"test", "prod"}).Draw(rt, "configuredEnv")
		queriedEnv := rapid.SampledFrom([]string{"test", "prod", "staging", "dev"}).
			Filter(func(s string) bool { return s != configuredEnv }).
			Draw(rt, "queriedEnv")

		configPath := filepath.Join(tmpDir, fmt.Sprintf("samconfig_missing_%s.toml", stackName))

		content := fmt.Sprintf(`version = 0.1
[%s]
[%s.deploy]
[%s.deploy.parameters]
stack_name = "%s"
`, configuredEnv, configuredEnv, configuredEnv, stackName)

		err := os.WriteFile(configPath, []byte(content), 0644)
		if err != nil {
			rt.Fatalf("Failed to write config file: %v", err)
		}

		config, err := LoadSAMConfig(configPath)
		if err != nil {
			rt.Fatalf("LoadSAMConfig failed: %v", err)
		}

		// Property: querying non-existent env should return empty strings
		if got := config.GetStackName(queriedEnv); got != "" {
			rt.Fatalf("GetStackName(%q) = %q, want empty string", queriedEnv, got)
		}
		if got := config.GetProfile(queriedEnv); got != "" {
			rt.Fatalf("GetProfile(%q) = %q, want empty string", queriedEnv, got)
		}
		if got := config.GetFunctionName(queriedEnv); got != "" {
			rt.Fatalf("GetFunctionName(%q) = %q, want empty string", queriedEnv, got)
		}
	})
}

// TestLoadSAMConfig_FileNotFound tests error handling for missing file
func TestLoadSAMConfig_FileNotFound(t *testing.T) {
	_, err := LoadSAMConfig("/nonexistent/path/samconfig.toml")
	if err == nil {
		t.Error("LoadSAMConfig should return error for non-existent file")
	}
}

// TestLoadSAMConfig_InvalidTOML tests error handling for invalid TOML
func TestLoadSAMConfig_InvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "samconfig.toml")

	// Write invalid TOML content
	err := os.WriteFile(configPath, []byte("invalid [ toml content"), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err = LoadSAMConfig(configPath)
	if err == nil {
		t.Error("LoadSAMConfig should return error for invalid TOML")
	}
}
