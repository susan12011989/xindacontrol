# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Teamgram Enterprise Control (总控后台) - A full-stack enterprise management system with Vue3 frontend and Go backend. Features merchant management, cloud integration (Aliyun, AWS), user authentication with 2FA.

## Development Commands

### Frontend (in `front/` directory)
```bash
pnpm dev              # Start dev server (http://localhost:3333)
pnpm build            # Production build
pnpm build:staging    # Staging build
pnpm lint             # ESLint fix
pnpm test             # Run Vitest tests
```

### Backend (in `server/` directory)
```bash
# Run the server
cd server/apps/server && go run main.go

# Build with embedded frontend (from project root)
bash build-embed.sh        # Linux/Mac
bash build-embed_win.sh    # Windows
```

## Architecture

### Frontend (`front/src/`)
- **Framework**: Vue 3 + TypeScript + Composition API (`<script setup>`)
- **UI**: Element Plus + VXE-Table (complex grids)
- **State**: Pinia
- **Styling**: SCSS + UnoCSS (atomic CSS)

**Path aliases**:
- `@` → `src/`
- `@@` → `src/common/`

**Key directories**:
- `pages/demo/` - **Reference examples for standard patterns**
- `common/apis/{module}/` - API definitions with `index.ts` + `type.ts`
- `common/composables/` - Reusable composition functions
- `pinia/stores/` - State management

### Backend (`server/`)
- **Framework**: Go + Gin + Xorm
- **Database**: MySQL + Redis

**Structure** (module-first organization):
```
internal/server/
├── model/       # Request/Response DTOs (flat, by module: auth.go, merchant.go)
├── router/      # Routes by module (auth/, merchant/, cloud_aliyun/)
├── service/     # Business logic by module (mirrors router structure)
├── middleware/  # Auth, CORS, logging
└── cloud/       # Cloud SDK wrappers (aliyun/, aws/)
pkg/
├── entity/      # Database models
├── result/      # Unified response format
└── token_manager/
```

## Key Patterns

### API Response Format
All endpoints return: `{ code: number, data: any, message: string }`
- 200: Success
- 400: Error
- 601: Parameter error

### Frontend API Pattern
```typescript
// src/common/apis/{module}/index.ts
import type * as Module from "./type"
import { request } from "@/http/axios"

export function getItems(params: Module.QueryReq) {
  return request<Module.ResponseData>({
    url: "items",
    method: "get",
    params
  })
}
```

### Backend Handler Pattern
```go
// router/{module}/ - Route definitions
// service/{module}/ - Business logic
// Use result.Success(c, data) and result.Error(c, code, message)
```

### Vue Component Pattern
Every component must have `defineOptions({ name: "ComponentName" })`.

## Important Reference

**`AI-开发文档.md`** - Comprehensive development guide with complete examples for:
- Full CRUD workflow (frontend + backend)
- VXE-Table patterns
- API definitions
- Pinia stores
- Go service patterns

## Code Style

- **Frontend**: ESLint with @antfu config, 2-space indent, double quotes, no semicolons
- **Backend**: Standard Go formatting
- **Commits**: `feat`, `fix`, `perf`, `refactor`, `docs`, `types`, `test`, `ci`, `chore`
