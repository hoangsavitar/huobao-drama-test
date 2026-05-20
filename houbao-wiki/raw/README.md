# Documentation Index (Optimized for AI Agents & Maintainers)

Thư mục này chứa toàn bộ kiến thức cốt lõi về dự án `huobao-drama-test` (hệ thống sinh video/drama tự động). Các tài liệu được tối ưu hóa theo nguyên tắc **Simplicity First** (ngắn gọn, tập trung) và **Goal-Driven** (hướng đến việc hiểu cấu trúc nhanh chóng để code/debug). 

Đặc biệt, tài liệu này hướng tới **AI Agents** và Developer mới vào dự án.

---

## 📂 Core Documents

Thay vì dàn trải, toàn bộ kiến thức dự án được cô đọng trong 3 tài liệu chính thức dưới đây:

### 1. `ARCHITECTURE.md` (Dành cho hiểu luồng & Dữ liệu)
- **Nội dung:** Sơ đồ kiến trúc tổng quan (Frontend -> Backend -> DB), luồng xử lý bất đồng bộ (Async/Polling), và **chi tiết về Database Schema** (các bảng cốt lõi như `dramas`, `episodes`, `storyboards`, `image_generations`...).
- **Khi nào đọc:** Khi cần truy vết lỗi luồng hệ thống, muốn biết dữ liệu được lưu ở bảng nào, hoặc trước khi thêm tính năng/API mới.

### 2. `AI_INTEGRATION.md` (Dành cho xử lý Prompts & AI Models)
- **Nội dung:** Chứa toàn bộ "trái tim" của hệ thống sinh nội dung: Các hợp đồng Prompt (Prompt Contracts), cấu trúc Payload khi gọi AI, quy tắc trích xuất Character/Scene từ Script, và luồng chia Shot (Storyboard generation).
- **Khi nào đọc:** Khi cần sửa đổi chất lượng text/hình ảnh sinh ra, thay đổi prompt template (`prompt_i18n.go`), thêm provider mới (Doubao, OpenAI), hoặc sửa lỗi liên quan đến nội dung sinh ra bị sai format.

### 3. `DEVELOPMENT_GUIDE.md` (Dành cho Phát triển & Fix Bugs)
- **Nội dung:** Hướng dẫn luồng phát triển frontend/backend, cách thức FFMPEG render video, cẩm nang khắc phục sự cố nhanh (Quick Fix Playbook) và hướng dẫn Database Migration.
- **Khi nào đọc:** Khi chạy hệ thống local, khi xử lý các bugs thường gặp (stale cache, mix-up prompt), hoặc khi cần sửa luồng ghép video.

---

## 🎯 Quy tắc bảo trì Docs (Dành cho AI Agents)

1. **Surgical Changes**: Nếu bạn sửa code làm thay đổi Database Schema, **phải** cập nhật `ARCHITECTURE.md`.
2. **Prompt Integrity**: Nếu bạn thay đổi logic tạo Prompt trong thư mục `application/services/`, **phải** cập nhật `AI_INTEGRATION.md`.
3. **Không tạo thêm file mới**: Hãy giữ cấu trúc 4 files này. Nếu có kiến thức mới, hãy bổ sung vào đúng chuyên mục của 1 trong 3 file trên.
