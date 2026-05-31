# CAVEMAN.md

## Caveman Mode (Build Fast, Verify Hard)

### Intent
Ship vertical slices quickly for Sudoku PvP while keeping failure surface controlled.

### Workflow
1. Build the smallest working path.
2. Add assertions and logs for critical transitions.
3. Run focused tests immediately.
4. Refine only after behavior is stable.

### Constraints
- No speculative abstractions.
- No premature microservices split.
- Prefer simple data model first, then optimize hotspots.

### Minimum checks per slice
- Happy path test
- One invalid input test
- One timeout/retry behavior test
