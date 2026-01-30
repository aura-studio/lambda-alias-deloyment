package patcher_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aura-studio/lad/internal/patcher"
	"pgregory.net/rapid"
)

// Test ValidateTemplate
func TestValidateTemplate(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "patcher_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name        string
		content     string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid_SAM_template",
			content: `AWSTemplateFormatVersion: '2010-09-09'
Transform: AWS::Serverless-2016-10-31
Resources:
  Function:
    Type: AWS::Serverless::Function
`,
			expectError: false,
		},
		{
			name: "missing_Serverless",
			content: `AWSTemplateFormatVersion: '2010-09-09'
Resources:
  Function:
    Type: AWS::Lambda::Function
`,
			expectError: true,
			errorMsg:    "不是有效的 SAM 模板文件",
		},
		{
			name: "missing_Resources_section",
			content: `AWSTemplateFormatVersion: '2010-09-09'
Transform: AWS::Serverless-2016-10-31
Parameters:
  Env:
    Type: String
`,
			expectError: true,
			errorMsg:    "模板文件缺少 Resources 部分",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			filePath := filepath.Join(tmpDir, tt.name+".yaml")
			if err := os.WriteFile(filePath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			err := patcher.ValidateTemplate(filePath)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}

	// Test non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		err := patcher.ValidateTemplate(filepath.Join(tmpDir, "non-existent.yaml"))
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
		if !strings.Contains(err.Error(), "文件不存在") {
			t.Errorf("Expected error about file not existing, got: %v", err)
		}
	})
}

