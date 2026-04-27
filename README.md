# gh-pr-feedback

[![CI](https://github.com/robert-crandall/gh-pr-feedback/actions/workflows/ci.yml/badge.svg)](https://github.com/robert-crandall/gh-pr-feedback/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/robert-crandall/gh-pr-feedback)](https://goreportcard.com/report/github.com/robert-crandall/gh-pr-feedback)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A GitHub CLI extension that collects every issue + review comment on the current pull request, renders them as Markdown or JSON, optionally routes feedback to Copilot CLI for summarization, and can even have Copilot **apply fixes** directly to your code.

![Demo](demo.gif)

## ✨ Features

- 📋 **Summarize** – Render all PR feedback as clean Markdown
- 🎯 **Unresolved Only** – Automatically filters to unresolved review threads (no noise from addressed feedback)
- 📦 **Export** – Dump raw JSON for scripting and automation
- 🤖 **Copilot Integration** – Summarize feedback or apply fixes with GitHub Copilot CLI
- 💬 **Reply** – Post a follow-up comment back to the PR
- ⏱️ **Watch** – Poll for new feedback and automatically apply it when it arrives

## 📦 Installation

### From GitHub (recommended)

```bash
gh extension install robert-crandall/gh-pr-feedback
```

### From source

```bash
git clone https://github.com/robert-crandall/gh-pr-feedback.git
cd gh-pr-feedback
go build .
gh extension install .
```

### From releases

Download the appropriate binary from [Releases](https://github.com/robert-crandall/gh-pr-feedback/releases), extract it, and run:

```bash
gh extension install .

# To reinstall:
go build . && gh extension remove gh-pr-feedback && gh extension install .
```

## 🚀 Usage

```bash
# Summarize feedback for the PR detected in the current repo
gh pr-feedback --action summary

# Dump raw JSON for PR #42 in another repo
gh pr-feedback --pr 42 --owner octo-org --repo demo --action raw

# Send feedback through Copilot CLI for summarization
gh pr-feedback --action summary

# Apply reviewer feedback with Copilot CLI (auto-approves tool usage)
gh pr-feedback --action apply

# Apply feedback using a specific model
gh pr-feedback --action apply --model claude-sonnet-4.5

# Watch for feedback every minute (up to 10 min) and apply it automatically
gh pr-feedback --watch

# Watch and apply with a specific model
gh pr-feedback --watch --model claude-sonnet-4.5

# Write Markdown to a file and post a "no action" reply
gh pr-feedback --action markdown --output feedback.md --post-reply "No additional changes required."
```

## ⚙️ Flags

| Flag | Description |
|------|-------------|
| `--pr` | Override the PR number. When set, provide `--owner` and `--repo` or run inside the target repo. |
| `--owner`, `--repo` | Repository coordinates used with `--pr`. |
| `--action` | `markdown` (default), `raw`, `summary`, or `apply`. |
| `--output` | Write the rendered output (Markdown or JSON) to a file. |
| `--post-reply` | Body of a new issue comment to post back to the PR. |
| `--copilot-cmd` | Command invoked for `summary` or `apply` actions. Default: `copilot` (for summarization) or `copilot --allow-all-tools --allow-all-paths` (apply). |
| `--model` | Model to pass to the copilot command via `--model` (e.g. `claude-sonnet-4.5`). |
| `--watch` | Poll for feedback every minute, up to 10 minutes, and run the configured action when found. Defaults to `apply`. |
| `--quiet` | Suppress log messages. |

## 🤖 Copilot Integration

> **Note:** Review comments are automatically filtered to **unresolved threads only**. Resolved feedback is excluded so Copilot focuses on what still needs attention.

### Summarize mode (`--action summary`)

Pipes all PR feedback to Copilot CLI for AI-powered summarization:

```bash
gh pr-feedback --action summary
```

### Apply mode (`--action apply`)

Has Copilot evaluate each piece of review feedback and either apply fixes or explain why no changes are needed:

```bash
gh pr-feedback --action apply
```

See [apply.md](apply.md) for an in-depth overview of how `apply` works.

> 💡 **Tip:** The apply mode uses `copilot --allow-all-tools` by default to auto-approve file edits. Override with `--copilot-cmd` if you want manual confirmation. Use `--model` to specify a model without rewriting the full command.

### Watch mode (`--watch`)

Polls for feedback every minute for up to 10 minutes. Runs the configured action (defaults to `apply`) as soon as feedback is found:

```bash
# Watch and apply feedback when it arrives
gh pr-feedback --watch

# Watch with a specific model
gh pr-feedback --watch --model claude-sonnet-4.5

# Watch and summarize instead of applying
gh pr-feedback --watch --action summary
```

## 🛠️ Development

```bash
# Build
make build

# Run tests
make test

# Run linter
make lint

# Format code
make fmt

# Run all checks
make check

# Generate coverage report
make coverage
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

## 🎬 Recording the Demo

This project uses [VHS](https://github.com/charmbracelet/vhs) for terminal recordings:

```bash
# Install VHS
brew install vhs  # macOS

# Record the demo
make demo
```

## 📄 License

[MIT](LICENSE) © Robert Crandall

## 🗑️ Removing

```bash
gh extension remove gh-pr-feedback
```

Or manually:

```bash
rm -rf ~/.config/gh/extensions/gh-pr-feedback
```
