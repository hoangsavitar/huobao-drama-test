# Feature Planning: 3-Agent Architecture & Upstream Data-First

## 1. Bối cảnh ban đầu (Context First)
*   **Trạng thái hiện tại (Luồng cũ):**
    *   Hệ thống dùng 1 lần gọi LLM duy nhất (Single-pass) để gánh cả 3 vai trò (Architect, Builder, Designer), dẫn đến quá tải token và ảo giác logic (không thể theo dõi state của các tập).
    *   Tại trang *Episode Production*, hình ảnh nhân vật được tạo ra bằng cách chạy hàm `ExtractCharactersFromScript` đọc lại toàn bộ kịch bản. Việc này chậm, tốn kém và dễ gây sai lệch khuôn mặt nhân vật giữa các tập do AI "đoán" từ text.
*   **Trạng thái đề xuất (Luồng mới - ĐÃ HOÀN THÀNH):**
    *   **Tách 3 Agent:**
        *   *Agent 1 (Architect):* Giữ Logic Toàn cục + Khuôn mặt tĩnh (Base Prompt Portrait A-Pose, Pure White Background).
        *   *Agent 2 (Builder):* Giữ Trạng thái cục bộ (State Tracking) + Trang phục động (Outfit Prompt) + Nhịp truyện (Beats).
        *   *Agent 3 (Designer):* Giữ Ngôn từ & Cảm xúc kịch bản (Markdown Screenplay).
    *   **Upstream Data-First:** Dữ liệu hình ảnh được Agent 1 & 2 chốt cứng và lưu thẳng vào Database trước khi kịch bản được viết ra.
    *   **3 Trụ cột Trang Episode Production:**
        1. *Load mặc định 0s:* Đổ Data từ DB ra UI.
        2. *Pro Mode (CRUD):* User có thể Thêm/Sửa/Xóa Nhân vật/Outfit bằng tay hoàn toàn không dùng AI.
        3. *Lazy Mode (Đũa thần/Magic Wand):* Nút bấm chỉ dùng AI để quét "phần dư" (Delta Extract) - tức là những nhân vật lạ mà User vừa gõ tay vào kịch bản nhưng chưa có ở DB.

---

## 2. Khóa Xác Nhận (Understanding Lock - Hard Gate)

**Tóm tắt Mục tiêu:**
Đập bỏ kiến trúc sinh kịch bản nguyên khối để chuyển sang kiến trúc Multi-Agent (Architect -> Builder -> Designer) có theo dõi trạng thái (State). Đồng thời, dời khâu sinh Prompt Hình ảnh lên đầu phễu (Upstream) để đảm bảo 100% tính nhất quán. Trang *Episode Production* sẽ được nâng cấp thành một Authoring Tool chuyên nghiệp có CRUD data, với nút Extract cũ được "hạ cấp" thành tính năng "Đũa thần" hỗ trợ tìm phần dư (Delta Extract).

---

## 3. BẢN KẾ HOẠCH & TRẠNG THÁI THỰC THI (Action Plan & Execution Status)

### 🟢 Giai đoạn 0: Chuẩn bị Hạ tầng Data (Database Migration & Schema Update) - ✅ HOÀN THÀNH
1. Thêm cột `base_image_prompt` (loại `TEXT`) vào bảng `characters` trong `domain/models/drama.go` để lưu Portrait A-Pose không background.
2. Thêm trường `StateSnapshot`, `Choices`, `Description` (Micro-beats) và `ParentNodeID` vào bảng `episodes` phục vụ thuật toán đồ thị rẽ nhánh.
3. **Mới bổ sung:** Định cấu trúc lưu trữ bảo toàn `plot_summary` (bản tóm tắt cốt truyện cốt lõi từ Agent 1) ngay bên trong cột JSON `state_snapshot` để tránh bị Agent 2 ghi đè trường `description` (dùng cho Micro-beats).

### 🟢 Giai đoạn 1: Nâng cấp luồng API thành Bất đồng bộ (Async Background Job) - ✅ HOÀN THÀNH
1. API `GenerateNarrativeEpisodes` trong `drama_service.go` trả về `task_id` ngay lập tức.
2. Frontend hiển thị Progress Bar và liên tục polling trạng thái tiến độ của các Agent (Architect -> Builder -> Designer) theo thời gian thực.

