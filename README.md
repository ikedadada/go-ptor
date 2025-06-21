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
