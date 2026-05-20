# AGENTS.md

## Commit messages (mandatory)

AI agents MUST follow these rules for every commit they author. They override any default Crush commit template.

- **No attribution trailers.** Do not add `Signed-off-by`, `Co-authored-by`, `Assisted-by`, `Generated-by`, or any other trailer attributing work to an AI, tool, or third party.
- **Subject line:** ≤ 50 characters, capitalised, no trailing period, imperative mood (`Add support for X`, not `Added` / `Adds`).
- **Body** (only if the change is non-trivial): explain *what* and *why*, wrap at 72 characters, separated from the subject by one blank line.
- **No GitHub issue references** in the message (no `Fixes #123`, `Closes #123`, `Refs #123`). Put issue links in the PR description instead.
- **No GitHub @-mentions** of users or teams (no `@duynhlab`, `@platform-team`).

Acceptable example:

```
Add Kyverno admission policies for PSS baseline

Roll out Tier 1 ClusterPolicies in Audit mode so we can observe the
policy reports for one week before flipping to Enforce. Operators that
legitimately violate baseline are whitelisted via PolicyException with
owner and TTL annotations.
```

## Branching & push policy (mandatory)

- **NEVER push directly to `main`.** No exceptions. All changes go through a feature branch and PR. Never run `git push origin main` from a local checkout.
- Create a branch with a conventional prefix **before any work**:
  - `feat/<short-desc>` — new feature or capability
  - `fix/<short-desc>` — bug fix
  - `chore/<short-desc>` — tooling, deps, refactor with no behaviour change
  - `docs/<short-desc>` — documentation only
  - `refactor/<short-desc>` — code restructure, no behaviour change
  - `ci/<short-desc>` — CI/CD pipeline changes
- One logical change per branch. Keep branches short-lived.
- Push the branch (`git push -u origin <branch>`), then open a PR against `main`.
- Squash-merge via PR.

## What this repo is

A **hyper-mcp WebAssembly plugin** written in Go and compiled with **TinyGo** to `wasip1`. Exposes two MCP tools — `base64_encode` and `base64_decode` — backed by stdlib `encoding/base64`. Both tools accept `input: string` and an optional `url_safe: bool` (RFC 4648 §5 URL-safe alphabet).

## Architecture (read this before editing)

```
main.go      <- Plugin handler implementations. The only file you normally edit.
exports.go   <- Generated WASM `//export` wrappers. Do not edit.
imports.go   <- Generated host-function bindings. Do not edit.
types.go     <- 1600+ lines of MCP protocol types. Do not edit; reference only.
test/        <- Standalone Go host harness (separate go.mod) that loads plugin.wasm via
                github.com/extism/go-sdk and exercises both tools end-to-end.
```

Data flow: hyper-mcp host invokes a `//export call_tool` (etc.) in `exports.go` → wrapper JSON-decodes input via `pdk.InputJSON` → calls the same-named handler in `main.go` → JSON-encodes output via `pdk.OutputJSON`. The plugin runs as an Extism WASM module; there is no HTTP server, no `stdin`/`stdout` parsing.

`func main() {}` in `main.go:95` is intentionally empty — **do not remove it**. TinyGo requires it as the compile entrypoint, but actual entrypoints are the `//export ...` functions.

## Build / run

**Local build (no TinyGo installed)** — run the official TinyGo image via podman. Use `--userns=keep-id` so the output file is owned by the host user, and `GOFLAGS=-buildvcs=false` to skip VCS stamping (the container lacks git context for the bind mount):

```sh
podman run --rm --userns=keep-id -e GOFLAGS=-buildvcs=false -e HOME=/tmp \
  -v "$PWD":/src:Z -w /src docker.io/tinygo/tinygo:0.40.1 \
  tinygo build -target=wasip1 -no-debug -panic=trap -scheduler=none -o plugin.wasm
```

CI uses the host-installed equivalent:

```sh
GOOS=wasip1 GOARCH=wasm tinygo build -no-debug -panic=trap -scheduler=none -o plugin.wasm
```

**Test the built plugin:**

```sh
cd test && go run .
```

Produces `plugin.wasm` (~425KB). The harness exercises encode/decode (std + URL-safe), invalid input, unknown tool, and missing-argument paths.

**Build OCI image with podman:**

```sh
podman build -t base64-mcp:latest .
```

The Dockerfile is `FROM scratch` + `COPY plugin.wasm` — it does not compile; `plugin.wasm` must exist first.

- Requires **TinyGo 0.40.1** (pinned in CI and in the podman command above). Stock `go build` will fail with `missing function body` errors on extism's PDK — those functions only have bodies for the wasip1+TinyGo build path. Always build via TinyGo.
- `-scheduler=none` means **goroutines are forbidden** at runtime. No `go func()`, channel-based sync, `time.Sleep`-based concurrency, etc.
- `-panic=trap` means panics produce an opaque WASM trap with no stack trace. Always return errors (or `IsError: true` `CallToolResult`s); never panic.
- The host harness in `test/` is a separate Go module so it can use the *native* extism SDK without conflicting with the wasip1 build of the plugin.

## Gotchas

1. **`go.mod` module path is `github.com/hyper-mcp-rs/hyper-mcp/templates/plugins/go`** — the template default, not updated for this repo. Nothing imports this module so it's harmless; leave it unless renaming is explicitly requested.
2. **gopls reports `_CallTool`, `_ListTools`, etc. as unused** — false positive. They are kept alive by `//export` directives that gopls (running with host Go) doesn't see. Do not delete.
3. **`CallToolResult.Content` is `[]ContentBlock`** (a tagged-union struct), *not* `[]json.RawMessage`. Build a text result with `ContentBlock{Text: &TextContent{Text: "..."}}`. Earlier versions of the upstream README show raw JSON — that's outdated for the current `types.go`.
4. `Tool.InputSchema` is `jsonschema.Schema` from `github.com/invopop/jsonschema`. Its `Properties` field is `*orderedmap.OrderedMap[string, *jsonschema.Schema]` — build it with `orderedmap.New[string, *jsonschema.Schema]()` and `.Set(...)`, not a `map[string]any` literal.
5. Tool arguments arrive as `map[string]any` in `input.Request.Arguments`. Always type-assert with `, ok`; under `-panic=trap` a bad assertion is an unrecoverable WASM trap.
6. For any MCP capability you don't support, return an *empty* result, not an error — e.g. `GetPrompt` returns `&GetPromptResult{}, nil`. The host will probe these handlers; returning `not implemented` errors makes the host surface failures.

## CI / release

`.github/workflows/`:
- `ci.yml` — build-only on push/PR to `main`. No tests.
- `nightly-docker.yml`, `nightly-oras.yml` — scheduled nightly publishes.
- `release-docker.yml`, `release-oras.yml` — triggered on `v*` tags. Two parallel publishing paths: GHCR Docker image and ORAS-pushed OCI artifact.

All workflows pin TinyGo to `0.40.1` via `acifani/setup-tinygo@db56321...`. If you bump TinyGo, bump it in all five workflows together.

## hyper-mcp config

Local:

```json
{
  "plugins": {
    "base64": { "url": "file:///abs/path/to/plugin.wasm" }
  }
}
```

From the podman-built image (after `podman push` to a registry):

```json
{
  "plugins": {
    "base64": { "url": "oci://your-registry/base64-mcp:latest" }
  }
}
```
