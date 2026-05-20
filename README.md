# base64-plugin

A [hyper-mcp](https://github.com/hyper-mcp-rs/hyper-mcp) WebAssembly plugin that exposes **base64 encode/decode** as MCP tools, backed by Go's stdlib `encoding/base64`.

Built with **TinyGo 0.40.1** targeting `wasip1`, distributed as an OCI artifact on `ghcr.io`.

## Tools

| Name | Description |
|---|---|
| `base64_encode` | Encode a UTF-8 string to base64. |
| `base64_decode` | Decode a base64-encoded string to UTF-8. |

### Arguments (both tools)

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `input` | string | yes | — | The string to encode/decode. |
| `url_safe` | boolean | no | `false` | When `true`, uses the RFC 4648 §5 URL-safe alphabet (`-`/`_` instead of `+`/`/`). |

Errors (invalid base64, missing argument, unknown tool) are returned as a `CallToolResult` with `isError: true` and a text content block describing the failure — the plugin never panics, because TinyGo is built with `-panic=trap` and a panic would become an opaque WASM trap.

## Install in hyper-mcp

Add to `~/.config/hyper-mcp/config.json`:

```json
{
  "plugins": {
    "base64": {
      "url": "oci://ghcr.io/duynhlab/base64-plugin:latest"
    }
  }
}
```

Pin to a digest for reproducibility (recommended for production):

```json
{
  "plugins": {
    "base64": {
      "url": "oci://ghcr.io/duynhlab/base64-plugin@sha256:..."
    }
  }
}
```

The published image is signed with [cosign](https://github.com/sigstore/cosign) keyless via GitHub OIDC. Verify a digest with:

```sh
cosign verify \
  --certificate-identity-regexp "https://github.com/duynhlab/base64-plugin/.github/workflows/release-(docker|oras)\.yml@refs/tags/v.*" \
  --certificate-oidc-issuer-regexp "https://token.actions.githubusercontent.com" \
  ghcr.io/duynhlab/base64-plugin@sha256:...
```

Once the plugin is loaded, hyper-mcp exposes the tools as `base64-base64_encode` and `base64-base64_decode` (plugin name is prefixed).

## Build from source

Requires **TinyGo 0.40.1**. The build flags are fixed by the upstream hyper-mcp Go plugin template — do not change them.

### Native TinyGo

```sh
GOOS=wasip1 GOARCH=wasm tinygo build -no-debug -panic=trap -scheduler=none -o plugin.wasm
```

### Via podman (no host TinyGo required)

```sh
podman run --rm --userns=keep-id -e GOFLAGS=-buildvcs=false -e HOME=/tmp \
  -v "$PWD":/src:Z -w /src docker.io/tinygo/tinygo:0.40.1 \
  tinygo build -target=wasip1 -no-debug -panic=trap -scheduler=none -o plugin.wasm
```

`--userns=keep-id` keeps the output file owned by your host user. `GOFLAGS=-buildvcs=false` skips Go's VCS stamping (the container does not have git context for the bind mount).

The resulting `plugin.wasm` is ~425 KB.

### OCI image

The `Dockerfile` is a `FROM scratch` wrapper that copies `plugin.wasm` into the image — it does not compile anything. Build it after producing `plugin.wasm`:

```sh
podman build -t base64-plugin:latest .
```

## Test

A host harness lives in `test/` (a separate Go module so it can use the native `extism/go-sdk` without conflicting with the wasip1 plugin build). It loads `plugin.wasm` and exercises both tools end-to-end:

```sh
cd test && go run .
```

Covers std + URL-safe alphabets, invalid base64, unknown tool, and missing-argument paths.

## Release

Releases are tag-driven. Push a SemVer tag prefixed with `v` and two parallel GitHub Actions workflows (`Release (Docker)` and `Release (ORAS)`) will:

1. Build `plugin.wasm` with TinyGo 0.40.1.
2. Push the image to `ghcr.io/duynhlab/base64-plugin:<tag>` and `:latest`.
3. Sign both refs with cosign (keyless, GitHub OIDC).
4. Create a GitHub Release with `plugin.wasm` attached.

```sh
git tag -a v0.3.0 -m "Release v0.3.0"
git push origin v0.3.0
```

## Repository layout

```
main.go      Plugin handler implementations (the only file you normally edit).
exports.go   Generated WASM //export wrappers. Do not edit.
imports.go   Generated host-function bindings. Do not edit.
types.go     MCP protocol types. Do not edit; reference only.
test/        Standalone Go host harness (extism/go-sdk) for end-to-end tests.
Dockerfile   FROM scratch wrapper around plugin.wasm.
```

See [AGENTS.md](./AGENTS.md) for build/test gotchas and project policies that apply to any contributor (human or AI).

## License

Apache 2.0 — same as the upstream [hyper-mcp Go plugin template](https://github.com/hyper-mcp-rs/go-plugin-template) this repository was generated from.
