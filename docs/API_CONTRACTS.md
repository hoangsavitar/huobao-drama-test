# API Contracts (Operational)

This document defines practical endpoint contracts for maintenance and feature development.
It focuses on high-impact endpoints used in episode production.

Base path:
- `/api/v1`

Response envelope (current convention):
- success:
  ```json
  { "success": true, "data": { ... } }
  ```
- error:
  ```json
  { "success": false, "error": { "message": "..." } }
  ```

---

## 1) Drama

### `GET /dramas/:id`
- Purpose: load drama + nested episode context.
- Used by: `EpisodeWorkflow.vue`, `ProfessionalEditor.vue`, `DramaList.vue`.
- Key response fields:
  - `id`, `title`, `style`, `aspect_ratio`
  - `episodes[]` (with storyboard summaries)

### `PUT /dramas/:id`
- Purpose: update project-level config.
- Request:
  ```json
  {
    "title": "string?",
    "description": "string?",
    "style": "ghibli|guoman|...|kdrama?",
    "aspect_ratio": "16:9|9:16?",
    "status": "draft|planning|production|completed|archived?"
  }
  ```
- Notes:
  - `aspect_ratio` is critical for downstream prompt/image/video behavior.

---

## 2) Storyboard Read Model

### `GET /episodes/:episode_id/storyboards`
- Purpose: episode shot composition payload for workflow table/status.
- Used by: `EpisodeWorkflow.vue` enrichment.
- Key fields per storyboard:
  - `id`, `storyboard_number`, `title`, `action`, `duration`
  - `characters[]`, `background`
  - `composed_image`
  - `image_generation_status`, `video_generation_status`

---

## 3) Frame Prompt

### `POST /storyboards/:id/frame-prompt`
- Purpose: submit async prompt generation for a frame type.
- Request:
  ```json
  { "frame_type": "first|key|last|panel|action", "panel_count": 3? }
  ```
- Response:
  ```json
  {
    "task_id": "string",
    "status": "pending|processing",
    "message": "..."
  }
  ```

### `PUT /storyboards/:id/frame-prompt`
- Purpose: manual prompt overwrite/upsert.
- Request:
  ```json
  { "frame_type": "first|key|last|panel|action", "prompt": "string" }
  ```
- Behavior:
  - upsert by `(storyboard_id, frame_type)`.
  - empty prompt may be treated as delete semantics by handler logic.

### `GET /storyboards/:id/frame-prompts`
- Purpose: fetch prompt records for one storyboard.
- Response:
  ```json
  { "frame_prompts": [ { "id": 1, "frame_type": "first", "prompt": "...", "layout": "" } ] }
  ```

### `GET /episodes/:episode_id/frame-prompts`
- Purpose: batch status map for shot table.
- Response:
  ```json
  {
    "frame_prompts_by_storyboard": {
      "167": [ { "frame_type": "first", "prompt": "..." } ]
    }
  }
  ```

---

## 4) Image Generation

### `POST /images`
- Purpose: create image generation task (character/scene/storyboard).
- Request (common):
  ```json
  {
    "drama_id": "5",
    "storyboard_id": 167,
    "scene_id": 21,
    "character_id": 9,
    "image_type": "storyboard|scene|character",
    "frame_type": "first|key|last|panel|action",
    "prompt": "string",
    "provider": "string?",
    "model": "string?",
    "size": "2560x1440?",
    "reference_images": ["/static/..", "http://..."]
  }
  ```
- Response:
  - `ImageGeneration` record with `status=pending|processing`.

### `GET /images`
- Purpose: list image generation records with pagination.
- Query:
  - `drama_id?`, `scene_id?`, `storyboard_id?`, `frame_type?`, `status?`
  - `page`, `page_size` (service currently caps to 100 per page).
- Response:
  ```json
  {
    "items": [ { "id": 1, "status": "completed", "image_url": "...", "local_path": "..." } ],
    "pagination": { "page": 1, "page_size": 100, "total": 500, "total_pages": 5 }
  }
  ```

### `POST /images/upload`
- Purpose: register manual upload as completed image generation.

---

## 5) Video Generation

### `POST /videos` (via frontend video API wrapper)
- Purpose: create async video generation task.
- Request (common):
  ```json
  {
    "drama_id": "5",
    "storyboard_id": 167,
    "prompt": "string",
    "provider": "string",
    "model": "string",
    "duration": 5,
    "aspect_ratio": "16:9|9:16",
    "reference_mode": "none|single|first_last|multiple",
    "image_url": "string?",
    "image_local_path": "string?",
    "first_frame_url": "string?",
    "last_frame_url": "string?",
    "reference_image_urls": ["string?"]
  }
  ```

---

## 6) Task Status

### `GET /tasks/:id` (frontend `taskAPI.getStatus`)
- Purpose: poll async state for prompt/image/video tasks.
- Expected `status`:
  - `pending`, `processing`, `completed`, `failed`

---

## 7) Contract Guardrails for New Features

1. Add endpoint contract in this file in the same PR.
2. Include example request + response with required/optional fields.
3. Document async behavior (immediate response vs final resource state).
4. Document ID type (`string` vs `number`) expectations explicitly.

