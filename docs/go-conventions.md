# Go engineering conventions for GoSungrow

Status: Normative for implementation quality and process.
Audience: Coding assistants, maintainers, and reviewers.
Last researched: 2026-09-13.

## PURPOSE, AUTHORITY, AND NON-NEGOTIABLE PRIORITIES

This standard tells an implementer HOW to translate accepted behavior into excellent
Go. It does not define WHAT the product does. Product behavior comes from specs/.

Interpret requirement words as follows:

  - MUST and MUST NOT are mandatory. A change that violates one is incomplete.
  - SHOULD and SHOULD NOT are strong defaults. Depart only for a concrete, documented
    reason that makes the result safer, clearer, more compatible, or measurably better.
  - MAY grants permission; it never creates a requirement.

When instructions conflict, use this order of authority:

  1. Accepted normative requirements under specs/.
  2. Security, data-integrity, and Go language/toolchain correctness.
  3. Repository-wide instructions in AGENTS.md and validated architecture decisions.
  4. This implementation standard.
  5. Established local code conventions in the affected package.
  6. Personal preference.

The spelling of AGENTS.md may differ by filesystem case. Locate and read the actual
repository file. Never use this precedence list to silently ignore a conflict. Report
the conflict and resolve it in the higher-authority document before implementing.

Optimize, in order, for:

  1. Correct conformance to the specifications.
  2. Safety and preservation of user data and secrets.
  3. Clarity to the next reader.
  4. Simplicity and explicit control flow.
  5. Maintainability and testability.
  6. Compatibility.
  7. Efficiency demonstrated by evidence.
  8. Concision.

"Professional" does not mean abstract, clever, compressed, or enterprise-shaped.
World-class Go is usually unsurprising: small coherent packages, explicit data flow,
useful errors, bounded work, obvious ownership, focused interfaces, and strong tests.

## REQUIRED SPEC-FIRST IMPLEMENTATION WORKFLOW

For every task, perform this sequence. Do not start from the code alone.

### Establish the contract

  1. Read AGENTS.md, specs/README.md, specs/governance.md, and the affected subsystem
     specifications. Then read specs/acceptance.md and relevant entries in
     specs/traceability.md.
  2. Identify the requirement IDs governing the requested behavior.
  3. Classify the request:
       a. Behavior change: externally observable logic changes.
       b. Refactor: behavior is intentionally unchanged.
       c. Repair: implementation is brought back into agreement with an existing spec.
       d. Tooling/docs-only: runtime behavior is unchanged.
  4. For a behavior change, edit the specifications and acceptance criteria before or
     in the same change as code. Add stable requirement IDs and update traceability.
  5. If the desired behavior is ambiguous, contradictory, or missing, do not infer it
     from incidental legacy code. Clarify or amend the spec first.

### Inspect before designing

  - Trace the complete call path, including command setup, API transport, caching,
    persistence, MQTT publication, logging, and recovery where applicable.
  - Find all implementations, tests, mocks, fixtures, configuration keys, serialized
    fields, and callers affected by a symbol or behavior.
  - Check the working tree and preserve unrelated user changes.
  - Identify invariants, ownership, cancellation, failure boundaries, compatibility
    constraints, and security-sensitive values before changing types or signatures.
  - Prefer extending a sound local pattern. Do not perpetuate a local pattern that is
    unsafe, incorrect, or explicitly superseded by this document.

### Design the smallest complete change

  - Write down inputs, outputs, state transitions, side effects, and error cases.
  - Keep policy at the layer that owns the decision and mechanics at the layer that
    owns the resource. Do not scatter one requirement across unrelated packages.
  - Make invalid states difficult to represent when doing so remains simple.
  - Separate pure decision logic from I/O. Pure logic should accept explicit values and
    return results that can be exhaustively tested.
  - Plan failure behavior, cancellation, cleanup, deterministic ordering, logging, and
    tests at the same time as the happy path.
  - Avoid unrelated cleanup. A focused diff is easier to validate and review.

### Implement and prove

  - Add or update tests with the implementation. A behavioral fix normally requires a
    regression test that fails before the fix and passes after it.
  - Format changed Go files and run all mandatory validation in section 24.
  - Review the diff, not just the final files. Confirm every changed line is intended.
  - Verify the implementation against each affected requirement ID and each relevant
    prohibited behavior, not merely against existing tests.
  - Update user-facing documentation if users can observe or configure the change.

No task is complete merely because it compiles or because tests happen to pass.

## LANGUAGE VERSION, TOOLCHAIN, AND PORTABILITY

  - The go directive in go.mod is the source of truth for the language version. This
    repository currently declares Go 1.19.
  - MUST NOT use syntax, standard-library APIs, or module features newer than the
    declared version unless the same accepted change deliberately raises the version,
    updates delivery requirements, and validates all supported builds.
  - MUST preserve the required CGO_ENABLED=0 Linux build and the supported amd64 and
    arm64 targets. Do not add cgo or platform-specific behavior accidentally.
  - Use build constraints only for real platform or build-mode differences. Include the
    modern //go:build form; if compatibility with an older toolchain requires the
    legacy form, keep both consistent.
  - Platform-dependent filenames and code MUST have a portable counterpart or an
    intentionally restricted package. Do not assume Unix path separators, rename
    behavior, signals, permissions, or case sensitivity where the specs allow Windows.
  - Do not hand-edit go.sum. Let the Go command maintain module metadata, then review
    the diff.

