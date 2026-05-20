# End-to-End Flow: Từ Ý Tưởng (User Idea) đến Chia Cảnh (Storyboard)

Tài liệu này mô tả chi tiết toàn bộ luồng xử lý của hệ thống Huobao Drama, bắt đầu từ khi người dùng nhập ý tưởng ban đầu (`user_idea`) cho đến khi hệ thống tạo ra kịch bản phân tập (Episode) và cuối cùng là chia nhỏ kịch bản đó thành các cảnh quay (Storyboard/Shot List).

---

## 1. Tổng quan Kiến trúc (High-level Architecture)

Luồng xử lý được chia làm hai giai đoạn (Phase) chính:
1. **Phase 1: Narrative Generation (Sinh Kịch bản)**: Chuyển đổi ý tưởng thô thành một đồ thị cốt truyện (Narrative Graph) gồm nhiều tập phim (Episodes), mỗi tập có kịch bản chi tiết (`script_content`).
2. **Phase 2: Storyboard Splitting (Chia Shot)**: Đọc kịch bản của từng tập và sử dụng AI để bóc tách thành các cảnh quay quay (Shots), tính toán thời lượng, và sinh prompt cho bước tạo ảnh/video tiếp theo.

---

## 2. Phase 1: Narrative Generation (Sinh Cốt truyện và Kịch bản)

### 2.1. Điểm bắt đầu (Trigger & Handler)
- **API Endpoint:** `POST /api/v1/dramas/:id/narrative/generate`
- **Handler:** `api/handlers/drama.go` -> `GenerateNarrativeEpisodes`
- **Nhiệm vụ:** Tiếp nhận `user_idea` từ client, gọi Service xử lý và trả về kết quả.

### 2.2. Core Service (`application/services/drama_service.go`)
Hàm `GenerateNarrativeEpisodes` chịu trách nhiệm điều phối:
1. Đọc thông tin `Drama` từ Database và tạo async task.
2. Nếu `agent_step=1`, chạy Agent 1 Architect để tạo đúng 15 node graph + characters.
3. Nếu `agent_step=2`, chạy Agent 2 Builder theo từng node, truyền parent `state_snapshot` để sinh micro-beats, outfits, scenes.
4. Nếu `agent_step=3`, chạy Agent 3 Designer theo từng node để ghi `script_content`.
5. Nếu bỏ trống `agent_step` hoặc gửi `0`, chạy full pipeline Agent 1 -> Agent 2 -> Agent 3 trong một task.
6. Single-pass `BuildPackage` chỉ còn là legacy/fallback path, không phải default cho flow multi-agent mới.

### 2.3. AI Orchestration (`application/services/narrative_package_service.go`)
Flow mới dùng các prompt embed riêng:
- `prompts/narrative/agent1_architect.md`: graph skeleton + global characters + `base_image_prompt`.
- `prompts/narrative/agent2_builder.md`: per-node micro-beats, state transition, outfits, scenes.
- `prompts/narrative/agent3_designer.md`: per-node Markdown screenplay.
- **Model Priority:** Hệ thống ưu tiên gọi model `gemini-2.5-pro` (Text Pro) để xử lý logic đồ thị dài. Nếu không có, fallback về model mặc định.
- **Data Normalization:** Sau khi AI trả về JSON, hàm `normalizeNarrativeGraph` sẽ kiểm tra tính toàn vẹn:
  - Loại bỏ các JSON markdown block.
  - Sắp xếp lại thứ tự các tập (BFS order).
  - Đảm bảo tất cả các node đều có thể đi tới được (Reachability).

### 2.4. Cấu trúc Dữ liệu Đầu ra (Database: `episodes`)
Dữ liệu được lưu vào bảng `episodes` (model `Episode`):
- `narrative_node_id`: ID của node trong đồ thị (VD: N101).
- `choices`: JSON mảng các nhánh lựa chọn sang tập tiếp theo.
- `script_content`: Nội dung kịch bản văn bản. Hệ thống có bước `stripUIUXBlock` để cắt bỏ các đoạn nháp UI/UX do AI tự đẻ ra, tránh ảnh hưởng đến bước chia shot.

---

## 3. Phase 2: Storyboard Splitting (Chia Shot List)

Sau khi đã có kịch bản cho tập (`script_content`), hệ thống tiến hành chia nhỏ thành Storyboard.

