# Runbook (Local Dev and Maintenance)

This runbook is for day-to-day operation, debugging, and safe change rollout.

---

## 1) Local Startup

### Backend
From repo root:
- `go run main.go`

Notes:
- AutoMigrate runs on startup.
- Restart backend after changing:
  - `domain/models/*`
  - `application/services/prompt_i18n.go`
  - service logic that changes runtime behavior.

### Frontend
From `web/`:
- `npm install`
- `npm run dev`

---

## 2) Build and Basic Validation

### Backend validation
- `go build ./application/... ./api/... ./domain/... ./cmd/...`

### Frontend validation
- lint/type checks may include pre-existing errors in this repo.
- For targeted validation, at least run:
  - changed-file lint checks in IDE,
  - manual page smoke tests for impacted flows.

---

## 3) Common Post-Change Smoke Tests

After touching prompt/image/video flows, run this minimal checklist:

1. Open episode workflow page.
2. Batch generate first-frame prompt for 2-3 shots.
3. Verify first-frame status updates.
4. Batch generate shot images.
5. Verify shot image status column (`Ready/Generating/Failed`).
6. Open professional editor:
   - confirm latest prompt text loads,
   - generate one image and one video.
7. If aspect-ratio-related change:
   - set drama to `9:16`,
   - verify prompt text ratio + generated output orientation.

---

## 4) Prompt Change Procedure

When changing prompt behavior:

1. Edit `application/services/prompt_i18n.go`.
2. Rebuild backend.
3. Restart backend.
4. Test one extraction/generation flow end-to-end.
5. Update docs:
   - `AI_PROMPT_AND_API_REFERENCE.md`
   - `AI_PROMPT_EXAMPLES_DB.md` if output format changed.

---

## 5) Migration Procedure (Schema Fields)

1. Add/update model field in `domain/models`.
2. Confirm model is in `infrastructure/database/database.go` AutoMigrate list.
3. Start backend to apply migration.
4. Validate persisted value via API read-back.
5. Add migration notes in `docs/DATA_MIGRATION.md` when needed.

---

## 6) Incident Response (Quick)

### Prompt stale/mismatch
- Check `PUT /storyboards/:id/frame-prompt` path.
- Verify DB and reload cache path.

### Wrong aspect ratio
- Verify `drama.aspect_ratio` persisted.
- Verify request payload includes ratio for video.
- Verify image size default mapping in backend.

### Batch status not moving
- Confirm async task state is polled.
- Confirm composition endpoint includes current status fields.

---

## 7) Release Safety Checklist

Before merging feature changes:

1. No contract mismatch between frontend API wrapper and backend handler.
2. No mixed ID comparison bugs (`string` vs `number`) in touched code.
3. Async flows handle partial failures (batch operations).
4. Docs updated for:
   - architecture impact,
   - contract updates,
   - known failure patterns.

