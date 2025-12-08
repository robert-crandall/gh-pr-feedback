package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robert-crandall/gh-pr-feedback/internal/model"
)

func TestPaginateAggregatesPages(t *testing.T) {
	chunk := make([]int, pageSize)
	for i := range chunk {
		chunk[i] = i
	}

	calls := 0
	items, err := paginate(func(page int) ([]int, error) {
		calls++
		switch page {
		case 1:
			return chunk, nil
		case 2:
			return []int{1, 2}, nil
		default:
			t.Fatalf("unexpected page %d", page)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != len(chunk)+2 {
		t.Fatalf("expected %d items, got %d", len(chunk)+2, len(items))
	}
	if calls != 2 {
		t.Fatalf("expected 2 pages, got %d", calls)
	}
}

func TestPaginateStopsOnEmpty(t *testing.T) {
	calls := 0
	items, err := paginate(func(page int) ([]int, error) {
		calls++
		switch page {
		case 1:
			return make([]int, pageSize), nil
		case 2:
			return []int{}, nil
		default:
			t.Fatalf("unexpected page %d", page)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != pageSize {
		t.Fatalf("expected %d items, got %d", pageSize, len(items))
	}
	if calls != 2 {
		t.Fatalf("expected to stop after encountering empty page, got %d calls", calls)
	}
}

func TestRunCopilotWritesPromptToCommand(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "copilot.sh")
	outputPath := filepath.Join(dir, "output.txt")

	script := "#!/bin/sh\ncat > \"" + outputPath + "\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	r := &runner{opts: Options{
		CopilotCmd: scriptPath,
		Stdout:     ioDiscard{},
		Stderr:     ioDiscard{},
	}}

	markdown := "## Issue Comments\n\n- test"
	if err := r.runCopilot(context.Background(), markdown); err != nil {
		t.Fatalf("runCopilot returned error: %v", err)
	}

	out, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	text := string(out)
	if !strings.Contains(text, markdown) {
		t.Fatalf("expected prompt to contain markdown, got: %s", text)
	}
	if !strings.Contains(text, "You are a code assistant") {
		t.Fatalf("expected prompt to contain instructions, got: %s", text)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

func TestGroupCommentsByFile(t *testing.T) {
	tests := []struct {
		name     string
		comments []model.ReviewComment
		want     map[string]int // path -> count
		wantPath []string       // expected order
	}{
		{
			name: "groups by file path",
			comments: []model.ReviewComment{
				{Path: "file1.go", Body: "comment 1"},
				{Path: "file2.go", Body: "comment 2"},
				{Path: "file1.go", Body: "comment 3"},
			},
			want:     map[string]int{"file1.go": 2, "file2.go": 1},
			wantPath: []string{"file1.go", "file2.go"},
		},
		{
			name: "handles empty path as general comment",
			comments: []model.ReviewComment{
				{Path: "", Body: "general comment"},
				{Path: "file1.go", Body: "file comment"},
			},
			want:     map[string]int{"(general comment)": 1, "file1.go": 1},
			wantPath: []string{"(general comment)", "file1.go"},
		},
		{
			name: "filters out empty body comments",
			comments: []model.ReviewComment{
				{Path: "file1.go", Body: ""},
				{Path: "file1.go", Body: "  "},
				{Path: "file1.go", Body: "valid comment"},
			},
			want:     map[string]int{"file1.go": 1},
			wantPath: []string{"file1.go"},
		},
		{
			name: "trims whitespace from paths",
			comments: []model.ReviewComment{
				{Path: "  file1.go  ", Body: "comment"},
			},
			want:     map[string]int{"file1.go": 1},
			wantPath: []string{"file1.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := groupCommentsByFile(tt.comments)

			if len(got.byFile) != len(tt.want) {
				t.Errorf("got %d groups, want %d", len(got.byFile), len(tt.want))
			}

			for path, wantCount := range tt.want {
				gotComments, exists := got.byFile[path]
				if !exists {
					t.Errorf("missing path %q", path)
					continue
				}
				if len(gotComments) != wantCount {
					t.Errorf("path %q: got %d comments, want %d", path, len(gotComments), wantCount)
				}
			}

			if len(got.order) != len(tt.wantPath) {
				t.Errorf("got order length %d, want %d", len(got.order), len(tt.wantPath))
			}
			for i, path := range tt.wantPath {
				if i >= len(got.order) || got.order[i] != path {
					t.Errorf("order[%d]: got %q, want %q", i, got.order[i], path)
				}
			}
		})
	}
}

func TestBuildApplyPromptWithAnalysis(t *testing.T) {
	tests := []struct {
		name         string
		comments     []model.ReviewComment
		wantContains []string
		wantNotEmpty bool
	}{
		{
			name: "includes instructions and file headers",
			comments: []model.ReviewComment{
				{Path: "file1.go", Body: "fix this", User: model.CommentUser{Login: "reviewer1"}},
			},
			wantContains: []string{
				"IMPORTANT INSTRUCTIONS:",
				"analyze the feedback",
				"## File: file1.go",
				"> fix this",
				"(@reviewer1)",
			},
			wantNotEmpty: true,
		},
		{
			name: "handles comments with line numbers",
			comments: []model.ReviewComment{
				{Path: "file1.go", Body: "fix", Line: intPtr(42)},
			},
			wantContains: []string{
				"Around line 42",
				"> fix",
			},
			wantNotEmpty: true,
		},
		{
			name: "includes reference URLs",
			comments: []model.ReviewComment{
				{Path: "file1.go", Body: "comment", HTMLURL: "https://github.com/owner/repo/pull/1#discussion_123"},
			},
			wantContains: []string{
				"Reference: https://github.com/owner/repo/pull/1#discussion_123",
			},
			wantNotEmpty: true,
		},
		{
			name: "filters empty body comments",
			comments: []model.ReviewComment{
				{Path: "file1.go", Body: ""},
				{Path: "file2.go", Body: "valid"},
			},
			wantContains: []string{
				"## File: file2.go",
				"> valid",
			},
			wantNotEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildApplyPromptWithAnalysis(tt.comments)

			if tt.wantNotEmpty && got == "" {
				t.Error("expected non-empty prompt")
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("prompt missing expected content %q", want)
				}
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}
