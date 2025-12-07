package render

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/robert-crandall/gh-pr-feedback/internal/model"
)

const timestampLayout = time.RFC3339

// Markdown renders combined feedback into a human-readable Markdown document.
func Markdown(data model.Feedback) string {
	var b strings.Builder
	b.WriteString("# Pull Request Feedback\n\n")

	writeIssueComments(&b, data.IssueComments)
	writeReviewComments(&b, data.ReviewComments)

	return b.String()
}

// JSON renders the feedback as pretty-printed JSON for piping/automation.
func JSON(data model.Feedback) (string, error) {
	if data.IssueComments == nil {
		data.IssueComments = []model.IssueComment{}
	}
	if data.ReviewComments == nil {
		data.ReviewComments = []model.ReviewComment{}
	}
	buf, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func writeIssueComments(b *strings.Builder, comments []model.IssueComment) {
	b.WriteString("## Issue Comments\n\n")
	if len(comments) == 0 {
		b.WriteString("_No issue comments._\n\n")
		return
	}

	for _, c := range comments {
		fmt.Fprintf(b, "### @%s — %s\n\n", safeLogin(c.User.Login), c.CreatedAt.Format(timestampLayout))
		writeQuotedBody(b, c.Body)
		fmt.Fprintf(b, "[View comment](%s)\n\n", c.HTMLURL)
	}
}

func writeReviewComments(b *strings.Builder, comments []model.ReviewComment) {
	b.WriteString("## Review Comments\n\n")
	if len(comments) == 0 {
		b.WriteString("_No review comments._\n")
		return
	}

	for _, c := range comments {
		lineInfo := formatLineLabel(c)
		fmt.Fprintf(b, "### @%s — %s (%s)\n\n", safeLogin(c.User.Login), c.CreatedAt.Format(timestampLayout), lineInfo)
		writeQuotedBody(b, c.Body)
		fmt.Fprintf(b, "[View comment](%s)\n\n", c.HTMLURL)
	}
}

func writeQuotedBody(b *strings.Builder, body string) {
	body = strings.TrimSpace(body)
	if body == "" {
		b.WriteString("(No comment body)\n\n")
		return
	}
	for _, line := range strings.Split(body, "\n") {
		if strings.TrimSpace(line) == "" {
			b.WriteString(">\n")
			continue
		}
		b.WriteString("> ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

func formatLineLabel(c model.ReviewComment) string {
	var line int
	switch {
	case c.Line != nil && *c.Line != 0:
		line = *c.Line
	case c.OriginalLine != nil && *c.OriginalLine != 0:
		line = *c.OriginalLine
	}

	if c.Path == "" {
		if line == 0 {
			return "general"
		}
		return fmt.Sprintf("line %d", line)
	}

	if line == 0 {
		return c.Path
	}
	return fmt.Sprintf("%s:%d", c.Path, line)
}

func safeLogin(login string) string {
	if strings.TrimSpace(login) == "" {
		return "unknown"
	}
	return login
}