## PACKAGE AND ARCHITECTURE DESIGN

  - A package MUST have one coherent purpose that can be described in one sentence.
  - Package names MUST be short, lowercase, singular where natural, and free of
    underscores. Avoid vague new packages such as util, common, misc, helpers, types,
    interfaces, or api. Existing package names are compatibility constraints; do not
    rename them casually.
  - Package names and exported identifiers MUST read naturally together. Prefer
    cache.Reader to cache.CacheReader and mqtt.Client to mqtt.MQTTClient.
  - Avoid import cycles by correcting ownership, not by creating a dumping-ground
    package.
  - Keep dependency direction clear: commands compose; domain/policy packages decide;
    adapters perform external I/O. Lower-level packages MUST NOT import command/UI
    layers to report progress or read global configuration.
  - Keep main and command handlers thin. They parse inputs, assemble dependencies,
    invoke application behavior, translate the final result, and choose process exit
    behavior. Business decisions belong in testable functions or types.
  - Use internal packages only when an import boundary provides real protection. Do
    not fragment code solely to imitate an architecture diagram.
  - Avoid cyclic runtime relationships, hidden registries, init-time wiring, and global
    service locators. Construct dependencies explicitly.
  - Do not create a new abstraction until there are concrete callers or a clear seam
    for external effects. Duplication of a few obvious lines is often cheaper than a
    premature framework.
  - Keep files organized by cohesive responsibility, not arbitrary maximum size. Split
    a file when distinct concepts can be named, tested, and understood independently.
  - Filenames MUST be lowercase. Underscores are acceptable for meaningful multiword
    filenames and required suffixes such as _test.go, _linux.go, and _amd64.go.

## API DESIGN, TYPES, AND INTERFACES

  - Minimize exported surface. Export only what another package genuinely needs.
  - Accept interfaces when the function needs behavior; return concrete types by
    default. Define a small consumer-side interface close to its use, not in the
    provider package merely "for mocking."
  - Interfaces SHOULD contain the minimum method set needed by the consumer. One- or
    two-method interfaces are preferable when they express a real capability.
  - Do not add methods to a public interface without treating it as a breaking change.
  - Constructors SHOULD return a usable value and an error when construction can fail.
    NewX SHOULD return *X unless value semantics are intentional. Do not return an
    interface just to hide a concrete implementation.
  - The zero value SHOULD be useful when this is natural and safe. Otherwise require a
    constructor and ensure direct zero-value use fails clearly, never unpredictably.
  - Use value receivers for small immutable value-like types. Use pointer receivers
    when methods mutate state, the type contains synchronization primitives, copying is
    expensive, or receiver consistency requires it. Do not mix receiver kinds without
    a reason.
  - Types containing sync.Mutex, sync.Once, atomic values, or other non-copyable state
    MUST NOT be copied after first use. Prefer pointer construction and methods.
  - Do not embed a type in a public struct unless promoting its entire method set is an
    intentional API commitment. Prefer named fields for implementation reuse.
  - Avoid boolean parameters whose meaning is unclear at the call site. Use a named
    type, separate methods, or an options struct when there are multiple independent
    options. Do not introduce functional options for one or two stable values.
  - Keep structs cohesive. Do not use one giant request/config/state struct as an
    implicit parameter bag shared by unrelated operations.
  - Prefer domain-specific types when values have different units or invariants and
    could otherwise be confused. Avoid wrapper types that add no safety or behavior.
  - Generics MAY be used only when one algorithm genuinely applies to multiple types
    and the result is clearer than an interface or small duplication. Go 1.19 syntax
    is the ceiling. Do not introduce generic containers already provided by maps,
    slices, or the standard library.
  - Reflection is a last resort for schema-driven behavior that cannot be expressed
    clearly with types. Isolate it, validate every assumption, preserve errors, and
    test nils, pointers, aliases, malformed tags, and unexpected kinds.

## NAMING

Names communicate role, not implementation trivia.

  - Use MixedCaps or mixedCaps, never snake_case in Go identifiers. Preserve external
    wire names in JSON/YAML tags and protocol constants.
  - Initialisms use consistent capitals: ID, API, URL, HTTP, JSON, MQTT, DNS, TLS, UUID.
    Examples: deviceID, APIClient, serveHTTP. Follow established exported names when a
    compatibility constraint prevents correction.
  - Name variables in proportion to scope. i and j are fine for tiny index loops; err,
    ctx, req, resp, and cfg are conventional when unambiguous. Use descriptive names
    when values coexist, live longer, or carry different units or identities.
  - Avoid meaningless names: data, info, item, object, manager, processor, handler, and
    result need a qualifier unless their meaning is unmistakable locally.
  - Do not encode a type in a name: usersByID, not userMap; timeout, not timeoutDuration.
    A unit suffix such as timeoutSeconds is appropriate only at an untyped boundary.
  - Avoid package-name repetition in exported identifiers.
  - Getters use Name(), not GetName(), unless Get is part of the domain operation.
    Setters use SetName only when mutation is a deliberate API.
  - Acronym-only and single-letter exported names are rarely acceptable. Exported
    names must be searchable and comprehensible in documentation.
  - Constants are MixedCaps, not SCREAMING_SNAKE_CASE.
  - Sentinel errors use ErrName. Concrete error types use NameError. Error variables
    MUST be stable only if callers are intended to inspect them.
  - Test names follow TestUnit_Scenario where the suffix helps, and subtest names
    describe behavior in human-readable terms. Do not use issue numbers as the only
    description.

