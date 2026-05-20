# CẨM NANG BẤM NÚT: HÀNH TRÌNH TỪNG NÚT BẤM CỦA NGƯỜI SÁNG TẠO (USER CLICK JOURNEY)

Tài liệu này là hướng dẫn thực hành từng bước (Step-by-step click guide) dành cho Người Sáng Tạo (User). Bất kể bạn muốn AI tự động sinh kịch bản hay tự tay thiết kế rẽ nhánh, cẩm nang này sẽ chỉ rõ: **"Bạn cần click nút nào, ở trang nào, nhập gì, và kết quả sẽ hiển thị ra sao."**

---

## PHẦN 1: HÀNH TRÌNH 1 - CHẠY DÀN Ý TỰ ĐỘNG BẰNG AI (AUTOMATED PIPELINE)

*Áp dụng khi bạn đã có một ý tưởng sơ bộ và muốn AI (3 Agent) tự thiết kế toàn bộ thế giới, nhân vật, 15-20 tập phim rẽ nhánh và viết kịch bản thô.*

### 🛠️ Các nút cần bấm & Quy trình:

#### Bước 1: Khởi tạo ý tưởng
1.  Tại màn hình **Project Overview** (Trang chính của dự án), cuộn xuống thẻ **Story generator**.
2.  Nhập ý tưởng ngắn của bạn vào ô textarea **"Idea / hook (can be short)"** (Ví dụ: *"Một lập trình viên phát hiện thế giới mình sống là giả lập và tìm cách hack hệ thống"*).
3.  Click nút **[Save Idea]** để lưu lại ý tưởng làm seed prompt cho AI.

#### Bước 2: Kích hoạt Agent sinh đồ thị
Bạn có 2 sự lựa chọn bấm nút:

*   **Lựa chọn A - Chạy từng Agent để kiểm soát (Khuyên dùng để học luồng):**
    1.  Click nút **[Agent 1: Architect World & Characters]** (Màu xanh dương) -> Hệ thống xuất hiện Progress Bar chạy ngầm. 
        *   *Kết quả:* Khi đạt 100%, ô **Pipeline Data (Preview)** bên dưới sẽ xuất hiện danh sách Nhân vật (Characters) và các Episode rẽ nhánh (N101, N102...) do Agent 1 thiết kế.
    2.  Click tiếp nút **[Agent 2: Build Beats & Outfits]** (Màu cam).
        *   *Kết quả:* AI sẽ chạy ngầm sinh nhịp truyện (Micro-beats) và Bối cảnh (Scenes) cho toàn bộ các tập. Bảng Preview sẽ xuất hiện thêm các Scene tag màu cam.
    3.  Click tiếp nút **[Agent 3: Design Markdown Scripts]** (Màu xanh lá).
        *   *Kết quả:* AI viết kịch bản chi tiết cho từng tập. Khi hoàn thành, hệ thống hiển thị thông báo: *"Agent 3 Complete..."*.
*   **Lựa chọn B - Chạy một mạch (Full Auto):**
    1.  Click nút **[Run Full Pipeline]** (Màu đỏ).
    2.  Hệ thống sẽ tự động chạy liên tiếp Agent 1 -> Agent 2 -> Agent 3. Bạn chỉ cần ngồi xem Progress Bar tăng dần lên 100%.

#### Bước 3: Xem và Duyệt thành quả
1.  **Xem Cốt truyện tổng:** Nhìn ngay vào ô **Pipeline Data (Preview)** -> Mục **🎬 GLOBAL STORYLINE & HOOK** sẽ hiển thị đoạn giới thiệu drama cuốn hút do AI trau chuốt.
2.  **Xem chi tiết phân mảnh:**
    *   Click Tab **Episodes** ở menu trên cùng để mở danh sách tập.
    *   Tại bảng danh sách Episode, tìm cột ngoài cùng bên trái, click vào biểu tượng **Mũi tên chỉ xuống (Expand)** ở dòng bất kỳ.
    *   *Kết quả:* Panel mở rộng sẽ hiển thị rõ ràng:
        *   **📖 PLOT OUTLINE (Agent 1):** Tóm tắt cốt truyện cốt lõi của tập đó.
        *   **🎬 MICRO-BEATS (Agent 2):** Các nhịp hành động chi tiết.
        *   **📋 STATE SNAPSHOT (Agent 2):** Trạng thái vật phẩm và các nhân vật đang ở đâu.

---

## PHẦN 2: HÀNH TRÌNH 2 - TỰ THIẾT KẾ & ĐẤU NỐI NHÁNH CỐT TRUYỆN THỦ CÔNG (MANUAL GRAPH CONNECT)

*Áp dụng khi bạn muốn tự tay viết cốt truyện nhánh, rẽ đôi ngả đường câu chuyện theo ý mình, hoặc thêm các phân cảnh ẩn mà AI không tự tạo ra.*

### 🛠️ Các nút cần bấm & Quy trình:

