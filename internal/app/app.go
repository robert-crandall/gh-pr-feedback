package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/google/shlex"
	"github.com/robert-crandall/gh-pr-feedback/internal/model"
	"github.com/robert-crandall/gh-pr-feedback/internal/render"
)

const (
	actionMarkdown = "markdown"
	actionRaw      = "raw"
	actionSummary  = "summary"
	actionApply    = "apply"

	pageSize = 100
)

// Options configures the CLI behavior.
type Options struct {
	PRNumber   int
	Owner      string
	Repo       string
	Action     string
	OutputPath string
	PostReply  string
	CopilotCmd string
	Quiet      bool
	Stdout     io.Writer
	Stderr     io.Writer
}

// Run executes the CLI flow based on provided options.
func Run(ctx context.Context, opts Options) error {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	opts.Action = strings.ToLower(strings.TrimSpace(opts.Action))
	if opts.Action == "" {
		opts.Action = actionSummary
	}

	if opts.CopilotCmd == "" {
		if opts.Action == actionApply {
			opts.CopilotCmd = "copilot --allow-all-tools --allow-all-paths"
		} else {
			opts.CopilotCmd = "copilot"
		}
	}

	r := &runner{opts: opts}
	return r.run(ctx)
}

type runner struct {
	opts          Options
	restClient    *api.RESTClient
	graphqlClient *api.GraphQLClient
}

func (r *runner) run(ctx context.Context) error {
	var err error
	r.restClient, err = api.NewRESTClient(api.ClientOptions{})
	if err != nil {
		return err
	}
	r.graphqlClient, err = api.NewGraphQLClient(api.ClientOptions{})
	if err != nil {
		return err
	}

	action, err := r.normalizeAction()
	if err != nil {
		return err
	}
	r.opts.Action = action

	prCtx, err := r.resolvePRContext(ctx)
	if err != nil {
		return err
	}
	r.log("Resolved PR context: %s/%s#%d", prCtx.Owner, prCtx.Repo, prCtx.Number)

	feedback, err := r.fetchFeedback(ctx, prCtx)
	if err != nil {
		return err
	}

	var rendered string
	switch action {
	case actionMarkdown:
		rendered = render.Markdown(feedback)
		if _, err := fmt.Fprint(r.opts.Stdout, rendered); err != nil {
			return err
		}
	case actionRaw:
		rendered, err = render.JSON(feedback)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(r.opts.Stdout, rendered); err != nil {
			return err
		}
	case actionSummary:
		rendered = render.Markdown(feedback)
		if err := r.runCopilot(ctx, rendered); err != nil {
			return err
		}
	case actionApply:
		rendered = render.Markdown(feedback)
		if err := r.runCopilotApply(ctx, feedback); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported action %q", action)
	}

	if r.opts.OutputPath != "" {
		if err := writeOutputFile(r.opts.OutputPath, rendered); err != nil {
			return err
		}
		r.log("Wrote output to %s", r.opts.OutputPath)
	}

	if strings.TrimSpace(r.opts.PostReply) != "" {
		if err := r.postReply(ctx, prCtx); err != nil {
			return err
		}
	}

	return nil
}

func (r *runner) normalizeAction() (string, error) {
	switch r.opts.Action {
	case actionMarkdown, actionRaw, actionSummary, actionApply:
		return r.opts.Action, nil
	default:
		return "", fmt.Errorf("invalid --action value %q", r.opts.Action)
	}
}

func (r *runner) resolvePRContext(ctx context.Context) (prContext, error) {
	if r.opts.PRNumber > 0 {
		owner := strings.TrimSpace(r.opts.Owner)
		repo := strings.TrimSpace(r.opts.Repo)
		if owner == "" || repo == "" {
			fallback, err := currentRepo(ctx)
			if err != nil {
				return prContext{}, errors.New("--owner and --repo are required when --pr is set outside a PR context")
			}
			if owner == "" {
				owner = fallback.Owner
			}
			if repo == "" {
				repo = fallback.Repo
			}
		}
		return prContext{Owner: owner, Repo: repo, Number: r.opts.PRNumber}, nil
	}

	return detectCurrentPR(ctx)
}