### 🟢 Giai đoạn 2: Xây dựng Agent 1 - The Architect (Global Setup) - ✅ HOÀN THÀNH
1. Tạo file prompt: `prompts/narrative/agent1_architect.md`.
2. Trả về cấu trúc Graph Skeleton và mảng `Characters` (mô tả ngoại hình dạng Front view portrait, strict A-pose, pure white background).
3. Tự động lưu phác thảo cốt truyện ban đầu vào `state_snapshot.plot_summary` để bảo toàn dữ liệu.

### 🟢 Giai đoạn 3: Xây dựng Agent 2 - The Builder (State & Outfits) - ✅ HOÀN THÀNH
1. Tạo file prompt: `prompts/narrative/agent2_builder.md`.
2. Vòng lặp duyệt đồ thị rẽ nhánh, sinh `Micro-beats` (lưu vào `description`), `State_Snapshot_T` (lưu dòng thời gian, trang thái, vật phẩm) và `episode_outfits` (lưu trang phục từng tập).
3. **Mới bổ sung:** Tách biệt hoàn toàn hiển thị giữa **Plot Outline (Agent 1)** và **Micro-beats (Agent 2)** tại Frontend (Expand dòng của Episode Management và mục Collapse của Episode Production) giúp người viết kịch bản nắm được bối cảnh vĩ mô lẫn vi mô.

### 🟢 Giai đoạn 4: Xây dựng Agent 3 - The Designer (Screenplay Writer) - ✅ HOÀN THÀNH
1. Tạo file prompt: `prompts/narrative/agent3_designer.md`.
2. LLM đọc các nhịp truyện `Micro-beats` để viết kịch bản dạng Markdown (`script_content`).
3. **Mới bổ sung:** Tích hợp bộ hiển thị **Global Storyline & Hook** ngay trong ô **Pipeline Data (Preview)** của trang Project Overview để làm điểm tựa định hướng nội dung cho creator trong suốt quá trình chạy pipeline.

### 🟡 Giai đoạn 5: Phát triển tính năng "Đũa thần" (Magic Wand - Delta Extract) - 🟢 ĐÃ TÍCH HỢP UI & BACKEND
1. Nút "Extract Characters & Scenes" được đặt tại Bước 1 của trang Episode Production.
2. Quét kịch bản thô và trích xuất thực thể cảnh/nhân vật mới phát sinh.
3. *Đang tối ưu tiếp:* Tăng cường khả năng Fuzzy Matching và Coreference Resolution trong Prompt của API để giảm tỷ lệ trùng lặp thực thể khi người dùng gõ sai chính tả.

---

## 4. NHẬT KÝ QUYẾT ĐỊNH (Decision Log)

| Chủ đề | Phương án đã chọn | Tại sao chọn? (Alternative considered) |
| :--- | :--- | :--- |
| **Bảo toàn Cốt truyện gốc (Plot Summary)** | Lưu `plot_summary` vào `state_snapshot` JSON, giải phóng `description` cho Micro-beats. | *(Cũ)*: Agent 2 ghi đè thẳng lên `description` làm mất hoàn toàn cốt truyện gốc do Agent 1 phác thảo khiến người viết kịch bản bị mất định hướng toàn cục. |
| **Giao diện Overview Thống nhất** | Hiển thị **Global Storyline & Hook** ở ngay ô Preview dưới các nút chạy Agent. | *(Cũ)*: Thông tin cốt truyện tổng chỉ nằm ở ô Project Info nhỏ ở trên, người sáng tạo khi theo dõi tiến độ sinh tag không thể có cái nhìn tổng quan tức thời về ý tưởng gốc. |
| **Bảo toàn Khuôn Mặt (Visual Consistency)** | Dùng Ảnh Base làm **Reference/FaceID** + Text Outfit. | Dùng Portrait A-pose với pure white background giúp mô hình Stable Diffusion/LoRA dễ dàng nhận dạng và thay đổi quần áo nhân vật chuẩn xác nhất. |
| **Luồng Thực thi Narrative** | **Bất đồng bộ (Async) với TaskID.** | Tránh việc request bị Timeout do 3 Agent chạy qua 15-20 tập tốn nhiều thời gian. |
| **Phân bổ Agent** | **Tách 3 Agent riêng biệt.** | Ngăn chặn quá tải Token và ảo giác logic (hallucination) so với luồng Single-pass cũ. |
