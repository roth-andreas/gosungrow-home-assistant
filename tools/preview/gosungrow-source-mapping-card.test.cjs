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
  const policy = { entity_id: oldEntity, selection_origin: "automatic", verification: "unverified", display_decision: "withheld", reason: "Source meaning could not be verified" };
  const metric = { key: "p13112", group: "today_energy", label: "Solar production today", default: oldEntity, confidence: "low", ...policy, validation: { schema_version: 2, rules: [] }, needs_review: true, recommendation: newEntity };
  const candidates = [
    { entity_id: oldEntity, recommended: true, confidence: "low", reason: "Current automatic match" },
    { entity_id: newEntity, recommended: true, confidence: "high", reason: "Canonical Sungrow point", canonical_point: "p13112", provenance: "native", compatibility: "compatible", verification: "verified" },
    { entity_id: "sensor.advanced_production", recommended: false, confidence: "medium", reason: "Compatible advanced result" },
  ];
  const config = {
    schema_version: 1, matcher_version: 5, mapping_id: "source-100", dashboard_url_path: "gosungrow-flow",
    defaults: { p13112: oldEntity }, pinned_defaults: { p13112: oldEntity }, overrides: {}, recommendations: { p13112: newEntity },
    bindings: { p13112: ["/views/0/cards/0/entity", "/views/0/cards/0/source_policy/metrics/p13112/entity_id"] }, metrics: [metric], candidates: { p13112: candidates }, labels: {},
  };
  const dashboard = { views: [{ cards: [{ entity: oldEntity, source_policy: { mapping_id: config.mapping_id, metrics: { p13112: structuredClone(policy) } } }] }, { cards: [{ type: "custom:gosungrow-source-mapping-card-v1", ...structuredClone(config) }] }] };
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