func (r *runner) fetchFeedback(ctx context.Context, pr prContext) (model.Feedback, error) {
	issue, err := r.fetchIssueComments(ctx, pr)
	if err != nil {
		return model.Feedback{}, fmt.Errorf("issue comments: %w", err)
	}
	review, err := r.fetchReviewComments(ctx, pr)
	if err != nil {
		return model.Feedback{}, fmt.Errorf("review comments: %w", err)
	}
	return model.Feedback{IssueComments: issue, ReviewComments: review}, nil
}

func (r *runner) fetchIssueComments(ctx context.Context, pr prContext) ([]model.IssueComment, error) {
	return paginate(func(page int) ([]model.IssueComment, error) {
		path := fmt.Sprintf("repos/%s/%s/issues/%d/comments?per_page=%d&page=%d", url.PathEscape(pr.Owner), url.PathEscape(pr.Repo), pr.Number, pageSize, page)
		var chunk []model.IssueComment
		if err := r.restClient.DoWithContext(ctx, http.MethodGet, path, nil, &chunk); err != nil {
			return nil, err
		}
		return chunk, nil
	})
}

func (r *runner) fetchReviewComments(ctx context.Context, pr prContext) ([]model.ReviewComment, error) {
	const query = `
query($owner: String!, $repo: String!, $pr: Int!, $cursor: String) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $pr) {
      reviewThreads(first: 100, after: $cursor) {
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          isResolved
          comments(first: 100) {
            nodes {
              id
              author { login }
              body
              path
              line
              originalLine
              url
              createdAt
            }
          }
        }
      }
    }
  }
}
`
	var allComments []model.ReviewComment
	var cursor *string

	for {
		variables := map[string]interface{}{
			"owner":  pr.Owner,
			"repo":   pr.Repo,
			"pr":     pr.Number,
			"cursor": cursor,
		}

		var resp struct {
			Repository struct {
				PullRequest struct {
					ReviewThreads struct {
						PageInfo struct {
							HasNextPage bool   `json:"hasNextPage"`
							EndCursor   string `json:"endCursor"`
						} `json:"pageInfo"`
						Nodes []struct {
							IsResolved bool `json:"isResolved"`
							Comments   struct {
								Nodes []struct {
									ID     string `json:"id"`
									Author struct {
										Login string `json:"login"`
									} `json:"author"`
									Body         string `json:"body"`
									Path         string `json:"path"`
									Line         *int   `json:"line"`
									OriginalLine *int   `json:"originalLine"`
									URL          string `json:"url"`
									CreatedAt    string `json:"createdAt"`
								} `json:"nodes"`
							} `json:"comments"`
						} `json:"nodes"`
					} `json:"reviewThreads"`
				} `json:"pullRequest"`
			} `json:"repository"`
		}

		if err := r.graphqlClient.DoWithContext(ctx, query, variables, &resp); err != nil {
			return nil, fmt.Errorf("graphql reviewThreads: %w", err)
		}

		for _, thread := range resp.Repository.PullRequest.ReviewThreads.Nodes {
			// Only include unresolved threads
			if thread.IsResolved {
				continue
			}
			for _, c := range thread.Comments.Nodes {
				comment := model.ReviewComment{
					User:         model.CommentUser{Login: c.Author.Login},
					Body:         c.Body,
					Path:         c.Path,
					Line:         c.Line,
					OriginalLine: c.OriginalLine,
					HTMLURL:      c.URL,
				}
				if c.CreatedAt != "" {
					if t, err := time.Parse(time.RFC3339, c.CreatedAt); err == nil {
						comment.CreatedAt = t
					}
				}
				allComments = append(allComments, comment)
			}
		}

		if !resp.Repository.PullRequest.ReviewThreads.PageInfo.HasNextPage {
			break
		}
		cursor = &resp.Repository.PullRequest.ReviewThreads.PageInfo.EndCursor
	}

	return allComments, nil
}