// Test HasPatchMarker
func TestHasPatchMarker(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "has patch marker",
			content:  "some content\n" + patcher.PatchStartMarker + "\nmore content",
			expected: true,
		},
		{
			name:     "no patch marker",
			content:  "some content\nno marker here",
			expected: false,
		},
		{
			name:     "empty content",
			content:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := patcher.HasPatchMarker(tt.content)
			if result != tt.expected {
				t.Errorf("HasPatchMarker() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test HasAliasResources
func TestHasAliasResources(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name: "has LiveAlias resource",
			content: `Resources:
  LiveAlias:
    Type: AWS::Lambda::Alias
`,
			expected: true,
		},
		{
			name: "has Lambda Alias type",
			content: `Resources:
  MyAlias:
    Type: AWS::Lambda::Alias
`,
			expected: true,
		},
		{
			name: "has Lambda Version type",
			content: `Resources:
  MyVersion:
    Type: AWS::Lambda::Version
`,
			expected: true,
		},
		{
			name: "no alias resources",
			content: `Resources:
  Function:
    Type: AWS::Serverless::Function
`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := patcher.HasAliasResources(tt.content)
			if result != tt.expected {
				t.Errorf("HasAliasResources() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test GetExistingAliasResources
func TestGetExistingAliasResources(t *testing.T) {
	content := `Resources:
  FunctionVersion:
    Type: AWS::Lambda::Version
    Properties:
      FunctionName: !Ref Function

  LiveAlias:
    Type: AWS::Lambda::Alias
    Properties:
      Name: live

  PreviousAlias:
    Type: AWS::Lambda::Alias
    Properties:
      Name: previous
`
	resources := patcher.GetExistingAliasResources(content)

	expected := []string{"FunctionVersion", "LiveAlias", "PreviousAlias"}
	if len(resources) != len(expected) {
		t.Errorf("Expected %d resources, got %d", len(expected), len(resources))
	}

	for _, exp := range expected {
		found := false
		for _, res := range resources {
			if res == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected resource %q not found in %v", exp, resources)
		}
	}
}

// Test CheckFunctionExists
func TestCheckFunctionExists(t *testing.T) {
	content := `Resources:
  Function:
    Type: AWS::Serverless::Function
    Properties:
      Handler: main

  MyFunction:
    Type: AWS::Lambda::Function
    Properties:
      Handler: main
`
	tests := []struct {
		name         string
		functionName string
		expected     bool
	}{
		{
			name:         "Function exists (Serverless)",
			functionName: "Function",
			expected:     true,
		},
		{
			name:         "MyFunction exists (Lambda)",
			functionName: "MyFunction",
			expected:     true,
		},
		{
			name:         "NonExistent function",
			functionName: "NonExistent",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := patcher.CheckFunctionExists(content, tt.functionName)
			if result != tt.expected {
				t.Errorf("CheckFunctionExists(%q) = %v, want %v", tt.functionName, result, tt.expected)
			}
		})
	}
}

// Test CheckDescriptionParam
func TestCheckDescriptionParam(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name: "has Description param",
			content: `Parameters:
  Description:
    Type: String
`,
			expected: true,
		},
		{
			name: "no Description param",
			content: `Parameters:
  Env:
    Type: String
`,
			expected: false,
		},
		{
			name:     "no Parameters section",
			content:  "Resources:\n  Function:\n",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := patcher.CheckDescriptionParam(tt.content)
			if result != tt.expected {
				t.Errorf("CheckDescriptionParam() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test DetectHttpApis
func TestDetectHttpApis(t *testing.T) {
	content := `Resources:
  MyHttpApi:
    Type: AWS::Serverless::HttpApi
    Properties:
      StageName: prod

  AnotherApi:
    Type: AWS::Serverless::HttpApi
    Properties:
      StageName: dev

  Function:
    Type: AWS::Serverless::Function
`
	apis := patcher.DetectHttpApis(content)

	expected := []string{"MyHttpApi", "AnotherApi"}
	if len(apis) != len(expected) {
		t.Errorf("Expected %d APIs, got %d: %v", len(expected), len(apis), apis)
	}

	for _, exp := range expected {
		found := false
		for _, api := range apis {
			if api == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected API %q not found in %v", exp, apis)
		}
	}
}

// Test DetectSchedules
func TestDetectSchedules(t *testing.T) {
	content := `Resources:
  DailySchedule:
    Type: AWS::Scheduler::Schedule
    Properties:
      ScheduleExpression: rate(1 day)

  HourlySchedule:
    Type: AWS::Scheduler::Schedule
    Properties:
      ScheduleExpression: rate(1 hour)

  Function:
    Type: AWS::Serverless::Function
`
	schedules := patcher.DetectSchedules(content)

	expected := []string{"DailySchedule", "HourlySchedule"}
	if len(schedules) != len(expected) {
		t.Errorf("Expected %d schedules, got %d: %v", len(expected), len(schedules), schedules)
	}

	for _, exp := range expected {
		found := false
		for _, sch := range schedules {
			if sch == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected schedule %q not found in %v", exp, schedules)
		}
	}
}

// Test GeneratePatchContent
func TestGeneratePatchContent(t *testing.T) {
	content := patcher.GeneratePatchContent("MyFunction")

	// Check that it contains the start marker
	if !strings.Contains(content, patcher.PatchStartMarker) {
		t.Error("Generated content should contain start marker")
	}

	// Check that it contains the function version resource
	if !strings.Contains(content, "MyFunctionVersion:") {
		t.Error("Generated content should contain function version resource")
	}

	// Check that it contains all three aliases
	if !strings.Contains(content, "LiveAlias:") {
		t.Error("Generated content should contain LiveAlias")
	}
	if !strings.Contains(content, "PreviousAlias:") {
		t.Error("Generated content should contain PreviousAlias")
	}
	if !strings.Contains(content, "LatestAlias:") {
		t.Error("Generated content should contain LatestAlias")
	}

	// Check that it references the function correctly
	if !strings.Contains(content, "!Ref MyFunction") {
		t.Error("Generated content should reference the function")
	}
	if !strings.Contains(content, "!GetAtt MyFunctionVersion.Version") {
		t.Error("Generated content should reference the function version")
	}
}

// Test GenerateDescriptionParam
func TestGenerateDescriptionParam(t *testing.T) {
	content := patcher.GenerateDescriptionParam()

	if !strings.Contains(content, "Description:") {
		t.Error("Generated content should contain Description parameter")
	}
	if !strings.Contains(content, "Type: String") {
		t.Error("Generated content should have Type: String")
	}
}

// Test GenerateHttpApiPatch
func TestGenerateHttpApiPatch(t *testing.T) {
	content := patcher.GenerateHttpApiPatch("Function", "MyHttpApi")

	// Check for permission resource
	if !strings.Contains(content, "LiveAliasHttpApiPermission:") {
		t.Error("Generated content should contain permission resource")
	}

	// Check for route resource
	if !strings.Contains(content, "HttpApiLiveRoute:") {
		t.Error("Generated content should contain route resource")
	}

	// Check for integration resource
	if !strings.Contains(content, "HttpApiLiveIntegration:") {
		t.Error("Generated content should contain integration resource")
	}

	// Check that it references the API correctly
	if !strings.Contains(content, "MyHttpApi") {
		t.Error("Generated content should reference the API name")
	}
}

// Test BackupFile
func TestBackupFile(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "backup_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "template.yaml")
	testContent := "test content"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Backup the file
	backupPath, err := patcher.BackupFile(testFile)
	if err != nil {
		t.Fatalf("BackupFile failed: %v", err)
	}

	// Check that backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("Backup file should exist")
	}

	// Check that backup file has correct content
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}
	if string(backupContent) != testContent {
		t.Errorf("Backup content = %q, want %q", string(backupContent), testContent)
	}

	// Check backup file name format
	if !strings.HasPrefix(backupPath, testFile+".bak.") {
		t.Errorf("Backup path should have format {path}.bak.{timestamp}, got %q", backupPath)
	}
}

// =============================================================================
// Property 6: 模板验证
// **Validates: Requirements 10.2**
//
// *For any* 模板文件，验证应检查：文件存在、包含 AWS::Serverless、包含 Resources 部分。
// 不满足任一条件应返回参数错误。
// =============================================================================

// TestProperty6_TemplateValidation tests Property 6: Template Validation
// **Validates: Requirements 10.2**
func TestProperty6_TemplateValidation(t *testing.T) {
	// Property 6a: Valid SAM templates should always pass validation
	t.Run("valid_SAM_templates_pass_validation", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			tmpDir := t.TempDir()

			// Generate random but valid SAM template content
			// Must contain AWS::Serverless and Resources section
			functionName := rapid.StringMatching(`[A-Z][a-zA-Z0-9]{2,15}`).Draw(rt, "functionName")
			handler := rapid.StringMatching(`[a-z][a-zA-Z0-9_]{2,20}`).Draw(rt, "handler")
			runtime := rapid.SampledFrom([]string{
				"python3.9", "python3.10", "python3.11",
				"nodejs18.x", "nodejs20.x",
				"go1.x", "provided.al2",
			}).Draw(rt, "runtime")

			content := generateValidSAMTemplate(functionName, handler, runtime)

			// Write to temp file
			filePath := filepath.Join(tmpDir, "template.yaml")
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				rt.Fatalf("Failed to write test file: %v", err)
			}

			// Validate should pass
			err := patcher.ValidateTemplate(filePath)
			if err != nil {
				rt.Errorf("Valid SAM template should pass validation, got error: %v", err)
			}
		})
	})

	// Property 6b: Non-existent files should always fail validation
	t.Run("non_existent_files_fail_validation", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			tmpDir := t.TempDir()

			// Generate random file name that doesn't exist
			fileName := rapid.StringMatching(`[a-z][a-z0-9_-]{2,20}\.yaml`).Draw(rt, "fileName")
			filePath := filepath.Join(tmpDir, fileName)

			// Validate should fail with "文件不存在" error
			err := patcher.ValidateTemplate(filePath)
			if err == nil {
				rt.Error("Non-existent file should fail validation")
			}
			if !strings.Contains(err.Error(), "文件不存在") {
				rt.Errorf("Error should mention file not existing, got: %v", err)
			}
		})
	})

	// Property 6c: Templates without AWS::Serverless should fail validation
	t.Run("templates_without_serverless_fail_validation", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			tmpDir := t.TempDir()

			// Generate template content WITHOUT AWS::Serverless
			functionName := rapid.StringMatching(`[A-Z][a-zA-Z0-9]{2,15}`).Draw(rt, "functionName")
			handler := rapid.StringMatching(`[a-z][a-zA-Z0-9_]{2,20}`).Draw(rt, "handler")

			// Use AWS::Lambda::Function instead of AWS::Serverless::Function
			content := generateNonServerlessTemplate(functionName, handler)

			// Write to temp file
			filePath := filepath.Join(tmpDir, "template.yaml")
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				rt.Fatalf("Failed to write test file: %v", err)
			}

			// Validate should fail with "不是有效的 SAM 模板文件" error
			err := patcher.ValidateTemplate(filePath)
			if err == nil {
				rt.Error("Template without AWS::Serverless should fail validation")
			}
			if !strings.Contains(err.Error(), "不是有效的 SAM 模板文件") {
				rt.Errorf("Error should mention invalid SAM template, got: %v", err)
			}
		})
	})

	// Property 6d: Templates without Resources section should fail validation
	t.Run("templates_without_resources_fail_validation", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			tmpDir := t.TempDir()

			// Generate template content WITHOUT Resources section
			description := rapid.StringMatching(`[A-Za-z][A-Za-z0-9 ]{5,30}`).Draw(rt, "description")
			paramName := rapid.StringMatching(`[A-Z][a-zA-Z0-9]{2,15}`).Draw(rt, "paramName")

			content := generateTemplateWithoutResources(description, paramName)

			// Write to temp file
			filePath := filepath.Join(tmpDir, "template.yaml")
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				rt.Fatalf("Failed to write test file: %v", err)
			}

			// Validate should fail with "模板文件缺少 Resources 部分" error
			err := patcher.ValidateTemplate(filePath)
			if err == nil {
				rt.Error("Template without Resources section should fail validation")
			}
			if !strings.Contains(err.Error(), "模板文件缺少 Resources 部分") {
				rt.Errorf("Error should mention missing Resources section, got: %v", err)
			}
		})
	})

	// Property 6e: Validation is deterministic - same file always produces same result
	t.Run("validation_is_deterministic", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			tmpDir := t.TempDir()

			// Generate random template (valid or invalid)
			isValid := rapid.Bool().Draw(rt, "isValid")
			hasServerless := rapid.Bool().Draw(rt, "hasServerless")
			hasResources := rapid.Bool().Draw(rt, "hasResources")

			var content string
			if isValid {
				functionName := rapid.StringMatching(`[A-Z][a-zA-Z0-9]{2,15}`).Draw(rt, "functionName")
				content = generateValidSAMTemplate(functionName, "handler", "python3.9")
			} else if !hasServerless {
				functionName := rapid.StringMatching(`[A-Z][a-zA-Z0-9]{2,15}`).Draw(rt, "functionName")
				content = generateNonServerlessTemplate(functionName, "handler")
			} else if !hasResources {
				content = generateTemplateWithoutResources("Test", "Param")
			} else {
				functionName := rapid.StringMatching(`[A-Z][a-zA-Z0-9]{2,15}`).Draw(rt, "functionName")
				content = generateValidSAMTemplate(functionName, "handler", "python3.9")
			}

			// Write to temp file
			filePath := filepath.Join(tmpDir, "template.yaml")
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				rt.Fatalf("Failed to write test file: %v", err)
			}

			// Run validation multiple times
			result1 := patcher.ValidateTemplate(filePath)
			result2 := patcher.ValidateTemplate(filePath)
			result3 := patcher.ValidateTemplate(filePath)

			// All results should be the same
			if (result1 == nil) != (result2 == nil) || (result2 == nil) != (result3 == nil) {
				rt.Error("Validation should be deterministic - same file should always produce same result")
			}

			// If there's an error, the error message should be the same
			if result1 != nil && result2 != nil && result3 != nil {
				if result1.Error() != result2.Error() || result2.Error() != result3.Error() {
					rt.Error("Error messages should be consistent for the same file")
				}
			}
		})
	})
}

