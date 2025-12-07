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

func TestBuildApplyInstructions(t *testing.T) {
	line := 42
	original := 21
	comments := []model.ReviewComment{
		{Path: "foo.go", Body: "Rename variable to match style", Line: &line, User: model.CommentUser{Login: "alice"}, HTMLURL: "https://example.com/1"},
		{Path: "foo.go", Body: "Cover the new branch with tests", OriginalLine: &original, User: model.CommentUser{Login: "bob"}},
		{Path: "", Body: "ignore me"},
	}

	inst := buildApplyInstructions(comments)
	if len(inst) != 1 {
		for _, i := range inst {
			t.Logf("instruction: %+v", i)
		}
		t.Fatalf("expected 1 instruction, got %d", len(inst))
	}
	if inst[0].Path != "foo.go" {
		t.Fatalf("expected path foo.go, got %s", inst[0].Path)
	}
	text := inst[0].Instructions
	for _, snippet := range []string{"Rename variable", "Cover the new branch", "Ensure other unrelated code remains unchanged"} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("expected instructions to contain %q, got: %s", snippet, text)
		}
	}
	if !strings.Contains(text, "Around line 42") || !strings.Contains(text, "Near original line 21") {
		t.Fatalf("expected instructions to mention line context, got: %s", text)
	}
	if !strings.Contains(text, "@alice") {
		t.Fatalf("expected instructions to mention reviewer, got: %s", text)
	}
}

func TestBuildApplyInstructionsSkipsEmpty(t *testing.T) {
	inst := buildApplyInstructions([]model.ReviewComment{{Path: "file.go", Body: "", User: model.CommentUser{Login: "alice"}}})
	if len(inst) != 0 {
		t.Fatalf("expected no instructions for empty body")
	}
}
