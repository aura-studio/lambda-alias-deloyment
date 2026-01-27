package cmd_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/aura-studio/lambda-alias-deployment/cmd"
	"pgregory.net/rapid"
)

// =============================================================================
// Unit Tests for RollbackLog
// =============================================================================

func TestRollbackLog_Format(t *testing.T) {
	tests := []struct {
		name     string
		log      cmd.RollbackLog
		contains []string
	}{
		{
			name: "basic format with all fields",
			log: cmd.RollbackLog{
				Timestamp:   time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				Env:         "test",
				FromVersion: "5",
				ToVersion:   "4",
				Reason:      "bug found",
				Operator:    "admin",
			},
			contains: []string{
				"[2024-01-15T10:30:00Z]",
				"ENV=test",
				"FROM_VERSION=5",
				"TO_VERSION=4",
				"REASON=\"bug found\"",
				"OPERATOR=admin",
			},
		},
		{
			name: "format with prod environment",
			log: cmd.RollbackLog{
				Timestamp:   time.Date(2024, 6, 20, 15, 45, 30, 0, time.UTC),
				Env:         "prod",
				FromVersion: "10",
				ToVersion:   "9",
				Reason:      "performance issue",
				Operator:    "ops-team",
			},
			contains: []string{
				"ENV=prod",
				"FROM_VERSION=10",
				"TO_VERSION=9",
				"REASON=\"performance issue\"",
				"OPERATOR=ops-team",
			},
		},
		{
			name: "format with default reason",
			log: cmd.RollbackLog{
				Timestamp:   time.Date(2024, 3, 10, 8, 0, 0, 0, time.UTC),
				Env:         "test",
				FromVersion: "3",
				ToVersion:   "2",
				Reason:      "未指定原因",
				Operator:    "user1",
			},
			contains: []string{
				"REASON=\"未指定原因\"",
			},
		},
		{
			name: "format with empty operator",
			log: cmd.RollbackLog{
				Timestamp:   time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
				Env:         "test",
				FromVersion: "100",
				ToVersion:   "99",
				Reason:      "emergency",
				Operator:    "unknown",
			},
			contains: []string{
				"OPERATOR=unknown",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.log.Format()
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Format() = %q, should contain %q", result, expected)
				}
			}
		})
	}
}

func TestRollbackLog_Format_Structure(t *testing.T) {
	log := cmd.RollbackLog{
		Timestamp:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Env:         "test",
		FromVersion: "2",
		ToVersion:   "1",
		Reason:      "test reason",
		Operator:    "tester",
	}

	result := log.Format()

	// Verify the format structure: [timestamp] ENV=... FROM_VERSION=... TO_VERSION=... REASON="..." OPERATOR=...
	// The order should be: timestamp, ENV, FROM_VERSION, TO_VERSION, REASON, OPERATOR
	envIdx := strings.Index(result, "ENV=")
	fromIdx := strings.Index(result, "FROM_VERSION=")
	toIdx := strings.Index(result, "TO_VERSION=")
	reasonIdx := strings.Index(result, "REASON=")
	operatorIdx := strings.Index(result, "OPERATOR=")

	if envIdx == -1 || fromIdx == -1 || toIdx == -1 || reasonIdx == -1 || operatorIdx == -1 {
		t.Fatalf("Format() missing required fields: %q", result)
	}

	// Verify order
	if !(envIdx < fromIdx && fromIdx < toIdx && toIdx < reasonIdx && reasonIdx < operatorIdx) {
		t.Errorf("Format() fields are not in correct order: %q", result)
	}

	// Verify timestamp is at the beginning in brackets
	if !strings.HasPrefix(result, "[") {
		t.Errorf("Format() should start with '[', got: %q", result)
	}
}

// =============================================================================
// Property-Based Tests
// =============================================================================

