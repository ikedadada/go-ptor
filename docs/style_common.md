# Common Coding Style Guide (Language-Agnostic)

## 🎯 Purpose

This guide defines universal coding conventions to ensure that an AI agent can generate code that aligns naturally with my personal style.
It serves as a reference point for consistent implementation regardless of programming language, framework, or project type.

Language-specific and domain-specific rules will be defined separately.

---

## 📁 Project Structure & Design Principles

- **Functional Folder Separation**
  Separate code, data, configurations, and other concerns into logically named directories.

- **Single Responsibility per Module**
  Each file, class, or module should have one clear responsibility. Avoid mixing unrelated logic in the same unit.

- **Decoupling Entry Points and Logic**
  CLI, UI, or HTTP handlers should only invoke logic defined in separate modules. Avoid embedding core logic in entry scripts.

---

## 🧠 Naming Conventions

- **Descriptive and Consistent Names**
  Names should clearly reflect purpose or behavior. Avoid cryptic or overly abbreviated names.

- **Consistent Style per Project**
  Use idiomatic naming styles (e.g., camelCase, snake_case, PascalCase) consistently within each project and language.

---

## 🔧 Implementation Style

- **Comments Should Capture Intent**
  Focus comments on _why_ something is implemented a certain way, not _what_ it does—code should explain the "what".

- **Readable, Modular Logic**
  Keep nesting shallow. If logic grows complex, split it into helper functions. Functions should aim to be side-effect free.

- **Avoid Hardcoded Constants**
  Extract configuration values (paths, thresholds, credentials, etc.) into a separate settings or config file.

- **Design for Idempotency**
  Code should be safe to re-run without harmful side effects (e.g., repeated file writes, residual state issues).

- **Expressive Layers of Reasoning**
  Write:
  - **How** in the code itself (expressing logic and flow)
  - **What** in the test code (describing expected behavior)
  - **Why** in the commit messages (rationale behind the change)
  - **Why not** in the code comments (avoided paths, trade-offs, rejected alternatives)

---

## 🛠️ Error Handling & Reliability

- **Validate Assumptions Up Front**
  When inputs or states have preconditions, check and fail early if violated.

- **Handle Failures Intentionally**
  Decide explicitly whether to exit, retry, skip, or fallback—and explain that decision in the code if non-obvious.

- **Prepare the Environment Proactively**
  Ensure directories exist, remove stale files, and prepare all prerequisites before executing logic.

---

## 🧾 Documentation Practices

- **README Is a Contract**
  Every project should have a README with clear: purpose, usage, requirements, setup instructions, and directory structure.

- **Minimal but Strategic Code Comments**
  Strive for self-explanatory code, but do leave comments for non-obvious decisions, assumptions, or tradeoffs.

---

## 🔁 Git & Workflow Conventions

- **Use Conventional Commits**
  Commit messages should follow this pattern:
  `feat:`, `fix:`, `refactor:`, `chore:`, etc., followed by a short, descriptive message.

- **One Commit = One Logical Change**
  Don’t mix unrelated changes in one commit. Keep logic, formatting, and configuration changes separate.

- **Automate Everything Repeatable**
  Use formatters, linters, tests, and CI/CD pipelines to ensure consistent builds and reduce manual work.

---
