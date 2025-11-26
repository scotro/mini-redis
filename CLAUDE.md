# CLAUDE.md

## Project Overview
Mini-Redis: A simple Redis clone implementing core string operations and the RESP protocol.

## Commands
- Build: `go build ./...`
- Test: `go test ./...`
- Run: `go run ./cmd/mini-redis`
- Lint: `go vet ./...`

## Project Structure
TBD

## Code Style
- Use standard Go conventions (gofmt, go vet)
- Error handling: return errors, don't panic
- Interfaces for testability
- Table-driven tests preferred
