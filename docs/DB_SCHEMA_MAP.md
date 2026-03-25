# DB Schema Map (Operational)

This document summarizes core entities, key fields, and relationships used by the production pipeline.
It is not a full SQL dump; it is a maintenance-focused map.

---

## 1) Core Entity Graph

- `Drama` 1 -> N `Episode`
- `Episode` 1 -> N `Storyboard`
- `Drama` 1 -> N `Character`
- `Drama` 1 -> N `Scene`
- `Storyboard` N <-> N `Character` (many-to-many)
- `Storyboard` N <-> N `Prop` (many-to-many)
- `Storyboard` 1 -> N `FramePrompt`
- `Storyboard/Scene/Character` -> N `ImageGeneration` (typed by `image_type`)
- `Storyboard` -> N `VideoGeneration`

---

## 2) Tables and Critical Fields

### `dramas`
- `id`
- `title`
- `style`
- `aspect_ratio` (`16:9` / `9:16`)
- `status`

### `episodes`
- `id`
- `drama_id`
- `episode_number`
- `script_content`
- `status`

### `storyboards`
- `id`
- `episode_id`
- `scene_id`
- `storyboard_number`
- `action`, `dialogue`, `atmosphere`
- `image_prompt`, `video_prompt`
- `duration`
- `composed_image`
- `video_url`

### `characters`
- `id`
- `drama_id`
- `name`
- `appearance`, `personality`, `description`
- `image_url`, `local_path`

### `scenes`
- `id`
- `drama_id`
- `episode_id`
- `location`, `time`
- `prompt` (background-only prompt)
- `image_url`, `local_path`
- `status`

### `frame_prompts`
- `id`
- `storyboard_id`
- `frame_type` (`first|key|last|panel|action`)
- `prompt`
- `layout`

### `image_generations`
- `id`
- `drama_id`
- `storyboard_id?`
- `scene_id?`
- `character_id?`
- `image_type` (`storyboard|scene|character`)
- `frame_type?`
- `prompt`
- `model`, `provider`
- `size`, `width`, `height`
- `status`
- `image_url`, `local_path`

### `video_generations`
- `id`
- `drama_id`
- `storyboard_id?`
- `prompt`
- `provider`, `model`
- `aspect_ratio`
- `duration`, `fps`
- `status`
- `video_url`, `local_path`

---

## 3) Source-of-Truth Rules

1. Drama-level visual config:
   - `dramas.style`, `dramas.aspect_ratio`
2. Prompt persistence:
   - `frame_prompts` is source of truth for frame prompts.
3. Shot image readiness:
   - prefer composed/status projection from storyboard composition read model.
4. Raw job history:
   - `image_generations` / `video_generations` contain async lifecycle records.

---

## 4) Index/Key Considerations for Features

Operationally important lookup keys:
- `episodes.drama_id`
- `storyboards.episode_id`
- `storyboards.scene_id`
- `frame_prompts.storyboard_id + frame_type`
- `image_generations.drama_id`
- `image_generations.storyboard_id`
- `video_generations.storyboard_id`

When adding new feature filters, ensure query path has practical index coverage.

---

## 5) Migration Notes

When introducing new field:
1. Add to model in `domain/models`.
2. Ensure model stays in AutoMigrate list.
3. Restart backend to apply migration.
4. Validate read/write path at API and UI levels.
5. Update:
   - `docs/DATA_MIGRATION.md`
   - this file if field is operationally important.