## FORMATTING, SOURCE LAYOUT, AND DECLARATIONS

  - Every changed Go file MUST match gofmt output. Use goimports when available to also
    maintain imports; gofmt remains authoritative for layout.
  - Do not impose a hard line-length limit. If a line is hard to read, first simplify
    the expression or extract a well-named value. Do not wrap mechanically.
  - Imports MUST be used directly and grouped by tooling. Avoid dot imports. Import
    aliases are only for real collisions, generated names, or a substantially clearer
    local name; the alias should describe the package, not its contents.
  - Group declarations when they form one concept. Do not create var or const blocks
    merely to save keywords.
  - Declare variables near first use. Prefer := for initialized locals and var when the
    zero value is meaningful or the type must be explicit.
  - Reduce scope aggressively but naturally. Initialization in if and switch is useful
    when the value has no meaning afterward.
  - Avoid naked returns except in very short functions where named results materially
    clarify documentation. Named results are not scratch variables for long functions.
  - Do not use blank identifier assignments to hide an error or silence dead code.
  - Remove commented-out code, obsolete branches, debug prints, and speculative TODOs.
    A TODO must state a concrete action and, when possible, an issue/owner reference.
  - Generated files MUST say they are generated and must not be edited, using the form
    recognized by Go tooling: "Code generated ... DO NOT EDIT."

## COMMENTS AND DOCUMENTATION

  - Comments explain WHY a decision, invariant, workaround, safety boundary, or
    non-obvious algorithm exists. Do not narrate syntax.
  - Every exported package, type, function, method, constant, and variable SHOULD have
    a proper doc comment; it MUST when its contract is not obvious or lint requires it.
  - A declaration doc comment begins with the declared name and is a complete sentence.
    A package comment begins with "Package name" and appears once per package.
  - Document caller-visible behavior: units, accepted ranges, ordering, ownership,
    mutation, concurrency safety, blocking, cancellation, nil behavior, errors, and
    side effects. Do not promise implementation details unnecessarily.
  - Use doc-comment syntax supported by go/doc: paragraphs, headings, lists, links, and
    indented code. Verify complex documentation with go doc or a doc preview.
  - Comments MUST remain true after the change. Delete a stale comment rather than
    preserving misinformation.
  - Workarounds MUST identify the external constraint and the condition under which the
    workaround can be removed.
  - Security-sensitive code deserves comments describing the threat or invariant, not
    secret material.
  - Examples named Example, ExampleType, or ExampleType_Method SHOULD be executable
    tests when they teach public behavior.

## CONTROL FLOW AND FUNCTION DESIGN

  - Prefer the happy path aligned at the left margin. Handle invalid input and errors
    early, then continue with the successful flow.
  - Keep functions at one level of abstraction. Extract a function when it gives a
    coherent operation a good name, isolates a side effect, or enables focused tests.
    Do not split linear readable code into tiny forwarding functions.
  - Avoid else after a branch that returns, continues, breaks, or panics.
  - Prefer switch when it expresses mutually exclusive classification more clearly
    than a long if/else chain. Always consider the default/unknown case.
  - Never rely on fallthrough unless the shared behavior is unmistakable and tested.
  - Loops MUST make termination and progress clear. Retries and polling MUST be bounded
    or explicitly lifecycle-bound by the spec, cancellable, and observable.
  - Defer cleanup immediately after successful acquisition. In a long loop, use a
    helper function when defers must run per iteration rather than at function exit.
  - Remember that deferred call arguments are evaluated at the defer statement. Use a
    closure only when later values are intentionally required.
  - Panic is for impossible programmer invariants or unrecoverable package initialization,
    not ordinary bad input, network errors, configuration, or library control flow.
    A library API presents errors even if tightly scoped internals use panic/recover.
  - Recover only at an explicit goroutine/process boundary that can restore a valid
    state, and never discard the panic silently. Tests may assert an intentional panic.

## DATA, COLLECTIONS, ZERO VALUES, AND OWNERSHIP

  - Decide and document whether nil and empty slices/maps are semantically equivalent,
    especially at JSON and public API boundaries. Preserve the specification's wire
    distinction where one exists.
  - Prefer nil slices when no allocated empty slice is required. Allocate maps before
    writes. A nil map is safe only for reads.
  - Preallocate slices/maps when the size is known or a hot path demonstrates benefit;
    avoid capacity arithmetic that obscures code for negligible gain.
  - Do not return internal mutable slices, maps, byte buffers, or pointers when callers
    must not mutate them. Clone at ownership boundaries.
  - State whether an input buffer is borrowed, retained, or modified when not obvious.
  - Use comma-ok map/type assertions when absence is possible. Do not confuse a missing
    entry with the element type's zero value.
  - Map iteration order is unspecified. Sort keys or maintain an ordered slice before
    any observable selection, serialization, hash, generated file, diagnostic, or test
    expectation that requires determinism.
  - Be explicit about numeric units and conversion. Check overflow, truncation, sign,
    precision, NaN, and infinity at untrusted boundaries.
  - Prefer time.Duration over untyped integer durations internally. Parse and validate
    once at configuration boundaries.
  - Use time.Time for instants and document timezone semantics. Compare instants with
    Time methods, not serialized strings. Inject a clock where deterministic tests or
    calendar policy requires it.
  - Never store pointers to short-lived loop variables in patterns whose semantics are
    ambiguous across Go versions. Create an explicit per-iteration value.

## ERRORS

