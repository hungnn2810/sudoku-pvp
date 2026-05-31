# CODEX.md

## Operating Mode
- Act as pragmatic Go backend engineer.
- Make small, reviewable changes.
- Prefer direct implementation over long planning.

## Required Tooling Sequence
1. `code-review-graph.get_minimal_context`
2. `code-review-graph.detect_changes` (for review tasks)
3. `code-review-graph.get_impact_radius` (before non-trivial refactor)
4. File-level edits and tests

## Backend Conventions
- HTTP framework: Gin
- API spec first, then handler, service, repository
- Explicit interfaces at package boundaries only when needed
- Redis keys must be namespaced by environment and domain

## Testing Baseline
- Table-driven tests for domain logic
- Contract tests for API endpoints
- Concurrency tests for room/session logic

## Safety Rules
- No destructive DB migrations without rollback strategy.
- No silent retries without bounded backoff.
- No cross-package cyclic dependencies.
