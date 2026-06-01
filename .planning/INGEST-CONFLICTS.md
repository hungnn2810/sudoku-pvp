## Conflict Detection Report

Mode: merge
Ingest set: 18 classification documents
Existing planning files checked: PROJECT.md, REQUIREMENTS.md, ROADMAP.md, STATE.md
Precedence applied: ADR > SPEC > PRD > DOC
Cycle detection: run on cross-ref graph — 0 cycles found
UNKNOWN/low-confidence documents: 0

---

### BLOCKERS (0)

No blockers detected.

No ADR-type documents exist in the ingest set, so no LOCKED-vs-LOCKED ADR
contradictions are possible. No ingest SPEC or PRD decisions contradict any
locked decision recorded in existing planning files. All project-locked decisions
in PROJECT.md are consistent with ingest document content.

---

### WARNINGS (1)

[WARNING] Missing referenced document — MATCHMAKING_SPEC.md
  Found: docs/CLAUDE_CODE_MASTER_PROMPT.md (DOC) cross-references "docs/MATCHMAKING_SPEC.md"
  as a required follow document for code generation.
  Expected: a classification JSON for MATCHMAKING_SPEC.md in CLASSIFICATIONS_DIR.
  Actual: no classification file found for docs/MATCHMAKING_SPEC.md in this ingest run.
  Impact: Matchmaking behavior is partially covered by BACKEND_ARCHITECTURE.md Section 5
  (Matchmaking Module) and ENGINE_TASKS.md Phase 5, but no dedicated authoritative SPEC
  exists for matchmaking algorithm, queue timeout thresholds, bot assignment logic, or
  rank-spread matching constraints. Code generation that follows CLAUDE_CODE_MASTER_PROMPT
  rules will lack a complete contract for this module.
  source: J:\sources\sudoku-pvp\docs\CLAUDE_CODE_MASTER_PROMPT.md (cross_refs field)
  -> Author docs/MATCHMAKING_SPEC.md and re-run ingest, or confirm BACKEND_ARCHITECTURE.md
     Section 5 is the sole authoritative source for matchmaking contracts.

---

### INFO (3)

[INFO] Auto-resolved: SPEC > SPEC on Redis matchmaking queue key pattern
  Found: docs/REDIS_SCHEMA.md (SPEC) defines queue key as "queue:{region}:{difficulty}:{stake}"
  including a region segment.
  Found: docs/BACKEND_ARCHITECTURE.md (SPEC) Section 7 defines queue key as "queue:{difficulty}:{stake}"
  without a region segment.
  Resolution: REDIS_SCHEMA.md is the dedicated schema contract document and takes precedence
  over the architecture overview shorthand. Synthesized intel uses queue:{region}:{difficulty}:{stake}.
  source: J:\sources\sudoku-pvp\docs\REDIS_SCHEMA.md, J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md
  Note: Implementation must align to REDIS_SCHEMA.md pattern. If region is not yet implemented,
  a placeholder region token (e.g., "global") should be used until multi-region is active.

[INFO] Auto-resolved: SPEC extends PRD on auth providers (Apple login)
  Found: docs/OPENAPI.yaml (SPEC) defines POST /auth/apple endpoint.
  Found: docs/PRODUCT_REQUIREMENTS.MD (PRD) MVP scope lists only Guest Login and Google Login;
  Apple Login is not mentioned.
  Resolution: PRD does not prohibit Apple login — it omits it from MVP scope listing.
  SPEC (OPENAPI.yaml) takes precedence over PRD per precedence rules. Apple login is included
  in synthesized requirements as a SPEC-sourced extension with a note on PRD omission.
  source: J:\sources\sudoku-pvp\docs\OPENAPI.yaml, J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD
  Note: Confirm with product owner whether Apple login is in-scope for MVP delivery or should
  be deferred. No blocking conflict — SPEC wins, but product alignment is recommended.

[INFO] Auto-resolved: SPEC > SPEC partial overlap on WebSocket server events
  Found: docs/API_CONTRACT.md (SPEC) server events list does not include "battle.error".
  Found: docs/WEBSOCKET_PROTOCOL.md (SPEC) and docs/EVENT_CATALOG.md (SPEC) both include
  "battle.error" in the server-to-client event list.
  Resolution: WEBSOCKET_PROTOCOL.md and EVENT_CATALOG.md are the dedicated protocol/event
  contract documents. They take precedence over the abbreviated listing in API_CONTRACT.md.
  "battle.error" is included in synthesized requirements. STATE.md already notes that
  API_CONTRACT.md is partial and may require additions during implementation.
  source: J:\sources\sudoku-pvp\docs\WEBSOCKET_PROTOCOL.md, J:\sources\sudoku-pvp\docs\EVENT_CATALOG.md,
          J:\sources\sudoku-pvp\docs\API_CONTRACT.md, J:\sources\sudoku-pvp\.planning\STATE.md
