---
name: feature-planning
description: Plan non-trivial Huobao Drama features before implementation, especially changes involving narrative graphs, AI prompt contracts, async tasks, Go service/handler/routes, GORM models, Vue views, API clients, media generation, or storyboard workflow.
---

# Huobao Drama Feature Planning

Use this skill to turn a feature request into a repo-aware implementation plan with explicit contracts and verification.

## Context First

Read the smallest relevant map before asking questions or proposing changes:
- General architecture: `docs/ARCHITECTURE.md`
- Development playbook: `docs/DEVELOPMENT_GUIDE.md`
- AI/prompt contracts: `docs/AI_INTEGRATION.md`
- Narrative graph flow: `docs/NARRATIVE_TO_STORYBOARD_FLOW.md`

Then inspect the code path likely to change:
- Drama/project workflow: `api/handlers/drama.go`, `application/services/drama_service.go`, `web/src/api/drama.ts`, `web/src/types/drama.ts`
- Narrative generation: `application/services/narrative_package_service.go`, `application/services/prompts/narrative/*.md`
- Storyboard splitting: `application/services/storyboard_service.go`, `api/handlers/storyboard.go`
- Frame prompts: `application/services/frame_prompt_service.go`, `api/handlers/frame_prompt*.go`
- Image/video generation: `application/services/image_generation_service.go`, `application/services/video_generation_service.go`
- Character/outfit/prop flows: `character_library_service.go`, `prop_service.go`, related handlers and views

## Planning Output

For a non-trivial feature, produce:
1. Understanding summary: what changes for the user and which production stage it affects.
2. Current behavior: route, handler, service, model, and frontend entry points.
3. Target contract: request/response shape, DB fields, task status/result, prompt output schema, and UI state.
4. Risk list: API compatibility, migrations, prompt JSON fragility, long-running tasks, stale polling/cache, media path/local storage handling.
5. Implementation steps with verification criteria.

Ask clarifying questions only when local context cannot resolve a risky ambiguity. Otherwise state assumptions and keep moving.

## Huobao Feature Checklist

For backend-visible features, plan every needed layer:
- `domain/models/**`: field, JSON tag, GORM tag, relation, default.
- AutoMigrate/migration impact: verify `infrastructure/database/database.go` covers the model.
- `application/services/**`: main behavior, transactions, async task creation, provider fallback, partial success semantics.
- `api/handlers/**`: request binding, status codes, thin delegation.
- `api/routes/routes.go`: route group and endpoint placement.
- `web/src/api/**`: client method.
- `web/src/types/**`: TypeScript interface.
- `web/src/views/**` or components: state, loading, task polling, error display.
- `web/src/locales/**`: visible copy for both locale files when adding UI text.

For AI/prompt features:
- Keep JSON-only output constraints explicit.
- Include `style` and `aspect_ratio` where image/video downstream depends on them.
- Keep character names canonical across episodes.
- Keep scene background prompts free of people/characters.
- Add parser/normalizer tests for generated JSON shapes when practical.

## Decision Log

Record decisions that affect architecture, DB contracts, prompt schemas, async behavior, or user workflow. Include alternatives only when they were plausible in this repo.
