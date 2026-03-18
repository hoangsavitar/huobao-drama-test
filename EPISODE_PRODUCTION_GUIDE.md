# 🎬 HƯỚNG DẪN TOÀN DIỆN: CÁCH VIẾT KỊCH BẢN TỐI ƯU CHO AI DRAMA GENERATOR

Tài liệu này tổng hợp toàn bộ nguyên tắc hoạt động, "luật ngầm" của AI (dựa trên các prompt hệ thống) và các mẹo thực chiến để viết ra một kịch bản Episode hoàn hảo, giúp hệ thống trích xuất Nhân vật, Bối cảnh và tạo Storyboard/Video với độ chính xác cao nhất, ít lỗi nhất.

---

## 📖 BÍ QUYẾT 1: TƯ DUY KỊCH BẢN "SHOW, DON'T TELL"
Hệ thống AI của chúng ta hoạt động giống như một **Đạo diễn Phân cảnh (Storyboard Artist)** thay vì một Tiểu thuyết gia. 
*   **Độ dài lý tưởng:** Khoảng **500 - 1000 chữ** cho mỗi Episode. Quá dài AI sẽ bị "nhớ nhớ quên quên" (hallucination), có xu hướng gộp cảnh ở phần cuối.
*   **Văn phong:** Sử dụng câu đơn, miêu tả NGÔN NGỮ CƠ THỂ, HÀNH ĐỘNG VẬT LÝ và KHÔNG GIAN.
*   ❌ **Sai lầm:** *"Anna cảm thấy hối hận vô cùng, cô nghĩ về những ngày tháng thanh xuân và thầm ao ước giá như mình không làm vậy."* (AI không thể quay phim "suy nghĩ").
*   ✅ **Chuẩn xác:** *"Anna quỳ gục xuống sàn nhà. Nước mắt cô tuôn rơi lấm lem lớp trang điểm. Cô nắm ngực áo, ngước mặt lên trời và hét lớn: 'Tôi sai rồi!'"*

---

## 👤 BÍ QUYẾT 2: TỐI ƯU TRÍCH XUẤT NHÂN VẬT (CHARACTER EXTRACTION)
Hệ thống quét toàn bộ kịch bản và **chỉ nhặt những người CÓ TÊN**.

1.  **Khai sinh rõ ràng:** Lần đầu nhân vật xuất hiện, hãy mô tả ngoại hình của họ ngay trong kịch bản.
    *   *Ví dụ:* *"Leon (nam, 30 tuổi, cao lớn, mặc áo khoác da màu nâu, tóc vuốt ngược) bước vào."*
    *   Điều này giúp AI tự động điền form "Appearance" chuẩn xác để sau này Gen ảnh không bị thay đổi ngoại hình giữa các cảnh.
2.  **Đồng nhất nhân xưng:** Xuyên suốt kịch bản, **hãy luôn gọi tên nhân vật**. Hạn chế tối đa dùng đại từ phiếm chỉ (hắn, y, ả, gã, người đàn ông đó...). AI dễ bị nhầm lẫn dẫn đến việc đẻ ra nhân vật phụ không mong muốn hoặc trộn lẫn hành động.

---

## 🖼️ BÍ QUYẾT 3: TỐI ƯU TRÍCH XUẤT BỐI CẢNH (SCENE EXTRACTION)
Prompt của AI có một lệnh bắt buộc: **Bối cảnh phải LÀ PHÔNG NỀN RỖNG (Pure Background)**, KHÔNG CÓ BÓNG NGƯỜI, CẤM NHỮNG TỪ NHƯ "PEOPLE", "CHARACTER".

1.  **Cú pháp phân cảnh:** Hãy dán nhãn [ĐỊA ĐIỂM] - [THỜI GIAN] rõ ràng ở đầu mỗi phân đoạn.
    *   *Ví dụ:* **[Tại Quán Bar cũ - Đêm khuya]**
2.  **Miêu tả vật thể vật lý:** Khuyên dùng 1-2 câu miêu tả đồ vật trong phòng để bản vẽ có vibe.
    *   *Ví dụ:* *"Quán bar cũ nát, ánh đèn neon đỏ nhấp nháy, bàn ghế gỗ xước xát, sàn nhà vương vãi vỏ chai bia."* (AI sẽ lấy chính câu này làm Background Prompt cực đẹp).

---

