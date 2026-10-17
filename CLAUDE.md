## Architecture rules
- The service follows hexagonal architecture.
- Domain code has zero imports from adapters, frameworks or infrastructure.
- Ports (interfaces) live next to the domain. Adapters implement them.
- `main.go` is the one composition root. Only the composition root constructs concrete adapters.
- The walking skeleton is: entry point -> inbound adapter -> domain -> outbound adapter.
- Every feature extends the existing walking skeleton.
- Create no new entry points, main functions or parallel wiring.
- A feature is complete when unit tests cover its domain logic and adapters and the composition root wires it.

## Domain design rules
- `World` and `Character` hold and call feature types. They are not where features live.
- A feature with its own state or rules (a skill, an activity, an interaction) gets its own domain type that owns that state. `World` holds an instance of that type, not the type's fields.
- A feature type depends on the small view of the world it needs (a narrow interface or plain values), not on `World` itself.
- Shared mechanics stay generic. Skills, XP, levels, level-ups and the save `Snapshot` take the skill name as data. Never add a field, method or branch named after one skill to shared code.
- The hero carries out one player order at a time (walk, chop, open a door, climb), held in one current-order value, not in separate "active" flags. Background effects that run alongside orders (level-up celebration, regrowth, stat restore, run energy) each keep their own state and advance every frame.
- The domain holds game state only. Camera and other view state belong to the terminal adapter.
- Ports stay stable as features grow. A feature plays sounds through a generic effect value or its own port, not through a new method on a shared port.
- Domain code takes time only from the `dt` passed to `Advance`. It never reads the wall clock.

## Feature workflow (test-driven development)
1. Name the domain type that owns the feature's state and rules. If the answer is `World` or `Character`, create a new type instead.
2. Define or extend the port the feature needs.
3. Write failing unit tests for the domain logic.
4. Implement the domain logic until unit tests pass.
5. Write failing unit tests for the adapter, then implement or extend it until they pass.
6. Wire the new pieces in the composition root.
7. Run the full verification. The feature is done when it passes.

## Service: lapresregne
- Entry point: `main.go` (`main`)
- Composition root: `main.go` (`run`, `startMusic`)
- Domain: `internal/world` (terrain, maps, character, movement), `internal/soundtrack` (shuffle, history, fades)
- Ports: `internal/soundtrack/ports.go` (`Samples`, `Decoder`, `Player`), `internal/world/ports.go` (`Sounds`, `Progress`)
- Inbound adapters: `internal/adapters/terminal` (`session.go`, `input.go`)
- Outbound adapters: `internal/adapters/terminal` (`render.go`, `foliage.go`), `internal/adapters/songfiles`, `internal/adapters/speaker`, `internal/adapters/progressfile`, `internal/adapters/viewfile`, `internal/adapters/mapfile`

## Commands
- Unit tests: `go test ./internal/...`
- Architecture check: `go test -run TestArchitecture .`
- Full verification: `go vet ./... && go test ./...`
