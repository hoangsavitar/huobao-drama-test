# Quick Fix Playbook

This playbook is for high-speed debugging and safe hotfixes.
Use it when something is broken and you need the shortest path to root cause.

---

## 0) Triage Checklist (Always)

1. Reproduce once with exact page + action path.
2. Confirm if bug is:
   - UI-only
   - API contract
   - service logic
   - async/polling/cache race
3. Trace flow:
   - page -> frontend API -> handler -> service -> model update
4. Verify IDs are not mixed (`string` vs `number`) across map keys/comparisons.

---

## 1) Prompt Not Updating Across Pages

### Symptoms
- Edit prompt in one page, open another page, old prompt appears.
- Regenerate prompt but UI still shows stale text.

### Fast checks
1. Verify `PUT /storyboards/:id/frame-prompt` is called on save/autosave.
2. Verify `GET /storyboards/:id/frame-prompts` returns the new prompt.
3. Verify frontend cache load path overwrites stale session/local cache from DB.

### Typical fix points
- Frontend autosave/watch in `ProfessionalEditor.vue`.
- Backend upsert handler in `api/handlers/frame_prompt_query.go`.

---

## 2) Prompt Mix-Up Between Shots

### Symptoms
- Shot 7 displays text from shot 6 (or vice versa).

### Fast checks
1. Cache key must include both `storyboard_id` and `frame_type`.
2. All map reads/writes should normalize ID type (`Number(id)` or `String(id)` consistently).
3. Verify grouped prompt map from episode endpoint uses same key normalization.

### Typical fix points
- `EpisodeWorkflow.vue` / `ProfessionalEditor.vue` keyed maps.
- compare operations like `s.id === storyboardId` with explicit casting.

---

## 3) Aspect Ratio Wrong (9:16 not applied)

### Symptoms
- Drama set to portrait but prompts/images/videos still appear landscape.

### Fast checks
1. Drama update request includes `aspect_ratio`.
2. DB row for drama has `aspect_ratio = 9:16`.
3. Server restarted after backend schema/service changes.
4. Prompt builder receives ratio argument (`prompt_i18n` function calls).
5. Video generation request includes `aspect_ratio`.
6. UI ratio rendering uses dynamic variable (not hardcoded CSS ratio).

### Typical fix points
- `domain/models/drama.go`, `drama_service.go`
- `frame_prompt_service.go`, `prompt_i18n.go`
- `image_generation_service.go`, `storyboard_composition_service.go`
- `GenerateVideoDialog.vue`, `ProfessionalEditor.vue`

---

## 4) Batch First-Frame Prompt Fails Partially

### Symptoms
- Some selected shots get prompts, others not.

### Fast checks
1. Confirm all selected storyboard IDs are valid for episode.
2. Check each submitted task ID exists and reaches terminal status.
3. Reload episode prompt map after all tasks complete.

### Typical fix
- Keep partial-success behavior:
  - do not fail whole batch when one shot fails.
  - show done/failed counts.
  - allow rerun on failed subset.

---

## 5) Batch Shot Image Status Not Updating

### Symptoms
- Jobs submitted but "Shot Image" status column stays `None`.

### Fast checks
1. Ensure status source includes `composed_image` and `image_generation_status`.
2. Ensure episode shot list is enriched from `/episodes/:id/storyboards`.
3. Poll/reload after submission long enough for async completion.

### Typical fix points
- `EpisodeWorkflow.vue` enrich function and status column logic.
- `StoryboardCompositionService.GetScenesForEpisode` response shape.

---

## 6) Export Shot Images ZIP Missing Files

### Symptoms
- ZIP downloaded but missing some shot images.

### Fast checks
1. `GET /images` pagination fetched all pages (not only first page).
2. Filter includes correct `storyboard_id` set and `status=completed`.
3. URLs are fetchable from browser origin (CORS/static path).
4. Dedupe logic did not accidentally drop composed image.

### Typical fix points
- `web/src/utils/exportShotImagesZip.ts`
- Export trigger in `EpisodeWorkflow.vue`

---

## 7) Character/Scene Prompt Missing Ratio

### Symptoms
- Character or scene images ignore portrait framing intent.

### Fast checks
1. Prompt string includes explicit ratio phrase.
2. Request `size` defaults are derived from `Drama.aspect_ratio`.
3. Prompt orchestration path still appends style and ratio after recent refactor.

### Typical fix points
- `character_library_service.go`
- `storyboard_composition_service.go`
- `image_generation_service.go`

---

## 8) Decision Tree (Fast Path)

1. **UI wrong only?**
   - Check computed/state/render path first.
2. **UI + API payload wrong?**
   - Fix frontend request builder.
3. **API payload correct but DB wrong?**
   - Fix handler/service persistence.
4. **DB correct but generated output wrong?**
   - Fix prompt contract or provider request assembly.
5. **Intermittent only?**
   - Suspect async polling/race/cache and ID normalization.

---

## 9) Safe Hotfix Protocol

1. Reproduce bug and capture one concrete failing ID.
2. Patch smallest responsible layer first.
3. Run build/lint for touched files.
4. Validate one happy path + one failure path.
5. Update docs immediately:
   - contract changes -> `AI_PROMPT_AND_API_REFERENCE.md`
   - incident pattern -> this file
   - ownership/flow change -> `EPISODE_PRODUCTION_GUIDE.md`