#### Bước 1: Tạo Episode Node mới nằm ngoài dòng chảy
1.  Vào Tab **Episodes** trên menu của trang Project Overview.
2.  Bên phải, click nút **[+ Create New Episode]** (màu xanh dương).
3.  Nhập tên tập phim (Ví dụ: *"Quyết định sinh tử"*) -> Click **Confirm** để tạo.
4.  *Kết quả:* Tập phim mới xuất hiện cuối bảng danh sách. Lúc này chưa có liên kết, Node ID sẽ hiện `N/A`.

#### Bước 2: Vào trang sản xuất tập phim vừa tạo
1.  Tại dòng Episode vừa tạo mới trong bảng, click nút **[Produce]** (biểu tượng Bút chì/Chỉnh sửa) ở cột thao tác bên phải.
2.  Giao diện **Episode Production** (3 Bước Sản Xuất) mở ra.

#### Bước 3: Mở Dialog Chỉnh Sửa Kết Nối
1.  Tại Bước 1 (Episode Content), nhìn xuống thanh điều hướng có nền xám/tối mang nhãn **← FROM** và **→ TO**.
2.  Click nút **[⚙ Edit Connections]** ở góc phải của thanh điều hướng này.
3.  *Kết quả:* Dialog **"Edit Narrative Connections"** mở ra với **2 phần riêng biệt**:

#### Bước 4: Cấu hình Kết Nối ĐẾN tập hiện tại (← INCOMING)

> *Phần này cho phép bạn cấu hình "tập nào CÓ LỰA CHỌN dẫn đến tập hiện tại". Đây là chức năng **HOÀN TOÀN MỚI** so với trước.*

*   Nhìn vào phần đầu dialog có nhãn màu xám **← INCOMING**.
*   Click nút **[+ Add Incoming Connection]**.
*   Tại ô **SOURCE EPISODE**, chọn tập cha từ dropdown (Ví dụ: `Ep 3: The Lion's Roar (N103)`).
*   Tại ô **CHOICE LABEL ON THAT EPISODE**, nhập tên lựa chọn sẽ hiển thị trên tập cha khi người xem đang ở tập đó (Ví dụ: *"Chọn đi đường hầm ẩn"*).
*   *Ý nghĩa:* Sau khi Save, tập N103 sẽ tự động được thêm một lựa chọn mới trỏ về tập hiện tại của bạn.
*   Muốn **xóa kết nối đến**: Click nút **Thùng rác** đỏ bên cạnh dòng tương ứng.

#### Bước 5: Cấu hình Lựa Chọn Ra từ tập hiện tại (→ OUTGOING)

> *Phần này cấu hình "từ tập hiện tại, người xem có thể đi đến những tập nào tiếp theo".*

*   Nhìn xuống phần dưới dialog có nhãn màu tím **→ OUTGOING**.
*   Click nút **[+ Add Outgoing Branch / Choice]**.
*   Tại ô **CHOICE LABEL**, nhập câu lựa chọn hiển thị cho người xem (Ví dụ: *"Tiếp tục theo dấu chân Elena"*).
*   Tại ô **TARGET EPISODE**, chọn tập đích từ dropdown (Ví dụ: `Ep 5: The Scholar's Gambit`).
*   Muốn **xóa lựa chọn ra**: Click nút **Thùng rác** đỏ bên cạnh.

#### Bước 6: Lưu lại tất cả thay đổi
1.  Click nút **[Save Connections]** ở footer dialog.
2.  *Kết quả:* Dialog đóng lại. Hệ thống đồng thời cập nhật **cả 2 đầu kết nối**:
    *   Tập cha (Source Episode từ INCOMING) sẽ có thêm một lựa chọn mới trong mảng `choices` của chúng.
    *   Tập hiện tại sẽ có danh sách lựa chọn ra mới (OUTGOING choices).
3.  Trên thanh điều hướng xám bên dưới kịch bản, nút **← FROM** và **→ TO** sẽ cập nhật ngay lập tức hiển thị đúng các kết nối vừa thiết lập. Bấm vào bất kỳ nút nào sẽ nhảy nhanh sang tập đó.
4.  Sơ đồ đồ thị **Story Graph** màn hình tối trên trang **Project Overview** cũng tự vẽ thêm các nhánh mới này theo thời gian thực.

---

## PHẦN 3: HÀNH TRÌNH 3 - VIẾT KỊCH BẢN & DÙNG "ĐŨA THẦN" CẬP NHẬT TRẠNG THÁI (SCRIPTING & MAGIC WAND)

*Áp dụng khi bạn tự gõ kịch bản cho một tập phim và muốn AI tự lọc xem trong kịch bản có nhân vật hay bối cảnh nào mới để thêm vào thư viện hình ảnh mà không làm hỏng dữ liệu cũ.*

### 🛠️ Các nút cần bấm & Quy trình:

