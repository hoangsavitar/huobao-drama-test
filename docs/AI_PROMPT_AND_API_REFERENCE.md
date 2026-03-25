# AI Prompt and API Reference

This file is a fast technical reference for prompt contracts and API payloads.
Use this together with:
- `docs/EPISODE_PRODUCTION_GUIDE.md` (system map),
- `docs/AI_PROMPT_EXAMPLES_DB.md` (real examples).

---

## 1) Prompt Source of Truth

Primary file:
- `application/services/prompt_i18n.go`

Prompt functions currently used:
- Storyboard breakdown:
  - `GetStoryboardSystemPrompt()`
- Scene extraction:
  - `GetSceneExtractionPrompt(style, aspectRatio string)`
- Character extraction:
  - `GetCharacterExtractionPrompt(style, aspectRatio string)`
- Prop extraction:
  - `GetPropExtractionPrompt(style, aspectRatio string)`
- Frame prompts:
  - `GetFirstFramePrompt(style, aspectRatio string)`
  - `GetKeyFramePrompt(style, aspectRatio string)`
  - `GetLastFramePrompt(style, aspectRatio string)`
  - `GetActionSequenceFramePrompt(style, aspectRatio string)`
- Video constraints:
  - `GetVideoConstraintPrompt(referenceMode string)`
- Style expansion:
  - `styleLabel(style string)`
  - `GetStylePrompt(style string)`

---

## 2) Prompt Contract Rules

### Frame prompt response contract
- Expected output: strict JSON object for prompt generation.
- `description` is no longer part of frame prompt contract.
- `frame_type` persistence happens in `FramePrompt` table via service layer.

### Scene extraction contract
- Must return background-only prompts.
- No people/characters in background prompt.
- Must include style and aspect-ratio intent.

### Character extraction contract
- Must produce stable appearance anchors for consistency across shots.
- Appearance text is reused in character image generation.

### Aspect ratio contract
- Use `16:9` or `9:16`.
- Ratio must flow from drama config into prompt builders and generation request defaults.

---

## 3) Key API Endpoints

### Frame prompt endpoints
- `POST /api/v1/storyboards/:id/frame-prompt`
  - async generation task for one frame type.
- `PUT /api/v1/storyboards/:id/frame-prompt`
  - manual overwrite/upsert prompt text.
- `GET /api/v1/storyboards/:id/frame-prompts`
  - list prompts for one storyboard.
- `GET /api/v1/episodes/:episode_id/frame-prompts`
  - grouped prompts for episode shot status rendering.

### Image endpoints
- `POST /api/v1/images`
  - create image generation task.
- `GET /api/v1/images`
  - paginated list (supports `drama_id`, `storyboard_id`, `status`, `frame_type`).
- `POST /api/v1/images/upload`
  - register uploaded image as completed generation record.

### Video endpoints
- `POST /api/v1/videos` (from frontend video API wrappers)
  - for shot video generation.

---

## 4) Common Request Shapes

### Create/update frame prompt
```json
{
  "frame_type": "first",
  "prompt": "string"
}
```

### Generate storyboard image
```json
{
  "drama_id": "5",
  "storyboard_id": 167,
  "image_type": "storyboard",
  "frame_type": "first",
  "prompt": "string",
  "reference_images": ["/static/scenes/xx.jpg", "/static/characters/a.jpg"]
}
```

### Generate video
```json
{
  "drama_id": "5",
  "storyboard_id": 167,
  "prompt": "string",
  "provider": "doubao",
  "model": "model_name",
  "duration": 5,
  "aspect_ratio": "9:16",
  "reference_mode": "single"
}
```

---

## 5) Camera and Visual Enumerations (Operational)

These values are controlled by frontend option sets and locale labels.

### Shot type (examples)
- Long Shot
- Full Shot
- Medium Shot
- Close Up
- Extreme Close Up

### Camera angle (examples)
- Eye level
- High angle
- Low angle
- Bird's-eye view
- Dutch angle
- Over-the-shoulder

### Camera movement (examples)
- Static shot
- Push in
- Pull out
- Pan
- Follow shot
- Handheld
- Dolly
- Drone

---

## 6) Style Keys

Current style keys used in prompts/UI:
- `ghibli`
- `guoman`
- `guoman3d`
- `wasteland`
- `nostalgia`
- `pixel`
- `voxel`
- `urban`
- `chibi3d`
- `kdrama`

---

## 7) Cost and Throughput Notes for Prompt Generation

For first-frame prompt generation:
- Current implementation submits per-shot tasks (one call per shot).
- If batching multiple shots in one model request, input-token duplication (system prompt repeated N times) is reduced.
- If using provider-level Batch API, token price can be reduced further but with async/latency tradeoffs.

Implementation recommendation:
- keep per-shot fallback path for reliability and retry granularity,
- add grouped/batched path for cost optimization in high-volume runs.
