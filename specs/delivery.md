# Build, packaging, and release

Status: Normative  
Scope: Binary/container build, Home Assistant manifest, validation, and publishing

## Requirements

- **REQ-REL-001** — The Go module MUST target the Go version declared in `go.mod`, build with `CGO_ENABLED=0`, and pass `go test ./...`.
- **REQ-REL-002** — The container builder MUST produce a stripped, trimpath Linux `GoSungrow` binary for `amd64` or `arm64`; unsupported target architecture MUST fail.
- **REQ-REL-003** — Runtime image MUST contain CA certificates, timezone data, the binary, `/run.sh`, and dashboard assets. It MUST set `HOME=/data`, `GOSUNGROW_CONFIG=/data/.GoSungrow/config.json`, asset directory `/opt/gosungrow/assets`, and working directory `/data`.
- **REQ-REL-004** — Home Assistant manifest MUST use slug `gosungrow`, automatic boot, service startup, no init, Home Assistant API access, optional MQTT service dependency, writable Home Assistant config mapping, and architectures `aarch64` and `amd64`.
- **REQ-REL-005** — App package version MUST start with the binary version. CI MUST validate this before publishing.
- **REQ-REL-006** — Validation MUST run spec checks, build the binary, run all Go and dashboard-card JavaScript tests, validate shell syntax, and smoke-build the amd64 app image. CI uses Node.js 22 for the dependency-free card tests.
- **REQ-REL-007** — Pushes to main/master/test branches, version tags, and relevant pull requests MUST validate. Non-PR successful validation publishes per-architecture images to GHCR with the manifest version; main/master and tags additionally receive `latest`.
- **REQ-REL-008** — Repository manifest name is `GoSungrow Apps` and points to this repository. Public image naming MUST remain `ghcr.io/<owner>/gosungrow-addon-{arch}`.
- **REQ-REL-009** — Specification, assistant skill, contributor contract, engineering-documentation, validation-script, and dashboard-card test changes MUST trigger validation.
- **REQ-REL-010** — Pull-request validation MUST reject normative-spec changes without derived artifacts and product implementation changes without normative-spec changes. Explicit `spec-docs-only` and `behavior-neutral` labels MAY waive the corresponding side only after reviewer confirmation that behavior does not change.

## Prohibited behavior

- Publishing when validation fails.
- Publishing an app version inconsistent with the binary version.
- Omitting specifications from the container build context in a way that affects validation; specs need not ship in the runtime image.