// TestProperty5_RollbackLogFormat tests Property 5: 回退日志格式
// **Validates: Requirements 7.4, 7.5**
//
// Property 5: For any rollback operation, the generated log entry should contain
// timestamp, environment, original version, target version, reason, and operator,
// formatted as `[timestamp] ENV=env FROM_VERSION=from TO_VERSION=to REASON="reason" OPERATOR=operator`.
func TestProperty5_RollbackLogFormat(t *testing.T) {
	// Property 5a: Format always contains all required fields
	t.Run("format_contains_all_required_fields", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			log := generateRollbackLog(t)
			result := log.Format()

			// Verify all required fields are present
			requiredFields := []string{
				"ENV=",
				"FROM_VERSION=",
				"TO_VERSION=",
				"REASON=",
				"OPERATOR=",
			}

			for _, field := range requiredFields {
				if !strings.Contains(result, field) {
					t.Fatalf("Format() = %q, missing required field %q", result, field)
				}
			}

			// Verify timestamp is present in brackets
			if !strings.HasPrefix(result, "[") {
				t.Fatalf("Format() should start with '[' for timestamp, got: %q", result)
			}
			if !strings.Contains(result, "]") {
				t.Fatalf("Format() should contain ']' to close timestamp, got: %q", result)
			}
		})
	})

	// Property 5b: Format preserves field values correctly
	t.Run("format_preserves_field_values", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			log := generateRollbackLog(t)
			result := log.Format()

			// Verify ENV value is preserved
			expectedEnv := "ENV=" + log.Env
			if !strings.Contains(result, expectedEnv) {
				t.Fatalf("Format() = %q, should contain %q", result, expectedEnv)
			}

			// Verify FROM_VERSION value is preserved
			expectedFrom := "FROM_VERSION=" + log.FromVersion
			if !strings.Contains(result, expectedFrom) {
				t.Fatalf("Format() = %q, should contain %q", result, expectedFrom)
			}

			// Verify TO_VERSION value is preserved
			expectedTo := "TO_VERSION=" + log.ToVersion
			if !strings.Contains(result, expectedTo) {
				t.Fatalf("Format() = %q, should contain %q", result, expectedTo)
			}

			// Verify REASON value is preserved (with quotes)
			expectedReason := "REASON=\"" + log.Reason + "\""
			if !strings.Contains(result, expectedReason) {
				t.Fatalf("Format() = %q, should contain %q", result, expectedReason)
			}

			// Verify OPERATOR value is preserved
			expectedOperator := "OPERATOR=" + log.Operator
			if !strings.Contains(result, expectedOperator) {
				t.Fatalf("Format() = %q, should contain %q", result, expectedOperator)
			}
		})
	})

	// Property 5c: Timestamp is in RFC3339 format
	t.Run("timestamp_is_rfc3339_format", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			log := generateRollbackLog(t)
			result := log.Format()

			// Extract timestamp from brackets
			startIdx := strings.Index(result, "[")
			endIdx := strings.Index(result, "]")
			if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
				t.Fatalf("Format() = %q, invalid timestamp brackets", result)
			}

			timestamp := result[startIdx+1 : endIdx]

			// Verify it's a valid RFC3339 timestamp
			_, err := time.Parse(time.RFC3339, timestamp)
			if err != nil {
				t.Fatalf("Format() timestamp %q is not valid RFC3339: %v", timestamp, err)
			}

			// Verify it matches the original timestamp
			expectedTimestamp := log.Timestamp.Format(time.RFC3339)
			if timestamp != expectedTimestamp {
				t.Fatalf("Format() timestamp = %q, expected %q", timestamp, expectedTimestamp)
			}
		})
	})

	// Property 5d: Fields are in correct order
	t.Run("fields_are_in_correct_order", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			log := generateRollbackLog(t)
			result := log.Format()

			// Get indices of each field
			timestampEndIdx := strings.Index(result, "]")
			envIdx := strings.Index(result, "ENV=")
			fromIdx := strings.Index(result, "FROM_VERSION=")
			toIdx := strings.Index(result, "TO_VERSION=")
			reasonIdx := strings.Index(result, "REASON=")
			operatorIdx := strings.Index(result, "OPERATOR=")

			// Verify order: [timestamp] ENV FROM_VERSION TO_VERSION REASON OPERATOR
			if !(timestampEndIdx < envIdx &&
				envIdx < fromIdx &&
				fromIdx < toIdx &&
				toIdx < reasonIdx &&
				reasonIdx < operatorIdx) {
				t.Fatalf("Format() fields are not in correct order: %q", result)
			}
		})
	})

	// Property 5e: REASON field is quoted
	t.Run("reason_field_is_quoted", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			log := generateRollbackLog(t)
			result := log.Format()

			// Find REASON field and verify it's quoted
			reasonIdx := strings.Index(result, "REASON=")
			if reasonIdx == -1 {
				t.Fatalf("Format() = %q, missing REASON field", result)
			}

			// The character after REASON= should be a quote
			afterReason := result[reasonIdx+len("REASON="):]
			if !strings.HasPrefix(afterReason, "\"") {
				t.Fatalf("Format() REASON value should be quoted, got: %q", result)
			}
		})
	})

	// Property 5f: Format is deterministic
	t.Run("format_is_deterministic", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			log := generateRollbackLog(t)

			// Call Format twice
			result1 := log.Format()
			result2 := log.Format()

			// Results should be identical
			if result1 != result2 {
				t.Fatalf("Format() is not deterministic: first = %q, second = %q", result1, result2)
			}
		})
	})

	// Property 5g: Format matches expected regex pattern
	t.Run("format_matches_expected_pattern", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			log := generateRollbackLog(t)
			result := log.Format()

			// Expected pattern: [timestamp] ENV=env FROM_VERSION=from TO_VERSION=to REASON="reason" OPERATOR=operator
			// Note: We use a flexible pattern to match various field values
			pattern := `^\[.+\] ENV=\S+ FROM_VERSION=\S+ TO_VERSION=\S+ REASON=".*" OPERATOR=\S+$`

			matched, err := regexp.MatchString(pattern, result)
			if err != nil {
				t.Fatalf("Regex error: %v", err)
			}
			if !matched {
				t.Fatalf("Format() = %q, does not match expected pattern %q", result, pattern)
			}
		})
	})

	// Property 5h: Environment values are preserved exactly
	t.Run("environment_values_preserved", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate with specific valid environments
			envs := []string{"test", "prod"}
			env := rapid.SampledFrom(envs).Draw(t, "env")

			log := cmd.RollbackLog{
				Timestamp:   generateTimestamp(t),
				Env:         env,
				FromVersion: generateVersion(t),
				ToVersion:   generateVersion(t),
				Reason:      generateReason(t),
				Operator:    generateOperator(t),
			}

			result := log.Format()

			// Verify the exact environment value is in the output
			expectedEnv := "ENV=" + env + " "
			if !strings.Contains(result, expectedEnv) {
				t.Fatalf("Format() = %q, should contain exact environment %q", result, expectedEnv)
			}
		})
	})

	// Property 5i: Version numbers are preserved exactly
	t.Run("version_numbers_preserved", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			fromVersion := generateVersion(t)
			toVersion := generateVersion(t)

			log := cmd.RollbackLog{
				Timestamp:   generateTimestamp(t),
				Env:         "test",
				FromVersion: fromVersion,
				ToVersion:   toVersion,
				Reason:      generateReason(t),
				Operator:    generateOperator(t),
			}

			result := log.Format()

			// Verify exact version values
			if !strings.Contains(result, "FROM_VERSION="+fromVersion) {
				t.Fatalf("Format() = %q, should contain FROM_VERSION=%s", result, fromVersion)
			}
			if !strings.Contains(result, "TO_VERSION="+toVersion) {
				t.Fatalf("Format() = %q, should contain TO_VERSION=%s", result, toVersion)
			}
		})
	})
}

