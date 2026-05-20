---
description: Start the development environment (backend and frontend)
---

This workflow starts the Go backend and the Vue frontend in development mode.

1. **Start Backend**
// turbo
```bash
go run main.go
```

2. **Start Frontend**
// turbo
```bash
cd web && npm run dev
```

> [!NOTE]
> Backend runs on `http://localhost:5678` and Frontend on `http://localhost:3012` by default.
