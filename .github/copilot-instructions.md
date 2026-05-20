# Copilot Instructions for Huobao Drama

## Project Architecture & Data Flow
This is a monorepo containing a full-stack application (Go backend + Vue 3 frontend) applying Domain-Driven Design (DDD).
- **Backend (Go 1.23+)**:
  - `api/routes` -> `api/handlers/` (HTTP layer using Gin). Use `pkg/response` for standardized API responses.
  - `application/services/` (Business Logic layer).
  - `domain/models/` (Core entities like `Drama`, `Character`, `Scene`, `Storyboard`).
  - `infrastructure/` (Database implementations, external services like FFmpeg).
- **Frontend (Vue 3, Element Plus, Tailwind, Pinia)**: Located in the `web/` directory. Uses `<script setup>` syntax and Vite.

## Critical Developer Workflows
- **Running the Backend**: `go run main.go` in the project root.
- **Running the Frontend**: `pnpm dev` or `npm run dev` inside the `web/` directory.
- **Database Migrations**: Run `go run cmd/migrate/main.go` to apply schema updates.
- **Development Tools**: Auxiliary scripts are kept in `cmd/tools/` (e.g., `cmd/tools/export_prompts/main.go`).

## Coding Conventions
- **Dependency Injection**: Always inject dependencies into constructors (e.g., pass `*gorm.DB`, `*config.Config`, `*logger.Logger` to `New[Name]Handler` or `New[Name]Service`). 
- **Error Handling**: Use `pkg/logger` to log errors with context. Return standardized error responses via `response.Error(c, err)` or similar wrappers in `pkg/response` in your Gin handlers.
- **AI Integrations**: Prompts and AI interactions are managed in `application/services/` (e.g., `prompt_i18n.go`). When updating AI generation flows, keep prompts optimized for single-task extraction (e.g., Characters, Scenes, Storyboards).
- **Video Processing**: FFmpeg is used heavily for generation and merging (`infrastructure/external/ffmpeg/`). Treat video tasks as async background processes, utilizing `domain/models/task.go`.

## Prompts & AI Extraction Workflows
- **Prompt Management**: AI Prompts are centrally managed in `application/services/prompt_i18n.go` (via `PromptI18n`). Always update these methods if you need to tweak what the AI generates.
- **Extraction Logic**: The system processes uploaded scripts (`EPISODE_PRODUCTION_GUIDE.md`) to extract specific entities: `Characters`, `Scenes`, and `Storyboards` (Frames).
- **Enforcing Output**: AI prompts strictly enforce JSON array outputs containing specific schema fields (e.g., `Appearance` for characters, pure background descriptions without people for `Scenes`).
- **Context Handling**: Avoid overloading the LLM. Provide only single Episodes (500-1000 words maximum) for extraction to prevent hallucination.
- **Duration Automation**: Shot duration (seconds) is internally estimated by the AI and saved in the JSON response before FFmpeg rendering.
- **AI Reference Document**: Whenever generating or changing AI workflow logic, strictly adhere to `docs/AI_PROMPT_AND_API_REFERENCE.md` for proper Enum configurations (shot boundaries, transitions, camera specs) and payload schemas. Do NOT generate arbitrary parameter values.

## Adding Features Safely (Zero Data Loss)
- **Database Migrations**: When changing models in `domain/models/`, **NEVER drop columns or tables**. Always write additive SQL in `migrations/` or rely on safe GORM AutoMigrate for appending new fields.
- **Soft Deletes**: Rely on GORM's `gorm.DeletedAt` for soft deletes. Do not permanently delete User or Generation data.
- **Backward Compatibility**: If adding new properties to JSON APIs or extraction parsers, make them optional or provide default values to avoid breaking legacy data (`is_new_feature := req.Property != nil`).

## API Interaction & Payload Data Flow
- **Receiving Data (Requests)**: Endpoints in `api/handlers/` expect standard JSON payloads bound via `c.ShouldBindJSON(&req)`. Keep input structs strictly validated (use `binding:"required"` carefully to not break old frontends).
- **Sending Data (Responses)**: ALL API responses must go through `pkg/response` (e.g., `response.Success(c, data)` or `response.Error(c, err)`). The frontend strictly expects `{ code, message, data }` format.
- **Async Tasks**: For long operations like Video Merge or Batch AI Gen, the API immediately creates a `Task` (in DB) and returns a `task_id` for the frontend to poll status via WebSocket or HTTP, running the actual job via `application/services/task_service.go`.

## Contextual Hints
- The platform relies on correctly parsing script episodes (see `EPISODE_PRODUCTION_GUIDE.md`). When working on script models, remember the core workflow: Script Upload -> AI extraction -> Generate Portraits/Scenes -> Generate Storyboard Actions -> Video Merge.
