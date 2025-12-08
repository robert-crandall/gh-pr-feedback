# Apply Action Documentation

## Overview
The `apply` action in `gh-pr-feedback` is designed to automatically send PR feedback to GitHub Copilot for resolution. This action uses a **pattern-aware, single-invocation approach** that analyzes feedback across all files before applying changes.

## How It Works

### 1. Feedback Collection
The `apply` action first collects **only unresolved review comments** (not issue comments) from the PR:
- Uses GraphQL to fetch review threads
- Filters to include only threads where `isResolved == false`
- Issue comments are collected but NOT used in the apply action
- Review comments include: author, body, file path, line number, URL, and creation time

### 2. Pattern Analysis & Comprehensive Prompting
The agent looks at all feedback and:

✅ **Groups all feedback by file** for better organization  
✅ **Sends everything in ONE Copilot invocation** (not N separate calls)  
✅ **Instructs Copilot to analyze patterns first** before applying changes  
✅ **Resolves cross-references** like "Same", "Ditto", etc. by providing full context

### 3. Prompt Structure

The prompt is built by `buildApplyPromptWithAnalysis()` with the following structure:

```
You are helping resolve PR review feedback. Multiple reviewers have left comments across multiple files.

IMPORTANT INSTRUCTIONS:
1. First, analyze the feedback to identify common patterns or repeated issues across files
2. When you see comments like 'Same', 'Ditto', 'Same issue here', etc., understand them in context of previous comments
3. Apply fixes consistently when the same pattern appears in multiple files
4. If feedback appears incorrect or unclear, explain why rather than making incorrect changes
5. Keep unrelated code unchanged

---

# Review Feedback by File

## File: src/app.go

**Around line 42** (@alice):
> This function should return an error instead of panicking

Reference: https://github.com/owner/repo/pull/123#discussion_r789

**Around line 58** (@bob):
> Consider using a constant here instead of a magic number

Reference: https://github.com/owner/repo/pull/123#discussion_r790

---

## File: src/utils.go

**Around line 15** (@alice):
> Same

Reference: https://github.com/owner/repo/pull/123#discussion_r791

---

Now analyze the patterns and apply the necessary fixes across all files. 
Remember to resolve cross-references like 'Same' by understanding the context from previous comments.
```

### 4. Copilot Execution (Single Invocation)

```go
func (r *runner) runCopilotApply(ctx context.Context, feedback model.Feedback)
```

The function now:
1. Builds one comprehensive prompt with all feedback
2. Executes **one command**: `copilot --allow-all-tools --allow-all-paths -p "<comprehensive_prompt>"`
3. Copilot receives full context and can identify patterns across files
4. All fixes are applied in a single session with consistent understanding

**Default command**: `copilot --allow-all-tools --allow-all-paths`
- Can be overridden with `--copilot-cmd` flag
- The `--allow-all-tools --allow-all-paths` flags ensure Copilot has full access to modify files

### 5. Key Characteristics

#### ✅ Analyzes patterns across files
The prompt explicitly instructs Copilot to identify common patterns before applying changes.

#### ✅ Resolves cross-references
Comments like "Same" or "Ditto" are understood in context because Copilot sees all the previous comments.

#### ✅ Single invocation
All feedback is sent together, reducing overhead and enabling consistency.

#### ✅ Better consistency
Since Copilot processes everything in one session, it applies the same fix pattern uniformly across all files.

#### ✅ More efficient
- One analysis instead of N rediscoveries
- Lower token costs (no repeated context)
- Faster execution (no sequential file processing)

#### ✅ Includes full context
Each comment includes:
- File path
- Line numbers (when available)
- Comment authors
- Full comment text
- Reference URLs to the original comments

#### ✅ Safety guards
The prompt includes:
- Instructions to analyze before acting
- Reminder that feedback "might not be correct"
- Option to explain instead of making incorrect changes
- Explicit instruction to keep "other unrelated code unchanged"

## Advantages Over Previous File-by-File Approach

| Aspect | Old (File-by-File) | New (Pattern-Aware) |
|--------|-------------------|---------------------|
| **Invocations** | N (one per file) | 1 (all at once) |
| **Pattern recognition** | ❌ Each file analyzed independently | ✅ Patterns identified across all files |
| **Cross-references** | ❌ "Same" is meaningless without context | ✅ "Same" resolved by looking at previous comments |
| **Consistency** | ⚠️ File 1 might be fixed differently than file 5 | ✅ Same pattern applied uniformly |
| **Efficiency** | ⚠️ Repeated analysis of similar issues | ✅ Analyze once, apply everywhere |
| **Token cost** | Higher (N invocations with setup overhead) | Lower (one invocation with shared context) |
| **Speed** | Slower (sequential processing) | Faster (single execution) |
| **Error handling** | Stops on first file failure | Copilot can make best effort across all files |

## Example Scenario: "Same" Comments

Consider a PR where a reviewer comments:
- **file1.go:42** - "Add error handling here"
- **file2.go:15** - "Same"
- **file3.go:88** - "Same issue"

**Old behavior:** 
- File 1: Copilot adds error handling ✅
- File 2: Copilot doesn't understand "Same" ❌
- File 3: Copilot doesn't understand "Same issue" ❌

**New behavior:**
- Copilot sees all three comments in context
- Recognizes the pattern
- Applies error handling consistently to all three locations ✅

## Error Handling

- If no review comments exist, returns error: "no review comments to apply"
- If Copilot command fails, returns: `"copilot apply: %w"`
- Much simpler error handling since there's only one invocation

## Comparison to Summary Action

| Feature | `summary` | `apply` |
|---------|-----------|---------|
| Copilot command | `copilot` | `copilot --allow-all-tools --allow-all-paths` |
| Input format | Full markdown (issue + review comments) | Review comments only, grouped by file with pattern analysis |
| Processing | Single invocation for analysis | Single invocation for analysis + application |
| Prompt style | "Summarize and propose actions" | "Analyze patterns and apply fixes" |
| Output | Analysis and suggestions | Direct code changes |
| Issue comments | Included | Excluded |
| Resolved threads | Included | Excluded |
| Pattern awareness | No explicit instruction | Yes, explicit pattern analysis |
