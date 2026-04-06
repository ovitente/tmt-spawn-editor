# Tests

Test strategy for the spawn editor.

See also [[architecture]], [[delivery]].

## Required checks

Automated gates that must pass before commit.

- `go build` must succeed
- `lat check` must pass (enforced by pre-commit hook)

## Failure signals

Conditions that must block a commit.

- `go build` failure
- `lat check` failure
