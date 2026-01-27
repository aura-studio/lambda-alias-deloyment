package patcher

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Test RemovePatchMarkerContent
func TestRemovePatchMarkerContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "removes content between markers",
			content: `Resources:
  Function:
    Type: AWS::Serverless::Function

` + PatchStartMarker + `
  FunctionVersion:
    Type: AWS::Lambda::Version
` + PatchEndMarker + `
`,
			expected: `Resources:
  Function:
    Type: AWS::Serverless::Function

`,
		},
		{
			name:     "no markers returns unchanged",
			content:  "some content without markers",
			expected: "some content without markers",
		},
		{
			name: "only start marker returns unchanged",
			content: `content
` + PatchStartMarker + `
more content`,
			expected: `content
` + PatchStartMarker + `
more content`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RemovePatchMarkerContent(tt.content)
			if result != tt.expected {
				t.Errorf("RemovePatchMarkerContent() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// Test RemoveAliasResources
func TestRemoveAliasResources(t *testing.T) {
	content := `Resources:
  Function:
    Type: AWS::Serverless::Function
    Properties:
      Handler: main

  FunctionVersion:
    Type: AWS::Lambda::Version
    Properties:
      FunctionName: !Ref Function

  LiveAlias:
    Type: AWS::Lambda::Alias
    Properties:
      Name: live
`
	resources := []string{"FunctionVersion", "LiveAlias"}
	result := RemoveAliasResources(content, resources)

	if strings.Contains(result, "FunctionVersion:") {
		t.Error("Should remove FunctionVersion resource")
	}
	if strings.Contains(result, "LiveAlias:") {
		t.Error("Should remove LiveAlias resource")
	}
	if !strings.Contains(result, "Function:") {
		t.Error("Should keep Function resource")
	}
}

// =============================================================================
// Property 8: 移除补丁标记内容
// **Validates: Requirements 11.3**
//
// *For any* 包含补丁标记的模板内容，移除操作应删除标记之间的所有内容（包括标记本身），
// 保留其他内容不变。
// =============================================================================

// TestProperty8_RemovePatchMarkerContent tests Property 8
// **Validates: Requirements 11.3**
func TestProperty8_RemovePatchMarkerContent(t *testing.T) {
	// Property 8a: Content before marker is preserved
	t.Run("content_before_marker_preserved", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			beforeContent := rapid.StringMatching(`[a-zA-Z0-9\s:_-]{10,100}`).Draw(rt, "beforeContent")
			patchContent := rapid.StringMatching(`[a-zA-Z0-9\s:_-]{10,50}`).Draw(rt, "patchContent")

			fullContent := beforeContent + "\n" + PatchStartMarker + "\n" + patchContent + "\n" + PatchEndMarker + "\n"

			result := RemovePatchMarkerContent(fullContent)

			if !strings.HasPrefix(result, beforeContent) {
				rt.Errorf("Content before marker should be preserved, got: %q", result)
			}
		})
	})

	// Property 8b: Content after marker is preserved
	t.Run("content_after_marker_preserved", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			beforeContent := rapid.StringMatching(`[a-zA-Z0-9\s:_-]{10,50}`).Draw(rt, "beforeContent")
			patchContent := rapid.StringMatching(`[a-zA-Z0-9\s:_-]{10,50}`).Draw(rt, "patchContent")
			afterContent := rapid.StringMatching(`[a-zA-Z0-9\s:_-]{10,100}`).Draw(rt, "afterContent")

			fullContent := beforeContent + "\n" + PatchStartMarker + "\n" + patchContent + "\n" + PatchEndMarker + "\n" + afterContent

			result := RemovePatchMarkerContent(fullContent)

			if !strings.HasSuffix(result, afterContent) {
				rt.Errorf("Content after marker should be preserved, got: %q", result)
			}
		})
	})

	// Property 8c: Patch markers are removed
	t.Run("patch_markers_removed", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			beforeContent := rapid.StringMatching(`[a-zA-Z0-9\s:_-]{10,50}`).Draw(rt, "beforeContent")
			patchContent := rapid.StringMatching(`[a-zA-Z0-9\s:_-]{10,50}`).Draw(rt, "patchContent")

			fullContent := beforeContent + "\n" + PatchStartMarker + "\n" + patchContent + "\n" + PatchEndMarker + "\n"

			result := RemovePatchMarkerContent(fullContent)

			if strings.Contains(result, PatchStartMarker) {
				rt.Error("Start marker should be removed")
			}
			if strings.Contains(result, PatchEndMarker) {
				rt.Error("End marker should be removed")
			}
		})
	})

	// Property 8d: Patch content is removed
	t.Run("patch_content_removed", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			beforeContent := "before"
			// Generate unique patch content that won't appear elsewhere
			uniqueId := rapid.StringMatching(`UNIQUE_[A-Z0-9]{10}`).Draw(rt, "uniqueId")
			patchContent := "patch_" + uniqueId

			fullContent := beforeContent + "\n" + PatchStartMarker + "\n" + patchContent + "\n" + PatchEndMarker + "\n"

			result := RemovePatchMarkerContent(fullContent)

			if strings.Contains(result, patchContent) {
				rt.Errorf("Patch content should be removed, but found %q in result", patchContent)
			}
		})
	})

	// Property 8e: No markers means no change
	t.Run("no_markers_no_change", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			content := rapid.StringMatching(`[a-zA-Z0-9\s:_-]{20,200}`).Draw(rt, "content")

			// Ensure no markers in content
			if strings.Contains(content, "DEPLOY_SCRIPT") {
				rt.Skip("Skipping - content accidentally contains marker-like text")
			}

			result := RemovePatchMarkerContent(content)

			if result != content {
				rt.Errorf("Content without markers should be unchanged, got: %q", result)
			}
		})
	})

	// Property 8f: Operation is idempotent
	t.Run("operation_is_idempotent", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			beforeContent := rapid.StringMatching(`[a-zA-Z0-9\s:_-]{10,50}`).Draw(rt, "beforeContent")
			patchContent := rapid.StringMatching(`[a-zA-Z0-9\s:_-]{10,50}`).Draw(rt, "patchContent")

			fullContent := beforeContent + "\n" + PatchStartMarker + "\n" + patchContent + "\n" + PatchEndMarker + "\n"

			result1 := RemovePatchMarkerContent(fullContent)
			result2 := RemovePatchMarkerContent(result1)

			if result1 != result2 {
				rt.Error("RemovePatchMarkerContent should be idempotent")
			}
		})
	})
}