Errors are values and part of the API contract.

  - Check every meaningful error. Never discard an error with _, use a blank defer, or
    overwrite err before handling it. The rare intentionally ignored error needs a
    comment proving why it cannot affect correctness or cleanup.
  - Error messages are lowercase sentence fragments without trailing punctuation,
    unless they contain a proper noun or quoted upstream sentence.
  - Add concise operational context at abstraction boundaries: what operation and which
    non-secret identity failed. Avoid redundant "failed to" chains.
  - Wrap with %w exactly when callers should retain programmatic access to the cause.
    Use %v or construct a new error when the underlying identity is deliberately not
    part of the API. Wrapping is an API decision, not automatic punctuation.
  - Inspect with errors.Is and errors.As. Do not compare error strings for control flow.
    If an upstream system exposes only strings, isolate classification in one tested
    adapter and convert it into stable local semantics.
  - Use sentinel errors only for stable categories that callers act upon. Use typed
    errors when callers need structured fields. Keep their Error output secret-safe.
  - Returning a typed nil as error produces a non-nil interface. Avoid it by returning
    nil explicitly.
  - Do not both log and return an error at every layer. Usually annotate and return;
    log once at the boundary that decides to retry, degrade, or report to the user.
  - Partial results plus an error require an explicit, documented contract. Otherwise
    return a safe zero result with the error.
  - Cleanup errors matter when they can lose or corrupt data. For writes, check Flush,
    Sync, and Close as applicable. Preserve the primary error while retaining relevant
    cleanup context; Go 1.19 code cannot use errors.Join.
  - Preserve context cancellation. Do not translate context.Canceled or
    context.DeadlineExceeded into an unrelated generic error.
  - Error text and logs MUST NOT contain passwords, tokens, encryption keys, private
    payloads, or other secrets named by specs/cross-cutting.md.

Example:

    body, err := client.fetch(ctx, endpoint)
    if err != nil {
        return nil, fmt.Errorf("fetch endpoint %q: %w", endpoint.Name, err)
    }

The endpoint name is useful and non-secret; credentials and full request bodies are not.

## CONTEXT, CANCELLATION, TIMEOUTS, AND RETRIES

  - context.Context is the first parameter, conventionally named ctx, on operations
    whose work may block, cross a process boundary, or outlive a trivial computation.
  - Do not store a Context in a struct. Pass it through the call chain. Do not pass nil;
    use context.Background only at a true root and context.TODO only as temporary
    migration scaffolding.
  - A function MUST propagate the caller's context to downstream requests. Do not
    replace it with Background and thereby sever cancellation.
  - The layer that creates a timeout owns its CancelFunc and MUST call it, normally with
    defer immediately after creation.
  - Put timeout policy at an application boundary or accepted subsystem policy, not
    independently in every helper. A child deadline may be shorter but never defeats
    an earlier parent deadline.
  - Context values are only for request-scoped cross-cutting metadata, not optional
    parameters, services, configuration, or mutable state. Use an unexported key type.
  - Cancellation is not proof a goroutine stopped. Design worker ownership and joining
    explicitly when shutdown must wait.
  - A retry MUST have: an explicitly classified transient error; safe/idempotent
    semantics or an idempotency mechanism; a limit or lifecycle rule; cancellation;
    specified backoff; and logs/metrics at the layer owning the retry.
  - Never retry all errors. Never multiply retries invisibly across layers. The
    GoSungrow retry and replay limits in specs/ are authoritative.
  - Respect context cancellation during backoff with a timer and select; do not use an
    unconditional Sleep in cancellable library logic.
  - Stop and drain timers correctly when reuse or concurrent selection makes stale
    ticks possible. Prefer time.NewTicker only for genuinely repeated schedules and
    always Stop it when owned work ends.

## CONCURRENCY AND GOROUTINE LIFECYCLE

  - Do not introduce concurrency without a concrete need. Synchronous code is easier to
    reason about and often sufficient for network-bound command applications.
  - Every goroutine MUST have a named owner, a termination condition, a cancellation or
    shutdown path, and an error/panic handling strategy. Never launch a goroutine whose
    lifetime is "eventually."
  - The creator of a channel owns closing it. Receivers normally do not close channels.
    Never close a channel to signal one receiver when context cancellation is clearer.
  - Do not send on a channel while holding a mutex when the send can block.
  - Choose channel capacity deliberately. An arbitrary buffered channel hides
    backpressure and does not solve a deadlock.
  - Shared mutable state MUST be serialized through channel ownership or protected by
    sync/atomic or a mutex. "It is probably not simultaneous" is not synchronization.
  - State guarded by a mutex SHOULD be adjacent to the mutex and identify the invariant.
    Keep critical sections small, but never unlock midway through an invariant merely
    to reduce lock duration.
  - Do not copy a mutex. Do not expose protected maps or slices after unlocking.
  - Use sync.Once for one-time concurrent initialization only when initialization cannot
    naturally occur during construction. Decide how initialization failures behave.
  - Use atomics for simple, well-documented independent state. Complex invariants need
    a mutex or single owner. Clever lock-free code is prohibited without evidence.
  - Worker groups MUST collect errors, cancel sibling work when policy requires it, and
    wait for every worker before returning. Bound fan-out; do not spawn per item without
    a concurrency limit for unbounded input.
  - Code that adds or changes concurrency MUST be tested with the race detector on a
    supported platform and should include cancellation/leak-sensitive tests.

