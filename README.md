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

When the client resolves a `.ptor` address, it sends a CONNECT cell once for the
current circuit before any streams are opened. After the CONNECT succeeds,
stream data continues to use BEGIN cells as before.

### Environment variables

The relay uses the `PTOR_HIDDEN_ADDR` variable to locate the hidden HTTP service
when processing CONNECT cells. If not set, it falls back to `hidden:5000` (the
Docker demo value). The older `HIDDEN_ADDR` variable is also checked for
backward compatibility.

### Hidden service

The hidden service proxies incoming connections to an upstream HTTP server. Use
the `-http` flag to specify the target address.

The provided `docker compose` configuration starts a small demo server from
`cmd/httpdemo` and points the hidden service at `httpdemo:8080`. Bring up the
demo stack with:

```bash
docker compose up --build
```

The hidden service will print its `.ptor` address on startup. You can access it
via the client once the stack is running.

## Testing

Execute all unit tests with:

```bash
go test ./...
```
