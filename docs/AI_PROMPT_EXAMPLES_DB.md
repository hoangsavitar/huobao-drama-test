# AI Prompt Examples from Real Data

This file contains practical examples (DB-shaped) for debugging output quality and contract regressions.
Use this with:
- `docs/EPISODE_PRODUCTION_GUIDE.md`
- `docs/AI_PROMPT_AND_API_REFERENCE.md`

---

## 1) Character Extraction Example

Source prompt function:
- `GetCharacterExtractionPrompt(style, aspectRatio)`

Expected output shape (example):
```yaml
name: Seo-yeon
role: female_lead
appearance: >
  A woman in her early 30s with a naturally graceful and slender build...
  She wears an elegant white evening gown.
personality: calm, resilient
description: polished public persona with hidden conflict
```

What to validate:
- stable visual anchors (face/clothes),
- no contradictory age/gender details,
- language is consistent with downstream image generation.

---

## 2) Scene Extraction Example (Background-Only)

Source prompt function:
- `GetSceneExtractionPrompt(style, aspectRatio)`

Expected output shape:
```yaml
location: Chairman's Office
time: Morning
prompt: >
  A modern luxury office bathed in sharp morning sunlight...
  pure background, no people, no characters, empty scene.
  image ratio 16:9.
```

What to validate:
- no person tokens,
- location/time specificity,
- explicit image ratio intent.

---

## 3) Storyboard Breakdown Example

Source prompt function:
- `GetStoryboardSystemPrompt()`

Expected shot sample:
```yaml
storyboard_number: 12
shot_type: Medium Shot
angle: Eye level
movement: Pan
action: Camera pans across press wall and stage lights.
dialogue: null
atmosphere: Warm golden launch-event lighting.
duration: 8
```

What to validate:
- one action unit per shot,
- duration in expected range,
- camera fields are populated where possible.

---

## 4) First-Frame Prompt Example

Source prompt function:
- `GetFirstFramePrompt(style, aspectRatio)`

Expected generated prompt sample:
```text
Photorealistic Korean drama style, 9:16 aspect ratio,
eye-level medium close-up of Jae-hyun standing at a lectern,
warm stage lighting, soft bokeh event hall background,
static pre-action frame, no motion blur.
```

What to validate:
- static "moment-before-action" phrasing,
- correct style and ratio,
- character consistency hints survive merge with scene refs.

---

## 5) Image Generation Request Example

Endpoint:
- `POST /api/v1/images`

Typical request:
```json
{
  "drama_id": "5",
  "storyboard_id": 167,
  "image_type": "storyboard",
  "frame_type": "first",
  "prompt": "Photorealistic Korean drama style, 9:16 aspect ratio, ...",
  "model": "gemini-2.5-flash-image",
  "reference_images": [
    "/static/scenes/scene_21.jpg",
    "/static/characters/seo_yeon.jpg",
    "/static/characters/jae_hyun.jpg"
  ]
}
```

Typical completed record fields:
```yaml
status: completed
image_url: /static/images/20260323_100935_gen_28.jpg
local_path: images/20260323_100935_gen_28.jpg
width: 1440
height: 2560
```

---

## 6) Video Generation Example

Endpoint:
- `POST /api/v1/videos`

Example payload:
```json
{
  "drama_id": "5",
  "storyboard_id": 167,
  "prompt": "Slow pan across stage before speech starts...",
  "reference_mode": "single",
  "image_url": "/static/images/20260323_100935_gen_28.jpg",
  "duration": 5,
  "aspect_ratio": "9:16"
}
```

Expected runtime pattern:
- returns task record quickly,
- frontend polls status,
- final `video_url` populated when completed.

---

## 7) Debug Checklist from Examples

If output quality drifts:
1. Compare generated text with this file's expected structure.
2. Verify style key and aspect ratio were included at prompt creation time.
3. Verify frame prompt upsert path saved the latest prompt.
4. Verify reference images were attached for storyboard image generation.
5. Verify no stale cache values override DB values on page reload.
