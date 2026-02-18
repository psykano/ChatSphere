# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ChatSphere is a real-time anonymous chat application with public/private rooms. The repo has two independent codebases: a React SPA frontend and a Go backend — no monorepo tooling.

## Commands

### Frontend (run from `frontend/`)
- **Dev server:** `npm run dev`
- **Build:** `npm run build`
- **Lint:** `npm run lint`
- **Run all tests:** `npm test`
- **Run tests in watch mode:** `npm run test:watch`
- **Run a single test file:** `npx vitest run src/components/SomeComponent.test.tsx`

### Backend (run from `backend/`)
- **Run server:** `go run ./cmd/server`
- **Run all tests:** `go test ./...`
- **Run a single test:** `go test ./internal/message/ -run TestSomething`
- **Build binary:** `go build -o chatsphere ./cmd/server`

### Docker (run from root)
- **Start all services:** `docker compose up --build`

## Architecture

### Backend (`backend/`)
- **Entrypoint:** `cmd/server/main.go`
- **`internal/server/`** — HTTP routes on `net/http` ServeMux (Go 1.22+ method-prefixed patterns), anonymous session management via HTTP-only cookie (`chatsphere_session`)
- **`internal/ws/`** — WebSocket hub, connection handling, JSON envelope protocol (`{type, payload}`)
- **`internal/message/`** — `MessageStore` interface with in-memory (ring buffer) and Redis implementations
- **`internal/room/`** — Room lifecycle and expiration (2hr idle or 15min empty)
- **`internal/ratelimit/`** — Sliding window IP-based rate limiting
- **`internal/user/`** — Anonymous session store
- Falls back to in-memory storage when `REDIS_ADDR` env var is not set

### Frontend (`frontend/`)
- React 19 + TypeScript + Vite + Tailwind CSS v4
- UI components from shadcn/ui live in `src/components/ui/` (generated, ESLint-ignored)
- No global state library — state lives in custom hooks (`src/hooks/`)
- `ReconnectingWS` class (`src/lib/reconnecting-ws.ts`) handles WebSocket with exponential backoff, session resumption, and message deduplication
- `use-chat.ts` hook aggregates WebSocket events into chat state
- `use-websocket.ts` wraps `ReconnectingWS` for React lifecycle
- Tests are co-located with components (e.g., `ChatMessage.test.tsx` next to `ChatMessage.tsx`)
- Path alias: `@/` maps to `src/`

### WebSocket Protocol
Client-to-server message types: `join`, `chat`, `typing`, `kick`, `ban`, `mute`, `set_username`, `history_fetch`, `leave`
Server-to-client message types: `session`, `history`, `history_batch`, `backfill`, `presence`, `chat`, `system`, `typing`, `mute_status`, `error`

### Environment Variables (Backend)
- `LISTEN_ADDR` — bind address (default `:8080`)
- `REDIS_ADDR` — Redis address; if unset, uses in-memory storage

## Key Conventions
- Frontend tests use Vitest + React Testing Library + jsdom
- Backend tests use Go standard `testing` package; Redis tests use `miniredis/v2` (in-process)
- No authentication system — anonymous users identified by session cookie with 24hr TTL
- Usernames are per-room, set during WebSocket join handshake
