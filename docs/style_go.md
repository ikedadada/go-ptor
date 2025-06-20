# Go Style Guide

## 🎯 Purpose

This guide defines Go-specific coding conventions to ensure that AI agents can produce code indistinguishable in style, structure, and intent from my own Go programming habits.
It reflects personal preferences regarding naming, file structure, idiomatic patterns, error handling, and test philosophy.

---

## 📁 Project Structure

- Organize code by **package** with clear boundaries based on functionality.
- Place executable entry points under `cmd/<appname>/`, and reusable logic under `internal/` or top-level packages.
- Keep directory names lowercase and aligned with their package name.
- Avoid over-nesting; aim for a flat and understandable folder hierarchy.
- Use `testdata/` for fixture files and `go.mod` to define modules.

---

## 🧠 Naming Conventions

- Use **camelCase** for variables and unexported functions.
- Use **PascalCase** for exported identifiers (types, functions).
- Always capitalize acronyms fully: `ID`, `HTTP`, `URL`, etc.
- Avoid redundant naming ("stutter"), e.g., prefer `user.ID` over `user.UserID`.
- Prefer short but meaningful names; avoid single-letter names outside short scopes.

---

## 🔧 Function & Type Design

- Keep functions short and focused; extract logic into helpers when complexity increases.
- Return `(value, error)` rather than panicking; use `fmt.Errorf("%w", err)` to wrap.
- Use **early returns** to handle errors and edge cases.
- Design types so that **zero values are usable** when appropriate.
- Prefer **interfaces defined at the consumer side** (not by the implementation).
- Use `context.Context` as the first argument in functions that involve I/O or cancellation.

---

## 🛠️ Error Handling

- Handle every error explicitly; do not ignore with `_` unless justified.
- Avoid `panic()` in normal control flow—only for truly unrecoverable states.
- Error messages should:
  - Not start with capital letters
  - Not end with a period
  - Be descriptive and composable

---

## 🧪 Testing Philosophy

- Use `_test.go` files in the same package unless black-box testing is required.
- Write tests as `func TestXxx(t *testing.T)` with descriptive names.
- Use **table-driven tests** with `t.Run(...)` where appropriate.
- Keep assertions explicit (`if got != want { t.Errorf(...) }`) without external assertion libraries.
- Prefer `t.Fatalf` for setup failures, `t.Errorf` for soft assertions.
- Write deterministic tests; avoid flaky behavior and external dependencies.
- Use `go test -race` and `-cover` as part of CI verification.

---

## 📦 Dependency Management

- Use Go modules (`go.mod` / `go.sum`) to manage dependencies.
- Keep dependencies **minimal** and prefer the standard library.
- Avoid unnecessary libraries when a few lines of custom logic suffice.
- Run `go mod tidy` regularly to prune unused requirements.
- Do not check in `vendor/` unless offline builds are required.

---

## 🗒️ Documentation & Comments

- Every exported identifier should have a **doc comment** starting with its name.
- Keep comments focused on **why**, not what (code should show what).
- Use proper grammar and punctuation in comments, except for error strings.
- Avoid redundant or outdated comments.
- Prefer writing **Example functions** for usage over inline code samples in comments.

---

## ⛳ Summary

This guide represents the stylistic and structural norms I follow when writing Go.
AI agents following this guide should be able to produce Go code that is idiomatic, minimalistic, modular, and highly readable — in line with my personal engineering standards.
