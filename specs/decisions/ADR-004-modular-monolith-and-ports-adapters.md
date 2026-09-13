# ADR-004: Modular monolith with ports and adapters

Status: Accepted

Date: 2026-09-13

## Context

GoSungrow is one deployable Home Assistant app with a supported diagnostic CLI. Its
runtime is an ordered integration pipeline:

1. Resolve and validate configuration.
2. Authenticate to iSolarCloud.
3. Discover plants, devices, metadata, and points.
4. Normalize remote data into stable telemetry semantics.
5. Publish Home Assistant MQTT discovery and retained state.
6. Generate and reconcile a managed Home Assistant dashboard.

The system crosses four unreliable or independently evolving boundaries:

- Sungrow iSolarCloud HTTP and encrypted wire protocols.
- MQTT broker communication and Home Assistant MQTT discovery.
- Home Assistant WebSocket, Lovelace, registry, state, and statistics APIs.
- Filesystem, container, and Home Assistant app process lifecycle.

Most important product rules are deterministic transformations and decisions rather
than external I/O: opaque identifier handling, unit normalization, virtual formulas,
target selection, entity projection, dashboard matching, capability pruning, source
validation, ownership hashing, and error classification.

The architecture must preserve those rules while making cancellation, retries,
persistence, external failures, and compatibility behavior explicit. It must also remain
natural Go. Architectural ceremony is not a goal.

## Decision

GoSungrow will be implemented as a **modular monolith** using **ports and adapters at
external boundaries**, a **functional core with an imperative shell**, and **explicit
state machines for lifecycle and recovery workflows**.

These terms describe dependency and ownership rules, not a mandatory framework or
directory template.

## Architectural principles

### One deployable modular monolith

The Go runtime remains one binary and one coordinated process from the product's point
of view. Internal modules have cohesive responsibilities and controlled dependencies,
but they are not independently deployed services.

The dashboard card remains a browser-side JavaScript artifact because Home Assistant
executes it in a separate environment. It is versioned, tested, and delivered with the
same product rather than operated as a separate service.

### Functional core and imperative shell

Deterministic product decisions should be expressed as pure or side-effect-free logic
where practical. Such logic receives all relevant inputs explicitly and returns values,
plans, classifications, or diagnostics without performing network, filesystem, process,
clock, random, or publication operations.

The imperative shell owns external effects and ordered workflow execution. It invokes
the functional core, applies returned decisions, and reports outcomes.

Examples of functional-core responsibilities include:

- Identifier normalization and validation.
- Typed value and unit conversion.
- Virtual-point calculations.
- Device and target selection.
- Endpoint batching decisions.
- MQTT topic, identity, metadata, and payload projection.
- Dashboard candidate gates, scoring, confidence, and tie-breaking.
- Capability pruning and localization fallback decisions.
- Canonical serialization inputs and ownership comparison.
- Error classification and retry-decision calculation.

Examples of imperative-shell responsibilities include:

- HTTP and WebSocket calls.
- MQTT connection, subscription, and publication.
- Filesystem reads and replacement-safe writes.
- Clocks, timers, sleeping, and cancellation.
- Process startup, shutdown, and restart.
- Applying and verifying Home Assistant mutations.

### Ports only at real boundaries

Interfaces are introduced where an application workflow consumes behavior supplied by
an external system or where multiple real implementations exist. Interfaces are small,
defined beside the consuming workflow, and shaped around the consumer's needs.

Expected port categories include:

- Cloud authentication, discovery, and point collection.
- MQTT publication and subscription.
- Home Assistant dashboard, registry, state, locale, and statistics operations.
- Persistent state replacement.
- Clock/timer behavior where deterministic workflow testing requires it.
- Secure randomness where protocol testing requires a controlled source.

Pure domain logic does not require interfaces merely to permit mocking. Providers do not
publish broad interfaces for consumers to adopt. There is no shared catch-all `ports`,
`services`, `repositories`, or `interfaces` package.

### Dependency direction

Dependencies point from orchestration and adapters toward stable models and policies:

```text
process and command entrypoints
             |
             v
application workflows and lifecycle state machines
        |                              |
        v                              v
deterministic policy/model       consumer-owned ports
                                       ^
                                       |
                          external-system adapters
```

Stable telemetry and dashboard policy must not import CLI, MQTT-client, HTTP-client,
WebSocket, filesystem, container, or process-supervision implementations.

