# Kế Hoạch Tính Năng: Đồ Thị Cốt Truyện Phức Tạp & Giọng Dẫn Truyện Liên Tục

## 1. Bối cảnh ban đầu (Context First)
- **Trạng thái hiện tại:**
  - **Đồ thị Cốt truyện (Narrative Graph):** Hệ thống đang tạo ra một đồ thị tuyến tính hoặc rẽ nhánh rất hạn chế theo hướng dẫn của file `drama_package_system.md`. Prompt hiện tại bắt buộc phải có "Một lần rẽ nhánh sớm... MỘT điểm nhập lại (merge) chung" để giữ cho cây cốt truyện nhỏ. Các kết thúc thường gộp lại với nhau chứ không rẽ thành các nhánh tách biệt kéo dài.
  - **Chia cảnh quay (Storyboard Splitting):** Hàm `GenerateStoryboard` chia kịch bản thành các cảnh quay (shot) chứa `action` (hành động), `dialogue` (thoại), `duration` (thời lượng 4-12s), v.v. Hiện tại không có giọng dẫn truyện liên tục. Video có thể tạo cảm giác im lặng hoặc rời rạc nếu thiếu đi người dẫn chuyện xuyên suốt.
  - **Cơ sở dữ liệu (Database):** Bảng `storyboards` đã có `dialogue` và `description` nhưng thiếu một cột `narration` chuyên dụng dành cho đoạn âm thanh dẫn truyện liên tục.
- **Thay đổi đề xuất:**
  - **Tính năng 1:** Cập nhật logic cốt truyện để sinh ra các cây tập phim rẽ nhánh sâu hơn, với các lựa chọn có ý nghĩa, nhiều hướng đi phức tạp, và các đoạn kết bị cô lập hoàn toàn (không nhập lại).
  - **Tính năng 2:** Thêm trường `narration` (lời dẫn truyện) vào quá trình sinh Storyboard. Tương ứng với mỗi cảnh quay (shot), sinh ra văn bản dẫn truyện liên tục có độ dài khớp với thời lượng (ví dụ: 4s = khoảng 10-12 chữ) để phục vụ cho các video phong cách TikTok/Reels/Douyin.

## 2. Nhóm câu hỏi làm rõ (Batch Clarifying Questions)
1. **Độ phức tạp của đồ thị:** Bạn muốn cây cốt truyện mới có tối đa bao nhiêu nhánh hoặc tổng số lượng tập là bao nhiêu? Chúng ta có nên cho phép các nhánh KHÔNG BAO GIỜ nhập lại với nhau, tạo ra các tuyến truyện hoàn toàn khác biệt từ tập 2 trở đi không?
2. **Phong cách dẫn truyện:** `narration` (lời dẫn) nên là một "người kể chuyện toàn tri" (ngôi thứ 3 kể về mọi sự kiện), hay là "độc thoại nội tâm" của nhân vật chính (ngôi thứ 1)?
3. **Thoại và Lời dẫn (Dialogue vs Narration):** Nếu trong một cảnh quay nhân vật đã cất lời thoại (`dialogue`), thì trường `narration` của cảnh đó nên để trống, hay giọng dẫn truyện vẫn cứ tiếp tục miêu tả hành động/giới thiệu song song với lời thoại?

## 3. Khóa Xác Nhận (Understanding Lock)
*(Đã được xác nhận bởi người dùng)*
- **Tóm tắt:** Nâng cấp engine kể chuyện để hỗ trợ đồ thị phức tạp hơn (khoảng 15 node, luôn có ít nhất 2 nhánh song song, logic chuẩn). Bổ sung giọng dẫn truyện (narration) ngôi thứ 3 phong cách review phim Trung Quốc; bỏ qua dẫn truyện nếu nhân vật đang có lời thoại.
- **Giả định:** 
  - Cần phải cập nhật Database (Migration) để thêm cột `narration` vào bảng `storyboards`.
  - Văn bản lời dẫn sẽ được viết liền mạch và tính toán khớp thời gian (khoảng 2.5 từ / giây) cho quy trình chuyển văn bản thành giọng nói (TTS) trong tương lai.
- **Ngoài phạm vi (Non-goals):** 
  - Không bao gồm việc xây dựng logic sinh Audio/TTS ở phase này (chỉ giới hạn ở việc sinh text kịch bản cho lời dẫn).

## 4. Kế hoạch hành động từng bước (Step-by-Step Action Plan)
**Giai đoạn 1: Bản nháp Prompt (Drafting Prompts - Hiện tại)**
1. Sửa đổi `drama_package_system.md`: Cập nhật logic ép buộc rẽ nhánh (luôn có nhánh song song) và chuẩn hóa logic lựa chọn.
2. Sửa đổi `drama_package_user.md`: (Sẽ không cần sửa nhiều vì logic rẽ nhánh chủ yếu nằm ở System Prompt).
3. Đề xuất prompt sửa đổi cho người dùng test thử.

*(Các giai đoạn DB Migration và Storyboard code update sẽ thực hiện sau khi test prompt thành công)*

## 5. Nhật ký quyết định (Decision Log)
- **Lưu trữ Lời dẫn (Narration Storage):** Quyết định thêm một cột `narration` riêng biệt thay vì nhồi nhét nó vào cột `dialogue`. Điều này đảm bảo hệ thống trích xuất audio ở hạ nguồn (downstream) có thể dễ dàng kéo một kịch bản sạch sẽ để nạp vào TTS.
- **Prompt vs Code:** Vấn đề độ phức tạp của đồ thị hoàn toàn là do giới hạn của Prompt Engineering. Chúng ta sẽ giải quyết bằng cách viết lại `drama_package_system.md` thay vì phải code những thuật toán sinh cây phức tạp ở tầng Go.
- **Quyết định từ người dùng (User Decisions):**
  - Giữ khoảng 15 nodes, nhưng BẮT BUỘC phải có tối thiểu 2 nhánh song song xuyên suốt, không được chạy tuyến tính. Lựa chọn phải cực logic.
  - Narration: Ngôi thứ 3 (như Review phim TikTok TQ).
  - Narration logic: Có thoại (`dialogue`) -> KHÔNG có `narration`. Không thoại -> Có `narration` giàu cảm xúc.