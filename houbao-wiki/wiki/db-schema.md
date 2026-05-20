# DB Schema

**Summary**: Details of the core database entities and their relationships.

**Sources**: [[raw/ARCHITECTURE.md]]

**Last updated**: 2026-04-23

---

The system uses GORM for persistence. Below is the entity graph and critical models.

## Entity Graph
- **Drama** `1 -> N` **Episode**
- **Episode** `1 -> N` **Storyboard** (Shot)
- **Drama** `1 -> N` **Character** / **Scene**
- **Storyboard** `1 -> N` **FramePrompt**

## Critical Models
### Drama
- `style`: Critical for image generation (e.g., `ghibli`).
- `aspect_ratio`: Defines the dimensions for all generated media.

### Storyboard (Shot)
- Contains `action`, `dialogue`, and `atmosphere` extracted from the script.
- Links to `composed_image` and `video_url`.

### FramePrompt
- The "Source of Truth" for individual frame prompts.
- `frame_type`: `first`, `key`, `last`, etc.

## Related pages
- [[architecture-overview]]
- [[prompt-contracts]]