// Helper functions for generating test templates

func generateValidSAMTemplate(functionName, handler, runtime string) string {
	return `AWSTemplateFormatVersion: '2010-09-09'
Transform: AWS::Serverless-2016-10-31
Description: Test SAM Application

Resources:
  ` + functionName + `:
    Type: AWS::Serverless::Function
    Properties:
      Handler: ` + handler + `
      Runtime: ` + runtime + `
      CodeUri: ./
`
}

func generateNonServerlessTemplate(functionName, handler string) string {
	return `AWSTemplateFormatVersion: '2010-09-09'
Description: Test CloudFormation Template (not SAM)

Resources:
  ` + functionName + `:
    Type: AWS::Lambda::Function
    Properties:
      Handler: ` + handler + `
      Runtime: python3.9
      Code:
        S3Bucket: my-bucket
        S3Key: code.zip
      Role: !GetAtt LambdaRole.Arn

  LambdaRole:
    Type: AWS::IAM::Role
    Properties:
      AssumeRolePolicyDocument:
        Version: '2012-10-17'
        Statement:
          - Effect: Allow
            Principal:
              Service: lambda.amazonaws.com
            Action: sts:AssumeRole
`
}

func generateTemplateWithoutResources(description, paramName string) string {
	return `AWSTemplateFormatVersion: '2010-09-09'
Transform: AWS::Serverless-2016-10-31
Description: ` + description + `

Parameters:
  ` + paramName + `:
    Type: String
    Default: default-value

Globals:
  Function:
    Timeout: 30
`
}

