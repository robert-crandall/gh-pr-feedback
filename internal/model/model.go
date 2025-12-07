package model

import "time"

// CommentUser is a minimal representation of a GitHub user on a comment.
type CommentUser struct {
	Login string `json:"login"`
}

// IssueComment holds metadata for regular issue-style comments on a PR thread.
type IssueComment struct {
	ID        int64       `json:"id"`
	User      CommentUser `json:"user"`
	Body      string      `json:"body"`
	CreatedAt time.Time   `json:"created_at"`
	HTMLURL   string      `json:"html_url"`
}

// ReviewComment holds metadata for code review comments tied to files/lines.
type ReviewComment struct {
	ID           int64       `json:"id"`
	User         CommentUser `json:"user"`
	Body         string      `json:"body"`
	Path         string      `json:"path"`
	Line         *int        `json:"line"`
	OriginalLine *int        `json:"original_line"`
	CreatedAt    time.Time   `json:"created_at"`
	HTMLURL      string      `json:"html_url"`
}

// Feedback combines issue and review comments for easier rendering/testing.
type Feedback struct {
	IssueComments  []IssueComment  `json:"issue_comments"`
	ReviewComments []ReviewComment `json:"review_comments"`
}
