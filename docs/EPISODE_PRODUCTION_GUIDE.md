# Drama Generator - System Guide for Development and AI Maintenance

This document is the canonical project map for developers and AI agents.
Goal: after reading this file once, you can safely debug, maintain, and add features without rediscovering architecture.

---

## 1) Product Scope and End-to-End Flow

The project builds episodic drama content through a staged pipeline:

1. Create drama project (title, style, aspect ratio).
2. Create/prepare episode script.
3. Extract character and scene background data from script.
4. Generate character images and scene images.
5. Split script into storyboard shots.
6. Generate first-frame prompts (and other frame prompts).
7. Generate shot images and videos.
8. Compose/export outputs.

Core UX pages:
- `web/src/views/drama/DramaList.vue`: project list + edit.
- `web/src/views/drama/DramaManagement.vue`: project-level management.
- `web/src/views/drama/EpisodeWorkflow.vue`: staged production workflow.
- `web/src/views/drama/ProfessionalEditor.vue`: per-shot fine editing and generation.

---

## 2) Backend Architecture

Backend stack:
- Gin (HTTP)
- GORM (DB model + query)
- Service layer (`application/services/*`)
- Handler layer (`api/handlers/*`)
- Route registration (`api/routes/routes.go`)

Important backend folders:
- `domain/models`: persistent schema.
- `application/services`: business logic and AI prompt orchestration.
- `api/handlers`: HTTP request binding + response.
- `infrastructure/database/database.go`: `AutoMigrate` model list.

Design rule:
- Handlers stay thin.
- Services hold flow logic and AI/provider calls.
- Frontend should not depend on provider-specific behavior.

---

## 3) Data Model Cheat Sheet

Primary entities:
- `Drama`: top-level project.
  - Important fields: `style`, `aspect_ratio`.
- `Episode`: script container under drama.
- `Storyboard`: shot-level unit.
  - Important fields: `action`, `image_prompt`, `video_prompt`, `composed_image`.
- `Character`: extracted cast and image anchors.
- `Scene`: extracted background-only scene records.
- `FramePrompt`: generated prompts by `frame_type` (`first`, `key`, `last`, `panel`, `action`).
- `ImageGeneration`: async image jobs and outputs.
- `VideoGeneration`: async video jobs and outputs.

New and critical behavior:
- `Drama.aspect_ratio` drives portrait/landscape behavior (`16:9` or `9:16`) across prompts and generation defaults.

---

## 4) API Surfaces You Will Touch Most

### Drama / episode / storyboard
- `GET /api/v1/dramas/:id`
- `PUT /api/v1/dramas/:id` (includes `aspect_ratio`)
- `GET /api/v1/episodes/:episode_id/storyboards`

### Frame prompts
- `POST /api/v1/storyboards/:id/frame-prompt` (async generation task)
- `PUT /api/v1/storyboards/:id/frame-prompt` (manual overwrite save)
- `GET /api/v1/storyboards/:id/frame-prompts`
- `GET /api/v1/episodes/:episode_id/frame-prompts`

### Images
- `POST /api/v1/images`
- `GET /api/v1/images` (pagination; use multiple pages for exports)
- `POST /api/v1/images/upload`

### Videos
- `POST /api/v1/videos` (through frontend `videoAPI.generateVideo`)

---

## 5) Prompt System and AI Orchestration

Single source of truth:
- `application/services/prompt_i18n.go`

Prompt families:
- Storyboard decomposition prompts.
- Character extraction prompts.
- Scene extraction prompts.
- Frame prompts (`first/key/last/panel/action`).
- Video constraint prompts.
- Style expansions (`styleLabel`, `GetStylePrompt`).

Important prompt contracts:
- Frame prompt JSON must be strict and parseable.
- Scene extraction must be background-only (no people).
- Aspect ratio must be explicitly present in prompt context for consistent framing.
- `description` field was removed from frame prompt payload contracts.

---

## 6) Aspect Ratio Rules (Critical)

Accepted values:
- `16:9` (landscape)
- `9:16` (portrait)

Where aspect ratio is applied:
- Drama create/update request and persistence.
- Prompt construction in `prompt_i18n.go`.
- Frame prompt generation chain in `frame_prompt_service.go`.
- Image generation defaults in `image_generation_service.go`.
- Scene generation in `storyboard_composition_service.go`.
- Video generation request defaults in frontend dialogs/editors.
- UI rendering ratio in `ProfessionalEditor.vue` via CSS variable.

