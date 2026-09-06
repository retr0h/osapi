# Contributing

Contributions to OSAPI are very welcome, but we ask that you read this document
before submitting a PR. It covers everything you need: prerequisites, setup, the
conventions code follows, how to add an API domain, and the pull request
workflow.

## Before you start

- Read the [Code of Conduct](CODE_OF_CONDUCT.md). It applies to every
  interaction in this repo.

- **Design records.** The conventions binding this repository are specified in
  [osapi-io/specs](https://github.com/osapi-io/specs) under `components/osapi/`,
  whose `.specify/memory/` is the standing record. Design reasoning for a change
  lives there, not here. A design document kept in this repository goes stale
  the moment the code moves past it, and nothing catches the drift.

- **Get familiar with the project.** Read the docs in this order:

  1. [Guiding Principles][principles]. Design philosophy and project values
  2. [System Architecture][system-architecture]. REST API, NATS, CLI
  3. [API Design Guidelines][api-guidelines]. REST conventions and endpoint
     structure
  4. [Job System Architecture][job-architecture]. KV-first job processing,
     subject routing, and the agent pipeline

- **Check existing work.** Is there an existing PR? Are there issues discussing
  the feature/change you want to make? Please make sure you consider/address
  these discussions in your work.

- **Backwards compatibility.** Will your change break existing OSAPI files? It
  is much more likely that your change will be merged if it is backwards
  compatible. Is there an approach you can take that maintains this
  compatibility? If not, consider opening an issue first so that API changes can
  be discussed before you invest your time into a PR.

## Prerequisites

Install tools using [mise]:

```bash
mise install
```

- **[Go].** OSAPI is written in Go. We always support the latest two major Go
  versions, so make sure your version is recent enough.
- **[Node.js].** Runtime for tools like `@redocly/cli`, for building the
  Docusaurus docs site, and for building the embedded React UI in `ui/`.
- **[Bun].** JavaScript package manager and script runner used for the
  Docusaurus docs and the React UI.
- **[just].** Task runner used for building, testing, formatting, and other
  development workflows.
- **[uv].** Python package runner. `just md-fmt` formats repository markdown
  with [mdformat] via `uvx`; nothing is installed into the repo.
- **[NATS CLI].** Command-line tools for interacting with NATS. Useful for
  debugging and monitoring during development. Install with
  `brew install nats-io/nats-tools/nats`.

### Claude Code

If you use [Claude Code] for development, install this plugin from the default
marketplace:

```
/plugin install commit-commands@claude-plugins-official
```

- **commit-commands.** provides `/commit` and `/commit-push-pr` slash commands
  that follow the project's commit conventions automatically.

**Do not use superpowers.** Spec Kit governs specification, planning, and
implementation, and the design record for a change lives in
[osapi-io/specs](https://github.com/osapi-io/specs). A second workflow over that
ground gives two answers to which artifact is authoritative, and the answer that
loses is the one nobody reads. Nothing superpowers produces is committed.

## Setup

Fetch shared justfiles and install all dependencies:

```bash
just fetch
just deps
```

## Project structure

- **`cmd/`.** Cobra CLI commands (`client`, `node agent`, `controller.api`,
  `nats server`).
- **`internal/controller/api/`.** Echo REST API. Node-targeted handlers nest
  under `node/{domain}/`; controller-only handlers are top-level (`job/`,
  `health/`). Each domain has its own `gen/` with an OpenAPI spec; the combined
  spec is `internal/controller/api/gen/api.yaml`.
- **`internal/job/`.** Job domain types and subject routing. `client/` holds the
  high-level operations.
- **`internal/agent/`.** Node agent: the consumer/handler/processor pipeline for
  job execution.
- **`internal/provider/`.** Operation implementations, organized by category
  then domain.
- **`internal/config/`.** Viper-based config from `osapi.yaml`.
- **`internal/telemetry/`.** Tracing, metrics, and agent self-metrics.
- **`internal/controller/notify/`.** Condition notification system.
- **`pkg/sdk/`.** Go SDK for programmatic REST API access.
- **`ui/`.** React 19 + TypeScript + Vite + Tailwind CSS v4, embedded into the
  Go binary at build time.

[System Architecture][system-architecture] describes the package layout, handler
structure, and provider pattern in full.

`nats-client` and `nats-server` are sibling repositories, consumed as pinned
module versions in `go.mod`. There is no `replace` directive.

## Code style

Go code is formatted by [gofumpt] and linted using [golangci-lint], enforced by
CI.

```bash
just go-fmt-check   # Check formatting
just go-fmt         # Auto-fix formatting
just go-vet         # Run linter
```

The linters that run are declared in `.golangci.yml`. Read them there rather
than looking for a list here. A copied list goes stale the first time the
configuration changes. Generated files (`*.gen.go`, `*.pb.go`) are excluded from
formatting.

TypeScript and CSS in `ui/` are formatted by [Prettier] and linted by [ESLint].

```bash
just react-fmt      # Auto-fix UI formatting
just react-lint     # Run ESLint
```

### Documentation

Markdown outside the Docusaurus site is formatted with [mdformat] through `uvx`.
The site itself is formatted by Prettier through the `docusaurus` module. Both
styles are enforced by CI.

```bash
just md-fmt-check         # Check formatting outside the site
just md-fmt               # Auto-fix formatting outside the site
just docusaurus-fmt-check # Check site formatting
just docusaurus-fmt       # Auto-fix site formatting
```

## Code standards

### Function signatures

Functions with parameters use multi-line format, one parameter per line, with
the closing parenthesis and the return types on a line of their own:

```go
func FunctionName(
    param1 type1,
    param2 type2,
) (returnType, error) {
}
```

Functions taking no parameters stay on one line:

```go
func Name() string {
}
```

Adding a parameter then shows as one added line rather than a rewritten
signature.

### File naming

Name a file for what it holds. Avoid `helpers.go`, `utils.go`, and names of that
kind: they describe where code was put rather than what it is, and they
accumulate whatever has no other home.

`types.go` holds only type declarations: structs, interfaces, constants, and
aliases. A function belongs in a file named for what it does.

A test file is named for the production file it tests. Where tests grow too
large to read, split the production file first so each test file keeps a
counterpart, rather than splitting tests away from the file they cover.

### Go patterns

- Error wrapping: `fmt.Errorf("context: %w", err)`, so the chain names each
  layer it passed through and stays inspectable with `errors.Is` and
  `errors.As`.
- Early returns rather than nesting the successful path inside conditionals.
- Unused parameters: rename to `_`.
- Import order: standard library, third party, then local, separated by blank
  lines.

### Test doubles

A double for an interface this organization defines is generated with `mockgen`
and committed. Do not write a struct by hand to satisfy one.

Generated mocks live in a `mocks` package beside the code they mock, produced by
a `generate.go` holding the directive:

```go
package mocks

//go:generate go tool go.uber.org/mock/mockgen -source=../types.go -destination=types.gen.go -package=mocks
```

The generator is resolved through the module's tool dependencies, so every
checkout runs the version `go.mod` records. Destination files end in `.gen.go`
and are committed. Do not use `gen/` for mocks. That name is taken by API code
generation.

When the interface is **unexported**, a sibling package cannot work: the mock
has to import the package to name the types in the interface, and the package's
own tests have to import the mock. Generate it into the package instead, with a
destination scoped to tests so the mocking library stays out of the dependency
graph of anything that imports the package:

```go
// generate.go, in the package that declares the interface
package thispackage

//go:generate go tool go.uber.org/mock/mockgen -source=thing.go -destination=thing.gen_test.go -package=thispackage
```

Either way the directives live in a `generate.go` that holds no code, and the
generated file carries `.gen` so a reader knows not to edit it.

Where call sites would otherwise repeat the same expectations, write a
constructor returning a configured mock rather than introducing a hand-written
type. The generated mock is still what satisfies the interface.

Three doubles are written by hand, because generating them buys nothing:

- One standing in for a standard library interface: `net.Conn`, `fs.File`,
  `io.Writer`, `slog.Handler`. Those do not move when our code does.
- One carrying a real implementation of the behavior under test, such as signing
  with a genuinely generated key pair.
- A recorder for a dependency called from a goroutine the test cannot join,
  where a generated mock would assert a call count at a moment the test cannot
  establish. State that reason where the recorder is defined.

The conventions below are specific to OSAPI.

### File headers

Every `.go` file MUST start with the MIT license header. See any existing Go
file in the repo for the exact format. Build-tagged files put `//go:build` on
line 1, blank line, then the header.

### Logging

All logging uses Go's `log/slog` structured logger.

- **Subsystem labels.** Every component that holds a logger MUST wrap it with
  `logger.With(slog.String("subsystem", "..."))` at construction time, which
  auto-tags every line from that component. Examples: `"agent"`, `"agent.seed"`,
  `"api.schedule"`, `"provider.file"`, `"job.client"`, `"metrics"`,
  `"controller.heartbeat"`.
- **Typed attributes.** Use `slog.String("key", val)`, `slog.Int`, `slog.Bool`,
  `slog.Any`. Never use positional pairs like `"key", val`; they compile but
  bypass type safety.
- **Standard field names.** `error` for errors, `hostname` for hosts, `path` for
  file paths, `job_id` for job IDs, `name` for entry names, `addr` for
  addresses.
- **Error fields.** `slog.String("error", err.Error())` for string context, or
  `slog.Any("error", err)` to preserve the error type.
- **Log levels.** `Debug` for operation dispatch and idempotency skips, `Info`
  for lifecycle events and state changes, `Warn` for degraded but functional
  states, `Error` for failures that need attention.

### Lifecycle

Components use a non-blocking lifecycle: `Start()` returns immediately, and
`Stop(ctx)` shuts down with a deadline.

### Filesystem access

Use [avfs]: `memfs.New()` for in-memory work and `failfs.New()` for targeted
error injection. Never use `afero`.

## Testing

```bash
just test           # Run all tests (lint + unit + coverage)
just go-unit        # Run unit tests only
just go-unit-int    # Run integration tests
go test -run TestName -v ./internal/job/...  # Run a single test
```

Coverage is gated at 99.9%. `just test` fails if total coverage drops below it,
so a change that adds untested code fails locally and in CI:

```bash
just go-unit-cov-check   # Report coverage and fail below the target
```

The target is declared in `.github/codecov.yml` and in the shared `go` justfile
module. Change both together.

### Test file conventions

- Public tests: `*_public_test.go` in the package's `_test` package, exercising
  the exported surface. This is the default.
- Internal tests: `*_test.go` in the same package, for what the exported surface
  cannot reach.
- Suite naming: `*_public_test.go` → `{Name}PublicTestSuite`, `*_test.go` →
  `{Name}TestSuite`.
- `testify/suite` with table-driven cases.
- One suite method per function under test. Success, errors, and edge cases are
  rows in one table, not separate methods.
- `export_test.go` exposes unexported symbols to external tests, by alias or by
  setter. Do not use an alias to re-cover behavior the caller's own test already
  reaches; a helper with its own contract is what the pattern is for.

### Test layers

- **Unit tests** (`*_test.go`, `*_public_test.go`). Fast, mocked dependencies.
  Public suites also carry HTTP wiring methods (`TestXxxHTTP`,
  `TestXxxRBACHTTP`) that send raw HTTP through the full Echo middleware stack
  with mocked backends.
- **Integration tests** (`test/integration/`). Build and start a real `osapi`
  binary and exercise CLI commands end-to-end. Guarded by a
  `//go:build integration` tag. The harness allocates random ports, generates a
  JWT, and starts the server, so no external setup is required.

A new API domain should include a `{domain}_test.go` smoke suite under
`test/integration/`. Mutating tests MUST be guarded by `skipWrite(s.T())` so CI
runs read-only tests by default; `OSAPI_INTEGRATION_WRITES=1` enables writes.

### Test helpers

- Use `export_test.go` to expose an unexported variable or function to the
  `_test` package, rather than writing an internal test or a hand-rolled stub.
- Use `suite.TearDownSubTest()` to reset swapped variables between table-driven
  sub-tests, not `defer` inside the loop.
- Platform stubs: test that the Darwin and Linux stubs return `ErrUnsupported`
  for every method.

## Building and running

```bash
just build     # Builds the React UI, then the Go binary
./osapi controller start -f configs/osapi.yaml
```

**Always build through `just`.** The `//go:embed dist/*` directive in
`ui/embed.go` requires `ui/dist/` to hold files at compile time, and
`just build` runs `just react-build` first to satisfy it. Running
`go build ./...` or `go test ./...` directly fails for this reason.
`just build`, `just test`, and `just ready` all build the UI first.

## Input validation

All user input is validated through the `internal/validation` package, which
wraps [go-playground/validator]. Rules are declared in OpenAPI specs via
`x-oapi-codegen-extra-tags` and enforced at runtime by handler calls to
`validation.Struct()` or `validation.Var()`.

Config struct fields in `internal/config/types.go` use the same `validate` tags.
Validation runs at startup after `viper.Unmarshal()`, so an invalid value exits
immediately with a clear error. Defaults are set via `viper.SetDefault()` in
`cmd/root.go`, so most fields can be omitted. Use `go_duration` for Go duration
strings, and add `required` to fields with no sensible default.

- **Required fields** use `validate:"required,..."`.
- **Optional fields** use `validate:"omitempty,..."`.
- **Enum constraints** use `validate:"oneof=a b c"`.
- **Cross-field validation** uses `required_without` / `excluded_with` for
  mutually exclusive fields, such as cron `schedule` versus `interval`.

### Update endpoints with all-optional fields

When a PUT endpoint has all-optional fields (user update, group update, cron
update), call `validation.AtLeastOneField(request.Body)` after
`validation.Struct()` to reject empty bodies with a 400. This prevents
meaningless no-op updates and, worse, triggering destructive defaults.

```go
if errMsg, ok := validation.Struct(request.Body); !ok {
    return gen.PutXxx400JSONResponse{Error: &errMsg}, nil
}

if errMsg, ok := validation.AtLeastOneField(request.Body); !ok {
    return gen.PutXxx400JSONResponse{Error: &errMsg}, nil
}
```

### Defense in depth

When `validation.Struct()` cannot currently fail because every field uses
`omitempty`, keep the call and comment why. This guards against a later field
addition silently breaking validation:

```go
// Defense in depth: current fields use omitempty so validation
// always passes, but guards against future field additions.
if errMsg, ok := validation.Struct(request.Body); !ok {
    return gen.PostXxx400JSONResponse{Error: &errMsg}, nil
}
```

This applies to action endpoints (power, docker stop) where an empty body is
valid, unlike update endpoints, which must use `AtLeastOneField`.

## Adding a new API domain

Every domain must be consistent across all layers: provider, agent processor,
API handler, SDK service, CLI commands, docs, and tests. The
[Adding an API Domain][adding-an-api-domain] guide walks the nine steps in
order, from the provider implementation through to verification.

The principle: **pick an existing domain and `find`/`grep` for it across the
codebase. Your new domain should appear in all the same places.** If something
exists for `sysctl` but not for yours, it is missing.

## UI contributions

The React management dashboard lives in `ui/` and is embedded into the Go binary
at build time. See [UI Development][ui-development] for prerequisites, the
development server, code style, and component conventions, and
[UI Architecture][ui-architecture] for the embedding mechanism, component
layers, and SDK generation flow.

## Documentation

OSAPI uses [Docusaurus] to host a documentation server. Content is written in
Markdown under `docs/docs/`, wrapped at 80 characters. Formatting is covered
under [Code style](#documentation).

```bash
just docusaurus-start     # Start local docs server
just docusaurus-build     # Build docs for production
```

## Before committing

Run `just ready` before committing. It regenerates code, formats, lints, and
builds both the Go binary and the UI, so your commit matches what CI verifies:

```bash
just ready
```

## Branching

All changes should be developed on feature branches. Create a branch from `main`
using the naming convention `type/short-description`, where `type` matches the
[Conventional Commits] type:

- `feat/add-retry-logic`
- `fix/null-pointer-crash`
- `docs/update-api-reference`
- `refactor/simplify-handler`
- `chore/update-dependencies`

When using Claude Code's `/commit` command, a branch will be created
automatically if you are on `main`.

## Commit messages

Follow [Conventional Commits] with the 50/72 rule:

- **Subject line**: max 50 characters, imperative mood, capitalized, no period
- **Body**: wrap at 72 characters, separated from subject by a blank line
- **Format**: `type(scope): description`
- **Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`
- Summarize the "what" and "why", not the "how"

Try to write meaningful commit messages and avoid having too many commits on a
PR. Most PRs should likely have a single commit (although for bigger PRs it may
be reasonable to split it in a few). Git squash and rebase is your friend!

## Submitting a PR

- **Describe your changes.** Say what changed and why. A reviewer should not
  have to read the diff to learn the reason for it.
- **Issue/PR links.** Link any previous work such as related issues or PRs.
  Please describe how your changes differ to/extend this work.
- **Examples.** Add any examples or screenshots that you think are useful to
  demonstrate the effect of your changes.
- **Draft PRs.** If your changes are incomplete, but you would like to discuss
  them, open the PR as a draft and add a comment to start a discussion. Using
  comments rather than the PR description allows the description to be updated
  later while preserving any discussions.

## AI usage

This repo is written with AI assistance. All contributions are subject to the
[AI Usage Policy](AI_POLICY.md). Disclose the tool you used, and make sure you
can explain what your change does without the aid of AI tools.

## FAQ

> I want to contribute, where do I start?

All kinds of contributions are welcome, whether it's a typo fix or a shiny new
feature. You can also contribute by upvoting/commenting on issues or helping to
answer questions.

> I'm stuck, where can I get help?

If you have questions, open a [Discussion] on GitHub.

[adding-an-api-domain]: docs/docs/sidebar/development/adding-an-api-domain.md
[api-guidelines]: docs/docs/sidebar/architecture/api-guidelines.md
[avfs]: https://github.com/avfs/avfs
[bun]: https://bun.sh
[claude code]: https://claude.ai/code
[conventional commits]: https://www.conventionalcommits.org
[discussion]: https://github.com/osapi-io/osapi/discussions
[docusaurus]: https://docusaurus.io
[eslint]: https://eslint.org
[go]: https://go.dev
[go-playground/validator]: https://github.com/go-playground/validator
[gofumpt]: https://github.com/mvdan/gofumpt
[golangci-lint]: https://golangci-lint.run
[job-architecture]: docs/docs/sidebar/architecture/job-architecture.md
[just]: https://just.systems
[mdformat]: https://pypi.org/project/mdformat/
[mise]: https://mise.jdx.dev
[nats cli]: https://github.com/nats-io/natscli
[node.js]: https://nodejs.org
[prettier]: https://prettier.io
[principles]: docs/docs/sidebar/architecture/principles.md
[system-architecture]: docs/docs/sidebar/architecture/system-architecture.md
[ui-architecture]: docs/docs/sidebar/architecture/ui.md
[ui-development]: docs/docs/sidebar/development/ui-development.md
[uv]: https://docs.astral.sh/uv/
