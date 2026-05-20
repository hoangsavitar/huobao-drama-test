---
name: brainstorming
description: Explore and shape Huobao Drama product, UX, narrative, prompt, AI workflow, branching story, storyboard, image/video generation, or production-pipeline ideas before implementation; use when the request is ambiguous, creative, or design-heavy.
---

# Huobao Drama Brainstorming

Use this skill before implementation when the user is shaping an idea rather than asking for a concrete code change.

## Ground The Discussion In The Repo

Quickly load the relevant project context:
- Product and architecture: `README.md`, `docs/ARCHITECTURE.md`
- Production flow: `docs/DEVELOPMENT_GUIDE.md`
- Prompt contracts: `docs/AI_INTEGRATION.md`
- Narrative flow: `docs/NARRATIVE_TO_STORYBOARD_FLOW.md`

Then identify which production stage the idea affects:
- Create drama and metadata
- Narrative graph generation
- Episode script authoring
- Character, outfit, prop, or scene extraction
- Storyboard splitting
- Frame prompt generation
- Image/video generation
- Timeline/export/finalization
- Settings and provider configuration

## Brainstorming Output

For creative/product requests, produce:
- Goal: what outcome the user wants.
- Current repo fit: where the idea plugs into existing flow.
- Options: 2-3 viable designs, with the recommended option first.
- Tradeoffs: complexity, prompt reliability, API/schema changes, UI effort, generation cost, regression risk.
- Assumptions and open questions.
- Decision log if the discussion results in a chosen direction.

## Project Constraints To Keep Visible

- The backend is the source of business rules; handlers stay thin.
- Long-running generation uses async tasks and polling.
- Prompt outputs must be machine-parseable JSON where downstream code expects JSON.
- Scene background generation must remain character-free.
- Character identity must stay canonical across episodes; aliases should not create duplicate people.
- `style` and `aspect_ratio` shape prompt/media output and should not be dropped.
- UI work should improve the actual production tool, not turn it into a marketing page.

## Handoff

When the user chooses an option, hand off to `feature-planning` for implementation steps and verification criteria. Do not start coding from a still-ambiguous brainstorm.