## 🎥 BÍ QUYẾT 4: TỐI ƯU PHÂN CẢNH VÀ GÓC MÁY (STORYBOARD BREAKDOWN)
Hệ thống sử dụng bộ luật của Robert McKee: **1 Hành Động = 1 Shot**. File xử lý việc gộp dữ liệu này nằm tại `application/services/storyboard_service.go`.

1.  **Dữ liệu sinh Video bao gồm gì?**
    Hàm `generateVideoPrompt` sẽ lấy và nối các trường sau để gom thành lệnh sinh video gửi cho Runway/Kling:
    *   **Action** (Hành động)
    *   **Dialogue** (Hội thoại)
    *   **Camera movement** (Chuyển động máy)
    *   **Shot type & Angle** (Cỡ cảnh và Góc máy)
    *   **Scene & Atmosphere** (Bối cảnh và Không khí)
    *   **Mood & Result** (Cảm xúc và Kết quả hành động)
    *   **BGM & Sound effects** (Nhạc nền và Hiệu ứng âm thanh)
    *   *Mẹo:* Bạn hoàn toàn có thể thêm các từ khóa về hiệu ứng âm thanh (tiếng nổ, tiếng mưa, tiếng gào thét) vào kịch bản để AI nhận diện và trích xuất vào `SoundEffect`.

2.  **Dữ liệu sinh Ảnh tĩnh (First Frame) bao gồm gì?**
    Ngược lại với Video, hàm `generateImagePrompt` sẽ cố gắng MÔ PHỎNG SỰ TĨNH LẶNG:
    *   Nó sẽ dùng hàm `extractInitialPose` để chặt bỏ toàn bộ các động từ mang tính quá trình (như "đang đi", "đang chạy", "nói") và chỉ giữ lại TRẠNG THÁI BẮT ĐẦU.
    *   *Mẹo:* Nếu bạn muốn ảnh đầu tiên đẹp, hãy miêu tả tư thế đứng/ngồi tĩnh của nhân vật ở ngay đầu câu hành động. Ví dụ: *"Leon đứng khoanh tay dựa vào tường. Sau đó, anh ta lao về phía trước..."*

3.  **Kích hoạt "Cường độ Cảm xúc" (Emotion Intensity):**
    *   Hệ thống có các mức độ cảm xúc: Mức 1 (↑), Mức 2 (↑↑), Mức 3 (↑↑↑).
    *   Hãy dùng các từ như "Kinh hoàng", "Phẫn nộ tột độ", "Cười điên dại" cùng với dấu chấm than (!), viết HOA để kích hoạt AI đánh tag cảm xúc (Intensity: 3). Từ đó kéo theo Camera tiến lại gần góc Extreme Close-Up (Cực cận).

---

## 🖼️ & 🎬 BÍ QUYẾT 5: GEN ẢNH VÀ VIDEO THÀNH CÔNG

1.  **Tránh hành động ảo thuật (Magic/Morphing):** AI Video Gen (Runway/Kling) hiện tại rất dở ở các hành động biến đổi hình thái. 
    *   *Hạn chế viết:* "... biến thành quái vật", "... lửa cháy rực quanh cơ thể", "... quay 360 độ trên không". 
    *   *Nên viết action trực diện nhỏ:* "... tung cú đấm mạnh", "... bước tới phía trước", "... nước mắt rơi".
2.  **Prompt tĩnh - Chuyển động động:** Việc tách bạch Background (Scene) và Nhân vật (Character) sau đó Composed lại sẽ giúp giữ phong độ hình ảnh. Nếu ảnh bị lỗi, hãy vào thẳng danh sách Scene/Character sửa tay Prompt tiếng Anh trước khi Gen lại.
3.  **Tỷ lệ khung hình (Aspect Ratio):** Luôn đảm bảo trong cài đặt Drama Setting, Tỷ lệ (16:9 ngang, 9:16 Tiktok dọc) được chọn đúng với nhu cầu kịch bản.

---

## 📝 TEMPLATE MẪU: KỊCH BẢN EPISODE CHUẨN ĐIỂM 10

