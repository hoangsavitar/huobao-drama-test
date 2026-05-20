# Prompt Contracts

**Summary**: Defined JSON shapes and rules for communication between the system and LLM providers.

**Sources**: [[raw/AI_INTEGRATION.md]]

**Last updated**: 2026-04-23

---

To ensure the system can parse AI responses correctly, all LLMs must return valid JSON without extra markdown formatting.

## Extraction Contracts

### Character Extraction
- **Goal**: Consistent character descriptions.
- **Rule**: `appearance` must not include background info.
- **Fields**: `name`, `role`, `appearance`, `personality`, `description`.

### Scene Background Extraction
- **CRITICAL RULE**: No humans or actions in scene prompts.
- **Fields**: `location`, `time`, `prompt`.

### Storyboard Generation
- **Goal**: Split script into timed shots.
- **Fields**: `storyboard_number`, `shot_type`, `angle`, `movement`, `action`, `dialogue`, `atmosphere`, `duration`.

## Payload Contracts
Requests to Image/Video providers must include:
- `drama_id`
- `prompt`
- `reference_images` (for consistency)
- `aspect_ratio` (forced by backend)

## Related pages
- [[ai-integration-details]]
- [[db-schema]]
