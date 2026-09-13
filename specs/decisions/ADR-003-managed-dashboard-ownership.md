# ADR-003: Managed dashboard ownership

Status: Accepted  
Date: 2026-09-13

## Decision

The app owns a storage-mode dashboard identified by URL path, while administrators may safely control metric sources through a constrained mapping card.

## Consequences

Full and source-insensitive structure hashes protect user edits. Source overrides persist separately by target and metric. Arbitrary dashboard edits are not overwritten without force; source-only edits made through the card survive reconciliation.
