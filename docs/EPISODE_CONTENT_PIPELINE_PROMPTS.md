# Episode Content -> Extract Characters/Scenes -> Split Shots

This document describes the exact backend/frontend flow and prompt construction used when you input script into Episode Content, then run:

- `Extracted Characters (This Episode)`
- `Extracted Scenes (This Episode)`
- `Split Shots`

All details below are based on the current code in this repo.

---

## 1) End-to-End Flow (what happens after you input Episode Content)

## A. Save Episode Content

Frontend (`EpisodeWorkflow.vue`) saves script to episode:

- API: `PUT /api/v1/dramas/:dramaId/episodes`
- Frontend API helper: `web/src/api/drama.ts` -> `saveEpisodes()`
- Script is stored in `episodes.script_content`.

---

## B. Click "Extract Characters And Scenes"

Frontend calls 2 async tasks in parallel:

1) Character extraction task
- API: `POST /api/v1/generation/characters`
- Frontend helper: `generationAPI.generateCharacters()`
- Payload shape:
  - `drama_id`
  - `episode_id`
  - `outline` (current episode script content)
  - `count` (0 -> default 5 in backend)
  - `model` (selected text model)

2) Scene/background extraction task
- API: `POST /api/v1/images/episode/:episode_id/backgrounds/extract`
- Frontend helper: `dramaAPI.extractBackgrounds()`
- Payload shape:
  - `model` (selected text model)
  - style may be auto-filled in backend from drama style

Then frontend polls both task IDs:
- API: `GET /api/v1/tasks/:taskId` (via `generationAPI.getTaskStatus()`)

When character task is completed:
- frontend parses task result JSON (`result.characters`)
- calls `PUT /api/v1/dramas/:dramaId/characters` (`saveCharacters`)
- backend dedupes by name per drama and associates characters to episode.

When scene task is completed:
- scenes are already persisted by backend task processor.

---

## C. Click "Split Shots"

Frontend:
- API: `POST /api/v1/episodes/:episode_id/storyboards`
- Payload: `{ model: selectedTextModel }`
- polls task until completed
- on success jumps to `ProfessionalEditor`.

Backend:
- reads episode script content
- loads available drama-level characters and scenes
- builds one large prompt
- calls text model
- parses JSON result
- regenerates storyboards for episode
- saves image/video prompt fields per shot.

---

## 2) Full Prompt Construction: Character Extraction

Source:
- `application/services/script_generation_service.go`
- `application/services/prompt_i18n.go`

Backend builds:
- `systemPrompt = GetCharacterExtractionPrompt(style, aspectRatio)`
- `userPrompt = FormatUserPrompt("character_request", outlineText, count)`

`outlineText` is:
- request outline if provided, else `drama_info_template` (title/summary/genre).

### 2.1 System Prompt (English, verbatim)

```text
You are a professional character analyst, skilled at extracting and analyzing character information from scripts.

Your task is to extract and organize detailed character settings for all characters appearing in the script based on the provided script content.

Requirements:
1. Extract all characters with names (ignore unnamed passersby or background characters)
2. For each character, extract:
   - name: Character name
   - role: Character role (main/supporting/minor)
   - appearance: Physical appearance description (150-300 words)
   - personality: Personality traits (100-200 words)
   - description: Background story and character relationships (100-200 words)
3. Appearance must be detailed enough for AI image generation, including: gender, age, body type, facial features, hairstyle, clothing style, etc. but do not include any scene, background, environment information
4. Main characters require more detailed descriptions, supporting characters can be simplified
- **Style Requirement**: <styleLabel(style)>
- **Image Ratio**: <aspectRatio>
Output Format:
**CRITICAL: Return ONLY a valid JSON array. Do NOT include any markdown code blocks, explanations, or other text. Start directly with [ and end with ].**
Each element is a character object containing the above fields.
```

### 2.2 User Prompt Template (English, verbatim)

```text
Script content:
%s

Please extract and organize detailed character profiles for up to %d main characters from the script.
```

---

## 3) Full Prompt Construction: Scene Extraction (Extracted Scenes This Episode)

Source:
- `application/services/image_generation_service.go`
- `application/services/prompt_i18n.go`

Backend builds:
- `systemPrompt = GetSceneExtractionPrompt(style, aspectRatio)`
- user message:
  - `script_content_label`
  - raw script content
  - strict JSON format instructions + examples

### 3.1 System Prompt (English, verbatim)

```text
[Task] Extract all unique scene backgrounds from the script

[Requirements]
1. Identify all different scenes (location + time combinations) in the script
2. Generate detailed **English** image generation prompts for each scene
3. **Important**: Scene descriptions must be **pure backgrounds** without any characters, people, or actions
4. Prompt requirements:
   - Must use **English**, no Chinese characters
   - Detailed description of scene, time, atmosphere, style
   - Must explicitly specify "no people, no characters, empty scene"
   - Must match the drama's genre and tone
   - **Style Requirement**: <styleLabel(style)>
   - **Image Ratio**: <aspectRatio>

[Output Format]
**CRITICAL: Return ONLY a valid JSON array. Do NOT include any markdown code blocks, explanations, or other text. Start directly with [ and end with ].**

Each element containing:
- location: Location (e.g., "luxurious office")
- time: Time period (e.g., "afternoon")
- prompt: Complete English image generation prompt (pure background, explicitly stating no people)
```