### 3.1. Điểm bắt đầu (Trigger)
- **Hàm xử lý:** `application/services/storyboard_service.go` -> `GenerateStoryboard(episodeID, model)`
- **Luồng bất đồng bộ:** Việc chia shot mất thời gian, do đó service sẽ tạo một `Task` (thông qua `TaskService`) và chạy goroutine `processStoryboardGeneration` dưới background, trả về `task_id` cho client.

### 3.2. Chuẩn bị Context
Trước khi gọi AI, hệ thống lấy toàn bộ context liên quan để prompt chính xác nhất:
1. `script_content`: Kịch bản gốc của tập.
2. Danh sách Nhân vật (`characters`): Bao gồm ID, Tên, và Danh sách trang phục (`outfits`). Đảm bảo AI chỉ dùng các nhân vật và quần áo có thật trong DB.
3. Danh sách Bối cảnh (`scenes`): Trích xuất từ project để AI map đúng ID bối cảnh.

### 3.3. Storyboard Prompt & Logic
Prompt được xây dựng động trong `GenerateStoryboard` với các quy tắc cực kỳ nghiêm ngặt:
- Bắt buộc phải **phá vỡ 100% kịch bản**, không được tóm tắt, không được bỏ sót thoại.
- Giới hạn thời lượng: **4-12 giây/shot**.
- Các field bắt buộc phải sinh cực chi tiết (Time, Location, Action, Result, Atmosphere, Dialogue, BGM, Sound Effect).
- `narration` là field riêng cho voiceover. Nếu shot có `dialogue`, `narration` phải rỗng. Nếu không có dialogue, narration là third-person English short-drama recap style và khớp duration.

### 3.4. Xử lý Kết quả (AI Output Parsing)
Trong hàm `processStoryboardGeneration`:
- Sử dụng hàm `utils.SafeParseAIJSON` để xử lý việc AI có thể trả về mảng `[...]` hoặc object `{"storyboards": [...]}`.
- **Tính toán thời lượng:** Tổng hợp lại thời lượng của tất cả các shot (`Duration`) và cập nhật ngược lại field `duration` của bảng `episodes`.

### 3.5. Xử lý Logic phụ & Lưu trữ (`saveStoryboards`)
- Xóa các storyboard cũ của tập đó.
- Tạo các Prompt chuyên dụng cho bước Generate Video/Image:
  - `generateImagePrompt(sb)`: Trích xuất Pose tĩnh đầu tiên (bỏ qua mô tả hành động động), kết hợp location và style để tối ưu cho bước sinh ảnh nền (Background) hoặc chân dung.
  - `generateVideoPrompt(sb)`: Kết hợp góc máy, chuyển động camera (Movement), hành động động để tối ưu cho AI sinh Video.
- **Mapping Nhân vật (Character Association):** Map các ID nhân vật AI trả về vào DB (`storyboard_characters`). AI có thể gán `outfit_id` cụ thể, nếu không có, hệ thống sẽ thực hiện **Keyword Matching** từ `Location/Action/Dialogue` xem có khớp với tên outfit nào của nhân vật không để gán tự động.

### 3.6. Cấu trúc Dữ liệu Đầu ra (Database: `storyboards`)
- Dữ liệu lưu bảng `storyboards` bao gồm: `shot_number`, `shot_type`, `action`, `dialogue`, `narration`, `duration`, `image_prompt`, `video_prompt`...
- Bảng trung gian `storyboard_characters` lưu quan hệ N-N kèm `outfit_id`.

---

## 4. Tổng kết Logic Dòng chảy (Flow Summary)

1. **User Idea** -> Gửi request tạo Narrative.
2. **Text AI (Pro)** -> Trả về `NarrativeDramaPackage` (JSON Graph bao gồm các `episodes`).
3. **Drama Service** -> Xóa episode cũ -> Lưu `episodes` mới vào DB (tính toán `resolveEpisodeChoiceNarrativeIDs` cho UI rẽ nhánh).
4. **User / System** -> Gọi lệnh Chia Shot cho từng Episode (`GenerateStoryboard`).
5. **Storyboard Service** -> Thu thập Nhân vật, Bối cảnh hiện có -> Gửi `script_content` cho AI bóc tách (Background Job).
6. **Text AI** -> Cắt kịch bản ra thành mảng các `storyboard` (JSON).
7. **Storyboard Service** -> Cập nhật AI JSON -> Sinh `ImagePrompt` & `VideoPrompt` -> Cập nhật DB.

*Toàn bộ quy trình được thiết kế với mức độ kiểm soát chặt chẽ, kết hợp Prompt Engineering (System + User Prompt) và Data Normalization trên Go để đảm bảo đầu ra chuẩn chỉnh nhất.*
