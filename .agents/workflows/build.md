---
description: Build the project for production
---

This workflow builds both the frontend and the backend.

1. **Build Frontend**
// turbo
```bash
cd web && npm run build
```

2. **Build Backend**
// turbo
```bash
go build -o huobao-drama .
```

> [!TIP]
> After building, the frontend static files are embedded in the backend binary or served from the `dist` folder depending on configuration.