func (r *runner) runCopilot(ctx context.Context, markdown string) error {
	args, err := shlex.Split(strings.TrimSpace(r.opts.CopilotCmd))
	if err != nil {
		return fmt.Errorf("parse --copilot-cmd: %w", err)
	}
	if len(args) == 0 {
		return errors.New("--copilot-cmd is empty")
	}

	prompt := buildCopilotPrompt(markdown)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Stdout = r.opts.Stdout
	cmd.Stderr = r.opts.Stderr
	r.log("Running %s for Copilot summary", args[0])
	return cmd.Run()
}

func (r *runner) runCopilotApply(ctx context.Context, feedback model.Feedback) error {
	if len(feedback.ReviewComments) == 0 {
		return errors.New("no review comments to apply")
	}

	args, err := shlex.Split(strings.TrimSpace(r.opts.CopilotCmd))
	if err != nil {
		return fmt.Errorf("parse --copilot-cmd: %w", err)
	}
	if len(args) == 0 {
		return errors.New("--copilot-cmd is empty")
	}

	// Build comprehensive prompt with pattern analysis
	prompt := buildApplyPromptWithAnalysis(feedback.ReviewComments)

	// Single invocation with all context
	cmdArgs := append([]string{}, args...)
	cmdArgs = append(cmdArgs, "-p", prompt)
	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdout = r.opts.Stdout
	cmd.Stderr = r.opts.Stderr
	r.log("Applying feedback with pattern analysis via %s", filepath.Base(cmdArgs[0]))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("copilot apply: %w", err)
	}

	return nil
}

func (r *runner) postReply(ctx context.Context, pr prContext) error {
	payload := map[string]string{"body": r.opts.PostReply}
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(payload); err != nil {
		return err
	}
	path := fmt.Sprintf("repos/%s/%s/issues/%d/comments", url.PathEscape(pr.Owner), url.PathEscape(pr.Repo), pr.Number)
	var resp struct {
		HTMLURL string `json:"html_url"`
	}
	if err := r.restClient.DoWithContext(ctx, http.MethodPost, path, buf, &resp); err != nil {
		return fmt.Errorf("post reply: %w", err)
	}
	if resp.HTMLURL != "" {
		_, _ = fmt.Fprintf(r.opts.Stdout, "Posted reply: %s\n", resp.HTMLURL)
	}
	return nil
}

