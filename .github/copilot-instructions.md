# Copilot Instructions — wspulse/metrics-otel

## Project Overview

wspulse/metrics-otel is an **OpenTelemetry adapter** for wspulse/server's `MetricsCollector` interface. It translates server lifecycle events into OTel instruments (counters, up-down counters, histograms, gauges). Module path: `github.com/wspulse/metrics-otel`. Package name: `otel`.

## Architecture

- **`collector.go`** — `Collector` struct implementing `wspulse.MetricsCollector`. Creates all OTel instruments on construction. Each interface method records to the corresponding instrument.
- **`options.go`** — `Option` functional options: `WithMeterProvider`, `WithNamespace`, `WithRoomAttribute`.
- **`collector_test.go`** — Unit tests using `sdkmetric.NewManualReader()` to verify instrument recordings.

## Dependencies

- `github.com/wspulse/server` — source of `MetricsCollector` interface
- `go.opentelemetry.io/otel/metric` — OTel metrics API
- `go.opentelemetry.io/otel/sdk/metric` — OTel metrics SDK (test only)

## Development Workflow

```bash
make fmt        # format source files
make check      # fmt + lint + test (pre-commit gate)
make test       # unit tests with race detector
make test-cover # tests with coverage report
make bench      # benchmarks
make tidy       # go mod tidy
```

## Conventions

- **Go style**: same as wspulse/server — `gofmt`/`goimports`, GoDoc on all public symbols.
- **Naming**: interface names use full words. Package name is `otel`.
- **Instrument naming**: dot-separated (`wspulse.connections.opened`), follows [OTel Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/).
- **Attribute naming**: dot-separated keys (`room.id`, not `room_id`).
- **Error format**: `fmt.Errorf("wspulse: <context>: %w", err)`.
- **Markdown**: no emojis in documentation files.
- **Git**: commit messages follow [commit-message-instructions.md](instructions/commit-message-instructions.md). Branch strategy: `feat/`, `fix/`, `chore/`. Never push directly to `main`.
- **File encoding**: all files must be UTF-8 without BOM. Do not use any other encoding.

## Critical Rules

1. **Read before write** — read the target file before editing.
2. **STOP — test first, fix second** — when a bug is discovered or reported, do NOT touch production code until a failing test exists. Follow this exact sequence: (1) write a failing test, (2) confirm it fails, (3) fix the code, (4) confirm it passes, (5) run `make check`.
3. **`make check` gates every commit** — fmt + lint + test must pass.
4. **Minimal changes** — one concern per edit.
5. **No breaking changes without version bump** — exported symbols are a public contract.
6. **Thread safety** — all `Collector` methods are called concurrently from server goroutines. OTel instruments are safe for concurrent use, but verify any custom state is properly synchronized.
7. **Accuracy** — verify instrument names, types, and attribute sets against the plan in the workspace `doc/local/plan/metrics-otel.md`.
8. **Documentation sync** — when changing public API or options, update `docs/reference/` and `docs/guides/metrics.md` in the docs repo.

## PR Comment Review — MANDATORY

When handling PR review comments, **every unresponded comment must be analyzed and responded to**. No comment may be silently ignored.

### 1. Fetch unresponded comments

Pull all comments that have not received a reply from the PR author. Bot-generated summaries (e.g. Copilot review overview) may be skipped; individual line comments from bots must still be evaluated.

### 2. Analyze each comment

Evaluate against:

| Criterion | Question |
|-----------|----------|
| **Validity** | Is the observation correct? Is the suggestion reasonable? |
| **Severity** | Is it a bug, a correctness issue, a design concern, or a style/preference nitpick? |
| **Cost** | What is the effort to address? Does the change introduce risk or scope creep? |

### 3. Decide and respond

Every comment must receive exactly one response:

- **`Fixed in {hash}. {what changed and why}`** — adopt and fix immediately. Bug and correctness issues must use this path unless the fix requires a separate PR due to scope.
- **`Tracked in TODOS.md — {reason for deferring}`** — adopt but defer. Add entry to repo root `TODOS.md` with context and PR comment link.
- **`Won't fix. {clear reasoning}`** — reject the suggestion with explanation.
- **`Not applicable — {explanation}`** — the comment does not apply (already handled, misunderstanding, duplicate, or already tracked in TODOS.md).

Duplicate or related comments may reference each other: `Same reasoning as {reference} above — {brief}`.

### 4. Zero unresponded comments before merge

The PR must have zero unaddressed comments before merge. This is a hard gate.

## Session Protocol

> Files under `doc/local/` are git-ignored and must **never** be committed.
> This includes plan files (`doc/local/plan/`), review records, and the AI learning log (`doc/local/ai-learning.md`).

### Start of every session — MANDATORY

**Do these steps before writing any code:**

1. Read `doc/local/ai-learning.md` **in full** to recall past mistakes. If the file is missing or empty, create it with the table header (see format below) before proceeding.
2. Check `doc/local/plan/` for any in-progress plan and read it fully.

### During feature work

For any new feature or multi-file fix: save a plan to `doc/local/plan/<feature-name>.md` **before starting**. Keep it updated with completed steps throughout the session.

### Review records

After conducting any review (code review, plan review, design review, PR review, etc.), record the findings for cross-session context:

- **Where to write**: this repo's `doc/local/`. If working in a multi-module workspace, also write to the workspace root's `doc/local/`.
- **Single truth**: write the full record in one location; the other location keeps a brief summary with a file path reference to the full record.
- **Acceptable formats**:
  1. Update the relevant plan file in `doc/local/plan/` with the review outcome.
  2. Dedicated review file in `doc/local/` if no relevant plan exists.
- **What to record**: review type, key findings, decisions made, action items, and resolution status.

### End of every session — MANDATORY

**Before closing the session, complete this checklist without exception:**

1. Append at least one entry to `doc/local/ai-learning.md` — **even if no mistakes were made**. Record what you confirmed, what technique worked, or what you observed. An empty file is a sign of non-compliance.
2. Update any in-progress plan in `doc/local/plan/` to reflect completed steps.
3. Verify `make check` passes in every module you edited.

**Entry format** for `doc/local/ai-learning.md`:

```
| Date       | Issue or Learning | Root Cause | Prevention Rule |
| ---------- | ----------------- | ---------- | --------------- |
| YYYY-MM-DD | <what happened or what you learned> | <why it happened> | <how to avoid it next time> |
```

**Writing to `ai-learning.md` is not optional. It is the primary cross-session improvement mechanism. An empty file proves the session protocol was ignored.**
