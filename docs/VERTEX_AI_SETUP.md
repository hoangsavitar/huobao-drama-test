# Hướng dẫn thiết lập Vertex AI (Google Cloud)

## Yêu cầu

- [Google Cloud CLI (gcloud)](https://cloud.google.com/sdk/docs/install) đã cài đặt
- Tài khoản Google Cloud có quyền truy cập Vertex AI API
- Đã enable Vertex AI API trên GCP project

---

## 1. Kích hoạt Vertex AI API

```bash
# Login vào Google Cloud
gcloud auth login

# Set project
gcloud config set project project-93aa7ef8-3fc1-4aa6-868

# Enable Vertex AI API
gcloud services enable aiplatform.googleapis.com
```

## 2. Thiết lập Application Default Credentials (ADC)

Vertex AI SDK dùng **ADC (Application Default Credentials)** để xác thực — **không cần API key**.

### Cách 1: Dùng user credentials (dành cho dev local)

```bash
gcloud auth application-default login
```

Sau đó làm theo hướng dẫn trên browser để đăng nhập.

### Cách 2: Dùng service account key (dành cho CI/server)

```bash
# Tạo service account (nếu chưa có)
gcloud iam service-accounts create vertex-ai-sa \
    --display-name="Vertex AI Service Account"

# Gán role
gcloud projects add-iam-policy-binding project-93aa7ef8-3fc1-4aa6-868 \
    --member="serviceAccount:vertex-ai-sa@project-93aa7ef8-3fc1-4aa6-868.iam.gserviceaccount.com" \
    --role="roles/aiplatform.user"

# Tạo key
gcloud iam service-accounts keys create ./vertex-ai-key.json \
    --iam-account="vertex-ai-sa@project-93aa7ef8-3fc1-4aa6-868.iam.gserviceaccount.com"
```

Set biến môi trường:

```bash
# Windows (Command Prompt)
set GOOGLE_APPLICATION_CREDENTIALS=C:\path\to\vertex-ai-key.json

# Windows (PowerShell)
$env:GOOGLE_APPLICATION_CREDENTIALS="C:\path\to\vertex-ai-key.json"

# Linux/Mac
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/vertex-ai-key.json"
```

## 3. Cấu hình Environment Variables

Vertex AI Go SDK (`google.golang.org/genai`) tự động đọc các biến môi trường sau:

| Biến | Giá trị | Bắt buộc |
|------|---------|----------|
| `GOOGLE_CLOUD_PROJECT` | `project-93aa7ef8-3fc1-4aa6-868` | **Có** |
| `GOOGLE_CLOUD_LOCATION` | `global` | **Có** |

> **Lưu ý**: Code hiện default `GOOGLE_CLOUD_LOCATION` về `global` nếu biến này rỗng, nhưng vẫn nên set rõ để tránh lệch môi trường. `GOOGLE_CLOUD_PROJECT` là bắt buộc.

`run_vertex.bat` chỉ là file shortcut đang làm đúng 2 việc:

1. Set `GOOGLE_CLOUD_PROJECT` và `GOOGLE_CLOUD_LOCATION` cho cửa sổ CMD đó.
2. Chạy `go run main.go`.

Để đồng nhất cách chạy, có thể bỏ qua `run_vertex.bat` và chạy trực tiếp:

```bash
go run main.go
```

Điều kiện là 2 biến môi trường ở trên phải tồn tại trong **cùng terminal đang chạy lệnh** hoặc đã được set vĩnh viễn ở Windows.

### Cách A: Set tạm thời trong terminal hiện tại

Biến chỉ có hiệu lực trong cửa sổ terminal hiện tại và các process được chạy từ cửa sổ đó. Nếu set ở terminal A nhưng chạy `go run main.go` ở terminal B thì terminal B sẽ không thấy biến.

#### Windows Command Prompt (CMD)

```bash
set GOOGLE_CLOUD_PROJECT=project-93aa7ef8-3fc1-4aa6-868
set GOOGLE_CLOUD_LOCATION=global
go run main.go
```

#### Windows PowerShell

```powershell
$env:GOOGLE_CLOUD_PROJECT="project-93aa7ef8-3fc1-4aa6-868"
$env:GOOGLE_CLOUD_LOCATION="global"
go run main.go
```

#### Conda env `whatif`

Nếu đang dùng conda, kích hoạt env trước rồi set biến trong đúng terminal đó:

```powershell
conda activate whatif
$env:GOOGLE_CLOUD_PROJECT="project-93aa7ef8-3fc1-4aa6-868"
$env:GOOGLE_CLOUD_LOCATION="global"
go run main.go
```

Nếu dùng CMD thay vì PowerShell thì dùng cú pháp `set ...` như phần CMD ở trên.

### Cách B: Set vĩnh viễn trên Windows

Dùng `setx` khi muốn các terminal mới tự có biến, không cần set lại mỗi lần:

```bash
setx GOOGLE_CLOUD_PROJECT "project-93aa7ef8-3fc1-4aa6-868"
setx GOOGLE_CLOUD_LOCATION "global"
```

> `setx` chỉ có hiệu lực với terminal mở sau khi chạy lệnh. Terminal đang mở sẵn chưa nhận giá trị mới, nên cần đóng/mở lại terminal rồi chạy `go run main.go`.

Kiểm tra biến trước khi chạy:

```powershell
# PowerShell
$env:GOOGLE_CLOUD_PROJECT
$env:GOOGLE_CLOUD_LOCATION
```

```bash
# CMD
echo %GOOGLE_CLOUD_PROJECT%
echo %GOOGLE_CLOUD_LOCATION%
```

## 4. Kiểm tra kết nối

### Cách 1: Dùng gcloud CLI (kiểm tra network + auth)

```powershell
$PROJECT_ID="project-93aa7ef8-3fc1-4aa6-868"
$LOCATION="global"
$MODEL="gemini-3.1-flash-lite"
$TOKEN=$(gcloud auth application-default print-access-token)

@'
{
  "contents": [
    {
      "role": "user",
      "parts": [
        {
          "text": "hello"
        }
      ]
    }
  ]
}
'@ | Out-File -FilePath body.json -Encoding utf8

curl.exe -X POST "https://aiplatform.googleapis.com/v1/projects/$PROJECT_ID/locations/$LOCATION/publishers/google/models/${MODEL}:generateContent" `
-H "Authorization: Bearer $TOKEN" `
-H "Content-Type: application/json" `
--data-binary "@body.json"
```

### Cách 2: Dùng UI

1. Vào **Settings > AI Config** hoặc mở dialog **AI Service Configuration**
2. Chọn tab **Text** hoặc **Image**
3. Bấm **Add Configuration**
4. Chọn provider: **Vertex AI (Google Cloud)**
5. Chọn model: `gemini-3.1-flash-lite` (text) hoặc `gemini-3.1-flash-image-preview` (image)
6. Base URL và API Key sẽ tự điền placeholder — không cần sửa
7. Bấm **Create**
8. (Chỉ text) Bấm **Test Connection** để kiểm tra

## 5. Cấu hình trong DB

Khi tạo config qua UI, dữ liệu lưu vào bảng `ai_service_configs`:

| Field | Text Config | Image Config |
|-------|-------------|--------------|
| `provider` | `vertex` | `vertex` |
| `service_type` | `text` | `image` |
| `name` | `VertexAI-Text-xxxx` | `VertexAI-Image-xxxx` |
| `model` | `["gemini-3.1-flash-lite"]` | `["gemini-3.1-flash-image-preview"]` |
| `base_url` | `vertex-ai` (placeholder) | `vertex-ai` (placeholder) |
| `api_key` | `ADC` (placeholder) | `ADC` (placeholder) |
| `priority` | Cao hơn các config khác | Cao hơn các config khác |
| `is_active` | `true` | `true` |

> **Quan trọng**: Set `priority` cao hơn các config Gemini API cũ để Vertex AI được ưu tiên dùng trước. Nếu muốn giữ Gemini API cũ làm fallback, set priority thấp hơn, hoặc tắt `is_active` của config cũ.

## 6. Cấu trúc code mới

### File mới

| File | Mô tả |
|------|-------|
| `pkg/ai/vertex_client.go` | Vertex AI text client (dùng SDK) |
| `pkg/image/vertex_image_client.go` | Vertex AI image client (dùng SDK) |

### File đã sửa

| File | Thay đổi |
|------|----------|
| `application/services/ai_service.go` | Thêm case `"vertex"` trong factory methods |
| `application/services/image_generation_service.go` | Thêm case `"vertex"` trong factory methods |
| `web/src/components/common/AIConfigDialog.vue` | Thêm provider `vertex` trong UI |
| `web/src/views/settings/AIConfig.vue` | Thêm provider `vertex` trong UI |
| `go.mod` / `go.sum` | Thêm dependency `google.golang.org/genai` |

### Provider toggle vẫn hoạt động

- `"gemini"` / `"google"` → Gemini REST API (cũ, dùng API key)
- `"vertex"` → Vertex AI (mới, dùng ADC)
- Các provider khác → không đổi

## 7. Xử lý sự cố

### "Vertex: create client: ..." error

- Kiểm tra `GOOGLE_CLOUD_PROJECT` đã set đúng
- Kiểm tra ADC: `gcloud auth application-default print-access-token`
- Kiểm tra Vertex AI API đã enable: `gcloud services enable aiplatform.googleapis.com`

### "Permission denied" error

- Gán role `roles/aiplatform.user` cho service account
- Hoặc dùng `gcloud auth application-default login` với user có quyền

### "Model not found" error

- Model name phải đúng: `gemini-3.1-flash-lite` (text), `gemini-3.1-flash-image-preview` (image)
- Kiểm tra model availability tại region đang dùng
