# Async Workflow

**Summary**: How the system handles long-running AI tasks via asynchronous processing and polling.

**Sources**: [[raw/ARCHITECTURE.md]]

**Last updated**: 2026-04-23

---

Generating images and videos is time-consuming, so the system uses an asynchronous job pattern.

## Life Cycle
1. **Client POST**: Sends a request to start a task (e.g., `POST /images`).
2. **Server Immediate Response**: Returns `pending` status and a `task_id`.
3. **Background Processing**: A Goroutine handles the actual AI provider call.
4. **Client Polling**: The frontend periodically checks status via `GET /api/v1/tasks/:taskId`.
5. **Completion**: Once status is `completed`, the client fetches the final resource URL.

## Partial Success
In batch operations (like generating all shots for an episode), the system is designed to allow **Partial Success**. A failure in one shot should not block the rest of the pipeline.

## Related pages
- [[architecture-overview]]
- [[db-schema]]
