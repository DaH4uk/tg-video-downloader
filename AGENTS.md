# Repository Guidelines

## Project Overview
- This repository contains a Go service that listens to Telegram updates, processes supported video links, and exposes Prometheus metrics on port `8080`.
- The main entrypoint is `cmd/service/main.go`.
- Core application logic lives under `internal/`, split by handlers, services, and logging infrastructure.

## Working Conventions
- Keep changes narrow and consistent with the existing package layout under `cmd/` and `internal/`.
- Prefer small, focused edits over broad refactors unless a refactor is required to fix the root cause.
- Follow existing Go naming and formatting conventions; use `gofmt` on edited Go files before finalizing.
- Avoid introducing new dependencies unless they are necessary for the requested change.

## Runtime Notes
- Local configuration is loaded from `.env` via `godotenv.Overload()`.
- Keep secrets out of the repository; update `.env.example` instead of committing `.env`.
- The metrics server binds to `:8080`, and `docker-compose.yml` maps it to host port `9988`.

## Useful Commands
- `go test ./...` runs the test suite.
- `go build ./cmd/service` builds the service binary.
- `docker compose up --build` builds and starts the containerized service.

## Files To Watch
- Check `.gitignore` before adding generated files or local artifacts.
- Do not commit build outputs, downloaded assets, or local-only environment files.