## I/O, HTTP, MQTT, AND RESOURCE SAFETY

  - External I/O MUST be context-aware and bounded by specification-approved timeouts.
  - Reuse a configured http.Client and its Transport. Do not create one per request.
    A zero client has no overall timeout, so enforce the repository's explicit timeout
    policy through client configuration and contexts.
  - Construct HTTP requests with NewRequestWithContext. Validate method, URL, headers,
    content type, and body at the adapter boundary.
  - After a successful Do, close resp.Body on every path. Read or drain it only when
    required for connection reuse and within a strict size bound.
  - Check HTTP status before decoding success payloads. Bound error-body reads and
    sanitize them before returning or logging.
  - Limit untrusted response/request sizes before decoding. A Decoder alone does not
    create a byte limit.
  - Configure server read, header, write, and idle timeouts if this program gains an
    HTTP server. Avoid bare http.ListenAndServe for an Internet-facing service.
  - Do not log complete headers or payloads. Authorization, cookies, tokens, app keys,
    encrypted material, and decrypted request data are sensitive.
  - MQTT calls MUST honor the QoS, retain, timeout, ordering, and outage semantics in
    specs/mqtt.md. Check token completion and token errors; do not treat enqueueing as
    confirmed publication.
  - Establish resource ownership immediately: the opener closes files/bodies/connections
    unless ownership is explicitly transferred and documented.

## SERIALIZATION, PROTOCOLS, AND UNTRUSTED INPUT

  - Treat network, environment, CLI, YAML, JSON, cached files, and Home Assistant/MQTT
    values as untrusted input.
  - Use typed wire structs for stable schemas and explicit tags for external names.
    Avoid map[string]any except where the schema is intentionally dynamic, unknown
    fields must be preserved, or dashboard tree manipulation requires it.
  - Separate wire representation from validated domain state when their invariants or
    types differ. Conversion is the validation boundary.
  - Validate required fields, ranges, units, enum values, mutually exclusive fields,
    and cross-field invariants before mutation or expensive work.
  - Decide unknown-field behavior from the specs. This repository preserves some
    unknown fields/options when safe but fails closed on unknown result codes, schemas,
    and semantic conflicts.
  - Do not add DisallowUnknownFields where forward compatibility requires preservation.
    Do use it for strict private configuration formats if the spec requires rejection.
  - Use json.Number or an explicitly typed field when float64 conversion could lose an
    integer identity or precision.
  - Do not rely on Go map iteration for canonical JSON. Build an ordered representation
    or otherwise implement the deterministic contract explicitly.
  - Escape and encode with established libraries. Do not assemble JSON, URLs, query
    strings, shell commands, or cryptographic framing through ad hoc concatenation.
  - Preserve protocol-required distinctions among absent, null, zero, false, and empty.
    Use pointers or custom marshaling only when the wire contract requires it.
  - Bound recursion, collection counts, token lengths, and allocations influenced by
    untrusted data where realistic inputs could exhaust resources.

## FILES, CACHE, CONFIGURATION, AND PERSISTENCE

  - All replacing writes MUST follow specs/cross-cutting.md: create a temporary sibling,
    write, apply required mode, sync, close, and atomically rename; clean up the temp on
    failure. Preserve the documented Windows replacement behavior.
  - Never truncate the destination before a replacement is fully written and durable.
  - Validate and normalize paths before mutation. Keep runtime state and public assets
    in the exact specified roots. Do not allow a relative or attacker-controlled path
    to escape its intended root.
  - Use filepath for filesystem paths and path for slash-separated protocol/URL paths.
  - Set permissions explicitly for sensitive state. Do not assume umask is sufficient.
  - Distinguish not-exist, permission, corruption, and transient I/O errors with
    errors.Is and typed errors. Remove corrupt cache entries only where specs permit it.
  - Configuration precedence and defaults are product behavior and MUST be specified.
    Parse once, validate completely, and pass typed configuration inward.
  - Environment lookup should distinguish unset from explicitly empty when behavior
    differs. Do not dump the process environment in diagnostics.
  - Avoid package init for filesystem, network, environment-dependent, or fallible work.
  - Persist only after upstream mutation/verification succeeds when the spec requires
    optimistic verification. A partial external failure must not masquerade as saved
    local truth.

## LOGGING, DIAGNOSTICS, AND OBSERVABILITY

  - Log at ownership boundaries: startup/shutdown, lifecycle transition, external
    operation result, retry decision, degraded mode, and final failure.
  - A log line SHOULD answer what operation occurred, which safe entity it affected,
    what category/result occurred, and what happens next.
  - Use stable structured fields if supported by the existing logger. Otherwise keep
    key terms consistent and messages actionable.
  - Do not log and return the same error at every layer. This creates duplicated noise
    and often leaks low-level details.
  - Choose levels consistently: debug for bounded diagnostic detail; info for normal
    lifecycle milestones; warning for recovered/degraded conditions requiring attention;
    error for an operation that ultimately failed.
  - Retry logs include the safe operation name, attempt/count when bounded, delay, and
    classified reason. Successful recovery reports resumption and outage duration as
    required by specs/cross-cutting.md.
  - Never emit credentials, tokens, passwords, API keys, encryption keys, Supervisor
    tokens, MQTT passwords, decrypted payloads, or unbounded external bodies.
  - Redaction happens before formatting and before data reaches a logger. Do not rely on
    a downstream sink to remove secrets.
  - Diagnostics MUST be bounded in cardinality and size. Sort fields whose ordering is
    observable. Avoid one log per item in an unbounded collection.
  - Do not use fmt.Print in library/runtime code for logging. Use the repository logger
    or return data to the command layer. Direct output is acceptable only when stdout or
    stderr is the defined command result.

