# Plant PV aggregation verification

Informative verification report for version 3.3.5, 2026-10-04. Product behavior is defined in specs/, not this report.

Implementation was prepared on `spec-change/preserve-causal-failure-diagnostics`, based on `066c54d119cbabe95875e13a7fb1b2a6c7275464`. The app and binary versions are aligned at 3.3.5. No reporter's live installation was accessed.

## Regression evidence

Synthetic API responses exercise decoding and normalization before aggregation. Coverage includes type-55 AC power, device-local copies, original IDs despite legacy decoder rewriting, copied native totals, precedence, null/invalid/zero handling, conflicting copies, independently discovered membership, AC/DC completeness, row-count independence, intermediate non-producing nodes, cycles, and plant-local UUIDs.

Publication tests use the actual retained discovery/state serializers with a supplied MQTT transport. They verify completion, config/state failures, timeout, filters, unknown parents, retry, retained-state preservation, plant-device attachment, and one final summary. The nine-microinverter fixture produces 1.8 kW through the dashboard template and source-mapping pipeline, rejects signed grid phases, counts nine inverter-like devices, and preserves existing pinned/manual choices.

These are regression fixtures, not a claim that the reporter's live plant has been verified.

## Checks and limitations

Environment: Windows/amd64, installed Go 1.26.0; go.mod remains at Go 1.19. No dependencies changed. The following implementation checks preceded the release-version bump; clean-snapshot release checks are recorded separately below.

| Check | Result |
|---|---|
| Repository Go packages | Passed after excluding only the ignored local investigation package named below |
| `go test ./...` | Blocked by ignored `notgit/issue-26-investigation/issue26_test.go`, which references undefined `cmds`, `CmdApi`, `NewCmdMqtt`, and `MqttEndPoints` |
| `python scripts/check_specs.py` | Existing local blocker: untracked `.agents/skills/research-issue/` lacks `agents/openai.yaml`; no other errors reported |
| Both required Node.js test suites | Passed: 6 source-mapping tests and 41 energy/card tests |
| `bash -n addon/gosungrow/run.sh` | Passed |
| CGO-disabled production build | Passed; binary written to an isolated system temporary directory, not the repository |
| `go mod verify` | Passed |
| Float JSON fuzzing | Passed, 15 seconds, more than one million executions |
| Query-point JSON/provenance fuzzing | Passed, 10-second target, 383,020 executions |
| `go vet` on repository packages | Existing warnings: unexported JSON-tagged fields in `valueTypes/datapoint.go` and lock-by-value `cmdHassio.New`; no newly introduced warnings remain |
| Race detector | Unavailable: CGO is disabled and no C compiler is present |
| Vulnerability scanner | Could not complete dependency downloads: repeated verified-TLS `bad record MAC` errors from the Go module proxy; TLS verification was not disabled |
| Container/browser release smoke | Not run locally: Docker daemon is unavailable; release CI requires both before image publication |

The unrelated skill edits and ignored investigation files were left untouched.

## Clean-snapshot release checks

An isolated detached worktree containing exactly the staged release files, without the unrelated local skill or ignored investigation, passed:

- `python scripts/check_specs.py` (17 normative documents, 357 requirements).
- Unfiltered `go test ./...`, a CGO-disabled production build, and `go mod verify`.
- Both required JavaScript suites (6 source-mapping and 41 energy/card tests).
- Startup/configuration/recovery shell syntax and both configuration/recovery policy test suites.

Local Docker remains unavailable. CI gates image publication on the container build and pinned Home Assistant browser/lifecycle smoke checks; their completion must be checked before describing this version as published.
