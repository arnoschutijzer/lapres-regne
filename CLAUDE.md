## Architecture rules
- The service follows hexagonal architecture.
- Domain code has zero imports from adapters, frameworks or infrastructure.
- Ports (interfaces) live next to the domain. Adapters implement them.
- `main.go` is the one composition root. Only the composition root constructs concrete adapters.
- The walking skeleton is: entry point -> inbound adapter -> domain -> outbound adapter.
- Every feature extends the existing walking skeleton.
- Create no new entry points, main functions or parallel wiring.
- A feature is complete when an acceptance test exercises it through the skeleton.

## Feature workflow (outside-in test-driven development)
1. Write a failing acceptance test in `terminal_test.go` that runs the compiled binary in a real pseudo-terminal and uses the real outbound adapters.
2. Define or extend the port the feature needs.
3. Write failing unit tests for the domain logic.
4. Implement the domain logic until unit tests pass.
5. Implement or extend the adapter.
6. Wire the new pieces in the composition root.
7. Run the full verification. The feature is done when it passes.

## Service: dither-cli
- Entry point: `main.go` (`main`)
- Composition root: `main.go` (`run`, `startMusic`)
- Domain: `internal/forest` (world, movement), `internal/soundtrack` (shuffle, history, fades)
- Ports: `internal/soundtrack/ports.go` (`Samples`, `Decoder`, `Player`)
- Inbound adapters: `internal/adapters/terminal` (`session.go`, `input.go`)
- Outbound adapters: `internal/adapters/terminal` (`render.go`, `foliage.go`), `internal/adapters/songfiles`, `internal/adapters/speaker`
- Acceptance tests: `terminal_test.go` (`TestTerminal`)

## Commands
- Unit tests: `go test ./internal/...`
- Acceptance tests: `go test -run TestTerminal .`
- Architecture check: `go test -run TestArchitecture .`
- Full verification: `go vet ./... && go test ./...`