Adapters may depend on stable models needed to translate data, but one adapter must not
reach through another adapter to reuse its internal representation. Cross-boundary data
passes through validated application or domain types.

### Explicit workflow state machines

Long-lived runtime behavior is modeled as explicit states and transitions rather than
scattered retry loops and flags. The model need not be a general state-machine
framework; ordinary typed Go control flow is preferred.

The synchronization lifecycle should make states equivalent to the following visible in
its design and tests:

```text
starting
  -> authenticating
  -> discovering
  -> connecting MQTT
  -> synchronizing
  -> waiting

synchronizing
  -> recovering session
  -> Docker DNS backoff
  -> waiting for the next normal cycle
  -> fatal termination

any active state
  -> stopping
```

The specification's recovery limits remain authoritative. An adapter reports a typed or
otherwise stable failure classification; the workflow that owns a retry decides whether
to replay, reauthenticate, defer to the next cycle, back off, or terminate. Lower layers
must not introduce hidden retries that multiply the attempts owned by higher layers.

Process-level restart remains the app wrapper's responsibility as specified. Where a
process boundary prevents returning Go error values directly, the binary should expose a
stable machine-readable exit classification. Human-readable log text should not be the
long-term interface for restart decisions.

### Snapshots and plans

Multi-stage work should be represented by complete values before externally visible
mutation where practical.

A telemetry synchronization constructs a normalized, plant-isolated snapshot before
projecting and publishing it. Publication may not be atomic at the broker, but completion
state such as `LastRefresh` advances only according to the normative synchronization
contract.

A dashboard reconciliation constructs a complete desired plan before mutation. The plan
contains target-specific configuration, resolved sources, warnings, ownership hashes,
and the reason a mutation is or is not permitted. Home Assistant mutation follows the
specified read, validate, save, reread, and verify protocol. Persistent managed state is
updated only at the point allowed by that protocol.

### Strong internal subsystem boundaries

The following responsibilities are conceptually distinct even when two responsibilities
initially share a Go package:

| Subsystem | Owns | Does not own |
|---|---|---|
| Configuration | Precedence, parsing, typed validation, secret-safe values | Runtime recovery |
| Runtime | Startup order, scheduling, cancellation, recovery transitions | Wire schemas or energy formulas |
| Cloud | iSolarCloud protocol, authentication, endpoint schemas, transport execution | MQTT or dashboard presentation |
| Telemetry | IDs, points, measurements, units, validity, normalization, virtual formulas | Network and persistence |
| Discovery | Plant/device inventory, deterministic target selection, batching | MQTT transport |
| MQTT | Broker lifecycle, HA discovery projection, retained publication, select commands | Dashboard overrides |
| Dashboard | Matching, pruning, rendering, source mappings, ownership, reconciliation plans | MQTT entity mutation |
| Home Assistant | WebSocket protocol and Home Assistant operations | Candidate-scoring policy |
| Persistence | Atomic replacement mechanics and typed state encoding | Product retry decisions |
| Browser cards | Presentation and specified browser transactions | Server-side candidate generation |

Subsystem boundaries exist to protect ownership and dependency direction. They do not
require one package per table row, and they do not prohibit later package splits.

## Illustrative Go organization

An implementation may converge on an organization similar to:

```text
cmd/gosungrow/          composition root and command/process boundary
internal/config/        configuration resolution and validation
internal/runtime/       startup, synchronization, recovery, shutdown
internal/cloud/         iSolarCloud transport, authentication, endpoint DTOs
internal/telemetry/     normalized points, units, formulas, validity
internal/discovery/     inventory, targets, ordering, batching
internal/mqtt/          MQTT and Home Assistant discovery publication
internal/dashboard/     resolution, mappings, rendering, ownership, plans
internal/homeassistant/ Home Assistant WebSocket adapter
internal/store/         atomic persistence adapters
web/cards/              browser-side custom cards
addon/                  thin Home Assistant app wrapper
```

This tree is illustrative, not normative. Package names, file names, private types, and
the number of packages may change without revising this decision as long as the
responsibility, dependency, and side-effect boundaries remain intact.

## Translation boundaries

External representations are validated and translated before entering stable policy:

```text
untrusted external representation
                -> validated wire model
                -> canonical internal model
                -> deterministic decision or projection
                -> external side effect
```

iSolarCloud response structs are transport models, not the canonical telemetry model.
Home Assistant state and registry responses are adapter inputs, not dashboard policy
objects. MQTT discovery payloads are projections, not the source of domain meaning.

