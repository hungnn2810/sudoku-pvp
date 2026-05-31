# AGENTS.md

## Project Profile
- Product: Sudoku PvP backend
- Language: Go (1.23+)
- Primary framework: Gin
- Realtime: WebSocket
- Storage: PostgreSQL + Redis

## Core Rules
- Keep APIs deterministic and idempotent where possible.
- Prefer small packages with explicit dependencies.
- Write tests for all match flow and scoring rules.
- Never merge code without passing lint, unit tests, and integration smoke tests.

## MCP Tools: code-review-graph

**IMPORTANT: Use code-review-graph BEFORE manual file scanning.**

### Default workflow
1. `get_minimal_context(task="<task>")`
2. `detect_changes`
3. `get_review_context`
4. `get_affected_flows`
5. `tests_for`
6. `get_impact_radius` before refactor

### Usage policy
- Prefer graph queries over grep for exploration.
- Fall back to manual scan only when graph coverage is missing.
- Keep `detail_level="minimal"` unless needed.

## Delivery Standards
- API contracts versioned under `/api/v1`.
- Domain errors mapped to stable error codes.
- Structured logging only; no free-form println in production paths.
- All feature PRs must include rollback notes.
