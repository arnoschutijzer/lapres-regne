---
name: feature
description: Build a new feature through the walking skeleton using test-driven development and hexagonal architecture. Use for any new behavior exposed through the CLI.
---

# Build a feature through the walking skeleton

Input: a feature description.

1. Follow the paths and commands in `CLAUDE.md`.
2. Restate the feature as observable behaviors at the entry point (keys or flags sent to the CLI and the expected terminal output or side effect).
3. Name the domain type that owns the feature's state and rules, following the domain design rules in `CLAUDE.md`. If the answer is `World` or `Character`, create a new type instead.
4. Define or extend the port.
5. Write failing unit tests for the domain logic.
6. Implement the domain logic until unit tests pass.
7. Write failing unit tests for the adapter, then implement or extend it until they pass.
8. Wire the new code in the composition root (`main.go`).
9. Run `go vet ./... && go test ./...`. Fix failures until it passes.
10. Report: files changed, the domain type that owns the feature, the tests added and confirmation that the new code is reachable from the composition root.
