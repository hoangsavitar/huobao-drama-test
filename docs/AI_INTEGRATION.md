# AI Integration & Prompt Contracts (For AI Agents)

Tài liệu này là "trái tim" của hệ thống sinh nội dung, định nghĩa toàn bộ **Luồng tích hợp LLM, Cấu trúc Prompt, và Payload Data**. Bất kỳ chỉnh sửa nào liên quan đến chất lượng gen Text/Image/Video đều phải tham chiếu file này.

---

## 1. Prompt Source of Truth

> [!WARNING]
> Toàn bộ logic tạo Prompt gốc nằm tại một file duy nhất: `application/services/prompt_i18n.go`. Tuyệt đối không hardcode prompt ở layer Handler hay Frontend.

Các hàm tạo Prompt cốt lõi:
- **Storyboards:** `GetStoryboardSystemPrompt()`
- **Characters:** `GetCharacterExtractionPrompt(style, aspectRatio)`
- **Scenes:** `GetSceneExtractionPrompt(style, aspectRatio)`
- **Frame Prompts:** `GetFirstFramePrompt(...)`, `GetKeyFramePrompt(...)`, `GetActionSequenceFramePrompt(...)`

Hai biến môi trường bắt buộc phải có trong mọi Prompt sinh ảnh/video là `style` (VD: `ghibli`, `guoman3d`) và `aspect_ratio` (VD: `16:9`, `9:16`).

---

## 2. LLM Prompt Contracts & Data Shapes

Hệ thống xử lý AI thông qua 3 pipeline chính sau khi user nhập kịch bản (Script Content). Yêu cầu bắt buộc của mọi Text LLM là **phải trả về JSON hợp lệ** (không kèm markdown dư thừa).

### 2.1. Character Extraction (Trích xuất Nhân Vật)
**Luồng API:** `POST /api/v1/generation/characters`
- **Nhiệm vụ:** Tìm và mô tả chi tiết nhân vật từ kịch bản để tạo tính nhất quán (anchor) khi gen ảnh.
- **Contract:** Phải trả về JSON Array chứa thông tin nhân vật.
- **Dữ liệu chuẩn (Expected Data Shape):**
  ```json
  {
    "name": "Seo-yeon",
    "role": "female_lead",
    "appearance": "A woman in her early 30s with a naturally graceful build. She wears an elegant white evening gown.",
    "personality": "calm, resilient",
    "description": "polished public persona with hidden conflict"
  }
  ```
- **Rule:** Phần `appearance` cấm đưa thông tin cảnh/background vào.

### 2.2. Scene Background Extraction (Trích xuất Bối Cảnh)
**Luồng API:** `POST /api/v1/images/episode/:episode_id/backgrounds/extract`
- **Nhiệm vụ:** Tìm các bối cảnh (không gian + thời gian) để sinh ảnh nền.
- **Contract:** Phải trả về JSON Array.
- **Quy tắc Tối Thượng (Critical Rule):** Prompt của Scene **TUYỆT ĐỐI KHÔNG CÓ NGƯỜI/NHÂN VẬT** (Pure backgrounds without any characters, people, or actions).
- **Dữ liệu chuẩn:**
  ```json
  {
    "location": "Chairman's Office",
    "time": "Morning",
    "prompt": "A modern luxury office bathed in sharp morning sunlight... pure background, no people, empty scene. image ratio 16:9."
  }
  ```

### 2.3. Storyboard Generation (Chia Shot)
**Luồng API:** `POST /api/v1/episodes/:episode_id/storyboards`
- **Nhiệm vụ:** Chặt kịch bản thành các shot nhỏ (4-12s) dựa trên danh sách Characters và Scenes đã có.
- **Contract:** JSON Array chứa các Shot.
- **Narration Rule:** `dialogue` and `narration` are separate fields. If `dialogue` is non-empty, `narration` must be empty. If `dialogue` is empty, `narration` should contain third-person English short-drama recap voiceover timed to the shot duration.
- **Dữ liệu chuẩn:**
  ```json
  {
    "storyboard_number": 12,
    "shot_type": "Medium Shot",
    "angle": "Eye level",
    "movement": "Pan",
    "action": "Camera pans across press wall and stage lights.",
    "dialogue": null,
    "narration": "The room holds its breath as the secret prepares to surface.",
    "atmosphere": "Warm golden launch-event lighting.",
    "duration": 8
  }
  ```

### 2.4. Multi-Agent Narrative Generation
**Luồng API:** `POST /api/v1/dramas/:id/narrative/generate`
- `agent_step=1`: Agent 1 Architect creates exactly 15 graph nodes and global characters with `base_image_prompt`.
- `agent_step=2`: Agent 2 Builder runs per graph node in execution order, using parent `state_snapshot` values to create micro-beats, outfits, scenes, and the next `state_snapshot`.
- `agent_step=3`: Agent 3 Designer runs per episode and writes Markdown screenplay into `episodes.script_content`.
- Omit `agent_step` or send `0` to run the full Agent 1 -> Agent 2 -> Agent 3 pipeline in one async task.
- Magic Wand character extraction must return `linked_character_ids` and `new_characters`, then backend post-processing dedupes near-identical character names before creating new rows.

---

## 3. Frame Prompts & Image/Video API Payload

### Frame Prompt
Thường dùng để sinh frame đầu tiên (`first`) hoặc frame chính (`key`) của Shot.
- **API Cập nhật:** `PUT /api/v1/storyboards/:id/frame-prompt`
- **Contract:** Prompt là chuỗi text. **Không còn dùng field `description`**.
- **Ví dụ Data:** `"Photorealistic Korean drama style, 9:16 aspect ratio, eye-level medium close-up of Jae-hyun standing at a lectern, warm stage lighting, static pre-action frame."`

### Image Generation Payload
- **API Request (`POST /api/v1/images`):**
  ```json
  {
    "drama_id": "5",
    "storyboard_id": 167,
    "image_type": "storyboard",
    "frame_type": "first",
    "prompt": "Text prompt...",
    "model": "gemini-2.5-flash-image",
    "reference_images": ["/static/scenes/scene_21.jpg", "/static/characters/seo_yeon.jpg"]
  }
  ```

### Video Generation Payload
- **API Request (`POST /api/v1/videos`):**
  ```json
  {
    "drama_id": "5",
    "storyboard_id": 167,
    "prompt": "Slow pan across stage before speech starts...",
    "reference_mode": "single",
    "image_url": "/static/images/shot_1.jpg",
    "duration": 5,
    "aspect_ratio": "9:16"
  }
  ```

---

## 4. Model Provider Selection

Hệ thống cho phép Frontend truyền `model` vào Request. Backend xử lý theo rules:
1. Thử gọi `GetAIClientForModel("text/image/video", model)`.
2. Nếu không tìm thấy, fallback về **default client**.
3. **Bắt buộc:** Service (Backend) phải chèn thêm thông tin `style` và `aspect_ratio` từ DB `dramas` trước khi gửi đi.

**Mẹo tối ưu API Tokens (Dành cho Dev/Agent):**
- Text Generation batch (chia shot, bóc tách): Ưu tiên model Reasoning mạnh (VD: Claude, GPT-4o) để đảm bảo định dạng JSON.
- Sinh Prompt Frame (First-frame): Có thể dùng Flash-lite để tối ưu chi phí, hiện tại hệ thống gọi 1 request/shot. Nếu cần scale lớn, hãy code thêm tính năng Provider-level Batch APIs.
