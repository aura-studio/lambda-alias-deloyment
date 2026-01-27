package aws_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aura-studio/lambda-alias-deployment/internal/aws"
	"github.com/aura-studio/lambda-alias-deployment/internal/exitcode"
	"pgregory.net/rapid"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		// nil error
		{
			name:     "nil error returns Success",
			err:      nil,
			expected: exitcode.Success,
		},
		// Network errors
		{
			name:     "unable to locate credentials",
			err:      errors.New("unable to locate credentials"),
			expected: exitcode.NetworkError,
		},
		{
			name:     "could not connect",
			err:      errors.New("could not connect to endpoint"),
			expected: exitcode.NetworkError,
		},
		{
			name:     "connection refused",
			err:      errors.New("connection refused"),
			expected: exitcode.NetworkError,
		},
		{
			name:     "network error",
			err:      errors.New("network error occurred"),
			expected: exitcode.NetworkError,
		},
		{
			name:     "timeout",
			err:      errors.New("request timeout"),
			expected: exitcode.NetworkError,
		},
		{
			name:     "timed out",
			err:      errors.New("operation timed out"),
			expected: exitcode.NetworkError,
		},
		{
			name:     "unreachable",
			err:      errors.New("host unreachable"),
			expected: exitcode.NetworkError,
		},
		// Resource not found errors
		{
			name:     "ResourceNotFoundException",
			err:      errors.New("ResourceNotFoundException: Function not found"),
			expected: exitcode.ResourceNotFound,
		},
		{
			name:     "does not exist",
			err:      errors.New("The function does not exist"),
			expected: exitcode.ResourceNotFound,
		},
		{
			name:     "not found",
			err:      errors.New("Alias not found"),
			expected: exitcode.ResourceNotFound,
		},
		{
			name:     "cannot find",
			err:      errors.New("cannot find the specified resource"),
			expected: exitcode.ResourceNotFound,
		},
		// Other AWS errors
		{
			name:     "generic AWS error",
			err:      errors.New("AccessDeniedException: User is not authorized"),
			expected: exitcode.AWSError,
		},
		{
			name:     "validation error",
			err:      errors.New("ValidationException: Invalid parameter"),
			expected: exitcode.AWSError,
		},
		// Case insensitivity tests
		{
			name:     "NETWORK uppercase",
			err:      errors.New("NETWORK ERROR"),
			expected: exitcode.NetworkError,
		},
		{
			name:     "NOT FOUND uppercase",
			err:      errors.New("Resource NOT FOUND"),
			expected: exitcode.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := aws.ClassifyError(tt.err)
			if result != tt.expected {
				t.Errorf("ClassifyError(%v) = %d, want %d", tt.err, result, tt.expected)
			}
		})
	}
}