func writeOutputFile(path string, content string) error {
	if path == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func buildCopilotPrompt(markdown string) string {
	var b strings.Builder
	b.WriteString("You are a code assistant. Summarize the following PR feedback and propose concrete actions. ")
	b.WriteString("If no action is required, explain why.\n\n")
	b.WriteString("# Feedback\n\n")
	b.WriteString(markdown)
	return b.String()
}

func buildApplyPromptWithAnalysis(comments []model.ReviewComment) string {
	if len(comments) == 0 {
		return ""
	}

	var b strings.Builder

	// Header with instructions for pattern analysis
	b.WriteString("You are helping resolve PR review feedback. Multiple reviewers have left comments across multiple files.\n\n")
	b.WriteString("IMPORTANT INSTRUCTIONS:\n")
	b.WriteString("1. First, analyze the feedback to identify common patterns or repeated issues across files\n")
	b.WriteString("2. When you see comments like 'Same', 'Ditto', 'Same issue here', etc., understand them in context of previous comments\n")
	b.WriteString("3. Apply fixes consistently when the same pattern appears in multiple files\n")
	b.WriteString("4. If feedback appears incorrect or unclear, explain why rather than making incorrect changes\n")
	b.WriteString("5. Keep unrelated code unchanged\n\n")

	b.WriteString("---\n\n")
	b.WriteString("# Review Feedback by File\n\n")

	// Group comments by file to show structure
	fileGroups := groupCommentsByFile(comments)

	for _, path := range fileGroups.order {
		fileComments := fileGroups.byFile[path]
		b.WriteString(fmt.Sprintf("## File: %s\n\n", path))

		for _, comment := range fileComments {
			lineDesc := describeCommentLocation(comment)
			if lineDesc != "" {
				b.WriteString(fmt.Sprintf("**%s** ", lineDesc))
			}

			author := strings.TrimSpace(comment.User.Login)
			if author != "" {
				b.WriteString(fmt.Sprintf("(@%s)", author))
			}
			b.WriteString(":\n")

			body := strings.TrimSpace(comment.Body)
			if body != "" {
				b.WriteString(fmt.Sprintf("> %s\n\n", body))
			}

			if comment.HTMLURL != "" {
				b.WriteString(fmt.Sprintf("Reference: %s\n\n", comment.HTMLURL))
			}
		}

		b.WriteString("---\n\n")
	}

	b.WriteString("\nNow analyze the patterns and apply the necessary fixes across all files. ")
	b.WriteString("Remember to resolve cross-references like 'Same' by understanding the context from previous comments.")

	return b.String()
}

type fileCommentGroup struct {
	order  []string
	byFile map[string][]model.ReviewComment
}

func groupCommentsByFile(comments []model.ReviewComment) fileCommentGroup {
	group := fileCommentGroup{
		order:  make([]string, 0),
		byFile: make(map[string][]model.ReviewComment),
	}

	for _, comment := range comments {
		path := strings.TrimSpace(comment.Path)
		if path == "" {
			path = "(no file specified)"
		}

		if _, exists := group.byFile[path]; !exists {
			group.order = append(group.order, path)
			group.byFile[path] = make([]model.ReviewComment, 0)
		}

		group.byFile[path] = append(group.byFile[path], comment)
	}

	return group
}

func describeCommentLocation(comment model.ReviewComment) string {
	if comment.Line != nil {
		return fmt.Sprintf("Around line %d", *comment.Line)
	}
	if comment.OriginalLine != nil {
		return fmt.Sprintf("Near original line %d", *comment.OriginalLine)
	}
	return ""
}

func (r *runner) log(format string, args ...interface{}) {
	if r.opts.Quiet || r.opts.Stderr == nil {
		return
	}
	_, _ = fmt.Fprintf(r.opts.Stderr, format+"\n", args...)
}

type prContext struct {
	Owner  string
	Repo   string
	Number int
}

type repoInfo struct {
	Owner string
	Repo  string
}

func detectCurrentPR(ctx context.Context) (prContext, error) {
	out, err := runGHJSON(ctx, "pr", "view", "--json", "number")
	if err != nil {
		return prContext{}, err
	}
	var parsed struct {
		Number int `json:"number"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return prContext{}, err
	}
	if parsed.Number == 0 {
		return prContext{}, errors.New("unable to determine PR number from gh pr view")
	}

	repo, err := currentRepo(ctx)
	if err != nil {
		return prContext{}, err
	}

	return prContext{Owner: repo.Owner, Repo: repo.Repo, Number: parsed.Number}, nil
}

func currentRepo(ctx context.Context) (repoInfo, error) {
	out, err := runGHJSON(ctx, "repo", "view", "--json", "owner,name")
	if err != nil {
		return repoInfo{}, err
	}
	var parsed struct {
		Name  string `json:"name"`
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return repoInfo{}, err
	}
	if parsed.Name == "" || parsed.Owner.Login == "" {
		return repoInfo{}, errors.New("gh repo view returned empty data")
	}
	return repoInfo{Owner: parsed.Owner.Login, Repo: parsed.Name}, nil
}

func runGHJSON(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return nil, fmt.Errorf("gh %s failed: %s", strings.Join(args, " "), detail)
	}
	return out, nil
}

func paginate[T any](fetch func(page int) ([]T, error)) ([]T, error) {
	var all []T
	for page := 1; ; page++ {
		chunk, err := fetch(page)
		if err != nil {
			return nil, err
		}
		if len(chunk) == 0 {
			break
		}
		all = append(all, chunk...)
		if len(chunk) < pageSize {
			break
		}
	}
	return all, nil
}
