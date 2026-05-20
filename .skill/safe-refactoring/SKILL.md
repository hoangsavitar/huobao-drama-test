---
name: safe-refactoring
description: Refactor Huobao Drama Go services, Gin handlers, GORM models, Vue components, TypeScript API/types, or prompt files while preserving behavior, database/API contracts, async task semantics, and generated media workflows.
---

# Huobao Drama Safe Refactoring

Use this skill when restructuring existing code without changing behavior.

## Behavior Lock

Before editing, identify the behavior that must remain identical:
- API path, method, request body, response shape, and status code.
- DB table, column, JSON tag, relation, and default.
- Async task type, status transitions, progress/result/error shape.
- Prompt input variables and model output JSON schema.
- Frontend visible workflow, loading state, and polling behavior.

If the requested refactor requires behavior change, split it into a refactor phase and a feature phase.

## Refactor Safely

- Run or identify a targeted verification before the refactor when possible.
- Keep file moves and symbol renames localized.
- Do not mix unrelated cleanup with the refactor.
- Remove only unused code created by your refactor.
- Preserve existing comments and bilingual/domain wording unless they are directly wrong.
- For Go, keep service logic in `application/services`, handlers thin, and model definitions in `domain/models`.
- For Vue, keep component state and API contracts synchronized with `web/src/api` and `web/src/types`.
- For prompts, preserve JSON-only constraints and downstream parser assumptions.

## High-Risk Areas

Treat these as high risk and verify carefully:
- `application/services/drama_service.go`: episode replacement, narrative graph, choices, character dedupe.
- `application/services/narrative_package_service.go`: embedded prompts, JSON parsing, normalization.
- `application/services/storyboard_service.go`: AI output parsing, character/outfit/scene mapping, duration totals.
- `application/services/frame_prompt_service.go`: source of truth for `frame_prompts`.
- `application/services/image_generation_service.go`: local storage, reference images, status updates.
- `api/routes/routes.go`: endpoint order and route collisions.
- `web/src/views/drama/EpisodeWorkflow.vue`: multi-step production UI and polling state.

## Verification

After refactoring, run the smallest command that proves behavior stayed intact:
- Backend package test: `go test ./application/services`
- Targeted test: `go test ./application/services -run TestNormalizeNarrativeGraph`
- Full backend test when scope warrants it: `go test ./...`
- Frontend type/build check: `cd web && npm run build:check`
- Backend build: `go build .`

If a verification fails for pre-existing unrelated changes, report that clearly and do not hide it with broad edits.
