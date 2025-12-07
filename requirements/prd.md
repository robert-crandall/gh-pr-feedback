**Overall goal of the extension**

A GitHub CLI extension (`gh-pr-feedback`) that, from your current PR context, fetches all PR feedback (issue comments + review comments), renders it as Markdown or JSON, optionally routes it through Copilot CLI for summarization/actionable suggestions, and can post a reply comment (e.g., rationale or “no action needed”) back to the PR.

---

**Step-by-step plan (Go)**

1) **Bootstrap the extension**
   - `gh extension create gh-pr-feedback --go`

2) **Define CLI surface**
   - Flags/env:
     - `--pr` (override PR number; otherwise auto-detect)
     - `--owner`, `--repo` (needed when `--pr` is manual)
     - `--action` (`summary` | `raw` | `copilot`) — default `summary`
     - `--output` (write Markdown to file)
     - `--post-reply` (string to post back to PR)
     - `--copilot-cmd` (default `github-copilot-cli`)
   - Optional future: `--filter-author`, `--since`, `--file`.

3) **Resolve PR context**
   - If `--pr` is unset: `gh pr view --json number,headRepositoryOwner,headRepository`.
   - If `--pr` is set: require/derive `--owner`, `--repo` (fallback `gh repo view --json owner,name`).

4) **Fetch comments (paginated)**
   - Issue comments: `GET /repos/{owner}/{repo}/issues/{issue_number}/comments`
   - Review comments: `GET /repos/{owner}/{repo}/pulls/{pull_number}/comments`
   - Loop `?per_page=100&page=N` until empty.
   - Parse user, body, path, line, created_at, html_url.

5) **Combine & render**
   - Combined struct: `{ issue_comments: [...], review_comments: [...] }`.
   - Renderers:
     - JSON (pretty-print).
     - Markdown (sections for issue vs review; include author, timestamp, path:line).

6) **Action modes**
   - `summary`: Markdown to stdout (and to `--output` if set).
   - `raw`: JSON to stdout.
   - `copilot`: Markdown piped to Copilot CLI with a prompt like:
     - “You are a code assistant. Summarize feedback and propose concrete code changes. If no changes are needed, explain why.”
   - Configurable `--copilot-cmd` to support `github-copilot-cli`, `copilot`, or `gh copilot`.

7) **Optional: post a reply**
   - If `--post-reply` provided: `POST /repos/{owner}/{repo}/issues/{issue_number}/comments` with the body; print the URL on success.

8) **Polish**
   - Validate flag combos; clear error messages.
   - `--quiet` to suppress nonessential logs.
   - Consider `--format` (`json|markdown`) if you want it orthogonal to `--action`.

9) **Testing**
   - Unit-test pagination, renderers, and Copilot command construction (mock subprocess).
   - Integration test against a test repo/PR with `GH_TOKEN`.

10) **Package & publish**
    - `go mod tidy`
    - `gh extension install .` (local)
    - `gh extension publish`
    - README with usage examples:
      - `gh pr-feedback --action summary > feedback.md`
      - `gh pr-feedback --action copilot --copilot-cmd github-copilot-cli`
      - `gh pr-feedback --post-reply "No changes needed because ..."`