// =============================================================================
// Helper functions for generating test data
// =============================================================================

// generateRollbackLog generates a random RollbackLog for property testing
func generateRollbackLog(t *rapid.T) cmd.RollbackLog {
	return cmd.RollbackLog{
		Timestamp:   generateTimestamp(t),
		Env:         generateEnv(t),
		FromVersion: generateVersion(t),
		ToVersion:   generateVersion(t),
		Reason:      generateReason(t),
		Operator:    generateOperator(t),
	}
}

// generateTimestamp generates a random timestamp
func generateTimestamp(t *rapid.T) time.Time {
	// Generate a timestamp between 2020 and 2030
	year := rapid.IntRange(2020, 2030).Draw(t, "year")
	month := rapid.IntRange(1, 12).Draw(t, "month")
	day := rapid.IntRange(1, 28).Draw(t, "day") // Use 28 to avoid invalid dates
	hour := rapid.IntRange(0, 23).Draw(t, "hour")
	minute := rapid.IntRange(0, 59).Draw(t, "minute")
	second := rapid.IntRange(0, 59).Draw(t, "second")

	return time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC)
}

// generateEnv generates a random environment value
func generateEnv(t *rapid.T) string {
	envs := []string{"test", "prod"}
	return rapid.SampledFrom(envs).Draw(t, "env")
}

// generateVersion generates a random version number
func generateVersion(t *rapid.T) string {
	// Lambda versions are typically numeric strings
	version := rapid.IntRange(1, 1000).Draw(t, "version")
	return fmt.Sprintf("%d", version)
}

// generateReason generates a random reason string
func generateReason(t *rapid.T) string {
	// Generate reasons that don't contain quotes to avoid escaping issues
	reasons := []string{
		"未指定原因",
		"bug found",
		"performance issue",
		"emergency rollback",
		"deployment failed",
		"customer reported issue",
		"memory leak detected",
		"high error rate",
		"timeout issues",
		"security vulnerability",
	}
	return rapid.SampledFrom(reasons).Draw(t, "reason")
}

// generateOperator generates a random operator name
func generateOperator(t *rapid.T) string {
	// Generate operator names that are valid (no spaces, alphanumeric with some special chars)
	operators := []string{
		"admin",
		"ops-team",
		"developer",
		"user1",
		"unknown",
		"system",
		"ci-cd",
		"root",
		"deploy-bot",
	}
	return rapid.SampledFrom(operators).Draw(t, "operator")
}
