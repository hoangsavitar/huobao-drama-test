# Architecture Map (Page -> API -> Handler -> Service -> Model)

This is the fastest lookup table for maintainers and AI agents.
Use this before changing any feature to avoid editing the wrong layer.

---

## 1) Core Runtime Layers

- **Frontend pages/components**: `web/src/views/**`, `web/src/components/**`
- **Frontend API wrappers**: `web/src/api/**`
- **HTTP handlers**: `api/handlers/**`
- **Business services**: `application/services/**`
- **Persistence models**: `domain/models/**`
- **Route registry**: `api/routes/routes.go`

---

## 2) Drama and Episode Management

### Create/Edit Drama (style, aspect ratio)
- Frontend:
  - `web/src/components/common/CreateDramaDialog.vue`
  - `web/src/views/drama/DramaList.vue`
  - `web/src/api/drama.ts`
- Backend:
  - Handler: `api/handlers/drama.go`
  - Service: `application/services/drama_service.go`
  - Model: `domain/models/drama.go`

### Episode workflow page data load
- Frontend:
  - `web/src/views/drama/EpisodeWorkflow.vue`
  - `web/src/api/drama.ts` (`get`, `getStoryboards`, etc.)
- Backend:
  - Handler: `api/handlers/drama.go`, `api/handlers/scene.go`
  - Service: `application/services/drama_service.go`, `application/services/storyboard_composition_service.go`
  - Models: `Drama`, `Episode`, `Storyboard`, `Scene`, `Character`

---

## 3) Prompt Generation Chain

### First-frame prompt (single shot)
- Frontend:
  - `web/src/api/frame.ts` -> `generateFirstFrame(storyboardId)`
- Backend:
  - Route: `POST /api/v1/storyboards/:id/frame-prompt`
  - Handler: frame prompt handler entry
  - Service: `application/services/frame_prompt_service.go`
  - Prompt template source: `application/services/prompt_i18n.go`
  - Model persistence: `domain/models/frame_prompt.go`

### Episode-level prompt status map
- Frontend:
  - `web/src/api/frame.ts` -> `getEpisodeFramePrompts(episodeId)`
  - UI: `EpisodeWorkflow.vue` first-frame status column
- Backend:
  - Route: `GET /api/v1/episodes/:episode_id/frame-prompts`
  - Handler: `api/handlers/frame_prompt_query.go`
  - Model: `FramePrompt`

### Prompt manual overwrite/autosave
- Frontend:
  - `web/src/api/frame.ts` -> `updateFramePrompt`
  - UI: `ProfessionalEditor.vue` autosave watcher
- Backend:
  - Route: `PUT /api/v1/storyboards/:id/frame-prompt`
  - Handler: `api/handlers/frame_prompt_query.go` (`UpdateFramePrompt`)
  - Model: `FramePrompt`

---

## 4) Character and Scene Extraction

### Character extraction from script
- Frontend:
  - `EpisodeWorkflow.vue` stage action
- Backend:
  - Handler: character generation/extraction endpoints
  - Service: `application/services/character_library_service.go`, `application/services/script_generation_service.go`
  - Prompt source: `prompt_i18n.go` (`GetCharacterExtractionPrompt`)
  - Model: `Character`

### Scene/background extraction from script
- Frontend:
  - `EpisodeWorkflow.vue` extract scenes flow
- Backend:
  - Handler: `api/handlers/image_generation.go` (`ExtractBackgroundsForEpisode`)
  - Service: `application/services/image_generation_service.go`
  - Prompt source: `prompt_i18n.go` (`GetSceneExtractionPrompt`)
  - Model: `Scene`

---

## 5) Image Generation Paths

### Character image generation
- Frontend:
  - `EpisodeWorkflow.vue` character card generate button
  - API: `web/src/api/character-library.ts`
- Backend:
  - Handler: `api/handlers/character_library_gen.go`
  - Service: `application/services/character_library_service.go`
  - Downstream image service: `application/services/image_generation_service.go`
  - Models: `Character`, `ImageGeneration`

### Scene image generation
- Frontend:
  - `EpisodeWorkflow.vue` scene card generate button
  - API: `web/src/api/drama.ts` / scene image routes
- Backend:
  - Handler: scene image endpoint in `api/handlers/scene.go`
  - Service: `application/services/storyboard_composition_service.go`
  - Downstream image service: `image_generation_service.go`
  - Models: `Scene`, `ImageGeneration`

### Shot image generation (single and batch)
- Frontend:
  - Single: `ProfessionalEditor.vue` Shot Image tab
  - Batch: `EpisodeWorkflow.vue` shot table batch action
  - API: `web/src/api/image.ts`
- Backend:
  - Handler: `api/handlers/image_generation.go` (`GenerateImage`)
  - Service: `application/services/image_generation_service.go`
  - Models: `ImageGeneration`, `Storyboard`

---

## 6) Video Generation Paths

### Shot video generation
- Frontend:
  - `ProfessionalEditor.vue` video tab
  - `web/src/views/generation/components/GenerateVideoDialog.vue`
  - API: `web/src/api/video.ts`
- Backend:
  - Handler: video generation endpoints
  - Service: `application/services/video_generation_service.go`
  - Prompt constraints: `prompt_i18n.go` (`GetVideoConstraintPrompt`)
  - Model: `domain/models/video_generation.go`

---

## 7) Storyboard Composition and Status Projection

### Episode storyboard table payload
- Frontend:
  - `EpisodeWorkflow.vue` shot table
- Backend:
  - Handler: `api/handlers/scene.go` (`GetStoryboardsForEpisode`)
  - Service: `application/services/storyboard_composition_service.go` (`GetScenesForEpisode`)
  - Returned composition fields include:
    - `characters`
    - `background`
    - `composed_image`
    - `image_generation_status`
    - `video_generation_status`

---

## 8) Export Features

### Export video prompts
- Frontend only:
  - `EpisodeWorkflow.vue` (`exportVideoPrompts`)

### Export shot images zip
- Frontend:
  - UI trigger: `EpisodeWorkflow.vue`
  - Utility: `web/src/utils/exportShotImagesZip.ts`
  - Data source: paginated `GET /api/v1/images`
- Backend:
  - Handler: `api/handlers/image_generation.go` (`ListImageGenerations`)
  - Service: `image_generation_service.go` (`ListImageGenerations`)

---

## 9) Cross-Cutting Contracts

### ID typing
- Frontend has mixed `string`/`number` IDs for storyboard/episode/drama.
- Normalize aggressively when joining map/list data:
  - compare with `Number(id)` or `String(id)` consistently.

### Aspect ratio
- `Drama.aspect_ratio` (`16:9`/`9:16`) is global config source.
- Must flow through:
  - prompt builders
  - image size defaults
  - video request payload
  - UI ratio rendering

### Async task and polling
- Most generation actions are async.
- UI should:
  - submit task(s)
  - poll status endpoint
  - refresh composition/prompt data maps

---

## 10) Change Impact Matrix (Quick)

- **Prompt text quality issue only**
  - likely `prompt_i18n.go` + caller service formatting.
- **Data saved but not visible**
  - frontend cache/reload or wrong endpoint for status.
- **Batch action partial failures**
  - per-item error handling and ID normalization.
- **Wrong ratio in output**
  - drama config persistence + prompt + generation request + UI display all must align.

