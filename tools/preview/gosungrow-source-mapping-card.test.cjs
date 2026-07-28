const assert = require("node:assert/strict");
const test = require("node:test");

class FakeShadowRoot {
  constructor() { this.innerHTML = ""; this.activeElement = null; }
  querySelector() { return null; }
  querySelectorAll() { return []; }
}

global.HTMLElement = class {
  attachShadow() { this.shadowRoot = new FakeShadowRoot(); return this.shadowRoot; }
  dispatchEvent() { return true; }
};
global.CustomEvent = class { constructor(type, init) { this.type = type; this.detail = init?.detail; } };
global.window = { customCards: [], confirm: () => true };
const registry = new Map();
global.customElements = {
  define: (name, implementation) => registry.set(name, implementation),
  get: (name) => registry.get(name),
};

require("../../addon/gosungrow/assets/gosungrow-energy-flow-card-v2.js");
const MappingCard = registry.get("gosungrow-source-mapping-card-v1");

function state(value, unit = "kWh", name = "Entity") {
  return { state: String(value), attributes: { unit_of_measurement: unit, friendly_name: name }, last_updated: new Date().toISOString() };
}

function fixture() {
  const oldEntity = "sensor.gosungrow_100_11_0_0_p13112";
  const newEntity = "sensor.renamed_production";
  const metric = { key: "p13112", group: "today_energy", label: "Solar production today", default: oldEntity, confidence: "low", reason: "Current automatic match", needs_review: true, recommendation: newEntity };
  const candidates = [
    { entity_id: oldEntity, recommended: true, confidence: "low", reason: "Current automatic match" },
    { entity_id: newEntity, recommended: true, confidence: "high", reason: "Canonical Sungrow point", canonical_point: "p13112", provenance: "native", compatibility: "compatible" },
    { entity_id: "sensor.advanced_production", recommended: false, confidence: "medium", reason: "Compatible advanced result" },
  ];
  const config = {
    schema_version: 1, mapping_id: "source-100", dashboard_url_path: "gosungrow-flow",
    defaults: { p13112: oldEntity }, pinned_defaults: { p13112: oldEntity }, overrides: {}, recommendations: { p13112: newEntity },
    bindings: { p13112: ["/views/0/cards/0/entity"] }, metrics: [metric], candidates: { p13112: candidates }, labels: {},
  };
  const dashboard = { views: [{ cards: [{ entity: oldEntity }] }, { cards: [{ type: "custom:gosungrow-source-mapping-card-v1", ...structuredClone(config) }] }] };
  return { oldEntity, newEntity, metric, candidates, config, dashboard };
}

test("live hass rerenders preserve expanded candidates and pending dialog state", () => {
  const { config, metric, oldEntity, newEntity } = fixture();
  const card = new MappingCard();
  card.setConfig(config);
  card._activeMetric = metric;
  card._pendingEntity = newEntity;
  card._search = "production";
  card._expandedOther.add("p13112");
  card._dialogScroll = 240;
  card.hass = { user: { is_admin: true }, states: { [oldEntity]: state(4), [newEntity]: state(5) } };
  assert.equal(card._activeMetric, metric);
  assert.equal(card._pendingEntity, newEntity);
  assert.equal(card._search, "production");
  assert.equal(card._dialogScroll, 240);
  assert.match(card.shadowRoot.innerHTML, /data-other-metric="p13112" open/);
});

test("dialog capture and restore keeps search focus, selection, and scroll", () => {
  const { config, metric } = fixture();
  const card = new MappingCard();
  card.setConfig(config);
  card._activeMetric = metric;
  const search = {
    selectionStart: 2, selectionEnd: 7, focused: false,
    matches: (selector) => selector === "input[type=search]",
    focus() { this.focused = true; },
    setSelectionRange(start, end) { this.restored = [start, end]; },
  };
  const scroller = { scrollTop: 184 };
  card.shadowRoot.activeElement = search;
  card.shadowRoot.querySelector = (selector) => selector === ".candidate-scroll" ? scroller : selector === "input[type=search]" ? search : null;
  card._captureDialogState();
  scroller.scrollTop = 0;
  card._restoreDialogState();
  assert.equal(scroller.scrollTop, 184);
  assert.equal(search.focused, true);
  assert.deepEqual(search.restored, [2, 7]);
});

test("legacy calculated sources are blocked and native-unavailable is explicit", () => {
  const { config } = fixture();
  const card = new MappingCard();
  card.setConfig(config);
  card._hass = { user: { is_admin: true }, states: { "sensor.calculated": state(10) } };
  const legacy = { key: "p13116", default: "sensor.calculated", unsupported_calculated: true };
  assert.equal(card._warning(legacy), "Unsupported calculated source");
  card._hass.states["sensor.native"] = state(8);
  assert.equal(card._warning(legacy, "sensor.native"), "");
  assert.match(card._candidate(legacy, { entity_id: "sensor.calculated", selectable: false }, []), / disabled>/);
  assert.equal(card._warning({ key: "p13116", default: "", native_unavailable: true }), "Native source unavailable");
});

test("accepting a recommendation atomically promotes the pinned automatic source", async () => {
  const { config, dashboard, oldEntity, newEntity } = fixture();
  let stored = structuredClone(dashboard);
  const card = new MappingCard();
  card.setConfig(structuredClone(config));
  card._hass = {
    user: { is_admin: true }, states: { [oldEntity]: state(4), [newEntity]: state(5) },
    callWS: async (request) => {
      if (request.type === "lovelace/config") return structuredClone(stored);
      if (request.type === "lovelace/config/save") { stored = structuredClone(request.config); return {}; }
      throw new Error(`unexpected request ${request.type}`);
    },
  };
  await card._save("p13112", newEntity);
  const storedCard = card._findMappingCard(stored, "source-100");
  assert.equal(stored.views[0].cards[0].entity, newEntity);
  assert.equal(storedCard.defaults.p13112, newEntity);
  assert.equal(storedCard.pinned_defaults.p13112, newEntity);
  assert.equal(storedCard.overrides.p13112, undefined);
  assert.equal(card._config.defaults.p13112, newEntity);
});

test("save failure leaves local mapping unchanged", async () => {
  const { config, dashboard, oldEntity, newEntity } = fixture();
  const card = new MappingCard();
  card.setConfig(structuredClone(config));
  card._hass = {
    user: { is_admin: true }, states: { [oldEntity]: state(4), [newEntity]: state(5) },
    callWS: async (request) => {
      if (request.type === "lovelace/config") return structuredClone(dashboard);
      throw new Error("save rejected");
    },
  };
  await card._save("p13112", newEntity);
  assert.equal(card._config.defaults.p13112, oldEntity);
  assert.deepEqual(card._config.overrides, {});
  assert.match(card._notice, /save rejected/);
});