## DETERMINISM AND REPRODUCIBILITY

  - Identical inputs MUST yield identical externally visible output except for the
    explicit nondeterministic values allowed by specs/product-and-architecture.md.
  - Sort map-derived keys before selection, output, hashing, comparison, diagnostics,
    generated configuration, or tests. Use a documented stable discovery order where
    that is the contract.
  - Define tie-breakers completely. Do not depend on filesystem enumeration, goroutine
    completion, randomized map order, or upstream incidental ordering.
  - Inject clocks, random sources, and external clients when logic depends on them.
    Production cryptographic randomness MUST remain cryptographically secure; tests may
    supply deterministic fakes at a seam below policy.
  - Do not make production behavior depend on test execution order, global mutable
    state, locale, local timezone, current working directory, or ambient HOME unless
    explicitly specified.
  - Canonical fingerprints include all behavior-relevant inputs with unambiguous framing
    and stable encoding. Never concatenate variable strings without delimiters/lengths.

## DEPENDENCIES AND MODULE HYGIENE

  - Prefer the standard library when it provides a clear, maintained solution.
  - A new dependency requires a concrete benefit exceeding its supply-chain, API,
    binary-size, licensing, maintenance, and transitive-dependency cost.
  - Before adding one, inspect its official documentation, repository health, license,
    release history, Go-version support, transitive graph, security history, and whether
    the needed feature can be implemented safely in a small amount of local code.
  - Pin through go.mod/go.sum. Do not use unreviewed replace or exclude directives in a
    committed production module.
  - Run go mod tidy only when imports/module metadata changed, and review every resulting
    direct/indirect dependency change.
  - Upgrades MUST be scoped, release-note reviewed, vulnerability checked, and tested.
    Do not combine a broad dependency refresh with unrelated behavior work.
  - Public API changes follow Go 1 compatibility principles. Avoid breaking exported
    signatures, removing names, changing interface method sets, narrowing accepted
    inputs, or changing serialized behavior without an explicit compatibility decision.

## TESTING STANDARD

Tests prove requirements and protect design. They are not implementation theater.

### What to test

  - Every behavior change MUST test the happy path, relevant boundaries, specified
    errors, and regression case. Safety, retries, ordering, redaction, persistence, and
    cancellation deserve first-class tests.
  - Tests SHOULD target exported or externally meaningful behavior and stable package
    seams. Test private helpers directly only when they encode complex policy that is
    clearer to exercise in isolation.
  - Every normative requirement affected by the change must be covered or have a clear
    reason why coverage belongs at another validation layer.

### Test construction

  - Use table-driven tests when cases share setup and assertion logic. Use separate tests
    when conditional test machinery would obscure behavior.
  - Each case has a descriptive name. Inputs, expected outputs, and expected error
    semantics are explicit.
  - Prefer direct Go comparisons for simple values. For large structures, report a
    readable diff with direction labeled -want +got. Do not add an assertion DSL that
    hides control flow or failure details.
  - Failure messages use: Function(input) = got, want want. Print got before want and
    include relevant inputs without secrets.
  - Use t.Helper in genuine helpers. Use t.Cleanup for cleanup registered during setup.
  - Use t.Fatal only when the current test/subtest cannot proceed safely. Use t.Error to
    report independent checks and continue collecting useful failures.
  - Test error identity with errors.Is/errors.As or stable categories, not full string
    equality. Check message fragments only when human-facing content is itself required.
  - Compare serialized formats semantically unless exact bytes/order are the contract.
  - Do not sleep to "wait" for concurrency. Synchronize on observable events and use a
    bounded timeout only to prevent a hung test.
  - Call t.Parallel only after proving the test and all subtests share no global state,
    environment, filesystem path, clock, port, logger, or mutable fixture. Capture a
    per-iteration test-case value explicitly for Go 1.19 loop semantics.
  - Use t.Setenv and t.TempDir. Never depend on a developer's real config, credentials,
    HOME, network, or current time.
  - Avoid package-level mutable mocks. Prefer small fakes that implement consumer-side
    interfaces and record calls under test ownership.

### Test layers

  - Unit tests: pure policy, validation, conversion, classification, ordering, and
    state transitions.
  - Adapter tests: use httptest and local fakes to verify HTTP method/path/headers/body,
    timeout/cancellation, response limits, status handling, and redaction. No live cloud
    dependency in the default suite.
  - Integration tests: verify package collaboration and persistence with isolated temp
    directories. Mark genuine external tests explicitly and keep them opt-in.
  - Golden tests: use only for stable, reviewable, intentionally exact output. Provide
    an explicit update mechanism, never update automatically in normal test runs, and
    inspect golden diffs.
  - Fuzz tests: add for parsers, decoders, protocol transforms, ID normalization, glob
    matching, and other untrusted inputs. Seed valid, boundary, and previously failing
    cases. Assert invariants, bounded behavior, and no panic—not merely non-crashing on
    trivial inputs.
  - Benchmarks: add only for a meaningful performance question. Report allocations when
    relevant, keep setup outside the timed region, and compare statistically.

### Test quality

  - A test must fail for the intended reason if the behavior is broken. Avoid assertions
    so broad that a different failure passes.
  - Tests MUST be deterministic, hermetic, order-independent, and safe to repeat.
  - Do not weaken or delete a valid test just to make a change pass. If the accepted
    specification changed, update the assertion and explain the behavioral transition.
  - Coverage is a diagnostic, not a target. Prioritize decision boundaries and failure
    modes over line percentage.

