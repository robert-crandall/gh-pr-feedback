# gh-pr-feedback

A GitHub CLI extension that collects every issue + review comment on the current pull request, renders them as Markdown or JSON, optionally routes the Markdown to Copilot CLI for summarization, and can post a follow-up reply back to the PR.

## Installation

```bash
# From this repository root
cd gh-pr-feedback
go build .
gh extension install .
```

## Usage

```bash
# Summarize feedback for the PR detected in the current repo
gh pr-feedback --action summary

# Dump raw JSON for PR #42 in another repo
gh pr-feedback --pr 42 --owner octo-org --repo demo --action raw

# Send feedback through Copilot CLI for summarization
gh pr-feedback --action copilot --copilot-cmd "copilot"

# Apply reviewer feedback with Copilot CLI (auto-approves tool usage)
gh pr-feedback --action apply --copilot-cmd "copilot --allow-all-tools"

# Write Markdown to a file and post a "no action" reply
gh pr-feedback --action summary --output feedback.md --post-reply "No additional changes required."
```

### Flags

- `--pr`: Override the PR number. When set, provide `--owner` and `--repo` or run inside the target repo.
- `--owner`, `--repo`: Repository coordinates used with `--pr`.
- `--action`: `summary` (default), `raw`, `copilot`, or `apply`.
- `--output`: Write the rendered output (Markdown or JSON) to a file.
- `--post-reply`: Body of a new issue comment to post back to the PR.
- `--copilot-cmd`: Command invoked when `--action` is `copilot` or `apply`. Quoted value is parsed with shell-style rules, so you can include arguments. For `apply`, the default is `copilot --allow-all-tools`, and the tool appends `-p "<instructions>"` for each file with review feedback.
- `--quiet`: Suppress log messages.

## Development

```bash
go test ./...
go fmt ./...
```

## Removing

Remove via CLI
```
gh extension remove gh-pr-feedback
```

Or delete manually
```
rm -rf ~/.config/gh/extensions/gh-pr-feedback
```

The extension uses the authenticated `gh` environment for both REST calls and shelling out to `gh pr view`/`gh repo view` when it needs context.
