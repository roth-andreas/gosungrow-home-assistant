# GoSungrow agent and contributor contract

## Specifications are authoritative

The normative product contract is [`specs/README.md`](specs/README.md). Before changing behavior, read its change protocol and the applicable specifications.

- Change the specification before or with behavior-changing code.
- Treat code and tests as derived artifacts; update them when they disagree with an accepted requirement.
- Do not turn accidental implementation behavior into a requirement without an explicit decision.
- Update `specs/traceability.md` when requirements, source ownership, or tests change.
- Run `python scripts/check_specs.py`, `go test ./...`, both JavaScript tests named in `specs/README.md`, and `bash -n addon/gosungrow/run.sh`.

## Repository map

- App startup: `addon/gosungrow/run.sh`
- CLI and orchestration: `cmd/`
- Home Assistant MQTT model: `cmdHassio/`
- iSolarCloud client and normalization: `iSolarCloud/`
- Dashboard template/cards/locales: `addon/gosungrow/assets/`
- App/version metadata: `addon/gosungrow/config.yaml`, `defaults/const.go`
- CI/publishing: `.github/workflows/homeassistant-app.yml`

## Working constraints

- Preserve dashboard-failure tolerance and layered network/session recovery.
- Preserve opaque composite `ps_id`/`ps_key` values.
- Keep private Home Assistant experiments and generated screenshots under ignored paths.
- App version must start with the binary version.
- Before release, also smoke-build the app image as described in `specs/delivery.md`.