test("safer automatic recommendation is visibly highlighted and explained", () => {
  const { config, metric, oldEntity, newEntity } = fixture();
  config.candidates.p13112.find((candidate) => candidate.entity_id === newEntity).recommended = false;
  const card = new MappingCard();
  card.setConfig(config);
  card._hass = { user: { is_admin: true }, states: { [oldEntity]: state(4), [newEntity]: state(5) } };
  card._activeMetric = metric;
  card._pendingEntity = oldEntity;
  const dialog = card._dialog(metric);
  assert.match(dialog, /class="recommendation-help"/);
  assert.match(dialog, /Select the highlighted source below, then choose Use this source\./);
  assert.match(dialog, new RegExp(`class="candidate recommended-update [^"]*"[^>]+data-select="${newEntity}"`));
  assert.match(dialog, /class="recommendation-badge"/);
  assert.ok(dialog.indexOf(`data-select="${newEntity}"`) < dialog.indexOf(`data-select="${oldEntity}"`), "recommended replacement should be listed first");
  assert.match(dialog, new RegExp(`aria-label="Recommended automatic source; [^"]+; ${newEntity.replaceAll(".", "\\.")};`));
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

test("matcher drift aborts selection and reset before any save", async () => {
  for (const [localVersion, freshVersion] of [[3, 4], [4, 3], [undefined, 4], [4, undefined]]) {
    for (const reset of [false, true]) {
      const { config, dashboard, oldEntity, newEntity } = fixture();
      config.matcher_version = localVersion;
      dashboard.views[1].cards[0].matcher_version = freshVersion;
      const calls = [];
      const card = new MappingCard();
      card.setConfig(structuredClone(config));
      const before = structuredClone(card._config);
      card._hass = {
        user: { is_admin: true }, states: { [oldEntity]: state(4), [newEntity]: state(5) },
        callWS: async (request) => {
          calls.push(request.type);
          if (request.type === "lovelace/config") return structuredClone(dashboard);
          throw new Error("unexpected mutation after matcher drift");
        },
      };
      await card._save("p13112", reset ? "" : newEntity);
      assert.deepEqual(calls, ["lovelace/config"]);
      assert.deepEqual(card._config, before);
      assert.match(card._notice, /Dashboard changed/);
    }
  }
});

function nativeFixture(kind) {
  const f = fixture();
  const projector = new MappingCard();
  const source = {
    mapping_id: f.config.mapping_id,
    metrics: { p13112: projector._policyFields(f.metric) },
    native_template: { type: kind, title: "Preserved title", layout_options: { grid_columns: 6 } },
    native_bindings: { p13112: ["/entity"] },
  };
  if (kind === "tile") source.native_template.entity = f.oldEntity;
  else {
    const other = { key: "p13199", default: f.oldEntity, entity_id: f.oldEntity, selection_origin: "automatic", verification: "verified", display_decision: "normal", reason: "Source meaning verified", validation: { schema_version: 2, rules: [] } };
    f.config.metrics.push(other);
    f.config.defaults.p13199 = f.oldEntity;
    f.config.pinned_defaults.p13199 = f.oldEntity;
    source.metrics.p13199 = projector._policyFields(other);
    source.native_template.hours_to_show = 24;
    source.native_template.entities = [{ entity: f.oldEntity, name: "Production" }, { entity: f.oldEntity, name: "Consumption" }];
    source.native_bindings = { p13112: ["/entities/0/entity"], p13199: ["/entities/1/entity"] };
  }
  const rendered = projector._projectNative(source);
  f.config.bindings = Object.fromEntries(Object.entries(rendered.bindings).map(([metric, paths]) => [metric, paths.map((path) => "/views/0/cards/0" + path)]));
  f.dashboard.views[0].cards[0] = rendered.card;
  f.dashboard.views[1].cards[0] = { type: "custom:gosungrow-source-mapping-card-v1", ...structuredClone(f.config) };
  return f;
}

test("native tile and graph adoption restore only the owned metric and regenerate bindings atomically", async () => {
  for (const kind of ["tile", "history-graph"]) {
    const { config, dashboard, oldEntity, newEntity } = nativeFixture(kind);
    let stored = structuredClone(dashboard);
    const card = new MappingCard();
    card.setConfig(config);
    card._hass = { user: { is_admin: true }, states: { [oldEntity]: state(4), [newEntity]: state(5) }, callWS: async (request) => {
      if (request.type === "lovelace/config") return structuredClone(stored);
      stored = structuredClone(request.config);
    } };
    await card._save("p13112", newEntity);
    assert.equal(card._config.defaults.p13112, newEntity, card._notice);
    const native = stored.views[0].cards[0];
    assert.equal(native.type, kind);
    assert.equal(native.title, "Preserved title");
    assert.deepEqual(native.layout_options, { grid_columns: 6 });
    if (kind === "history-graph") {
      assert.deepEqual(native.entities, [{ entity: newEntity, name: "Production" }, { entity: oldEntity, name: "Consumption" }]);
      assert.equal(native.hours_to_show, 24);
      assert.equal(native.source_policy.metrics.p13199.entity_id, oldEntity);
      assert.ok(card._config.bindings.p13199.includes("/views/0/cards/0/entities/1/entity"));
    } else assert.equal(native.entity, newEntity);
    for (const path of card._config.bindings.p13112) assert.equal(card._getPointer(stored, path), newEntity, path);
    assert.equal(native.source_policy.metrics.p13112.verification, "verified");
    assert.equal(native.source_policy.metrics.p13112.selection_origin, "automatic");
    assert.equal(native.source_policy.metrics.p13112.display_decision, "normal");
    assert.equal(JSON.stringify(stored).includes('"type":"conditional"'), false);
  }
});

test("native stale, missing, cross-card and multiply-owned templates cannot save", async () => {
  for (const corrupt of [
    (source) => { delete source.native_template; },
    (source) => { delete source.native_bindings; },
    (source) => { source.native_bindings.p13112 = ["/views/1/entity"]; },
    (source) => { source.native_bindings.p13112 = ["/entities/1/entity"]; },
    (source) => { source.native_template.entities[0].entity = "sensor.another_target"; },
    (source) => { source.native_template.source_policy = {}; },
  ]) {
    const { config, dashboard, oldEntity, newEntity } = nativeFixture("history-graph");
    corrupt(dashboard.views[0].cards[0].source_policy);
    const card = new MappingCard();
    card.setConfig(config);
    const before = structuredClone(card._config);
    let saves = 0;
    card._hass = { user: { is_admin: true }, states: { [oldEntity]: state(4), [newEntity]: state(5) }, callWS: async (request) => {
      if (request.type === "lovelace/config") return structuredClone(dashboard);
      saves++;
    } };
    await card._save("p13112", newEntity);
    assert.equal(saves, 0);
    assert.deepEqual(card._config, before);
    assert.match(card._notice, /Dashboard changed/);
  }
});

test("policy migration and live eligibility are checked again before selection", async () => {
  for (const mode of ["old_validation", "policy_drift", "missing_verification", "unavailable", "unit_changed", "stale"]) {
    const { config, dashboard, oldEntity, newEntity } = fixture();
    if (mode === "old_validation") config.metrics[0].validation.schema_version = dashboard.views[1].cards[0].metrics[0].validation.schema_version = 1;
    if (mode === "policy_drift") dashboard.views[1].cards[0].metrics[0].verification = "verified";
    if (mode === "missing_verification") delete dashboard.views[1].cards[0].metrics[0].verification;
    const live = state(mode === "unavailable" ? "unavailable" : 5, mode === "unit_changed" ? "W" : "kWh");
    if (mode === "stale") live.last_updated = new Date(Date.now() - 37 * 3600000).toISOString();
    const card = new MappingCard();
    card.setConfig(config);
    const before = structuredClone(card._config);
    let saves = 0;
    card._hass = { user: { is_admin: true }, states: { [oldEntity]: state(4), [newEntity]: live }, callWS: async (request) => {
      if (request.type === "lovelace/config") return structuredClone(dashboard);
      saves++;
    } };
    await card._save("p13112", newEntity);
    assert.equal(saves, 0, mode);
    assert.deepEqual(card._config, before, mode);
  }
});

test("a failed native save retains both saved projection and local mapping", async () => {
  const { config, dashboard, oldEntity, newEntity } = nativeFixture("history-graph");
  const before = structuredClone(dashboard);
  const card = new MappingCard();
  card.setConfig(config);
  card._hass = { user: { is_admin: true }, states: { [oldEntity]: state(4), [newEntity]: state(5) }, callWS: async (request) => {
    if (request.type === "lovelace/config") return structuredClone(dashboard);
    throw new Error("save rejected");
  } };
  await card._save("p13112", newEntity);
  assert.deepEqual(dashboard, before);
  assert.deepEqual(card._config, config);
  assert.match(card._notice, /save rejected/);
});

test("bindings cannot cross targets or metric ownership even when entities are identical", async () => {
  for (const mode of ["other_target", "other_metric", "missing_owned_path"]) {
    const { config, dashboard, oldEntity, newEntity } = nativeFixture("history-graph");
    if (mode === "other_target") {
      const other = structuredClone(dashboard.views[0].cards[0]);
      other.source_policy.mapping_id = "another-target";
      dashboard.views[0].cards.push(other);
      config.bindings.p13112.push("/views/0/cards/1/source_policy/metrics/p13112/entity_id");
    } else if (mode === "other_metric") {
      config.bindings.p13112.push("/views/0/cards/0/source_policy/metrics/p13199/entity_id");
    } else config.bindings.p13112 = config.bindings.p13112.slice(1);
    dashboard.views[1].cards[0] = { type: "custom:gosungrow-source-mapping-card-v1", ...structuredClone(config) };
    const card = new MappingCard();
    card.setConfig(config);
    let saves = 0;
    card._hass = { user: { is_admin: true }, states: { [oldEntity]: state(4), [newEntity]: state(5) }, callWS: async (request) => {
      if (request.type === "lovelace/config") return structuredClone(dashboard);
      saves++;
    } };
    await card._save("p13112", newEntity);
    assert.equal(saves, 0, mode);
    assert.deepEqual(card._config, config, mode);
    assert.match(card._notice, /Dashboard changed/, mode);
  }
});

test("a concurrent dashboard edit during source preparation aborts the save", async () => {
  const { config, dashboard, oldEntity, newEntity } = fixture();
  const card = new MappingCard();
  card.setConfig(config);
  let reads = 0, saves = 0;
  card._hass = { user: { is_admin: true }, states: { [oldEntity]: state(4), [newEntity]: state(5) }, callWS: async (request) => {
    if (request.type === "lovelace/config") {
      const current = structuredClone(dashboard);
      if (++reads > 1) current.views[0].title = "Concurrent user edit";
      return current;
    }
    saves++;
  } };
  await card._save("p13112", newEntity);
  assert.equal(saves, 0);
  assert.deepEqual(card._config, config);
  assert.match(card._notice, /Dashboard changed/);
});
