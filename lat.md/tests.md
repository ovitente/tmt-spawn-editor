# Tests

Test strategy for the spawn editor.

See also [[architecture]], [[delivery]].

## Required checks

Automated gates that must pass before commit.

- `go build` must succeed
- `lat check` must pass (enforced by pre-commit hook)

## Manual sanity

- Open a mission, ensure preview updates when moving list selection.
- Add/duplicate/delete without saving, ensure disk unchanged until `s`.
- Restore (`R`) clears modified dot on current entry.

## Failure signals

Conditions that must block a commit.

- `go build` failure
- `lat check` failure