Unknown-field preservation required for compatibility belongs at the relevant wire
boundary. Known semantics must be represented explicitly after translation.

## Concurrency and ownership

Concurrency is introduced only for a specified lifecycle or demonstrated latency need.
Sequential orchestration is the default.

Every goroutine has one owner, a cancellation path, a termination condition, and an
error-handling strategy. Background dashboard reconciliation is independent from MQTT
startup failure semantics but is owned and stopped by the app lifecycle.

Shared mutable runtime state has one clear owner or explicit synchronization. Channels
are coordination mechanisms, not a substitute for subsystem APIs. The architecture does
not assign one actor or goroutine to every plant, device, or point.

## Testing consequences

The architecture should allow most normative decisions to be tested without network,
broker, Home Assistant, filesystem, wall-clock, or process dependencies.

- Pure policy uses table-driven and property/invariant tests.
- Workflow state machines use small fake ports and controlled clocks.
- Cloud, MQTT, WebSocket, and persistence adapters use protocol-focused integration
  tests against local test servers, temporary directories, or narrow fakes.
- End-to-end acceptance tests verify subsystem composition and the normative lifecycle.
- Tests assert observable behavior and stable classifications rather than private
  package structure.

Architecture tests or dependency checks may enforce prohibited import directions, but
they must not freeze the illustrative directory tree unnecessarily.

## Rejected alternatives

### Microservices

Rejected because the product has one deployment unit, one operational lifecycle, and
tightly coordinated state. Service boundaries would add network failure, versioning,
deployment, and observability costs without an independent scaling or ownership need.

### Heavy domain-driven design

Rejected as the dominant architecture because GoSungrow is primarily an integration,
normalization, projection, and reconciliation system. Domain-specific value types and
vocabulary are useful; aggregate repositories, domain services for every operation, and
other tactical ceremony are not.

### Traditional handler-service-repository layering

Rejected because it does not express the system's real boundaries. Most persistence is
cache or managed state rather than an interchangeable business repository, and the key
complexity lies in protocol adapters, deterministic transformations, and lifecycle
orchestration.

### Event sourcing, CQRS, or a generic event bus

Rejected because the specifications do not require command reconstruction, independent
read/write scaling, or asynchronous integration among independently owned components.
The primary pipeline is ordered and its completion and recovery semantics must remain
obvious.

### Generic pipeline, actor, or dependency-injection frameworks

Rejected because stages are not arbitrary plugins, per-device actors are not required,
and ordinary Go constructors can assemble the dependency graph. Framework indirection
would obscure control flow and failure ownership.

### Interfaces around all components

Rejected because interfaces are most useful at consumer-owned behavioral seams.
Interface-first design for pure calculations and single concrete implementations would
increase indirection, mock coupling, and API surface without improving substitution.

## Consequences

### Positive

- Normative decision logic can be tested deterministically and exhaustively.
- External protocol changes remain localized to adapters and translation boundaries.
- Retry and cancellation ownership becomes inspectable rather than emergent.
- The single deployable remains operationally simple.
- Dashboard complexity receives a strong boundary without becoming another service.
- Go code can remain concrete and straightforward within each module.
- Package organization may evolve without changing product behavior.

### Costs and risks

- Maintainers must actively prevent adapters from leaking wire types into policy code.
- A modular monolith can still become a tightly coupled monolith if dependency direction
  is not reviewed.
- Explicit state transitions require more initial design than ad hoc retry loops.
- Snapshot and plan construction may use more memory than streaming every intermediate
  result directly, though correctness and deterministic behavior take priority unless
  measurement proves the cost material.
- Browser-side dashboard transactions remain a separate execution environment and need
  their own pure-core/effect-boundary discipline.

## Compliance guidance

This ADR governs structural decisions but does not redefine observable product behavior.
Normative requirements under `specs/` remain authoritative.

A change should be challenged during review when it:

- Makes stable policy depend directly on an external client or global process state.
- Introduces an adapter-to-adapter dependency instead of translating through a stable
  model or application workflow.
- Adds retries, sleeps, or goroutines without clear lifecycle ownership.
- Introduces broad provider-owned interfaces or architectural framework machinery.
- Couples dashboard policy to cloud wire schemas or MQTT client implementation details.
- Makes human-readable error strings the only internal classification mechanism.
- Splits a component into a separately deployed service without a new accepted ADR.

Departures are permitted when a concrete requirement cannot otherwise be implemented
safely or clearly. A material departure requires a superseding ADR that explains the
constraint and consequences.
