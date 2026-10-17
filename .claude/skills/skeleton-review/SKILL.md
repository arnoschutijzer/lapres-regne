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
6. Does the change add a feature's state or methods to `World`, `Character` or another shared type instead of giving the feature its own domain type?
7. Does shared code (`Snapshot`, level-ups, ports) gain a field, method or branch named after one skill or activity?
8. Does the change add a separate "active" flag for an activity instead of using the current-activity value?
9. Does the domain gain camera or other view state?
10. Does a shared port gain a method for one feature instead of taking a generic effect value?
11. Does domain code read the wall clock instead of using the `dt` passed to `Advance`?

List each problem with the file, the line and the required fix.
