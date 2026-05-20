---
name: skill-orchestrator
description: Primary routing skill for the Huobao Drama repo. Use for any project request that may involve planning, implementing, debugging, reviewing, refactoring, prompt work, AI media workflow, Go/Gin/GORM backend changes, or Vue 3 frontend changes; selects the right project skill and source files before work begins.
---

# Huobao Drama Skill Orchestrator

Use this skill as the repo-aware entry point. Route the request, load the smallest useful set of project files, then switch to the matching project skill.

## Project Map

Huobao Drama is a Go 1.23 + Gin + GORM backend with a Vue 3 + TypeScript + Vite + Element Plus frontend.

Core layers:
- Frontend views and API clients: `web/src/views/**`, `web/src/api/**`, `web/src/types/**`
- HTTP handlers: `api/handlers/**`
- Routes: `api/routes/routes.go`
- Business logic and AI orchestration: `application/services/**`
- Domain schema: `domain/models/**`
- Storage, database, FFmpeg: `infrastructure/**`
- Prompt contracts: `application/services/prompt_i18n.go`, `application/services/prompts/narrative/*.md`

Read only what the task needs:
- Architecture and feature-to-code map: `docs/ARCHITECTURE.md`
- Safe development flow and common bug fixes: `docs/DEVELOPMENT_GUIDE.md`
- AI and prompt contracts: `docs/AI_INTEGRATION.md`
- Narrative-to-storyboard flow: `docs/NARRATIVE_TO_STORYBOARD_FLOW.md`
- Current build/start commands: `.agents/workflows/build.md`, `.agents/workflows/start-dev.md`

## Routing

Classify the request first:
- Vague product, UX, narrative, or prompt idea: use `brainstorming`.
- Non-trivial new capability or schema/API/UI workflow: use `feature-planning`.
- Add a scoped feature to existing code: use `safe-feature-extension`.
- Clean up, move, split, or restructure code without behavior change: use `safe-refactoring`.
- Bug fix, regression, suspicious behavior, or final verification: use `test-driven-verification`.
- Any coding/review task that needs discipline against overreach: apply `karpathy-guidelines` as a guardrail.

If the task crosses categories, sequence skills in this order:
1. `feature-planning` for scope and contracts.
2. `safe-refactoring` only if existing code must be reshaped before extension.
3. `safe-feature-extension` for implementation.
4. `test-driven-verification` for targeted checks.

## Repo Non-Negotiables

- Keep handlers thin. Put business rules and AI/provider logic in `application/services/`.
- Preserve JSON/API contracts between `domain/models`, `api/handlers`, `web/src/api`, and `web/src/types`.
- Treat `style` and `aspect_ratio` as required context for image/video prompt and media generation flows.
- Keep long-running AI/media work asynchronous through `TaskService` and frontend polling.
- Do not hardcode prompt logic in handlers or frontend. Use `prompt_i18n.go` or embedded markdown prompts.
- Maintain source-of-truth tables: `frame_prompts` for frame prompt text, `storyboards` for shot data, `image_generations` and `video_generations` for async media history.
- Avoid touching `data/*.db` unless the user explicitly asks for data inspection or migration work.

## Before Editing

Check `git status --short`. If unrelated user changes exist, leave them alone. If a touched file already has changes, read it carefully and work with the current state.

State the selected skill path and why. Then proceed with the smallest project-specific context needed.