// =============================================================================
// Property 7: 补丁内容生成
// **Validates: Requirements 10.8**
//
// *For any* 有效的函数资源名称，生成的补丁内容应包含 Lambda Version 资源和三个 Alias 资源
// （live、previous、latest），且使用正确的补丁标记包裹。
// =============================================================================

// TestProperty7_PatchContentGeneration tests Property 7: Patch Content Generation
// **Validates: Requirements 10.8**
func TestProperty7_PatchContentGeneration(t *testing.T) {
	// Property 7a: Generated patch content always contains start marker
	t.Run("patch_content_contains_start_marker", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			// Generate valid function resource name (must start with letter, alphanumeric)
			functionName := rapid.StringMatching(`[A-Z][a-zA-Z0-9]{1,30}`).Draw(rt, "functionName")

			content := patcher.GeneratePatchContent(functionName)

			if !strings.Contains(content, patcher.PatchStartMarker) {
				rt.Errorf("Generated patch content should contain start marker %q, got:\n%s", patcher.PatchStartMarker, content)
			}
		})
	})

	// Property 7b: Generated patch content always contains Lambda Version resource
	t.Run("patch_content_contains_version_resource", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			functionName := rapid.StringMatching(`[A-Z][a-zA-Z0-9]{1,30}`).Draw(rt, "functionName")

			content := patcher.GeneratePatchContent(functionName)

			// Check for Version resource with correct name
			expectedVersionName := functionName + "Version:"
			if !strings.Contains(content, expectedVersionName) {
				rt.Errorf("Generated patch content should contain version resource %q, got:\n%s", expectedVersionName, content)
			}

			// Check for AWS::Lambda::Version type
			if !strings.Contains(content, "Type: AWS::Lambda::Version") {
				rt.Errorf("Generated patch content should contain AWS::Lambda::Version type, got:\n%s", content)
			}

			// Check that version references the function
			expectedFunctionRef := "FunctionName: !Ref " + functionName
			if !strings.Contains(content, expectedFunctionRef) {
				rt.Errorf("Generated patch content should reference function %q, got:\n%s", expectedFunctionRef, content)
			}
		})
	})

	// Property 7c: Generated patch content always contains all three aliases
	t.Run("patch_content_contains_all_aliases", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			functionName := rapid.StringMatching(`[A-Z][a-zA-Z0-9]{1,30}`).Draw(rt, "functionName")

			content := patcher.GeneratePatchContent(functionName)

			// Check for all three aliases
			aliases := []string{"LiveAlias:", "PreviousAlias:", "LatestAlias:"}
			for _, alias := range aliases {
				if !strings.Contains(content, alias) {
					rt.Errorf("Generated patch content should contain %s, got:\n%s", alias, content)
				}
			}

			// Check for alias names
			aliasNames := []string{"Name: live", "Name: previous", "Name: latest"}
			for _, name := range aliasNames {
				if !strings.Contains(content, name) {
					rt.Errorf("Generated patch content should contain %q, got:\n%s", name, content)
				}
			}
		})
	})

	// Property 7d: Generated content is deterministic for same function name
	t.Run("patch_generation_is_deterministic", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			functionName := rapid.StringMatching(`[A-Z][a-zA-Z0-9]{1,30}`).Draw(rt, "functionName")

			// Generate content multiple times
			content1 := patcher.GeneratePatchContent(functionName)
			content2 := patcher.GeneratePatchContent(functionName)
			content3 := patcher.GeneratePatchContent(functionName)

			// All should be identical
			if content1 != content2 || content2 != content3 {
				rt.Error("GeneratePatchContent should be deterministic - same input should produce same output")
			}
		})
	})

	// Property 7e: Generated content contains exactly 4 resources (1 version + 3 aliases)
	t.Run("patch_content_contains_exactly_four_resources", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			functionName := rapid.StringMatching(`[A-Z][a-zA-Z0-9]{1,30}`).Draw(rt, "functionName")

			content := patcher.GeneratePatchContent(functionName)

			// Count AWS::Lambda::Version occurrences
			versionCount := strings.Count(content, "Type: AWS::Lambda::Version")
			if versionCount != 1 {
				rt.Errorf("Expected exactly 1 AWS::Lambda::Version, got %d", versionCount)
			}

			// Count AWS::Lambda::Alias occurrences
			aliasCount := strings.Count(content, "Type: AWS::Lambda::Alias")
			if aliasCount != 3 {
				rt.Errorf("Expected exactly 3 AWS::Lambda::Alias, got %d", aliasCount)
			}
		})
	})
}
