# ADR-002: MQTT discovery integration

Status: Accepted  
Date: 2026-09-13

## Decision

GoSungrow integrates with Home Assistant through MQTT discovery rather than a native integration.

## Consequences

An MQTT broker and Home Assistant MQTT integration are prerequisites. Stable discovery IDs and retained states are compatibility contracts. Remote iSolarCloud outages should not tear down the broker connection or erase retained readings.