Size mapping (image service helper):
- `16:9` -> `2560x1440`
- `9:16` -> `1440x2560`

If a generated result looks like old ratio:
- verify drama `aspect_ratio` persisted in DB,
- verify server restarted after backend changes,
- verify request payload includes expected `aspect_ratio` for video endpoints.

---

## 7) Feature-to-Code Map

### A) Batch first-frame prompt generation (Episode Workflow)
- UI: `EpisodeWorkflow.vue`
- API: `generateFirstFrame`, `getEpisodeFramePrompts`
- Behavior: select shot rows, submit per-shot tasks, poll, reload prompt status column.

### B) Batch shot image generation (moved to Episode Workflow)
- UI: `EpisodeWorkflow.vue`
- Logic: selected storyboard IDs -> resolve first-frame prompt -> call `imageAPI.generateImage`.
- Status column: uses `composed_image` and `image_generation_status`.

### C) Professional editor per-shot operations
- File: `ProfessionalEditor.vue`
- Responsibilities: manual shot editing, prompt editing/saving, single-shot image/video generation, timeline operations.

### D) Prompt save consistency and overwrite flow
- Frontend: `updateFramePrompt` + debounced autosave.
- Backend: `PUT /storyboards/:id/frame-prompt` upsert behavior.
- Cache strategy: session storage mirror is overwritten from DB on load/switch.

### E) Exports
- Video prompts export: `EpisodeWorkflow.vue` text file.
- Shot images export: `EpisodeWorkflow.vue` + `web/src/utils/exportShotImagesZip.ts`.
  - Zip structure:
    - `Ep{n}_{episodeTitle}/{dramaTitle}/shot_{num}_{title}/...images`

---

## 8) Frontend Page Responsibilities

### `DramaList.vue`
- create/edit/delete projects.
- must include style + aspect ratio edit consistency.

### `DramaManagement.vue`
- overview and entity management at drama level.

### `EpisodeWorkflow.vue`
- staged production line (script -> extraction -> storyboard -> prompts/images/video).
- batch operations entry point.
- status visibility for extraction, prompts, and shot image readiness.

### `ProfessionalEditor.vue`
- advanced per-shot editing.
- not the place for episode-level batch orchestration.

---

## 9) Common Failure Modes and Fast Diagnosis

### 1) Prompt not updated between pages
Check:
- `PUT /storyboards/:id/frame-prompt` is called.
- DB reload path overwrites stale session cache.

### 2) Shot A shows prompt from shot B
Check:
- cache key includes storyboard ID + frame type.
- data reload merges by numeric storyboard IDs, not mixed string/number keys.

### 3) Aspect ratio still old after switching to portrait
Check:
- drama update included `aspect_ratio`.
- backend restarted after schema/prompt logic update.
- generation request path actually reads drama ratio in current flow.

### 4) Batch export zip missing images
Check:
- image records are `completed`.
- source URL/local path fetchable from browser origin.
- pagination fetched enough pages for the episode.

---

## 10) Safe Extension Playbook (How to Add Features)

When adding any new generation feature:
1. Add/confirm domain field in `domain/models`.
2. Add service logic in `application/services`.
3. Add API handler + route.
4. Add frontend API function.
5. Add UI state + explicit status rendering.
6. Add i18n keys in `en-US.ts` (and optionally other locales).
7. Add docs in this folder.

For AI prompt changes:
1. Update `prompt_i18n.go`.
2. Validate parser/JSON contracts.
3. Rebuild backend and restart service.
4. Verify one-shot and batch paths separately.

---

## 11) Operational Notes for AI Agents

When maintaining this project, always:
- read `prompt_i18n.go` first for behavioral intent,
- trace from page -> frontend API -> handler -> service -> model,
- verify string/number ID conversions in Vue state paths,
- check asynchronous polling completion and stale cache risks,
- keep docs updated when changing route contracts or generation flow.

---

## 12) Related Docs

- `docs/AI_PROMPT_AND_API_REFERENCE.md`: compact prompt/API contract reference.
- `docs/AI_PROMPT_EXAMPLES_DB.md`: concrete examples and payload/response samples.
- `docs/DATA_MIGRATION.md`: migration procedure and schema considerations.

