---
name: plan-issue-learnings
description: Turn a resolved or investigated GoSungrow issue into a generalized, specs-only improvement plan for diagnostics, observability, recovery, prevention, and acceptance coverage. Use after considering an issue or incident when the user asks what should change so similar problems are easier to diagnose or prevent. Invoke plan-spec-change for the final approval packet; never inspect implementation or modify files.
---

# Plan Issue Learnings

Convert the concrete issue into the smallest reusable product-contract improvement that would make the next similar failure easier to identify, explain, recover from, or prevent. Finish by invoking `$plan-spec-change`; do not duplicate or weaken its approval and drift controls.

## Evidence boundary

- Use issue evidence already present in the conversation or explicitly supplied by the user, including symptoms, logs, diagnosis, resolution, and remaining uncertainty.
- Clearly separate observed facts, supported inferences, and unknowns. Do not encode an unconfirmed root cause as a requirement.
- For repository product logic, read only `AGENTS.md` and files under `specs/`. Do not inspect source, tests, configuration, manifests, workflows, scripts, generated assets, runtime files, GitHub issues, or external systems.
- Use read-only operations exclusively. Never modify files or external state, and do not run validators, builds, tests, generators, or scripts.
- If the available issue evidence is too thin to identify a safe general lesson, ask one concise question instead of inventing one.

## Generalize the learning

Consider only dimensions supported by the issue:

- whether the user-visible symptom preserved the primary failure and causal chain;
- whether diagnostics identify the failing stage, attempted target, failure class, retry decision, and useful next action;
- whether logs remain concise, redact secrets, and distinguish application, dependency, network, authentication, configuration, and recovery failures;
- whether fallback and retry behavior preserves earlier, more informative errors and terminates or degrades predictably;
- whether acceptance scenarios cover the happy path, the observed failure, adjacent failure classes, boundaries, regressions, and prohibited misleading outcomes;
- whether user-facing recovery guidance is required by the product contract.

Prefer a general invariant over an issue-specific patch. Keep endpoint names, exact messages, retry counts, or special cases only when they are intentional domain behavior. Avoid turning one incident into a broad logging framework or an unrelated reliability project.

Rank candidate improvements by recurrence risk, diagnostic value, user impact, and specification fit. Separate the cohesive change worth planning now from optional follow-ups and non-goals. "More tests" means stronger normative acceptance scenarios and an implementation envelope from which implementation tests will later be derived; never inspect or prescribe individual test files here.

## Hand off to the specification planner

Synthesize one concise requested behavior containing:

1. the generalized failure pattern and evidence;
2. the desired observable diagnostic, recovery, or prevention behavior;
3. acceptance cases and prohibited misleading outcomes;
4. compatibility, redaction, and non-goal constraints;
5. uncertainties that must remain undecided.

Then invoke `$plan-spec-change` with that synthesized request and follow its `SKILL.md` exactly. The final response must be its drift-safe approval packet, preceded only by a short issue-learning rationale and any optional follow-ups that were deliberately excluded.

If the learnings contain independent product decisions that should not share one approval or pull request, do not bundle them. Ask the user which one to plan first, or produce separate `$plan-spec-change` packets only when the user explicitly requests multiple plans.
