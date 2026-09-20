# Repository Guidelines

## Project Structure & Module Organization

This is a Go 1.26 module: `github.com/nightnoryu/go-kita`. Each top-level
directory is an importable package focused on one infrastructure concern:
`env/`, `jsonlog/`, `log/`, `maybe/`, `postgresql/`, `redis/`, `slices/`,
and `transactional/`. Keep package code and its `*_test.go` files
together. Package-specific usage notes belong in that package's `README.md`;
the root `README.md` describes the SDK as a whole.

## Build, Test, and Development Commands

Use [mise](https://mise.jdx.dev) to install the pinned Go and golangci-lint
versions and to run the normal validation flow:

```sh
mise run          # download modules, build all packages, lint, and test
mise run build    # compile ./... without creating a binary
mise run lint     # run golangci-lint, including formatters
mise run test     # run normal tests across ./...
mise run test:race # run focused race-detector tests
mise run check    # lint plus normal and race-detector tests
mise run tidy     # update go.mod and go.sum when dependencies change
```

Run the narrowest relevant test while iterating, for example
`go test ./jsonlog`. Do not commit generated build output.

## Coding Style & Naming Conventions

Follow idiomatic Go and let `gofmt`, `goimports`, and `gci` set formatting and
import order; the lint configuration places standard-library imports first,
then third-party imports, then `github.com/nightnoryu/go-kita/...`. Use tabs
for indentation. Name exported APIs in PascalCase and document them when the
meaning is not obvious; keep unexported identifiers concise camelCase. Prefer
small, package-scoped APIs over cross-package abstractions.

## Testing Guidelines

Use the standard `testing` package with `testify` where assertions improve
readability. Name test files `*_test.go` and test functions `TestThing` or
`TestThing_Condition`. Cover success paths, errors, and boundary cases for
changed behavior. Assess changes involving concurrency or shared mutable state
for race-detector coverage. Put race-only tests in `*_race_test.go` files with
the `//go:build race` constraint, then add their package to the `test:race`
Mise task; `go test -race` enables that build tag. There is no stated coverage
threshold; all affected package tests and `mise run check` must pass before
review.

## Commit & Pull Request Guidelines

Recent history uses brief imperative subjects, such as `Add jsonlog package`,
`Fix error handling`, and `Update README.md`. Keep commits focused and avoid
unrelated formatting churn. Pull requests should explain the behavior change,
identify affected packages, link relevant issues when available, and include
tests. Add examples or documentation updates for public API changes; attach
screenshots only when a documentation or visual asset change makes them useful.
