# AI Integration Details

**Summary**: Deep dive into the LLM integration, prompt contracts, and data shapes for content generation.

**Sources**: [[raw/AI_INTEGRATION.md]]

**Last updated**: 2026-04-23

---

The "heart" of the system is the AI integration layer, which manages how the script is transformed into visual media.

## Prompt Source of Truth
All base prompts are located in `application/services/prompt_i18n.go`. This ensures consistency across different styles and languages.

## Key Pipelines
1. **Character Extraction**: Isolating character traits for consistency.
2. **Scene Extraction**: Creating pure backgrounds without human elements.
3. **Storyboard Generation**: Splitting the script into timed shots.

## Contracts
Every AI interaction must follow strict [[prompt-contracts]] to ensure valid JSON responses and proper media generation.

## Related pages
- [[prompt-contracts]]
- [[db-schema]]
- [[architecture-overview]]
