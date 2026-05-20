# Architecture & Data Flow (For AI Agents)

Tài liệu này cung cấp bản đồ kỹ thuật (Technical Map) của dự án `huobao-drama-test`. Nó kết hợp sơ đồ code (Feature-to-Code), Database Schema, và luồng Async API, giúp AI Agent truy vết lỗi và phát triển tính năng cực nhanh.

---

## 1. Core Architecture Layers

Hệ thống tuân theo mô hình phân lớp rõ ràng (Clean Architecture pattern):
- **Frontend (Vue 3)**: `web/src/views/**` (Giao diện), `web/src/api/**` (Gọi API)
- **HTTP Handlers (Gin)**: `api/handlers/**` (Chỉ xử lý Request/Response)
- **Business Services**: `application/services/**` (Logic lõi & Gọi AI LLM)
- **Persistence (GORM)**: `domain/models/**` (Định nghĩa Schema DB)
- **Routing**: `api/routes/routes.go`

> [!IMPORTANT]
> **Quy tắc thiết kế:** Handlers phải luôn "mỏng" (thin). Toàn bộ logic nghiệp vụ và luồng gọi AI/Provider phải nằm trong Service (`application/services/`).

---

## 2. Database Schema (Operational Map)

Dưới đây là các Model cốt lõi và quan hệ thực tế trong Database.

### 2.1. Entity Graph
- **Drama** `1 -> N` **Episode** (1 Drama có nhiều Tập)
- **Episode** `1 -> N` **Storyboard** (1 Tập chia thành nhiều Shot/Storyboard)
- **Drama** `1 -> N` **Character** / **Scene** (Nhân vật và Cảnh được quản lý ở cấp Drama)
- **Storyboard** `N <-> N` **Character** (Nhiều-Nhiều)
- **Storyboard** `1 -> N` **FramePrompt** (1 Shot có thể có nhiều prompt cho các loại frame: `first`, `key`, `last`...)
- **Storyboard/Scene/Character** `-> N` **ImageGeneration** (Lịch sử sinh ảnh)

### 2.2. Critical Models (`domain/models/`)

**Drama** (`dramas`)
- `id`, `title`, `status`
- `style`: Cực kỳ quan trọng để gen ảnh (vd: `ghibli`, `guoman3d`).
- `aspect_ratio`: Định dạng video (`16:9` hoặc `9:16`). Quy định size ảnh sinh ra.

**Episode** (`episodes`)
- `id`, `drama_id`, `episode_number`
- `script_content`: Chứa toàn bộ kịch bản thô của tập.
- `narrative_node_id`, `parent_node_id`, `choices`, `state_snapshot`, `is_entry`: state for the multi-agent branching narrative graph. Agent 1 writes skeleton/choices, Agent 2 writes state snapshots, Agent 3 writes final script content.

**Storyboard** (`storyboards` - Tương đương 1 Shot)
- `id`, `episode_id`, `scene_id`
- `storyboard_number`, `duration`
- `action`, `dialogue`, `narration`, `atmosphere` (Được AI bóc tách từ Script). `narration` is separate from `dialogue`; dialogue shots must keep narration empty.
- `image_prompt`, `video_prompt` (Các prompt chính thức)
- `composed_image`, `video_url` (Kết quả cuối)

**FramePrompt** (`frame_prompts`)
- `id`, `storyboard_id`
- `frame_type`: `first`, `key`, `last`, `panel`, `action`
- `prompt`: Text dùng để gen frame đó. (Lưu ý: Bảng này là Nguồn Sự Thật cho frame prompt).

**ImageGeneration** & **VideoGeneration**
- Theo dõi tiến trình (Async Job). Gồm các field: `prompt`, `model`, `provider`, `status` (`pending`, `processing`, `completed`, `failed`), `image_url/video_url`.

---

## 3. Asynchronous Tasks & Polling

Quá trình sinh Prompt/Ảnh/Video tốn nhiều thời gian, do đó hệ thống sử dụng **Async Tasks**.

### Vòng đời chuẩn:
1. Giao diện (Client) gửi request tạo. (`POST /images`)
2. Server trả về ngay lập tức (Status: `pending`, kèm `task_id` nếu có).
3. Server (Goroutine) chạy ngầm hàm sinh nội dung gọi LLM/ComfyUI/Runway...
4. Client poll trạng thái (`GET /api/v1/tasks/:taskId` hoặc list images).
5. Khi `status` thành `completed`/`failed`, Client fetch lại dữ liệu gốc.

### Batch Operations (Xử lý hàng loạt)
- Khi chia shot (Split Shots) hoặc sinh nhiều ảnh một lúc, thiết kế phải hỗ trợ **Partial Success** (thành công một phần). Tức là lỗi 1 shot không được làm hỏng tiến trình của các shot khác.

---

## 4. API Contracts (Top Endpoints)

Base URL: `/api/v1/`

### Drama & Storyboard
- `GET /dramas/:id`: Lấy thông tin dự án.
- `PUT /dramas/:id`: Update config (Cực kỳ lưu ý `aspect_ratio` nếu thay đổi phải update cả DB).
- `POST /dramas/:id/narrative/generate`: Async multi-agent narrative generation. Body `{ "user_idea": "...", "agent_step": 1 }`; `agent_step` may be `1`, `2`, or `3` for a single agent. Omit it or send `0` to run Agent 1 -> Agent 2 -> Agent 3 as one full pipeline.
- `GET /episodes/:episode_id/storyboards`: Lấy toàn bộ Shot của tập, bao gồm cả mảng `characters[]`, `background`, và các URL kết quả (`composed_image`). Đây là API nguồn cho bảng Workflow trên UI.

### Prompts
- `POST /storyboards/:id/frame-prompt`: Sinh tự động bằng AI (Async).
- `PUT /storyboards/:id/frame-prompt`: User ghi đè/chỉnh sửa prompt bằng tay.
- `GET /episodes/:episode_id/frame-prompts`: Trả về `map[storyboard_id] -> []frame_prompts` để UI render nhanh cột trạng thái.

### Sinh Media (Image/Video)
- `POST /images`: Yêu cầu sinh ảnh (cần `drama_id`, `image_type` (`storyboard|scene|character`), `frame_type`, `prompt`).
- `POST /videos`: Yêu cầu sinh video (cần `prompt`, `provider`, `model`, `duration`, `aspect_ratio`). 

---

## 5. Trace Code Nhanh (Feature-to-Code Map)

Nếu cần sửa lỗi theo Feature, hãy tìm theo luồng sau:

1. **Quản lý Drama (Tỷ lệ khung hình, Style)**
   - UI: `web/src/views/drama/DramaList.vue`, `web/src/components/common/CreateDramaDialog.vue`
   - Backend: `api/handlers/drama.go` -> `application/services/drama_service.go`
2. **Luồng Cắt Cảnh (Split Shots)**
   - UI: `web/src/views/drama/EpisodeWorkflow.vue`
   - Backend: `POST /episodes/:id/storyboards` -> `application/services/storyboard_service.go`
3. **Sinh Prompt cho Frame Đầu (First-frame)**
   - UI: `EpisodeWorkflow.vue` (Batch) hoặc `ProfessionalEditor.vue` (Single)
   - Backend: `api/handlers/frame_prompt_query.go` -> `application/services/frame_prompt_service.go`
4. **Luồng sinh Ảnh Character / Scene**
   - Backend: `api/handlers/character_library_gen.go` / `api/handlers/scene.go` -> `application/services/image_generation_service.go`
