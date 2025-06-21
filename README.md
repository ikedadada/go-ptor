# go-ptor

This repository demonstrates a simple multi-layer Go project. Code is organized following the layered layout described in `docs/style_backend.md`:

```
internal/
  domain/        # core models and business logic
  usecase/       # application orchestration logic
  handler/       # HTTP/CLI adapters (thin layer)
  infrastructure/ # external systems and drivers
```

The separation keeps business rules isolated from entry points and infrastructure code.

## Requirements

- Go **1.24** or higher (see `mise.toml`)

## Setup

1. Install Go and ensure `go` is in your `PATH`.
2. Clone this repository and download dependencies:

   ```bash
   go mod download
   ```

## Usage

Each entry in `cmd/` is a standalone binary. You can run them directly with `go run` or build them using `go build`.
For example, to start the client:

```bash
go run ./cmd/client
```

## Testing

Execute all unit tests with:

```bash
go test ./...
```