```text
[Bên trong Nhà Kho Bỏ Hoang - Đêm Khuya]
Nhà kho tối tăm, lạnh lẽo. Ánh trăng chiếu qua khe hở trên mái tôn xuống sàn bê tông dính đầy dầu mỡ. Các thùng phuy rỉ sét nằm ngổn ngang ở góc tường.

John (nam, 25 tuổi, dáng người cao gầy, tóc đen bờm xờm, mặc áo hoodie xám) bị trói chặt vào một chiếc ghế gỗ. 
John cố gắng giãy giụa. Sợi dây thừng cứa vào cổ tay khiến anh nhăn nhó vì đau đớn. [SFX: Tiếng ghế gỗ cọt kẹt, nhịp thở dốc]
John thở hổn hển, mồ hôi nhễ nhại trên trán.

[Tiếng bước chân vang lên]
Marcus (nam, 50 tuổi, tướng mạo bệ vệ, khuôn mặt có vết sẹo dài ở đuôi mắt trái, mặc vest đen sang trọng) từ từ bước từ trong bóng tối ra vùng sáng.
Marcus cầm một điếu xì gà đang cháy dở tỏa khói nghi ngút.
Marcus nhếch mép cười khinh bỉ. Lão ném điếu xì gà xuống đất.
Lão dùng mũi giày da bóng lộn vò nát điếu xì gà. [SFX: Tiếng giày da di xì gà]

Marcus từ từ cúi người người xuống, ghé sát mặt vào John.
Marcus gầm lên với ánh mắt đầy sát khí: "Mày cất bản đồ ở đâu?!"

John trừng mắt nhìn lại. Anh nghiến răng cắn chặt môi, kiên quyết rít lên: "Mày có giết tao... tao cũng không nói!"
Marcus vung tay tát mạnh mặt John. [SFX: Tiếng tát vang dội]
Đầu John bật sang một bên, khóe môi rỉ máu đỏ tươi. Mắt anh vẫn rực lửa hận thù.
```

---

## 🛠️ DÀNH CHO DEVELOPER: NẾU MUỐN CHỈNH SỬA THÌ SỬA Ở ĐÂU?

Hệ thống được thiết kế rất module hỏa tại thư mục `application/services/`. Dưới đây là các file bạn cần quan tâm khi muốn "độ" lại AI:

1. **Thay đổi câu lệnh AI (Prompt Engineering):**
   * **File:** `prompt_i18n.go`
   * **Mục đích:** Đây là trái tim của hệ thống. Tất cả các prompt ("Vai trò của bạn là...", "Luật breakdown shot...", "Format JSON...") đều nằm ở đây.
   * **Cách sửa:** Bạn có thể tìm các hàm như `GetStoryboardSystemPrompt()`, `GetCharacterExtractionPrompt()`, thay đổi quy tắc phân rã, hoặc bắt AI trả thêm các trường tùy chỉnh (như màu sắc, ánh sáng riêng).

2. **Cách ghép các thông số thành đoạn hội thoại gửi AI tạo Ảnh (Image Generation):**
   * **File:** `storyboard_service.go`
   * **Hàm cần chú ý:** `generateImagePrompt(sb Storyboard)`
   * **Mục đích:** Nếu bạn cảm thấy ảnh tĩnh AI sinh ra chưa đủ đẹp hoặc muốn truyền thêm góc máy (`Angle`) vào prompt tạo ảnh (hiện tại hàm này đang ẩn góc máy đi để tránh lỗi), hãy sửa ở đây.

3. **Cách ghép các thông số thành lệnh gửi AI tạo Video (Video Generation):**
   * **File:** `storyboard_service.go`
   * **Hàm cần chú ý:** `generateVideoPrompt(sb Storyboard)`
   * **Mục đích:** Tương tự như trên. Nếu bạn sử dụng model Video khác (Runway, Kling) mà chúng yêu cầu format prompt riêng (ví dụ thêm prefix `Animate this:`), hãy nối chuỗi cứng trực tiếp vào hàm này.

4. **Sửa tham số Model (Nhiệt độ, Max Token):**
   * **File:** Tại từng file chạy nhiệm vụ tương ứng như `storyboard_service.go`, `script_generation_service.go`.
   * **Cách sửa:** Tìm hàm nội bộ gọi API như `client.GenerateText(prompt, "", ai.WithTemperature(0.7), ai.WithMaxTokens(8000))` để điều chỉnh khả năng sáng tạo hoặc giới hạn độ dài của AI.

5. **Xử lý hậu kỳ file AI trả về (Lưu Base64, Download):**
   * **File:** `image_generation_service.go` (ảnh) và `video_generation_service.go` (video).
   * **Hàm cần chú ý:** `completeImageGeneration(...)`
   * **Mục đích:** Chỉnh sửa logic xử lý nếu API đổi host, trả về loại dữ liệu khác (`data:image/...` hoặc URL) và quản lý tiến trình lưu file xuống thư mục `storage`.