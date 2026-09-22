import assert from "node:assert/strict";

const baseURL = (process.env.HA_E2E_URL || "http://127.0.0.1:8123").replace(/\/$/, "");
const resourceURL = process.env.HA_E2E_RESOURCE_URL;
const expectedPhase = process.env.HA_E2E_BOOTSTRAP_PHASE;

assert(resourceURL, "HA_E2E_RESOURCE_URL is required");
assert(["unavailable", "available"].includes(expectedPhase), "HA_E2E_BOOTSTRAP_PHASE must be unavailable or available");

const response = await fetch(`${baseURL}${resourceURL}`, { cache: "reload" });
if (expectedPhase === "unavailable") {
  assert.equal(response.status, 404, "asset staged after Core startup must demonstrate the unregistered /local route");
} else {
  assert.equal(response.status, 200, "Core restart must register /local and expose the previously staged asset");
}
await response.arrayBuffer();

console.log(`Home Assistant /local bootstrap phase ${expectedPhase} passed`);
