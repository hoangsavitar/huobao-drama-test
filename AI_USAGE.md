# Phân Tích AI trong Project Huobao Drama

> **Tài liệu này tổng hợp toàn bộ cách sử dụng AI trong project: các prompt, kỹ thuật, và pipeline tạo text, ảnh, video.**

---

## Mục Lục

1. [Tổng quan kiến trúc AI](#1-tổng-quan-kiến-trúc-ai)
2. [Pipeline tạo TEXT (LLM)](#2-pipeline-tạo-text-llm)
3. [Pipeline tạo ẢNH (Image Generation)](#3-pipeline-tạo-ảnh-image-generation)
4. [Pipeline tạo VIDEO (Video Generation)](#4-pipeline-tạo-video-video-generation)
5. [Catalog toàn bộ Prompt](#5-catalog-toàn-bộ-prompt)
6. [Kỹ thuật Prompt Engineering](#6-kỹ-thuật-prompt-engineering)
7. [Luồng hoạt động tổng thể (End-to-End)](#7-luồng-hoạt-động-tổng-thể-end-to-end)

---

## 1. Tổng Quan Kiến Trúc AI

Project hỗ trợ **3 loại AI service**:

| Service Type | Mục đích | Providers hỗ trợ |
|---|---|---|
| `text` | Tạo kịch bản, phân cảnh, prompt | OpenAI-compatible, Gemini |
| `image` | Tạo ảnh nhân vật, bối cảnh, frame | OpenAI/DALL-E, Gemini Imagen, VolcEngine Doubao |
| `video` | Tạo video từ ảnh + prompt | Minimax Hailuo, VolcesArk (Doubao Seedance), Chatfire, OpenAI Sora |

**Cấu hình AI** được lưu trong database (`ai_service_configs` table) với các thuộc tính:
- `provider`: tên nhà cung cấp (`openai`, `gemini`, `chatfire`, `doubao`, `minimax`, v.v.)
- `base_url`, `api_key`, `model`, `endpoint`, `query_endpoint`
- `priority`: ưu tiên chọn config khi có nhiều config cùng loại
- `is_active`, `is_default`

---

## 2. Pipeline Tạo TEXT (LLM)

### 2.1 Các AI Client cho Text

**File:** `pkg/ai/client.go`, `pkg/ai/openai_client.go`, `pkg/ai/gemini_client.go`

```
Interface AIClient:
  GenerateText(prompt, systemPrompt, options...) → string
  GenerateImage(prompt, size, n) → []string
  TestConnection() → error
```

**OpenAI-compatible Client** (`openai_client.go`):
- Endpoint: `/v1/chat/completions` (hoặc `/chat/completions`)
- Request format: `{ model, messages: [{role, content}], temperature, max_tokens }`
- System prompt → role `"system"`, user prompt → role `"user"`
- Tự động retry với `max_completion_tokens` nếu gặp lỗi "Unsupported parameter: 'max_tokens'"

**Gemini Client** (`gemini_client.go`):
- Endpoint: `/v1beta/models/{model}:generateContent`
- Request format: `{ contents: [{parts: [{text}], role: "user"}], systemInstruction: {parts: [{text}]} }`
- API key được gắn vào URL: `?key=<API_KEY>`

### 2.2 Luồng tạo TEXT

Tất cả các tác vụ tạo text đều chạy **bất đồng bộ (async goroutine)**:

```
API Request → CreateTask() → goroutine (processXxxGeneration) → UpdateTaskStatus
                ↓                                                      ↓
           Return task_id                                     Return via polling
```

### 2.3 Các tác vụ TEXT cụ thể

#### A. Tạo đại cương (Outline Generation)

**Service:** `drama_service.go` → `GenerateOutline()`

**System Prompt** (`GetOutlineGenerationPrompt`):
```
[Tiếng Việt dùng zh prompt]
Bạn là biên kịch phim ngắn chuyên nghiệp. Dựa trên chủ đề và số tập, tạo đại cương phim ngắn hoàn chỉnh,
lên kế hoạch hướng phát triển cho từng tập.

Yêu cầu:
1. Cốt truyện chặt chẽ, mâu thuẫn mạnh, tiết tấu nhanh
2. Mỗi tập có mâu thuẫn độc lập, đồng thời đẩy cốt truyện chính
3. Arc nhân vật rõ ràng, biến đổi rõ rệt
4. Cliffhanger hợp lý, hấp dẫn khán giả
5. Chủ đề rõ ràng, cảm xúc cốt lõi rõ ràng

Output: JSON { title, episodes: [{episode_number, title, summary, conflict, cliffhanger}] }
```

**User Prompt** template: `"Hãy tạo đại cương phim ngắn cho chủ đề: {theme}"`

---

#### B. Tạo kịch bản phân tập (Episode Script Generation)

**Service:** `drama_service.go` → `GenerateEpisodeScript()`

**System Prompt** (`GetEpisodeScriptPrompt`):
```
Bạn là biên kịch phim ngắn chuyên nghiệp. Mở rộng đại cương thành kịch bản chi tiết.
- Mỗi tập ~3 phút (150-300 giây)
- 800-1200 từ mỗi tập, nhiều đối thoại
- Có lời thoại và hành động, không chỉ mô tả

Output: JSON { episodes: [{episode_number, title, script_content}] }
```

**User Prompt** template:
```
Đại cương kịch bản: {outline}
{character_info}
Hãy viết kịch bản chi tiết cho {n} tập...
Lưu ý: phải tạo đủ {n} tập, mảng episodes phải có {n} phần tử
```

---

#### C. Trích xuất nhân vật (Character Extraction)

**Service:** `script_generation_service.go` → `GenerateCharacters()`

**System Prompt** (`GetCharacterExtractionPrompt(style)`):
```
Bạn là chuyên gia phân tích nhân vật, giỏi trích xuất thông tin nhân vật từ kịch bản.

Yêu cầu:
1. Trích xuất tất cả nhân vật có tên (bỏ qua người qua đường vô danh)
2. Với mỗi nhân vật:
   - name: tên nhân vật
   - role: main/supporting/minor
   - appearance: mô tả ngoại hình (150-300 từ, đủ chi tiết cho AI tạo ảnh)
   - personality: tính cách (100-200 từ)
   - description: câu chuyện nền và mối quan hệ (100-200 từ)
3. Mô tả ngoại hình phải đủ chi tiết cho AI tạo ảnh: giới tính, tuổi, vóc dáng, khuôn mặt,
   kiểu tóc, trang phục - KHÔNG gồm bối cảnh, môi trường
4. Style yêu cầu: {style}
5. Tỷ lệ ảnh: 16:9

Output: JSON array thuần túy, bắt đầu bằng [, kết thúc bằng ]
```

**User Prompt**: `"Nội dung kịch bản:\n{script}\n\nHãy trích xuất tối đa {count} nhân vật chính..."`

---

#### D. Trích xuất cảnh/bối cảnh (Scene Extraction)

**Service:** `storyboard_composition_service.go` / `drama_service.go`

**System Prompt** (`GetSceneExtractionPrompt(style)`):
```
[Nhiệm vụ] Trích xuất tất cả bối cảnh nền độc đáo từ kịch bản

[Yêu cầu]
1. Xác định các cảnh khác nhau (địa điểm + thời gian)
2. Tạo prompt ảnh tiếng Trung chi tiết cho mỗi cảnh
3. QUAN TRỌNG: Mô tả cảnh phải là NỀN THUẦN TÚY - không có nhân vật, không có hành động
4. Prompt phải: dùng tiếng Trung, mô tả chi tiết cảnh/thời gian/bầu không khí/phong cách,
   ghi rõ "không có người, không có nhân vật, cảnh trống"
5. Phong cách: {style}, Tỷ lệ: 16:9

Output: JSON array [ {location, time, prompt} ]
```

---

#### E. Tạo phân cảnh (Storyboard Generation)

**Service:** `storyboard_service.go` → `GenerateStoryboard()`

**System Prompt** (`GetStoryboardSystemPrompt()`):
```
[Vai trò] Bạn là đạo diễn phân cảnh phim kỳ cựu, tinh thông lý thuyết phân cảnh của Robert McKee,
giỏi xây dựng nhịp điệu cảm xúc.

[Nhiệm vụ] Phân tách kịch bản theo ĐƠN VỊ HÀNH ĐỘNG ĐỘC LẬP thành phương án phân cảnh.

[Nguyên tắc phân cảnh]
1. Phân đơn vị hành động: Mỗi cảnh phải tương ứng với một hành động hoàn chỉnh và độc lập
   - 1 hành động = 1 cảnh (đứng dậy, đi lại, nói một câu, phản ứng bằng biểu cảm, v.v.)
   - Cấm gộp nhiều hành động (đứng dậy + đi lại = 2 cảnh riêng)

2. Chuẩn loại cảnh (chọn theo nhu cầu kể chuyện):
   - Toàn cảnh xa: môi trường, xây dựng không khí
   - Toàn cảnh: hành động toàn thân, quan hệ không gian
   - Trung cảnh: đối thoại tương tác, giao lưu cảm xúc
   - Cận cảnh: chi tiết, biểu đạt cảm xúc
   - Đặc tả: đạo cụ quan trọng, cảm xúc mạnh

3. Yêu cầu góc máy & di chuyển camera:
   - Cố định, Đẩy vào, Kéo ra, Quét ngang, Theo đuổi, Di chuyển

4. Chỉ số cảm xúc: ↑↑↑(3), ↑↑(2), ↑(1), →(0), ↓(-1)

Output: JSON array thuần túy, bắt đầu bằng [, kết thúc bằng ]
Mỗi cảnh: shot_number, scene_description, shot_type, camera_angle, camera_movement,
           action, result, dialogue, emotion, emotion_intensity
```

**User Prompt** (được xây dựng động với context đầy đủ):
```
{system_prompt}

【Nội dung kịch bản】
{script_content}

【Nhiệm vụ】
Phân tách kịch bản theo đơn vị hành động độc lập...

【Danh sách nhân vật có thể dùng】
[{"id": 159, "name": "Trần Tranh"}, {"id": 160, "name": "Lý Phương"}]

QUAN TRỌNG: Trong trường characters, chỉ dùng ID nhân vật (số) từ danh sách trên.

【Danh sách bối cảnh đã trích xuất】
[{"id": 1, "location": "Kho bến cảng bỏ hoang", "time": "Đêm khuya"}]

QUAN TRỌNG: Trong trường scene_id, phải chọn ID bối cảnh phù hợp nhất từ danh sách trên.
```

**max_tokens:** 16000 (để đảm bảo trả về đầy đủ JSON)

---

#### F. Tạo Prompt Frame (Frame Prompt Generation)

**Service:** `frame_prompt_service.go` → `GenerateFramePrompt()`

Hỗ trợ 5 loại frame:

| Loại Frame | Mô tả |
|---|---|
| `first` | Cảnh tĩnh đầu tiên - trạng thái ban đầu trước khi hành động |
| `key` | Khoảnh khắc cao trào nhất của hành động |
| `last` | Cảnh tĩnh cuối - trạng thái sau khi hành động kết thúc |
| `panel` | Dải phân cảnh 3-4 ô ngang (first+key+last) |
| `action` | Sequence động tác 3x3 = 9 ô (xem chi tiết bên dưới) |

**System Prompt - First Frame** (`GetFirstFramePrompt(style)`):
```
Bạn là chuyên gia prompt tạo ảnh. Dựa trên thông tin cảnh quay, tạo prompt cho AI tạo ảnh.

QUAN TRỌNG: Đây là frame đầu tiên - cảnh hoàn toàn tĩnh, thể hiện trạng thái ban đầu trước hành động.

Điểm chính:
1. Tập trung vào trạng thái ban đầu - khoảnh khắc trước khi hành động
2. KHÔNG gồm bất kỳ hành động hay chuyển động nào
3. Mô tả tư thế ban đầu, vị trí và biểu cảm của nhân vật
4. Có thể gồm không khí cảnh và chi tiết môi trường
5. Loại cảnh quay quyết định bố cục và góc nhìn
Phong cách: {style}, Tỷ lệ: 16:9

Output: JSON { prompt: "...", description: "..." }
```

**System Prompt - Key Frame** (`GetKeyFramePrompt(style)`):
```
QUAN TRỌNG: Đây là frame chính - bắt khoảnh khắc căng thẳng và hấp dẫn nhất của hành động.

Điểm chính:
1. Tập trung vào khoảnh khắc hấp dẫn nhất của hành động
2. Bắt đỉnh điểm biểu đạt cảm xúc
3. Nhấn mạnh sức căng động
4. Thể hiện trạng thái đỉnh điểm hành động và biểu cảm nhân vật
5. Có thể gồm motion blur hoặc hiệu ứng động
```

**System Prompt - Last Frame** (`GetLastFramePrompt(style)`):
```
QUAN TRỌNG: Đây là frame cuối cùng - cảnh tĩnh thể hiện trạng thái và kết quả sau khi hành động kết thúc.

Điểm chính:
1. Tập trung vào trạng thái cuối sau khi hành động hoàn tất
2. Thể hiện kết quả của hành động
3. Mô tả tư thế và biểu cảm cuối của nhân vật
4. Nhấn mạnh trạng thái cảm xúc sau hành động
5. Bắt khoảnh khắc bình lặng sau khi hành động kết thúc
```

**System Prompt - Action Sequence (3x3 Grid)** (`GetActionSequenceFramePrompt(style)`):
```
[Vai trò] Bạn là chuyên gia kể chuyện bằng hình ảnh. Cần tạo MỘT prompt mô tả sequence 3x3.

[Logic cốt lõi]
1. Toàn thể: Đây là MỘT hình ảnh hoàn chỉnh gồm bố cục 3x3, thể hiện 9 hành động liên tiếp.
2. Nhất quán: Nhân vật, trang phục, phong cách phải hoàn toàn nhất quán trong tất cả 9 ô.
3. Tiến trình: Ô 1→9 thể hiện sequence hành động hoàn chỉnh.
4. Prompt Engineering: Dùng từ vựng hình ảnh chất lượng cao (ánh sáng, kết cấu, bố cục, độ sâu trường ảnh).

[Quy tắc từng ô]
- Ô 1: Chuẩn bị / Tư thế ban đầu
- Ô 2: Chuẩn bị trước / Điều chỉnh thân thể
- Ô 3: Khởi động / Bắt đầu chuyển động
- Ô 4: Tăng tốc / Tích lũy năng lượng
- Ô 5: Đỉnh sức mạnh / Trước khi bùng nổ
- Ô 6: Bùng nổ hành động / Khoảnh khắc cao trào
- Ô 7: Giải phóng năng lượng / Quán tính tiếp tục
- Ô 8: Phanh lại / Dần kết thúc
- Ô 9: Hoàn toàn kết thúc / Trở về tĩnh

Tỷ lệ: {imageRatio}

Output: JSON { prompt: "...", description: "..." }
```

**User Prompt** cho các frame (được xây dựng từ context storyboard):
```
Thông tin cảnh quay:
Mô tả cảnh: {description}
Bối cảnh: {location}, {time}
Nhân vật: {character_names}
Hành động: {action}
Kết quả: {result}
Đối thoại: {dialogue}
Không khí: {atmosphere}
Loại cảnh: {shot_type}
Góc: {angle}
Di chuyển: {movement}

Hãy tạo prompt ảnh [cho frame đầu/key/cuối]...
```

---

#### G. Trích xuất đạo cụ (Prop Extraction)

**System Prompt** (`GetPropExtractionPrompt(style)`):
```
Hãy trích xuất đạo cụ quan trọng từ kịch bản sau.

[Yêu cầu]
1. Chỉ trích xuất đạo cụ quan trọng với cốt truyện hoặc có đặc điểm thị giác đặc biệt
2. KHÔNG trích xuất vật dụng hàng ngày thông thường
3. Nếu đạo cụ có chủ sở hữu rõ ràng, ghi chú trong mô tả
4. Trường "image_prompt" dùng cho AI tạo ảnh, mô tả chi tiết ngoại hình, chất liệu, màu sắc
5. Phong cách: {style}, Tỷ lệ: 1:1

Output: JSON array [ {name, type, description, image_prompt} ]
```

---

## 3. Pipeline Tạo ẢNH (Image Generation)

### 3.1 Các Image Client

**File:** `pkg/image/image_client.go`, `openai_image_client.go`, `gemini_image_client.go`, `volcengine_image_client.go`

```
Interface ImageClient:
  GenerateImage(prompt, opts...) → *ImageResult
  GetTaskStatus(taskID) → *ImageResult
```

#### OpenAI/DALL-E Client
- Endpoint: `/v1/images/generations`
- Request: `{ model, prompt, size, quality, n, image: [referenceImages] }`
- Trả về URL hoặc base64

#### Gemini Image Client
- Endpoint: `/v1beta/models/{model}:generateContent`
- Request: `{ contents: [{parts: [inlineData + text]}], generationConfig: {responseModalities: ["IMAGE"]} }`
- Hỗ trợ ảnh tham chiếu: tải về và convert sang base64 để gửi qua `inlineData`
- Trả về base64, không có URL

#### VolcEngine (Doubao Seedream) Client
- Endpoint: `/api/v3/images/generations`
- Request: `{ model, prompt, image: [referenceImages], size, watermark: false }`
- Models: `doubao-seedream-4-5-251128` (mặc định size 2K), các model khác size 1K
- Hỗ trợ `sequential_image_generation: "disabled"`

### 3.2 Luồng tạo ảnh

```
GenerateImage(request) → lưu DB (status: pending) → goroutine ProcessImageGeneration
                                                              ↓
                                              getImageClientWithModel(provider, model)
                                                              ↓
                                        Xây dựng prompt đầy đủ:
                                          1. Style prompt (GetStylePrompt)
                                          2. User prompt
                                          3. + imageRatio
                                          4. + consistency instruction (nếu có ref images)
                                                              ↓
                                              client.GenerateImage(fullPrompt, opts)
                                                              ↓
                                          Lưu ảnh về local storage → update DB
```

### 3.3 Xây dựng Prompt ảnh đầy đủ

```go
// Thêm style prompt (nếu drama.Style != "" và != "realistic")
stylePrompt := GetStylePrompt(drama.Style)
fullPrompt = stylePrompt + "\n\n" + userPrompt

// Thêm tỷ lệ ảnh
fullPrompt += ", imageRatio:16:9"

// Nếu có ảnh tham chiếu
fullPrompt += "\n\n**Quan trọng:** Phải tuân thủ nghiêm ngặt các yếu tố trong ảnh tham chiếu, " +
    "giữ tính nhất quán của cảnh và nhân vật"
```

> **Lưu ý bảo mật:** Các chuỗi user prompt (ví dụ: tên nhân vật, nội dung kịch bản) được truyền thẳng vào prompt mà không qua sanitization chuyên biệt. Hệ thống dựa vào cơ chế phân tách system/user message của LLM API để giảm thiểu rủi ro prompt injection.

### 3.4 Các loại ảnh được tạo

| `image_type` | Mục đích |
|---|---|
| `character` | Ảnh nhân vật |
| `scene` | Ảnh bối cảnh (nền thuần túy, không nhân vật) |
| `storyboard` | Ảnh frame phân cảnh |

### 3.5 Các Option khi tạo ảnh

```go
WithNegativePrompt(prompt)   // Prompt phủ định
WithSize(size)               // Kích thước ảnh
WithQuality(quality)         // Chất lượng
WithStyle(style)             // Phong cách
WithSteps(steps)             // Số bước (cho Stable Diffusion)
WithCfgScale(scale)          // Guidance scale
WithSeed(seed)               // Seed để tái tạo
WithModel(model)             // Override model
WithDimensions(width, height) // Kích thước pixel
WithReferenceImages(images)  // Ảnh tham chiếu
```

---

## 4. Pipeline Tạo VIDEO (Video Generation)

### 4.1 Các Video Client

**File:** `pkg/video/video_client.go`, `minimax_client.go`, `volces_ark_client.go`, `chatfire_client.go`, `openai_sora_client.go`

```
Interface VideoClient:
  GenerateVideo(imageURL, prompt, opts...) → *VideoResult
  GetTaskStatus(taskID) → *VideoResult
```

#### Minimax (Hailuo) Client
- Endpoint tạo: `/video_generation` (POST)
- Endpoint query: `/query/video_generation?task_id={id}` (GET)
- Endpoint file: `/files/retrieve?file_id={id}` (GET)
- Models: `MiniMax-Hailuo-2.3`, `MiniMax-Hailuo-2.3-Fast`, `MiniMax-Hailuo-02`
- **3-bước pipeline:**
  1. Tạo task → nhận `task_id`
  2. Polling query task → nhận `file_id`
  3. Lấy `download_url` từ file_id
- Hỗ trợ `first_frame_image`, `last_frame_image`, `subject_reference`

#### VolcesArk (Doubao Seedance) Client
- Endpoint tạo: `/api/v3/contents/generations/tasks` (POST)
- Endpoint query: `/api/v3/contents/generations/tasks/{taskId}` (GET)
- Request format:
  ```json
  {
    "model": "seedance-1-5-pro",
    "content": [
      {"type": "text", "text": "{prompt}  --ratio 16:9  --dur 5"},
      {"type": "image_url", "image_url": {"url": "..."}, "role": "reference_image"},
      {"type": "image_url", "image_url": {"url": "..."}, "role": "first_frame"},
      {"type": "image_url", "image_url": {"url": "..."}, "role": "last_frame"}
    ],
    "generate_audio": true  // chỉ với seedance-1-5-pro
  }
  ```
- **Tham số nhúng trong prompt text:** `--ratio {aspectRatio}  --dur {duration}`
- Hỗ trợ 4 chế độ ảnh: single, first+last frame, reference images, text-only

#### Chatfire Client
- Endpoint tạo: `/video/generations`, Query: `/video/task/{taskId}`
- Tự động chọn format request theo model:
  - Model chứa `doubao` hoặc `seedance` → VolcesArk-compatible format (với `--ratio` và `--dur` trong prompt)
  - Model chứa `sora` → Sora format với `multipart/form-data`
  - Các model khác → default format

#### OpenAI Sora Client
- Endpoint tạo: `/videos` (POST, multipart/form-data)
- Endpoint query: `/videos/{id}` (GET)
- Ảnh tham chiếu được gửi dưới dạng **file binary** (không phải URL) với đúng MIME type
- Xử lý cả ảnh URL (tải về) và base64 Data URI

### 4.2 Luồng tạo video

```
GenerateVideo(request) → lưu DB (status: pending) → goroutine ProcessVideoGeneration
                                                              ↓
                                              Convert ảnh tham chiếu → base64
                                                              ↓
                              Xây dựng prompt đầy đủ:
                                1. GetVideoConstraintPrompt(referenceMode) [xem bên dưới]
                                2. + "\n\n" + userPrompt
                                              ↓
                         client.GenerateVideo(imageURL, fullPrompt, opts)
                                              ↓
                              Nhận task_id → goroutine pollTaskStatus
                                              ↓
                         Polling mỗi 10s, tối đa 300 lần (50 phút)
                                              ↓
                              Completed → tải video về local → update DB
```

### 4.3 Constraint Prompt cho Video

**Prompt tiêu chuẩn (single/first_last/multiple mode):**
```
### Định nghĩa vai trò

Bạn là chuyên gia phân tích động lực video và tổng hợp hàng đầu. Bạn có thể chỉ từ một ảnh tĩnh
hoặc một tập hợp frame đầu/cuối, nhận biết chính xác các thuộc tính vật lý, hướng ánh sáng và
xu hướng chuyển động tiềm ẩn, tạo video chất lượng cao tuân theo quy luật vật lý.

### Logic thực thi cốt lõi

1. Nhận dạng chế độ:
   * Chế độ 1 ảnh (Single Image): Xem ảnh đầu vào là Frame 0. Phân tích "điểm căng" trong khung
     hình và tiếp tục hành động theo hướng đó.
   * Chế độ 2 ảnh (First & Last): Gắn chặt ảnh đầu là điểm bắt đầu, ảnh thứ hai là điểm kết thúc.
     Tính quỹ đạo dịch chuyển của tất cả các phần tử.

2. Tính nhất quán vật lý:
   * Bảo tồn khối lượng: Đảm bảo vật thể không đột biến về thể tích, mật độ, kết cấu.
   * Quán tính chuyển động: Theo cơ học cổ điển, khởi đầu ổn định, tăng tốc tự nhiên.

3. Ngoại suy môi trường: Tự động bổ sung phần nền mở rộng bên ngoài khung hình chính.
```

**Prompt cho Action Sequence (3x3 grid mode):**
```
### Định nghĩa vai trò

Bạn là chuyên gia tạo video độ chính xác cực cao, chuyên chuyển đổi ảnh sequence 9 ô (3x3)
thành video liên tục với chất lượng điện ảnh.

### Logic thực thi cốt lõi

1. Gắn frame đầu-cuối: Trích xuất ô 1 (góc trên trái) làm frame đầu (Frame 0),
   ô 9 (góc dưới phải) làm frame cuối (Final Frame).
2. Nội suy sequence: Ô 2-8 xác định path hành động chính. Phân tích dịch chuyển logic,
   thay đổi ánh sáng và biến dạng vật thể giữa các keyframe.
3. Nhất quán: Đảm bảo đặc điểm nhân vật, chi tiết cảnh, phong cách nghệ thuật giữ 100%
   ổn định xuyên suốt video.
4. Bổ sung động: Tự động fill transition frame mượt mà, 24fps hoặc 30fps.

### Lệnh ràng buộc có cấu trúc
* Cấm ảo giác: Không thêm phần tử mới hoặc chuyển đổi nền không có trong 9 ô và prompt.
```

### 4.4 Chế độ tham chiếu ảnh cho Video

| `reference_mode` | Cách truyền ảnh | Mô tả |
|---|---|---|
| `single` | `imageURL` | 1 ảnh tham chiếu |
| `first_last` | `first_frame_url` + `last_frame_url` | Frame đầu và cuối |
| `multiple` | `reference_image_urls[]` | Nhiều ảnh tham chiếu |
| `none` | (không ảnh) | Tạo video từ text thuần |

---

## 5. Catalog Toàn Bộ Prompt

### 5.1 System Prompts (theo loại tác vụ)

| Tác vụ | Hàm lấy prompt | Đặc điểm |
|---|---|---|
| Tạo đại cương | `GetOutlineGenerationPrompt()` | Không phụ thuộc style |
| Tạo kịch bản phân tập | `GetEpisodeScriptPrompt()` | Không phụ thuộc style |
| Trích xuất nhân vật | `GetCharacterExtractionPrompt(style)` | Style-aware |
| Trích xuất cảnh | `GetSceneExtractionPrompt(style)` | Style-aware |
| Trích xuất đạo cụ | `GetPropExtractionPrompt(style)` | Style-aware |
| Tạo phân cảnh | `GetStoryboardSystemPrompt()` | Không phụ thuộc style |
| Prompt frame đầu | `GetFirstFramePrompt(style)` | Style-aware |
| Prompt frame chính | `GetKeyFramePrompt(style)` | Style-aware |
| Prompt frame cuối | `GetLastFramePrompt(style)` | Style-aware |
| Prompt action sequence | `GetActionSequenceFramePrompt(style)` | Style-aware |
| Ràng buộc video | `GetVideoConstraintPrompt(referenceMode)` | Mode-aware |

### 5.2 Style Prompts (cho ảnh và video)

`GetStylePrompt(style)` trả về prompt định nghĩa phong cách nghệ thuật, được **prepend** trước user prompt:

| Style | Phong cách |
|---|---|
| `ghibli` | Studio Ghibli - watercolor, cel-shading, pastoral, warm |
| `guoman` | Quốc phong TQ - particle effects, collision colors, fluorescent elements |
| `wasteland` | Hậu tận thế - hard line-art, limited palette, grainy texture |
| `nostalgia` | Hoài niệm 90s - film grain, chromatic aberration, muted pastel |
| `pixel` | Pixel art 8/16-bit - dithering, flat shading, aliased lines |
| `voxel` | 3D voxel - cube units, global illumination, cinematic lighting |
| `urban` | Đô thị hiện đại - webtoon, crisp line art, neon glow, rim lighting |
| `guoman3d` | Tiên hiệp 3D - PBR, SSS skin, cinematic lighting, Eastern aesthetic |
| `chibi3d` | Chibi 3D - blind box style, plastic/resin texture, chibi proportions |
| `realistic` | (không thêm style prompt) |

### 5.3 User Prompt Templates

Tất cả template được quản lý trong `FormatUserPrompt(key, args...)`:

| Key | Template |
|---|---|
| `outline_request` | `"Hãy tạo đại cương phim ngắn cho chủ đề: {theme}"` |
| `character_request` | `"Nội dung kịch bản:\n{content}\n\nTrích xuất tối đa {count} nhân vật chính..."` |
| `episode_script_request` | `"Đại cương:\n{outline}\n{chars}\nTạo kịch bản chi tiết {n} tập..."` |
| `frame_info` | `"Thông tin cảnh quay:\n{context}\n\nTạo prompt cho frame đầu..."` |
| `key_frame_info` | `"Thông tin cảnh quay:\n{context}\n\nTạo prompt cho key frame..."` |
| `last_frame_info` | `"Thông tin cảnh quay:\n{context}\n\nTạo prompt cho frame cuối..."` |
| `drama_info_template` | `"Tên phim: {title}\nTóm tắt: {desc}\nThể loại: {genre}"` |

---

## 6. Kỹ Thuật Prompt Engineering

### 6.1 System Prompt + User Prompt Separation
Mọi cuộc gọi LLM đều tách biệt:
- **System Prompt:** Định nghĩa vai trò, quy tắc, ràng buộc, format output
- **User Prompt:** Nội dung cụ thể cần xử lý

### 6.2 JSON Output Enforcement
Tất cả các prompt đều yêu cầu output dạng JSON thuần túy:
```
**QUAN TRỌNG: Chỉ trả về JSON array/object thuần túy. KHÔNG gồm markdown code block,
giải thích, hay nội dung khác. Bắt đầu trực tiếp bằng [ hoặc {.**
```
Sau đó được parse bằng `utils.SafeParseAIJSON()` - tự động làm sạch markdown code fence (`\`\`\`json...\`\`\``) trước khi parse.

### 6.3 Bilingual Support (zh/en)
`PromptI18n` phát hiện ngôn ngữ từ config (`config.App.Language`):
- `"zh"` → prompt tiếng Trung
- `"en"` → prompt tiếng Anh

Mỗi hàm `GetXxxPrompt()` đều có hai phiên bản ngôn ngữ.

### 6.4 Style-Aware Prompting
Khi drama có `style` khác `"realistic"`:
1. LLM prompts nhận style làm tham số → yêu cầu AI hiểu context phong cách
2. Image generation: `GetStylePrompt(style)` được prepend trước user prompt với mô tả chi tiết phong cách nghệ thuật

### 6.5 Constraint Prompt Injection (Video)
Khi tạo video, một constraint prompt được **tự động thêm vào trước** user prompt:
```go
fullPrompt = constraintPrompt + "\n\n" + userPrompt
```
Constraint prompt khác nhau theo `referenceMode` (xem mục 4.3).

### 6.6 Reference Image Consistency Instruction
Khi tạo ảnh có ảnh tham chiếu:
```
**Quan trọng:** Phải tuân thủ nghiêm ngặt các yếu tố trong ảnh tham chiếu,
giữ tính nhất quán của cảnh và nhân vật
```

### 6.7 max_tokens Retry Logic
Khi gặp lỗi `"Unsupported parameter: 'max_tokens'"` (một số model OpenAI mới):
- Tự động retry với `max_completion_tokens` thay vì `max_tokens`

### 6.8 Structured Context Building (Storyboard Frame)
Trước khi gọi AI tạo frame prompt, service xây dựng context chi tiết từ database:
```
Shot description: {description}
Scene: {location}, {time}
Characters: {char1}, {char2}
Action: {action}
Result: {result}
Dialogue: {dialogue}
Atmosphere: {atmosphere}
Shot type: {shot_type}
Angle: {angle}
Movement: {movement}
```

### 6.9 Fallback Mechanism
Nếu AI thất bại hoặc JSON parse lỗi → fallback về template đơn giản:
```go
fallbackPrompt = scene.Location + ", " + scene.Time + ", " + characters + ", " +
                 atmosphere + ", anime style, first frame, static shot"
```

### 6.10 Async Processing + Task Polling
Tất cả tác vụ AI đều chạy async:
- **Text/Image:** goroutine → update `tasks` table → client polling `/api/tasks/{id}`
- **Video:** goroutine → polling `GetTaskStatus()` mỗi 10s → tối đa 300 lần (50 phút)

---

## 7. Luồng Hoạt Động Tổng Thể (End-to-End)

```
┌─────────────────────────────────────────────────────────────────┐
│                    TẠO PHIM NGẮN (End-to-End)                   │
└─────────────────────────────────────────────────────────────────┘

[1] TẠO ĐẠI CƯƠNG
    User nhập chủ đề, thể loại, số tập
    → LLM: GetOutlineGenerationPrompt + "Tạo đại cương cho {theme}..."
    → Output: JSON { title, episodes: [{summary, conflict, cliffhanger}] }

[2] TẠO KỊCH BẢN
    User chọn tập từ đại cương
    → LLM: GetEpisodeScriptPrompt + "Dựa trên đại cương, viết kịch bản {n} tập..."
    → Output: JSON { episodes: [{script_content}] }

[3] TRÍCH XUẤT NHÂN VẬT
    Từ kịch bản
    → LLM: GetCharacterExtractionPrompt(style) + kịch bản
    → Output: JSON [{ name, role, appearance, personality, description }]
    → Lưu vào DB (characters table)

[4] TRÍCH XUẤT CẢNH
    Từ kịch bản
    → LLM: GetSceneExtractionPrompt(style) + kịch bản
    → Output: JSON [{ location, time, prompt }]
    → Lưu vào DB (scenes table)

[5] TẠO ẢNH NHÂN VẬT
    Từ appearance mô tả trong DB
    → Image Client (DALL-E/Gemini/VolcEngine):
      prompt = GetStylePrompt(style) + appearance_description
    → Lưu ảnh local

[6] TẠO ẢNH BỐI CẢNH
    Từ scene.prompt trong DB
    → Image Client:
      prompt = GetStylePrompt(style) + scene.prompt
    → Lưu ảnh local

[7] TẠO PHÂN CẢNH (STORYBOARD)
    Từ script_content + characters + scenes
    → LLM: GetStoryboardSystemPrompt + prompt phức tạp (xem mục 2.3.E)
    → Output: JSON [{ shot_number, shot_type, action, dialogue, scene_id, characters[] ... }]
    → Lưu vào DB (storyboards table)

[8] TẠO FRAME PROMPT
    Từ từng storyboard record trong DB
    → LLM: GetXxxFramePrompt(style) + context cảnh quay
    → Output: JSON { prompt, description }
    → Lưu vào DB (frame_prompts table)

[9] TẠO ẢNH FRAME
    Từ frame_prompt.prompt
    → Image Client:
      prompt = GetStylePrompt(style) + frame_prompt + consistency instruction
      opts: reference_images = [scene_image, character_image]
    → Lưu ảnh local

[10] TẠO VIDEO
     Từ frame image + storyboard prompt
     → Video Client:
       fullPrompt = GetVideoConstraintPrompt(mode) + "\n\n" + user_prompt
       imageURL = frame_image (single/first/last/multiple)
     → Polling GetTaskStatus() → tải video về local

[11] GHÉP VIDEO (Video Merge)
     FFmpeg ghép nhiều video segment thành tập phim hoàn chỉnh
```

---

### Tóm tắt nhanh

| Bước | Tác vụ | AI Type | Provider | Output |
|---|---|---|---|---|
| 1 | Tạo đại cương | Text LLM | OpenAI/Gemini | JSON outline |
| 2 | Tạo kịch bản | Text LLM | OpenAI/Gemini | JSON scripts |
| 3 | Trích xuất nhân vật | Text LLM | OpenAI/Gemini | JSON characters |
| 4 | Trích xuất cảnh | Text LLM | OpenAI/Gemini | JSON scenes |
| 5 | Tạo ảnh nhân vật | Image Gen | DALL-E/Gemini/VolcEngine | JPEG/PNG |
| 6 | Tạo ảnh bối cảnh | Image Gen | DALL-E/Gemini/VolcEngine | JPEG/PNG |
| 7 | Tạo phân cảnh | Text LLM | OpenAI/Gemini | JSON storyboards |
| 8 | Tạo frame prompt | Text LLM | OpenAI/Gemini | JSON prompts |
| 9 | Tạo ảnh frame | Image Gen | DALL-E/Gemini/VolcEngine | JPEG/PNG |
| 10 | Tạo video | Video Gen | Minimax/VolcesArk/Chatfire/Sora | MP4 |
| 11 | Ghép video | FFmpeg | (local) | MP4 hoàn chỉnh |
