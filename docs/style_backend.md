# Backend Style Guide

## Purpose

This guide defines a unified backend coding style that reflects my current development practices. The goal is to allow AI Agents to write backend code that matches my expectations in structure, naming, testing, and separation of concerns — regardless of language or framework. Language-specific deviations will be defined in separate style guides.

---

## 1. Project Structure

- Follow a **layered architecture**:
  - `domain/`: core models and business logic
  - `usecase/` or `service/`: application-specific orchestration logic
  - `handler/` or `controller/`: external interfaces (e.g., HTTP APIs)
  - `infrastructure/`: external systems (DB, APIs, etc.)
- **Avoid overengineering** in small projects but maintain separation of logic.
- Do not mix interface, domain, and infrastructure responsibilities.

---

## 2. API Design

- Use **RESTful conventions** (resource-based URIs, proper HTTP methods).
- Prefer consistent, flat, predictable routes like:
  ```
  GET    /users
  GET    /users/{id}
  POST   /users
  PUT    /users/{id}
  DELETE /users/{id}
  ```
- Return **JSON** for all APIs unless otherwise required.
- Use appropriate **status codes** (200, 201, 400, 404, 500, etc.).
- Maintain **clear separation** between routing and actual logic.

---

## 3. Logic and Handler Separation

- Keep **handlers thin** — only validate inputs, invoke logic, and format outputs.
- Move business logic into **services or use cases**.
- Abstract I/O operations away from domain logic.
- This enables **testing business rules in isolation**.

---

## 4. Request/Response Modeling

- Define **typed request/response schemas** (DTOs or structs).
- Validate inputs strictly.
- Never expose raw internal models or DB structures directly in API responses.

---

## 5. Error Handling

- Return human-readable error messages with appropriate codes.
- Avoid exposing stack traces or internal errors to clients.
- Use a **centralized error handling** approach if the framework allows.
- Define a standard error format like:
  ```json
  {
    "error": {
      "code": "INVALID_INPUT",
      "message": "Email is required."
    }
  }
  ```

---

## 6. Dependency Management

- Use the **language’s standard dependency tool** (e.g., Go modules, npm, Maven).
- Pin or lock versions to ensure reproducibility.
- Avoid unnecessary libraries and be skeptical of large frameworks.
- Organize third-party dependencies clearly, and isolate them where useful.

---

## 7. Testing

- Write **unit tests** for all non-trivial logic.
- Keep tests fast, isolated, and deterministic.
- Use **parameterized tests** to cover edge cases.
- Business logic should be **testable without booting servers or DBs**.
- Include **integration tests** for critical flows.

---

## 8. Logging and Monitoring

- Use structured logging where possible (key-value or JSON).
- Log startup/shutdown events, API requests, and errors.
- Never log sensitive data (passwords, tokens, PII).
- Separate logs by level: `INFO`, `DEBUG`, `WARN`, `ERROR`.

---

## 9. Configuration and Environment Management

- Configuration should be **externalized**, not hardcoded.
- Use `.env`, config files, or environment variables for:
  - DB credentials
  - Ports
  - API keys
  - Feature flags
- Support multiple environments: `dev`, `stg`, `prod`, with separate configs.

---

## 10. Documentation

- Every project must have a **README.md** with:
  - Setup instructions
  - Run/test commands
  - Overview of folder structure and responsibilities
- Public functions/modules should be **self-explanatory or commented**.
- Avoid redundant or outdated comments.

---

## 11. Style and Naming

- Use consistent naming across files, packages, variables, and routes.
- Prefer `camelCase` or `PascalCase` depending on language conventions.
- Avoid abbreviations unless they are widely accepted.
- Stick to standard formatting tools (e.g., `gofmt`, `prettier`, etc.).

---

## 12. Summary

The backend code should be:

- **Modular**: logically separated into layers
- **Minimal**: no overengineering, but clear separation of concerns
- **Testable**: logic should be covered with unit and integration tests
- **Consistent**: naming, formatting, and API conventions should follow the same principles across projects
- **Secure and Maintainable**: avoid leaking internals, hardcoding secrets, or exposing unstable behavior

This document defines the baseline that future AI Agents should use when generating backend code to match my development preferences.
