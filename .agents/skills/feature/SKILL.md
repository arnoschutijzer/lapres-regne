---
name: feature
description: Build a new feature outside-in through the walking skeleton using test-driven development and hexagonal architecture. Use for any new behavior exposed through the CLI.
---

# Build a feature through the walking skeleton

Input: a feature description.

1. Read `AGENTS.md` for paths and commands.
2. Restate the feature as observable behaviors at the entry point (keys or flags sent to the CLI and the expected terminal output or side effect).
3. Write a failing acceptance test in `terminal_test.go` that runs the compiled binary in a pseudo-terminal and uses the real outbound adapters. Run it and confirm it fails for the expected reason.
4. Stop and show me the acceptance test. Wait for approval.
5. Define or extend the port.
6. Write failing unit tests for the domain logic.
7. Implement the domain logic until unit tests pass.
8. Implement or extend the adapter.
9. Wire the new code in the composition root (`main.go`).
10. Run `go vet ./... && go test ./...`. Fix failures until it passes.
11. Report: files changed, the acceptance test name and confirmation that the new code is reachable from the composition root.
