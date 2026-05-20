---
name: test-driven-verification
description: Verify Huobao Drama changes with targeted Go tests, frontend type/build checks, parser/normalizer tests, async task checks, and manual smoke paths for Go/Gin/GORM services, Vue 3 UI, AI prompt contracts, narrative graphs, storyboard generation, and media workflows.
---

# Huobao Drama Test-Driven Verification

Use this skill for bug fixes, regression checks, and final validation.

## Verification Ladder

Choose the lowest level that proves the change:
1. Static contract check: route, JSON tag, TypeScript type, prompt schema, and service call path all agree.
2. Targeted unit test or existing package test.
3. Backend build/test.
4. Frontend type/build check.
5. Manual smoke of the affected workflow.

Do not call live AI providers or generate media unless the user requested it and configuration is available. Prefer parser, normalizer, service, and UI contract tests for repeatability.

## Useful Commands

Backend:
- `go test ./application/services`
- `go test ./application/services -run TestNormalizeNarrativeGraph`
- `go test ./pkg/utils`
- `go test ./...`
- `go build .`

Frontend:
- `cd web && npm run build:check`
- `cd web && npm run build`

Development smoke:
- Backend default: `go run main.go` on `http://localhost:5678`
- Frontend default: `cd web && npm run dev` on `http://localhost:3012`

## What To Verify By Feature Type

Narrative graph:
- Output has one `start_narrative_node_id`.
- Every `choices[].next_narrative_node_id` exists.
- BFS ordering and `episode_number` are stable.
- Exactly one entry episode is marked `is_entry`.
- `choices` preserve both narrative node id and resolved episode id where required.

Prompt contracts:
- Model output can be parsed as JSON without markdown fences.
- Character extraction keeps appearance separate from scene/background.
- Scene/background prompts contain no people or actions.
- Frame prompt writes to `frame_prompts.prompt`, not obsolete fields.
- `style` and `aspect_ratio` are present in downstream prompt/media context.

Async tasks:
- Request returns `task_id` quickly.
- Task moves pending -> processing -> completed/failed.
- Error paths update task error and do not leave the UI stuck.
- Batch paths tolerate partial failures when the current workflow expects it.

Frontend:
- `web/src/api` return type matches handler response.
- `web/src/types` matches backend JSON fields.
- Loading, disabled, empty, retry, and polling states remain coherent.
- New visible text is added to locale files when the existing UI path uses i18n.

## Reporting

State exactly which checks ran and which were skipped. If a check fails, include the failing package/command and whether it appears related to the current change.