## SECURITY AND PRIVACY

  - Follow specs/cross-cutting.md first. Secrets must not enter logs, errors, specs,
    fixtures, snapshots, command arguments, generated dashboards, or diagnostics.
  - Use crypto/rand for security material. Never use math/rand for keys, nonces, tokens,
    session identifiers, or other security decisions.
  - Use maintained standard or golang.org/x/crypto primitives and established protocol
    constructions. Do not invent cryptography, padding, signing, key derivation, or
    constant-time comparisons.
  - Validate URLs and redirect behavior where credentials could be forwarded. Keep TLS
    verification enabled. An insecure development escape hatch must be explicit,
    isolated, off by default, documented, and prohibited for production if allowed at all.
  - Bound all attacker- or remote-controlled reads, allocations, decompression,
    recursion, concurrency, retries, and diagnostic output.
  - Defend file operations against traversal and unintended symlink/path escape when a
    path component is not fully trusted.
  - Avoid shell execution. If unavoidable, pass fixed executable and argument vectors;
    never interpolate untrusted text into a shell command.
  - Minimize privileges, writable paths, credential lifetime, and secret copies. Clear
    references when practical, while recognizing Go does not guarantee memory erasure.
  - Run govulncheck on relevant changes and before releases. A finding requires call-path
    and exposure analysis, remediation or a documented accepted risk—not blind ignoring.
  - Run the race detector for concurrency changes. Fuzz parsers and protocol boundaries.
  - Never "fix" a certificate, authentication, or authorization failure by disabling
    verification or widening permission without an explicit security requirement.

## PERFORMANCE AND RELIABILITY

  - Correctness and clarity precede optimization. Optimize from profiles, benchmarks,
    production evidence, or a clear algorithmic bound—not intuition.
  - Choose the right algorithm before micro-optimizing allocations. Document important
    complexity and input bounds.
  - Avoid repeated parsing, sorting, reflection, allocation, or network calls in hot
    loops when a measured improvement can keep ownership and invalidation simple.
  - Do not pool cheap or long-lived objects. sync.Pool is for temporary values shared
    across independent clients under measured allocation pressure; pooled values must
    be fully reset and must not escape ownership.
  - Streaming is preferred for large data, but size limits still apply. Avoid reading an
    unbounded response entirely into memory.
  - Backpressure must be explicit. Bound queues and worker counts; decide what happens
    when capacity is exhausted.
  - Reliability includes degraded behavior specified by the product: temporary cloud
    failure must not destroy retained MQTT state, optional capability failure must not
    disable unrelated outputs, and dashboard setup failure must not kill MQTT startup.
  - Never trade correctness, deterministic behavior, cancellation, or secret safety for
    an unmeasured speed improvement.

## GOSUNGROW-SPECIFIC IMPLEMENTATION RULES

These rules point implementations back to existing product contracts. The referenced
specifications remain authoritative if wording here becomes stale.

  - Preserve composite ps_id values. Do not truncate identifiers during parsing,
    filtering, sorting, logging, cache identity, or target selection.
  - Centralize iSolarCloud transport and recovery decisions. Normal endpoint paths must
    not bypass the existing recovery boundary or create nested unbounded retries.
  - Classify Docker DNS failures separately. They do not trigger host rotation or login
    refresh and follow their specified MQTT outage backoff.
  - API requests use bounded timeouts, encrypted protocol framing, exact required
    headers, secret-safe diagnostics, strict response classification, and at most the
    specified clock-skew retry/session replay.
  - Cache identity includes endpoint identity and a deterministic fingerprint of all
    endpoint-specific request data. A map's iteration order is never part of a key.
  - Persist cache, token, dashboard, and configuration replacements with the safe-write
    sequence in specs/cross-cutting.md. A cache write failure is not silent success.
  - MQTT target selection, batching, scheduling, retained publications, and retry timing
    are normative algorithms. Make tie-breakers and ordering explicit in pure helpers.
  - Home Assistant dashboard mutation must preserve administrator checks, fresh
    optimistic verification, per-target pruning, and nonfatal installation behavior.
  - Unknown API/MQTT values follow the explicit preservation/fail-closed rules. Do not
    "clean up" unknown data unless the relevant spec permits it.
  - Keep command flags and user configuration backward compatible unless a spec defines
    migration. New required configuration is a last resort.
  - User-visible lists, generated YAML/JSON, diagnostics, candidate traces, endpoint
    choices, and hashes require deterministic ordering.
  - Maintain version alignment between defaults/const.go and addon/gosungrow/config.yaml
    whenever a release change requires a version bump.

## REQUIRED VALIDATION AND DEFINITION OF DONE

Run commands from the module root. Use the toolchain version declared by go.mod.

Mandatory for every Go change:

    gofmt -w <changed-go-files>
    go test ./...

Mandatory repository validation for behavior/spec changes:

    python scripts/check_specs.py
    go test ./...
    bash -n addon/gosungrow/run.sh

Also validate the build contract when production paths, dependencies, build files, or
release behavior change:

    CGO_ENABLED=0 go build .

On PowerShell, set the environment without changing the meaning of the command:

    $env:CGO_ENABLED = "0"
    go build .

High-value checks that SHOULD run locally and SHOULD be automated in CI as the repository
modernizes its quality gates:

    go vet ./...
    go test -race ./...
    govulncheck ./...
    go mod verify

Use the race detector on a supported CGO-enabled host; it is distinct from the required
CGO-disabled production build. Run focused fuzz targets for changed parsers/protocol
boundaries for a time proportional to risk. Run the documented Docker amd64 smoke build
when container, asset, startup, architecture, or dependency behavior changes.

