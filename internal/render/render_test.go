package render

import (
	"strings"
	"testing"
	"time"

	"github.com/robert-crandall/gh-pr-feedback/internal/model"
)

func TestMarkdown(t *testing.T) {
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	data := model.Feedback{
		IssueComments: []model.IssueComment{{
			User:      model.CommentUser{Login: "octocat"},
			Body:      "Looks good to me",
			CreatedAt: now,
			HTMLURL:   "https://example.com/issue/1",
		}},
		ReviewComments: []model.ReviewComment{{
			User:      model.CommentUser{Login: "hubot"},
			Body:      "Nit: rename variable",
			Path:      "main.go",
			Line:      ptr(10),
			CreatedAt: now,
			HTMLURL:   "https://example.com/review/2",
		}},
	}

	doc := Markdown(data)

	checks := []string{
		"# Pull Request Feedback",
		"## Issue Comments",
		"@octocat",
		"> Looks good to me",
		"## Review Comments",
		"main.go:10",
		"@hubot",
	}

	for _, c := range checks {
		if !strings.Contains(doc, c) {
			t.Fatalf("expected markdown to contain %q, got:\n%s", c, doc)
		}
	}
}

func TestJSON(t *testing.T) {
	data := model.Feedback{}
	out, err := JSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "{\n  \"issue_comments\": [],\n  \"review_comments\": []\n}"
	if strings.TrimSpace(out) != expected {
		t.Fatalf("unexpected json: %s", out)
	}
}

func ptr[T any](v T) *T {
	return &v
}