1.  Tại trang **Episode Production (Bước 1: Episode Content)**, click nút **[✏️ Edit Script]** (Màu cam) nằm bên phải tiêu đề kịch bản.
2.  Khung text editor sẽ mở khóa cho phép chỉnh sửa. Hãy tự do gõ kịch bản kịch tính dạng Markdown của bạn vào đây.
3.  Click nút **[💾 Save Script]** (Màu xanh dương) để lưu lại nội dung kịch bản vừa viết.
4.  Bây giờ, click nút **[✨ Extract Characters & Scenes]** (Nút Đũa thần - Magic Wand).
5.  *Kết quả:* AI sẽ bắt đầu quét bất đồng bộ phần kịch bản bạn vừa gõ:
    *   Nếu phát hiện nhân vật cũ (Ví dụ: *Mi-ra*), AI sẽ giữ nguyên ID và FaceID đã có từ trước trong thư viện.
    *   Nếu phát hiện thực thể mới tinh chưa từng có (Ví dụ nhân vật: *Gã Ninja bí ẩn*), AI sẽ tự tạo một thực thể nhân vật mới trong thư viện, gán cho họ một `base_image_prompt` A-Pose chuẩn xác để phục vụ cho khâu sinh ảnh tiếp theo.
6.  Cuộn xuống dưới cùng của trang, mở panel collapse **Plot Outline (Agent 1 Architect)** hoặc **Micro-beats (Agent 2 Builder)** để xem tóm tắt hoặc nhịp phân tích của AI.

---

## PHẦN 4: HÀNH TRÌNH 4 - LÀM MỊN & SINH ẢNH NHÂN VẬT/BỐI CẢNH (STEP 2: GENERATE IMAGES)

*Áp dụng khi bạn đã có kịch bản và danh sách thực thể bối cảnh/nhân vật, giờ là lúc tạo ảnh minh họa để làm phim.*

### 🛠️ Các nút cần bấm & Quy trình:

1.  Tại giao diện sản xuất, click Tab **[2. Generate Images]** ở menu phân bước phía trên.
2.  Giao diện hiển thị danh sách các thẻ nhân vật (Character Cards) dạng A-Pose nền trắng và thẻ bối cảnh (Scene Cards) góc rộng cực kỳ premium.
3.  **Cách sinh ảnh đơn lẻ:**
    *   Hover chuột vào card của Nhân vật hoặc Bối cảnh chưa có ảnh.
    *   Click vào nút **[✨ Generate]** trên Card. Hệ thống sẽ kết nối với GPU/AI để vẽ ảnh. Khi hoàn thành, ảnh chân dung/bối cảnh sẽ hiện lên thay thế cho placeholder.
4.  **Cách sinh ảnh hàng loạt để tiết kiệm thời gian:**
    *   Nhìn lên góc trên bên phải trang, click nút **[✨ Batch Generate Images]**.
    *   *Kết quả:* Hệ thống sẽ tự động chạy song song sinh ảnh cho tất cả những nhân vật và bối cảnh nào chưa có ảnh minh họa.
5.  **Cách thay đổi ảnh thủ công (Pro Mode):**
    *   Nếu không thích ảnh AI sinh, click nút **[📤 Upload Image]** trên Card để chọn file ảnh chân dung của riêng bạn tải lên.
    *   Click nút **[✏️ Edit Prompt]** trên Card để thay đổi câu lệnh prompt của nhân vật hoặc bối cảnh (Ví dụ: thêm tags *"blonde hair, red jacket"*), sau đó click **[Generate]** lại để lấy kết quả mới.

---

## PHẦN 5: PHÂN CHIA SHOT LIST & LÀM VIDEO (STEP 3: SPLIT STORYBOARD)

*Bước cuối cùng để chia nhỏ kịch bản thô thành các phân cảnh quay (Shots), chọn góc máy và dựng thành phim hoàn chỉnh.*

### 🛠️ Các nút cần bấm & Quy trình:

1.  Click Tab **[3. Split Storyboard]** ở menu phân bước.
2.  Nếu đây là lần đầu tiên sản xuất tập này, màn hình hiển thị trạng thái trống. Click nút **[🎬 Split Storyboard]** (Màu xanh dương) ở giữa trang.
3.  *Kết quả:* Hệ thống tự động phân tích kịch bản thô và chia thành một Shot List (danh sách cảnh quay) chi tiết bên dưới. Mỗi Shot sẽ được gán sẵn: góc máy (Angle), chuyển động (Movement), nhân vật xuất hiện, trang phục (Outfit ID), và lời thoại tương ứng.
4.  **Khởi tạo khung hình tĩnh cho Shot (First-frame Prompt):**
    *   Với mỗi Shot, click nút **[✨ Gen Prompt]** tại cột First Frame. AI sẽ tự động sinh prompt khung hình đầu tiên dựa vào bối cảnh + quần áo nhân vật đã định ở Step 2.
    *   Click nút **[🎨 Gen Image]** để sinh ảnh tĩnh cho Shot đó.
5.  **Dựng động (Generate Video):**
    *   Sau khi Shot đã có ảnh tĩnh First Frame, click nút **[🎥 Gen Video]** ngay bên cạnh.
    *   *Kết quả:* AI sẽ nhận ảnh tĩnh đầu vào và prompt chuyển động để dựng thành clip video ngắn 3-5 giây sinh động.
6.  Click nút **[Play All / Preview Episode]** ở góc trên để chạy thử toàn bộ tập phim hoàn chỉnh từ đầu đến cuối trước khi xuất bản!