A change is done only when all applicable statements are true:

  [ ] Accepted specs describe the intended behavior and contain no unresolved ambiguity.
  [ ] Acceptance criteria and traceability are updated where required.
  [ ] The implementation covers success, failure, cancellation, cleanup, security,
      determinism, and compatibility—not only the happy path.
  [ ] New APIs are minimal, idiomatic, documented, and compatible with Go 1.19.
  [ ] Tests would fail if the new behavior regressed and include relevant boundaries.
  [ ] Changed files are formatted and all mandatory checks pass.
  [ ] Race, vet, vulnerability, fuzz, build, shell, and container checks were run when
      relevant, or their environment limitation is reported explicitly.
  [ ] No secret, personal payload, debug output, accidental generated file, or unrelated
      workspace change entered the diff.
  [ ] The final diff was reviewed against requirement IDs and prohibited behaviors.
  [ ] User documentation and release/version artifacts are aligned when applicable.

## REVIEW REJECTION CHECKLIST

Reject or revise a change containing any of these unless a higher-authority requirement
explicitly justifies it:

  - Behavior invented from legacy code instead of specified intent.
  - A behavior change without specs, acceptance coverage, or traceability updates.
  - New language/library features incompatible with go.mod's Go version.
  - Unformatted Go, vague names, needless exported symbols, or a vague new package.
  - An interface defined by the producer or expanded only to facilitate mocking.
  - Hidden global state, fallible init work, or dependency lookup through globals.
  - A goroutine without ownership and shutdown; a retry without classification/bounds;
    a network operation without cancellation/timeout.
  - An error ignored, string-matched across business layers, logged repeatedly, or
    stripped of semantics accidentally.
  - A response body/file/ticker/timer/connection not closed or stopped on every path.
  - Direct map-order dependence in output, selection, hashing, or tests.
  - Writes that can truncate/corrupt the prior good file on failure.
  - Unbounded remote reads, allocations, concurrency, retries, or diagnostics.
  - Credentials or sensitive payloads in errors, logs, fixtures, or documentation.
  - Tests that sleep, depend on live services, share ambient state, assert unstable text,
    update goldens silently, or pass without proving the requirement.
  - Premature abstractions, reflection, generics, functional options, factories, or
    dependency injection frameworks that make a simple flow harder to follow.
  - Opportunistic unrelated rewrites mixed into a targeted functional change.
  - A performance claim without a benchmark/profile or a security claim without a
    stated threat and verification.

## DECISION RULES FOR COMMON AMBIGUITIES

When two valid Go techniques compete, use these defaults:

  - Concrete type or interface? Accept the smallest consumer-owned interface only when
    multiple implementations or an external-effect seam exists; otherwise concrete.
  - Value or pointer? Value for small immutable values with meaningful copies; pointer
    for mutation, identity, large values, or non-copyable internals.
  - Nil or empty slice? Nil internally by default; exact form at a boundary follows the
    wire/product contract.
  - Sentinel or typed error? Sentinel for stable category only; typed when structured
    programmatic detail is required; plain wrapped error otherwise.
  - Wrap or format an error? %w when cause inspection is promised/useful; %v/new error
    when intentionally creating a new abstraction boundary.
  - Table test or separate tests? Table when assertion mechanics are uniform; separate
    when table conditionals obscure the scenario.
  - Method or function? Method when behavior belongs to the receiver's cohesive domain;
    function when inputs are peers or no state/identity is owned.
  - Add a package or a file? Package only for a coherent dependency boundary; file for
    organization within the same cohesive package.
  - Cache or recompute? Recompute until evidence and a sound invalidation/ownership model
    justify caching.
  - Parallelize or stay sequential? Sequential until latency/throughput requirements and
    bounded ownership justify concurrency.
  - Refactor or patch? Make the smallest complete clear change; refactor first only when
    the current structure prevents a safe implementation, and preserve behavior with
    tests.

## AUTHORITATIVE RESEARCH BASELINE

These sources informed this standard. Consult the version applicable to go.mod whenever
newer documentation describes unavailable APIs. Repository specifications override all
style advice on product behavior.

Primary Go sources:

  - Go language specification: https://go.dev/ref/spec
  - Go memory model: https://go.dev/ref/mem
  - Effective Go: https://go.dev/doc/effective_go
    Note: still foundational, but officially not actively updated for newer features.
  - Go Code Review Comments: https://go.dev/wiki/CodeReviewComments
  - Go Test Comments: https://go.dev/wiki/TestComments
  - Go doc comments: https://go.dev/doc/comment
  - Context package contract: https://pkg.go.dev/context
  - Errors package contract: https://pkg.go.dev/errors
  - HTTP package contract: https://pkg.go.dev/net/http
  - Go modules reference: https://go.dev/ref/mod
  - Module compatibility: https://go.dev/blog/module-compatibility
  - Go security best practices: https://go.dev/doc/security/best-practices
  - Race detector: https://go.dev/doc/articles/race_detector
  - Fuzzing: https://go.dev/doc/security/fuzz
  - govulncheck: https://go.dev/doc/security/vuln/
  - Defer, panic, and recover: https://go.dev/blog/defer-panic-and-recover
  - Errors are values: https://go.dev/blog/errors-are-values
  - Package naming: https://go.dev/blog/package-names

Detailed readability guidance:

  - Google Go Style Guide overview: https://google.github.io/styleguide/go/
  - Canonical guide: https://google.github.io/styleguide/go/guide
  - Style decisions: https://google.github.io/styleguide/go/decisions
  - Best practices: https://google.github.io/styleguide/go/best-practices

Do not cargo-cult any external guide. Apply its principles through this repository's
spec-first governance, supported Go version, operational constraints, and existing public
contracts. When guidance evolves, update this file deliberately, with a research date
and a review for contradictions.
