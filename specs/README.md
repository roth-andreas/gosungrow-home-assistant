# GoSungrow normative specification

This directory is the source of truth for GoSungrow behavior. The Go, shell, YAML, and JavaScript sources and their tests are implementations of these documents.

## Authority

Normative documents use `MUST`, `MUST NOT`, `SHOULD`, and `MAY` as defined in [governance.md](governance.md). If a normative requirement conflicts with implementation, the requirement wins after confirming that the requirement was deliberately accepted. Implementation details not stated here are replaceable.

## Reading order

1. [governance.md](governance.md)
2. [completeness.md](completeness.md)
3. [product-and-architecture.md](product-and-architecture.md)
4. [domain-model.md](domain-model.md)
5. The affected subsystem specification
6. [acceptance.md](acceptance.md), [traceability.md](traceability.md), and [source-inventory.md](source-inventory.md)

## Specifications

| Area | Normative document |
|---|---|
| Product boundary and lifecycle | [product-and-architecture.md](product-and-architecture.md) |
| Completeness and implementation boundary | [completeness.md](completeness.md) |
| Domain vocabulary and energy semantics | [domain-model.md](domain-model.md) |
| Configuration sources and environment | [configuration.md](configuration.md) |
| iSolarCloud transport, authentication, endpoints | [isolarcloud-api.md](isolarcloud-api.md) |
| Response conversion, typed values, virtual points | [data-normalization.md](data-normalization.md) |
| MQTT runtime, discovery, scheduling | [mqtt.md](mqtt.md) |
| Home Assistant entity contract | [home-assistant-entities.md](home-assistant-entities.md) |
| Managed dashboard lifecycle | [dashboard-management.md](dashboard-management.md) |
| Dashboard metric resolution | [dashboard-resolution.md](dashboard-resolution.md) |
| Data-source overrides | [dashboard-source-overrides.md](dashboard-source-overrides.md) |
| Custom dashboard cards | [dashboard-frontend.md](dashboard-frontend.md) |
| CLI and Home Assistant app | [cli-and-addon.md](cli-and-addon.md) |
| Persistence, errors, recovery, security | [cross-cutting.md](cross-cutting.md) |
| Build, packaging, release | [delivery.md](delivery.md) |
| Executable examples | [acceptance.md](acceptance.md) |

Architecture decisions are recorded under [decisions/](decisions/README.md). Implementation coverage is recorded in [traceability.md](traceability.md).
The one-time migration classification is recorded in [migration-inventory.md](migration-inventory.md).

## Change protocol

For any observable behavior change, keep all steps on one feature branch and submit them in one pull request:

1. Identify affected requirement IDs.
2. Modify or add normative requirements and acceptance examples.
3. Update implementation and tests.
4. Update `traceability.md` if ownership moved or a requirement was added.
5. Update user documentation when users can observe or configure the change.
6. Run `python scripts/check_specs.py`, `go test ./...`, `node tools/preview/gosungrow-source-mapping-card.test.cjs`, `node tools/test_energy_summary_card.mjs`, and `bash -n addon/gosungrow/run.sh`.

Use `$plan-spec-change` for a read-only approval packet. After reviewing it, reply `do this` to authorize `$plan-spec-change-implementation` to apply the normative delta and derive implementation. The implementation turn must reject a stale packet. See [the workflow guide](../docs/spec-change-workflow.md).

A change that intentionally has no behavioral effect MAY omit a spec edit, but its review description must say why.

## Completeness boundary

[completeness.md](completeness.md) defines the auditable boundary. In short, these specifications determine all intentional behavior while leaving behavior-neutral implementation mechanics replaceable.
