# Async Tasks and Polling

This document standardizes how async generation tasks are submitted, tracked, and surfaced in UI.

---

## 1) Why Async

Prompt/image/video generation can take seconds to minutes.
API must return quickly, so generation is modeled as async tasks with polling.

---

## 2) Task Lifecycle (Canonical)

States:
- `pending`
- `processing`
- `completed`
- `failed`

General lifecycle:
1. Client submits generation request.
2. Server creates task/job record.
3. Worker/service executes generation.
4. Task status moves to terminal state.
5. Client polls status and refreshes dependent resources.

---

## 3) Current Async Flows

### A) Frame prompt generation
- Submit:
  - `POST /storyboards/:id/frame-prompt`
- Poll:
  - `taskAPI.getStatus(task_id)`
- Refresh:
  - `GET /episodes/:episode_id/frame-prompts`
  - `GET /storyboards/:id/frame-prompts`

### B) Image generation
- Submit:
  - `POST /images`
- Poll:
  - image status list or task status depending UI flow
- Refresh:
  - composition/shot list and image lists

### C) Video generation
- Submit:
  - video generation endpoint
- Poll:
  - video status records
- Refresh:
  - video list / timeline assets

---

## 4) Polling Strategy Guidelines

Recommended defaults:
- interval: 2-3 seconds
- max attempts: 60 (or feature-specific)
- stop on terminal status
- tolerate transient request failures during polling

UI rules:
- show progress/loading state during polling
- avoid duplicate polling loops per same task
- clear timers on unmount/navigation

---

## 5) Batch Operations (Partial Success Contract)

Batch operations must not be all-or-nothing by default.

Required behavior:
1. submit/process per item independently
2. collect `done/failed` counts
3. surface summary to user
4. support rerun on failed subset

Examples:
- batch first-frame prompt
- batch shot image generation

---

## 6) Failure Patterns and Mitigation

### Duplicate submissions
- Cause: repeated clicks while loading not disabled.
- Mitigation: disable button while loading; idempotency checks if possible.

### Stale status after completion
- Cause: polling ends but dependent list not refreshed.
- Mitigation: always refresh source read model after terminal state.

### Lost updates due to cache
- Cause: local/session state overrides DB.
- Mitigation: prefer DB refresh after task completion; deterministic cache keying.

---

## 7) Implementation Checklist for New Async Feature

1. Define submit endpoint and task payload.
2. Define terminal output resource and refresh endpoint.
3. Add UI loading/progress states.
4. Add polling loop with max attempts and cleanup.
5. Add partial failure handling for batch.
6. Add doc entry in this file and `API_CONTRACTS.md`.

