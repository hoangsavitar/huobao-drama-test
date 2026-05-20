# Development & Operational Guide (For Agents & Devs)

Tài liệu này là cẩm nang (Playbook) hướng dẫn cách thêm tính năng mới, cách hệ thống render video bằng FFMPEG, cách migrate data và xử lý nhanh các bug thường gặp.

---

## 1. End-to-End Production Flow

Hiểu luồng này để biết bạn đang sửa code ở giai đoạn nào:

1. **Create Drama**: Tạo dự án (Xác định `style`, `aspect_ratio`).
2. **Episode Script**: Nhập kịch bản (`episodes.script_content`).
3. **Extract Characters/Scenes**: AI bóc tách nhân vật & bối cảnh.
4. **Generate Characters/Scenes Images**: Dùng LLM Image model sinh ảnh gốc.
5. **Split Shots**: AI chia kịch bản thành các Storyboard (Shot).
6. **Generate Prompts**: Sinh prompt cho từng frame (`first_frame`, `key_frame`).
7. **Generate Shot Images & Videos**: Dựa vào ảnh gốc và prompt, sinh ảnh và video cho từng Shot.
8. **Compose/Export**: Dùng FFMPEG ghép các video shot lại thành 1 video tập hoàn chỉnh.

### Multi-Agent Narrative Flow
- Narrative generation is async through `POST /api/v1/dramas/:id/narrative/generate`.
- Send `agent_step=1`, `2`, or `3` to rerun one agent. Omit `agent_step` or send `0` to run the full Agent 1 -> Agent 2 -> Agent 3 pipeline.
- Agent 1 must produce exactly 15 reachable graph nodes with at least one real branch, save `base_image_prompt`, and queue base character image generation best-effort.
- Agent 2 must run per episode/node and use parent `state_snapshot` values; reruns replace generated outfits/scenes for that episode.
- Agent 3 must run per episode/node and write Markdown screenplay without UI/UX button copy.

---

## 2. Safe Extension Playbook (Cách thêm tính năng)

Khi AI Agent hoặc Dev cần thêm 1 tính năng sinh nội dung (Gen AI) mới, phải tuân thủ chuẩn sau:
1. Thêm field vào Model (`domain/models/`).
2. Viết logic tạo Prompt tại `application/services/prompt_i18n.go`.
3. Viết Service gọi AI tại `application/services/`. Không để logic ở Handler.
4. Viết Handler tại `api/handlers/` và đăng ký route ở `api/routes/routes.go`.
5. Bổ sung hàm call API ở Frontend (`web/src/api/`).
6. Thêm trạng thái Async/Polling trên UI để user không bị block.
7. Thêm Text I18n tại `en-US.ts` / `zh-CN.ts`.

For storyboard voiceover extensions, add the data field through model -> service DTO -> API response -> frontend type/editor. Keep `dialogue` and `narration` separate; do not store narration text inside `dialogue`.

---

## 3. FFMPEG Rendering Workflow

Hệ thống **không** dùng API ngoài để ghép video, mà gọi trực tiếp FFMPEG CLI tại Backend (`infrastructure/external/ffmpeg/ffmpeg.go`).

- **Logic Merge (`MergeVideos`)**: Gom tất cả `scene.VideoURL` của một Episode.
- **Toán học Duration**: Hệ thống sẽ ép cứng tổng thời lượng video bằng cách cộng các `Duration` cấu hình của từng shot. Nếu AI sinh clip dài 7s nhưng config là 5s, FFMPEG sẽ dùng `-ss` và `-to` để cắt chuẩn 5s.
- **Transitions (Xfade)**: Nếu có hiệu ứng chuyển cảnh (Cross-fade), FFMPEG dùng filter `xfade`. Lưu ý: Transition sẽ ăn lẹm vào thời lượng tổng (VD: 2 clip 5s, fade 1s -> Tổng là 9s).
- **Codecs ép buộc**: `-c:v libx264 -preset fast -crf 23` và `-c:a aac`.

---

## 4. Data Migration Service

Hệ thống có một service tự động chạy lúc khởi động (`application/services/data_migration_service.go`) để tải file media từ URL ngoài về Local.

- **Nhiệm vụ**: Quét các bảng `scenes`, `characters`, `storyboards`, `video_generations` xem record nào có URL nhưng `local_path` rỗng.
- **Xử lý**: Tải file ngầm (Async) và lưu vào `data/storage/...`.
- **An toàn**: Lỗi tải 1 file không làm crash App, không block quá trình startup.
- **Fix lỗi**: Nếu Local Path không hiện ảnh, kiểm tra folder `data/storage/` có quyền Ghi/Đọc không, hoặc URL gốc đã die (404/403).

---

## 5. Quick Fix Playbook (Troubleshooting Nhanh)

### Bug 1: Sửa Prompt ở Shot 1, sang Shot 2 thấy bị dính text cũ
- **Nguyên nhân**: Lỗi Cache Key trên UI.
- **Khắc phục**: Cache key phải là tổ hợp `storyboard_id` + `frame_type`. Hãy chắc chắn Ép kiểu ID đồng nhất (dùng `String(id)` hoặc `Number(id)` khi so sánh `s.id === storyboardId` trên Vue).

### Bug 2: Edit Prompt xong, F5 lại bị mất (Stale Prompt)
- **Nguyên nhân**: Không gọi API lưu hoặc DB chưa kịp overwrite session cache.
- **Khắc phục**: Kiểm tra `PUT /storyboards/:id/frame-prompt` đã được gọi chưa (`ProfessionalEditor.vue` autosave watcher).

### Bug 3: Đổi Aspect Ratio sang 9:16 nhưng ảnh gen ra vẫn 16:9
- **Khắc phục**:
  1. Kiểm tra API `PUT /dramas/:id` đã lưu `aspect_ratio` thành công vào DB chưa.
  2. Kiểm tra log Backend xem biến `aspectRatio` có được push vào `GetFirstFramePrompt` và Image Generation Request không.

### Bug 4: Export Zip báo thiếu file
- **Khắc phục**: API `/api/v1/images` đang có Pagination (page_size mặc định thường là 100). Đảm bảo script Export trên frontend đã loop qua tất cả các page để gom đủ danh sách ảnh.
