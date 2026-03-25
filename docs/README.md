# Documentation Index

This folder is the primary knowledge base for maintainers and AI agents.
If you are new to the codebase, read files in this order.

---

## Recommended Reading Order

1. `EPISODE_PRODUCTION_GUIDE.md`
   - System architecture
   - Feature map
   - Page-to-service ownership
   - Failure diagnosis and extension playbook

2. `ARCHITECTURE_MAP.md`
   - Fast lookup table:
     page -> API -> handler -> service -> model
   - Best starting point for targeted fixes

3. `AI_PROMPT_AND_API_REFERENCE.md`
   - Prompt source-of-truth and contracts
   - API endpoint contracts
   - Payload shapes and enums

4. `API_CONTRACTS.md`
   - Endpoint-level request/response contracts
   - Required/optional field expectations

5. `ASYNC_TASKS_AND_POLLING.md`
   - Task lifecycle and polling strategy
   - Partial-failure behavior for batch flows

6. `MODEL_PROVIDER_MATRIX.md`
   - Feature-to-model/provider strategy
   - Cost/quality throughput guidance

7. `DB_SCHEMA_MAP.md`
   - Operational entity relationships
   - Source-of-truth field map

8. `AI_PROMPT_EXAMPLES_DB.md`
   - Real-shaped examples for output quality and debugging
   - Common validation checkpoints

9. `QUICK_FIX_PLAYBOOK.md`
   - Fast troubleshooting decision tree

10. `DATA_MIGRATION.md`
   - Schema and migration guidance
   - Operational migration procedures

---

## Role-Based Quick Entry

### Product / PM
- Start: `EPISODE_PRODUCTION_GUIDE.md` sections:
  - Product flow
  - Frontend page responsibilities
  - Feature-to-code map

### Backend Developer
- Start:
  - `EPISODE_PRODUCTION_GUIDE.md` (backend architecture + model/API map)
  - `AI_PROMPT_AND_API_REFERENCE.md` (prompt/API contracts)
- Then:
  - `AI_PROMPT_EXAMPLES_DB.md` for regression comparison

### Frontend Developer
- Start:
  - `EPISODE_PRODUCTION_GUIDE.md` (page responsibilities, status flows)
  - `AI_PROMPT_AND_API_REFERENCE.md` (request/response shapes)

### AI Agent / Automation
- Start:
  - `EPISODE_PRODUCTION_GUIDE.md`
  - `ARCHITECTURE_MAP.md`
  - `QUICK_FIX_PLAYBOOK.md`
- Use:
  - `AI_PROMPT_AND_API_REFERENCE.md` for contract-safe edits
  - `AI_PROMPT_EXAMPLES_DB.md` for output sanity checks

---

## Doc Maintenance Rules

When changing behavior, update docs in the same PR:

1. **Route/API changed** -> update `AI_PROMPT_AND_API_REFERENCE.md`.
2. **Prompt contract changed** -> update `AI_PROMPT_AND_API_REFERENCE.md` and examples if needed.
3. **Flow/page ownership changed** -> update `EPISODE_PRODUCTION_GUIDE.md`.
4. **New recurring incident pattern** -> add to `QUICK_FIX_PLAYBOOK.md`.

Keep docs concise, contract-first, and implementation-accurate.

