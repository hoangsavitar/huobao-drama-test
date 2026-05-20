---
name: karpathy-guidelines
description: Project coding guardrails for Huobao Drama. Use when writing, reviewing, or refactoring code to keep changes surgical, simple, verifiable, and aligned with the existing Go/Gin/GORM backend, Vue 3 frontend, AI prompt contracts, and async media workflow.
---

# Huobao Drama Coding Guardrails

Use these guardrails alongside implementation, review, and refactoring work.

## Think Before Editing

Before changing files:
- State what behavior is being changed or preserved.
- Identify the exact code path: Vue view/API/type -> route -> handler -> service -> model.
- Surface any ambiguity around DB schema, API contract, prompt output, async task result, or provider behavior.
- Prefer local code and docs over assumptions.

## Keep It Surgical

- Touch only files needed for the request.
- Match existing Go, Vue, TypeScript, Element Plus, and service patterns.
- Avoid speculative abstractions and broad rewrites.
- Do not mix feature work with refactoring unless a small preparatory refactor is required and verified.
- Clean up imports, variables, and helpers made unused by your own change.
- Leave unrelated user changes in place.

## Preserve Huobao Contracts

- Handlers parse/validate and delegate; services own business logic.
- `domain/models` JSON tags must align with frontend TypeScript fields.
- Route changes must be reflected in `web/src/api`.
- New visible UI text should follow existing i18n usage.
- AI text output that feeds Go parsers must remain JSON-only.
- Prompt changes must preserve required downstream fields and `style`/`aspect_ratio`.
- Async tasks must have clear pending/processing/completed/failed behavior.

## Simplicity Test

Before finishing, ask:
- Can this be implemented by extending the current service instead of creating a parallel system?
- Is this field/helper used by more than one place? If not, avoid an abstraction.
- Did the change add a new state that the UI cannot display?
- Did the change create a code path that cannot be tested without live AI?
- Did any old workflow lose a field, preload, relation, or polling update?

## Verification Mindset

Define success as a concrete check:
- Parser behavior: targeted Go test.
- Model/service behavior: package test.
- API/type sync: backend build plus frontend build/type check.
- UI behavior: focused manual smoke after build when needed.

Report skipped checks and known residual risk directly.
