---
name: skeleton-review
description: Review the current branch to confirm new code is wired through the walking skeleton and respects hexagonal boundaries. Use before opening a pull request.
---

# Review the change against the walking skeleton

Use `git diff` against the main branch. Use the paths in `CLAUDE.md` and answer with yes or no plus file references:

1. Is every new domain type, port and adapter reachable from the composition root (`main.go`)?
2. Do unit tests cover the new domain logic and adapter behavior?
3. Does the domain import anything from adapters, frameworks or infrastructure?
4. Did the change add a new entry point, main function or wiring location?
5. Is there new code that only unit tests call?

List each problem with the file, the line and the required fix.