// **Validates: Requirements 12.1**
// Property 9: 错误分类和退出码
// For any AWS API 错误，应根据错误消息中的关键词正确分类：
// 网络相关关键词返回退出码 4，资源不存在关键词返回退出码 3，其他返回退出码 2。
func TestProperty_ErrorClassificationAndExitCode(t *testing.T) {
	// Network error keywords (exit code 4)
	networkKeywords := []string{
		"unable to locate credentials",
		"could not connect",
		"connection refused",
		"network",
		"timeout",
		"timed out",
		"unreachable",
	}

	// Resource not found keywords (exit code 3)
	resourceNotFoundKeywords := []string{
		"resourcenotfoundexception",
		"does not exist",
		"not found",
		"cannot find",
	}

	// Property 9.1: Network error keywords should return exit code 4
	t.Run("NetworkErrorKeywords", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate a random network keyword
			keywordIndex := rapid.IntRange(0, len(networkKeywords)-1).Draw(t, "keywordIndex")
			keyword := networkKeywords[keywordIndex]

			// Generate random prefix and suffix
			prefix := rapid.StringMatching(`[a-zA-Z0-9 ]{0,20}`).Draw(t, "prefix")
			suffix := rapid.StringMatching(`[a-zA-Z0-9 ]{0,20}`).Draw(t, "suffix")

			// Create error message with the keyword
			errMsg := fmt.Sprintf("%s%s%s", prefix, keyword, suffix)
			err := errors.New(errMsg)

			result := aws.ClassifyError(err)
			if result != exitcode.NetworkError {
				t.Fatalf("ClassifyError(%q) = %d, want %d (NetworkError)", errMsg, result, exitcode.NetworkError)
			}
		})
	})

	// Property 9.2: Resource not found keywords should return exit code 3
	t.Run("ResourceNotFoundKeywords", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate a random resource not found keyword
			keywordIndex := rapid.IntRange(0, len(resourceNotFoundKeywords)-1).Draw(t, "keywordIndex")
			keyword := resourceNotFoundKeywords[keywordIndex]

			// Generate random prefix and suffix (avoiding network keywords)
			prefix := rapid.StringMatching(`[a-zA-Z0-9]{0,10}`).Draw(t, "prefix")
			suffix := rapid.StringMatching(`[a-zA-Z0-9]{0,10}`).Draw(t, "suffix")

			// Create error message with the keyword
			errMsg := fmt.Sprintf("%s%s%s", prefix, keyword, suffix)
			err := errors.New(errMsg)

			result := aws.ClassifyError(err)
			if result != exitcode.ResourceNotFound {
				t.Fatalf("ClassifyError(%q) = %d, want %d (ResourceNotFound)", errMsg, result, exitcode.ResourceNotFound)
			}
		})
	})

	// Property 9.3: Other errors (without network or resource not found keywords) should return exit code 2
	t.Run("OtherAWSErrors", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate error messages that don't contain any of the special keywords
			// Use a limited character set to avoid accidentally generating keywords
			errMsg := rapid.StringMatching(`(AccessDenied|ValidationError|InvalidParameter|ThrottlingException|ServiceException)[a-zA-Z0-9]{0,20}`).Draw(t, "errMsg")

			// Ensure the message doesn't contain any network or resource not found keywords
			err := errors.New(errMsg)

			result := aws.ClassifyError(err)
			if result != exitcode.AWSError {
				t.Fatalf("ClassifyError(%q) = %d, want %d (AWSError)", errMsg, result, exitcode.AWSError)
			}
		})
	})

	// Property 9.4: Case insensitivity - keywords should match regardless of case
	t.Run("CaseInsensitivity", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Test network keywords with random case
			keywordIndex := rapid.IntRange(0, len(networkKeywords)-1).Draw(t, "keywordIndex")
			keyword := networkKeywords[keywordIndex]

			// Randomly change case of each character
			var modifiedKeyword []byte
			for _, c := range []byte(keyword) {
				if rapid.Bool().Draw(t, "uppercase") {
					if c >= 'a' && c <= 'z' {
						c = c - 32 // Convert to uppercase
					}
				}
				modifiedKeyword = append(modifiedKeyword, c)
			}

			errMsg := string(modifiedKeyword)
			err := errors.New(errMsg)

			result := aws.ClassifyError(err)
			if result != exitcode.NetworkError {
				t.Fatalf("ClassifyError(%q) = %d, want %d (NetworkError) - case insensitivity failed", errMsg, result, exitcode.NetworkError)
			}
		})
	})

	// Property 9.5: nil error should return exit code 0 (Success)
	t.Run("NilError", func(t *testing.T) {
		result := aws.ClassifyError(nil)
		if result != exitcode.Success {
			t.Fatalf("ClassifyError(nil) = %d, want %d (Success)", result, exitcode.Success)
		}
	})

	// Property 9.6: Network keywords take precedence over resource not found keywords
	t.Run("NetworkPrecedence", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate a message containing both network and resource not found keywords
			networkKeywordIndex := rapid.IntRange(0, len(networkKeywords)-1).Draw(t, "networkKeywordIndex")
			resourceKeywordIndex := rapid.IntRange(0, len(resourceNotFoundKeywords)-1).Draw(t, "resourceKeywordIndex")

			networkKeyword := networkKeywords[networkKeywordIndex]
			resourceKeyword := resourceNotFoundKeywords[resourceKeywordIndex]

			// Create error message with both keywords (network first)
			errMsg := fmt.Sprintf("%s and %s", networkKeyword, resourceKeyword)
			err := errors.New(errMsg)

			result := aws.ClassifyError(err)
			// Network keywords are checked first, so should return NetworkError
			if result != exitcode.NetworkError {
				t.Fatalf("ClassifyError(%q) = %d, want %d (NetworkError) - network should take precedence", errMsg, result, exitcode.NetworkError)
			}
		})
	})
}
