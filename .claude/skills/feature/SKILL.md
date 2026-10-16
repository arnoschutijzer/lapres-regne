---
name: feature
description: Build a new feature through the walking skeleton using test-driven development and hexagonal architecture. Use for any new behavior exposed through the CLI.
---

# Build a feature through the walking skeleton

Input: a feature description.

1. Follow the paths and commands in `CLAUDE.md`.
2. Restate the feature as observable behaviors at the entry point (keys or flags sent to the CLI and the expected terminal output or side effect).
3. Define or extend the port.
4. Write failing unit tests for the domain logic.
5. Implement the domain logic until unit tests pass.
6. Write failing unit tests for the adapter, then implement or extend it until they pass.
7. Wire the new code in the composition root (`main.go`).
8. Run `go vet ./... && go test ./...`. Fix failures until it passes.
9. Report: files changed, the tests added and confirmation that the new code is reachable from the composition root.
