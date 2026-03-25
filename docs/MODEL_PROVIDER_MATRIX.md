# Model and Provider Matrix

This file records which model/provider paths are used by feature type and how to choose them.
Keep it updated when changing defaults or adding providers.

---

## 1) Feature to Model Type Mapping

| Feature | Layer | Typical API Type | Source of Config |
|---|---|---|---|
| Storyboard / extraction text | Backend service | text generation | AI config + selected text model |
| Frame prompt generation | Backend service | text generation | selected model or default text model |
| Character/scene/shot image generation | Backend image service | image generation | selected image model + provider |
| Shot video generation | Backend video service | video generation | selected video model + provider |

---

## 2) Prompt Builder Ownership

- Prompt templates: `application/services/prompt_i18n.go`
- Feature orchestration:
  - frame prompts: `frame_prompt_service.go`
  - image: `image_generation_service.go`
  - video: `video_generation_service.go`

Model/provider changes should not bypass prompt contract rules.

---

## 3) Provider Selection Rules (Current Pattern)

1. Frontend allows model selection per flow (text/image/video where applicable).
2. Backend resolves provider/model:
   - explicit request model/provider if provided,
   - otherwise fallback to configured default.
3. Service appends style/aspect-ratio constraints from drama context.

---

## 4) Cost/Quality Throughput Guidance

### Text generation (prompts/extraction)
- Cost-sensitive batch workloads:
  - prefer lower-cost flash-lite variants where quality is acceptable.
- Quality-sensitive storyboard decomposition:
  - prefer stronger reasoning models when scripts are complex.

### Image generation
- Use consistent references for character/scene stability.
- High-quality model for key assets, cheaper model for exploratory reruns.

### Video generation
- Keep prompt concise and physically feasible.
- Duration and reference mode strongly affect result consistency.

---

## 5) Batch vs Single Request Guidance

For first-frame prompt generation:
- Per-shot requests:
  - better fault isolation and retries.
- Grouped requests:
  - less repeated system context token usage.
- Provider-level Batch APIs:
  - usually lower token unit price but async completion tradeoff.

Recommended architecture:
- keep single-shot path as fallback,
- add grouped path for cost optimization,
- include robust parser + partial retry strategy.

---

## 6) Change Checklist for Model Path Updates

When changing model/provider defaults:

1. Update frontend selector defaults where used.
2. Update backend fallback logic.
3. Verify prompt contract still satisfied.
4. Run one smoke test per affected feature:
   - extraction
   - frame prompt
   - image
   - video
5. Update this matrix and relevant docs.

