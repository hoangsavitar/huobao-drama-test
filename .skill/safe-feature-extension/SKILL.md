---
name: safe-feature-extension
description: Safely implement scoped Huobao Drama features in the existing Go/Gin/GORM and Vue 3 codebase without breaking current production flows, API contracts, async task behavior, prompt contracts, or media generation workflows.
---

# Huobao Drama Safe Feature Extension

Use this skill when adding a concrete feature to the existing project.

## Start With The Existing Flow

Trace the feature through the repo before editing:
1. UI view or component in `web/src/views/**` or `web/src/components/**`
2. Frontend API client in `web/src/api/**`
3. Type contract in `web/src/types/**`
4. Gin route in `api/routes/routes.go`
5. Handler in `api/handlers/**`
6. Business logic in `application/services/**`
7. GORM model in `domain/models/**`

Use `rg` for symbols, endpoint paths, JSON fields, and task types.

## Implementation Rules

- Add the smallest behavior that satisfies the request.
- Keep handlers thin; put logic in services.
- Avoid unrelated refactors, formatting churn, and broad component rewrites.
- Preserve old API fields unless a breaking change is explicitly requested.
- For new long-running AI/media operations, create an async task with `TaskService`, return `task_id`, update progress, and let the frontend poll.
- For batch generation, prefer partial success over failing the whole batch when one shot/asset fails.
- Do not hardcode prompt strings in handlers or Vue components. Use `prompt_i18n.go` or embedded prompt markdown.
- Use DB transactions when replacing episode/storyboard/scene sets.
- Keep `data/*.db` and generated media untouched unless the user explicitly asks.

## Common Extension Paths

Drama metadata:
- Backend: `application/services/drama_service.go`, `api/handlers/drama.go`, `domain/models/drama.go`
- Frontend: `web/src/api/drama.ts`, `web/src/types/drama.ts`, `web/src/views/drama/**`

Narrative graph generation:
- Backend: `NarrativeGenerateRequest`, `GenerateNarrativeEpisodes`, `processMultiAgentNarrative`, `NarrativePackageService`
- Prompts: `application/services/prompts/narrative/*.md`
- Important fields: `narrative_node_id`, `choices`, `state_snapshot`, `is_entry`

Storyboard and frame prompts:
- Backend: `storyboard_service.go`, `frame_prompt_service.go`, `frame_prompt_query.go`
- Important tables: `storyboards`, `frame_prompts`, join tables for characters/props/outfits
- Keep shot duration, action, result, dialogue, atmosphere, image prompt, and video prompt coherent.

Image/video generation:
- Backend: `image_generation_service.go`, `video_generation_service.go`, provider clients in `pkg/image`, `pkg/video`
- Always propagate `drama.Style` and `drama.AspectRatio`.
- Preserve local storage download/update behavior for `image_url`, `video_url`, and `local_path`.

Frontend workflow:
- Main production UI: `web/src/views/drama/EpisodeWorkflow.vue`
- Shared clients: `web/src/api/**`
- UI library: Element Plus with existing styles and i18n.
- Keep loading, disabled, polling, error, and empty states consistent with existing workflow pages.

## Finish Criteria

Verify the new path and at least one old path that could regress. Prefer targeted tests and type/build checks over manual inspection alone.
