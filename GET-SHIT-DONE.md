# GET-SHIT-DONE.md

## GSD Workflow for Sudoku PvP Backend

### Objective
Deliver backend features end-to-end with strict execution discipline.

### Cycle
1. Clarify scope in 3 lines (input, output, acceptance).
2. Implement minimal end-to-end path.
3. Add tests for new behavior.
4. Verify locally.
5. Summarize impact and risks.

### Per-Task Checklist
- API contract updated
- Domain logic covered by tests
- Redis/Postgres interactions validated
- Metrics/logging added for new flow
- Rollback note documented

### Code Review Gate
Always run `code-review-graph` flow before finalizing:
- `detect_changes`
- `get_review_context`
- `get_affected_flows`
- `tests_for`

### Completion Standard
A task is done only when behavior is verified, observable, and review-ready.
