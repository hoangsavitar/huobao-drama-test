# Architecture Overview

**Summary**: Technical map of the system including layers, data models, and asynchronous flow.

**Sources**: [[raw/ARCHITECTURE.md]]

**Last updated**: 2026-04-23

---

The system follows a **Clean Architecture** pattern with a clear separation between the Vue 3 frontend and the Gin-based Go backend.

## System Layers
- **Frontend**: Vue 3 components and API callers.
- **HTTP Handlers**: Thin layer for request/response handling.
- **Business Services**: The core logic and AI provider integrations.
- **Persistence**: GORM models defining the [[db-schema]].

## Data Flow
The system heavily relies on [[async-workflow]] for long-running tasks like image and video generation.

- **Entity Hierarchy**: Drama -> Episode -> Storyboard (Shot).
- **Media Mapping**: Storyboards are the primary unit for prompt and media generation.

## Related pages
- [[db-schema]]
- [[async-workflow]]
- [[ai-integration-details]]
