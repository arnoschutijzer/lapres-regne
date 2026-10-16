## Architecture rules
- The service follows hexagonal architecture.
- Domain code has zero imports from adapters, frameworks or infrastructure.
- Ports (interfaces) live next to the domain. Adapters implement them.
- `main.go` is the one composition root. Only the composition root constructs concrete adapters.
- The walking skeleton is: entry point -> inbound adapter -> domain -> outbound adapter.
- Every feature extends the existing walking skeleton.
- Create no new entry points, main functions or parallel wiring.
- A feature is complete when unit tests cover its domain logic and adapters and the composition root wires it.

## Feature workflow (test-driven development)
1. Define or extend the port the feature needs.
2. Write failing unit tests for the domain logic.
3. Implement the domain logic until unit tests pass.
4. Write failing unit tests for the adapter, then implement or extend it until they pass.
5. Wire the new pieces in the composition root.
6. Run the full verification. The feature is done when it passes.

## Service: dither-cli
- Entry point: `main.go` (`main`)
- Composition root: `main.go` (`run`, `startMusic`)
- Domain: `internal/world` (terrain, character, movement), `internal/soundtrack` (shuffle, history, fades)
- Ports: `internal/soundtrack/ports.go` (`Samples`, `Decoder`, `Player`)
- Inbound adapters: `internal/adapters/terminal` (`session.go`, `input.go`)
- Outbound adapters: `internal/adapters/terminal` (`render.go`, `foliage.go`), `internal/adapters/songfiles`, `internal/adapters/speaker`

## Commands
- Unit tests: `go test ./internal/...`
- Architecture check: `go test -run TestArchitecture .`
- Full verification: `go vet ./... && go test ./...`