### 3.2 User Prompt Format Block (English path, verbatim structure)

The backend appends this block (contains expected JSON and examples):

- `[Output JSON Format]`
- object with `backgrounds: [ { location, time, atmosphere, prompt } ]`
- one correct example
- multiple wrong examples that include characters
- final instruction: strict JSON, all fields English.

The final user prompt sent to model is:

```text
<systemPrompt from 3.1>

【Script Content】
<script content>

<format instructions + examples>
```

---

## 4) Full Prompt Construction: Split Shots (Storyboard Generation)

Source:
- `application/services/storyboard_service.go`
- `application/services/prompt_i18n.go`

Backend collects:
- script content from `episodes.script_content` (or fallback episode description)
- all drama characters (`id`, `name`)
- all drama scenes (`id`, `location`, `time`)

Then builds:
- `systemPrompt = GetStoryboardSystemPrompt()`
- labels/constraints from `FormatUserPrompt(...)`
- huge fixed instruction block in English (shot schema, dialogue rules, duration rules, quality constraints, anti-omission rules).

### 4.1 Storyboard System Prompt (English, verbatim summary)

`GetStoryboardSystemPrompt()` defines:
- role: senior film storyboard artist
- independent action unit splitting
- shot-type standards, camera movement rules
- emotion/intensity schema
- required JSON fields and output constraints
- strict JSON-only output.

(See full literal in `prompt_i18n.go` under `GetStoryboardSystemPrompt()`.)

### 4.2 User Prompt Composition (exact structure)

The user prompt is assembled like:

```text
<systemPrompt>

<script_content_label>
<scriptContent>

<task_label><task_instruction>

<character_list_label>
<characterListJson>

<character_constraint>

<scene_list_label>
<sceneListJson>

<scene_constraint>

【Original Script】
<scriptContent>

<VERY LARGE FIXED STORYBOARD SPEC BLOCK>
```

That fixed block includes:
- required storyboard fields (`shot_number`, `title`, `shot_type`, `angle`, `time`, `location`, `scene_id`, `movement`, `action`, `dialogue`, `result`, `atmosphere`, `emotion`, `duration`, `bgm_prompt`, `sound_effect`, `characters`, `is_primary`)
- dialogue formatting requirements
- strict character/scene ID constraints
- explicit duration estimation algorithm (4-12s range with formula)
- anti-omission rules (must not skip plot/dialogue)
- minimum detail constraints for `time/location/action/result/atmosphere`.

---

## 5) Technical Handling Details (important behavior)

## 5.1 Task and Async model

- Character extraction: creates task `character_generation`, processes in goroutine.
- Scene extraction: creates task `background_extraction`, processes in goroutine.
- Storyboard split: creates task `storyboard_generation`, processes in goroutine.

All are polled by frontend via `/tasks/:id`.

## 5.2 Parsing strategy

- Uses `utils.SafeParseAIJSON(...)`.
- Supports both:
  - pure array format
  - object-wrapped format (e.g. `{ backgrounds: [...] }` or `{ storyboards: [...] }` depending service).

## 5.3 Save strategy

Characters:
- dedupe by `(drama_id, name)` before create.
- episode association done via many-to-many append.

Scenes:
- during re-extract, backend deletes existing episode scenes first, then recreates from AI result.

Storyboards:
- generation recreates episode storyboards from AI result.
- sets per-shot image and video prompts (`ImagePrompt`, `VideoPrompt`) using internal helper logic.

## 5.4 Model selection behavior

- If frontend passes a model, backend tries `GetAIClientForModel("text", model)`.
- If model-specific lookup fails, backend falls back to default text client.

---

## 6) Endpoint Map Used in This Pipeline

- `POST /api/v1/generation/characters`
- `POST /api/v1/images/episode/:episode_id/backgrounds/extract`
- `POST /api/v1/episodes/:episode_id/storyboards`
- `GET /api/v1/tasks/:taskId`
- `PUT /api/v1/dramas/:dramaId/characters` (frontend persistence after character task done)
- `PUT /api/v1/dramas/:dramaId/episodes` (save script content)

---

## 7) Quick Debug Checklist

If extracted characters/scenes/shots look wrong:

1. Check `episodes.script_content` is non-empty.
2. Confirm selected text model exists and is active.
3. Check task result JSON parse path (array vs object wrapper).
4. For scenes, verify re-extract deletion behavior did not remove expected old scenes.
5. For shots, inspect generated `scene_id` and `characters` numeric IDs against available lists.
6. Inspect backend logs where it prints full prompt for scene extraction:
   - `"=== AI Prompt for Background Extraction (extractBackgroundsFromScript) ==="`

