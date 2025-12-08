package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

