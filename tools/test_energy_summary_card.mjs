import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import vm from "node:vm";
import { fileURLToPath } from "node:url";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const assetPath = path.join(repoRoot, "addon", "gosungrow", "assets", "gosungrow-energy-flow-card-v2.js");
const registry = new Map();
const sandbox = {
  console,
  CustomEvent: class CustomEvent { constructor(type, options = {}) { this.type = type; this.detail = options.detail; this.bubbles = Boolean(options.bubbles); this.composed = Boolean(options.composed); } },
  HTMLElement: class HTMLElement {
    attachShadow() {
      this.shadowRoot = {
        innerHTML: "",
        querySelectorAll: () => [],
        querySelector: () => null,
      };
      return this.shadowRoot;
    }
    dispatchEvent() { return true; }
  },
  Intl,
  Math,
  Number,
  Object,
  Promise,
  queueMicrotask,
  String,
  URLSearchParams,
  customElements: {
    define: (name, elementClass) => registry.set(name, elementClass),
    get: (name) => registry.get(name),
  },
  navigator: { language: "en-US" },
  window: { customCards: [], confirm: () => true, location: { reload: () => {} } },
};

vm.createContext(sandbox);
const bundleSource = fs.readFileSync(assetPath, "utf8");
vm.runInContext(bundleSource, sandbox, { filename: assetPath });

const SummaryCard = registry.get("gosungrow-energy-summary-card-v1");
const SourceCard = registry.get("gosungrow-source-mapping-card-v1");
const FlowCard = registry.get("gosungrow-energy-flow-card-v2");
const entityID = "sensor.gosungrow_daily_production";
const metricEntities = {
  production: "sensor.gosungrow_daily_production",
  consumption: "sensor.gosungrow_daily_consumption",
  to_grid: "sensor.gosungrow_daily_to_grid",
  from_grid: "sensor.gosungrow_daily_from_grid",
  to_battery: "sensor.gosungrow_daily_to_battery",
  from_battery: "sensor.gosungrow_daily_from_battery",
};
const summaryMetricKeys = { production: "p13112", consumption: "p13199", to_grid: "p13173", from_grid: "p13147", to_battery: "p13174", from_battery: "p13029" };
function verifiedPolicy(entities) {
  return { mapping_id: "source-test", metrics: Object.fromEntries(Object.entries(entities).map(([key, entity_id]) => [summaryMetricKeys[key], { entity_id, selection_origin: "automatic", verification: "verified", display_decision: "normal", reason: "Source meaning verified" }])) };
}

test("bundle registration is synchronous and idempotent across compatible upgrades", () => {
  assert.deepEqual([...registry.keys()].sort(), [
    "gosungrow-energy-flow-card-v2",
    "gosungrow-energy-summary-card-v1",
    "gosungrow-source-mapping-card-v1",
  ]);
  assert.deepEqual(sandbox.window.customCards.map((entry) => entry.type).sort(), [...registry.keys()].sort());

  const upgradeSandbox = { ...sandbox };
  vm.createContext(upgradeSandbox);
  vm.runInContext(bundleSource, upgradeSandbox, { filename: assetPath });
  assert.equal(registry.size, 3);
  assert.equal(sandbox.window.customCards.length, 3);
});

function createCard(now, liveValue) {
  const card = new SummaryCard();
  card._config = {
    buckets: { day: 14, month: 12, year: 5 },
    entities: { production: entityID },
    source_policy: verifiedPolicy({ production: entityID }),
  };
  card._labels = {};
  card._hass = {
    locale: { language: "en-US" },
    states: Number.isFinite(liveValue)
      ? {
          [entityID]: {
            state: String(liveValue),
            attributes: { unit_of_measurement: "kWh" },
          },
        }
      : {},
  };
  card._now = () => new Date(now);
  return card;
}

function createMultiMetricCard(now, liveValues) {
  const card = new SummaryCard();
  card._config = {
    buckets: { day: 14, month: 12, year: 5 },
    entities: metricEntities,
    source_policy: verifiedPolicy(metricEntities),
  };
  card._labels = {};
  card._hass = {
    locale: { language: "en-US" },
    states: Object.fromEntries(
      Object.entries(liveValues)
        .filter(([, value]) => Number.isFinite(value))
        .map(([key, value]) => [
          metricEntities[key],
          {
            state: String(value),
            attributes: { unit_of_measurement: "kWh" },
          },
        ]),
    ),
  };
  card._now = () => new Date(now);
  return card;
}

function stateRow(date, state) {
  return { start: `${date}T00:00:00.000Z`, state };
}

test("automatic unverified daily values are withheld from totals, history and tooltips", async () => {
  const card = createMultiMetricCard("2026-05-17T12:00:00Z", { production: 4, consumption: 9.92 });
  const policy = card._config.source_policy.metrics.p13199;
  Object.assign(policy, { verification: "unverified", display_decision: "withheld", reason: "Daily meaning not verified" });
  card._hass.states[metricEntities.consumption].attributes.unit_of_measurement = "MWh";
  for (const period of ["day", "month", "year"]) {
    card._statsCache[card._cacheKey(period)] = card._parseStatistics({ [metricEntities.production]: [stateRow("2026-05-16", 3)], [metricEntities.consumption]: [stateRow("2026-05-16", 1234)] }, Object.values(metricEntities), period);
    const metrics = card._metricDefinitions();
    const consumption = card._metricDisplay(metrics.find((metric) => metric.key === "consumption"), period);
    assert.equal(consumption.available, false);
    assert.match(consumption.warning, /Review Data Sources/);
    const chart = card._chartDisplay(period, metrics);
    assert.equal(chart.series.some((series) => series.key === "consumption"), false);
    assert.equal(card._tooltipRows(chart, 0).some((row) => row.label === "Consumption"), false);
    assert.equal(card._metricDisplay(metrics.find((metric) => metric.key === "production"), period).available, true);
  }
  const requested = [];
  card._statsCache = {};
  card._hass.callWS = async (request) => { requested.push(request); return {}; };
  card._ensureStatistics();
  await Promise.resolve();
  assert.equal(requested.length, 1);
  assert.equal(requested[0].statistic_ids.includes(metricEntities.consumption), false);
});

test("manual unverified daily sources retain warnings, while incompatible ones remain withheld", () => {
  const card = createCard("2026-05-17T12:00:00Z", 0);
  const policy = card._config.source_policy.metrics.p13112;
  Object.assign(policy, { selection_origin: "manual", verification: "unverified", display_decision: "warned" });
  const metric = card._metricDefinitions()[0];
  assert.equal(card._metricDisplay(metric, "day").available, true);
  assert.match(card._metricDisplay(metric, "day").warning, /Unverified manual source/);
  const chart = card._liveChartDisplay("day", [metric]);
  assert.match(chart.series[0].label, /Unverified manual source/);
  assert.match(card._tooltipRows(chart, 0)[0].label, /Unverified manual source/);
  Object.assign(policy, { verification: "incompatible", display_decision: "withheld" });
  assert.equal(card._metricDisplay(metric, "day").available, false);
  assert.equal(card._chartDisplay("day", [metric]).series.length, 0);
  delete card._config.source_policy;
  assert.equal(card._metricDisplay(metric, "day").available, false, "legacy metadata must not imply verified meaning");
});

test("source changes discard stale recorder responses and capture the requested period", async () => {
  const card = new SummaryCard();
  card._now = () => new Date("2026-05-17T12:00:00Z");
  card.setConfig({ entities: { production: entityID }, source_policy: verifiedPolicy({ production: entityID }) });
  const resolvers = [];
  card._hass = { states: {}, callWS: () => new Promise((resolve) => resolvers.push(resolve)) };
  card._period = "month";
  card._ensureStatistics();
  const oldKey = card._cacheKey("month");
  const replacement = "sensor.replacement_daily";
  card.setConfig({ entities: { production: replacement }, source_policy: verifiedPolicy({ production: replacement }) });
  card._period = "year";
  card._ensureStatistics();
  const yearKey = card._cacheKey("year");
  card._period = "day";
  resolvers[0]({ [entityID]: [stateRow("2026-05-16", 999)] });
  resolvers[1]({ [replacement]: [stateRow("2026-05-16", 2)] });
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(card._statsCache[oldKey], undefined);
  assert.ok(card._statsCache[yearKey]);
  assert.ok(card._statsCache[yearKey].buckets.some((bucket) => bucket.key === "2026" && bucket.values[replacement] === 2));
  assert.equal(JSON.stringify(card._statsCache).includes("999"), false);
});

test("partial directional flows use only a verified direct fallback, including genuine zero", () => {
  const card = new FlowCard();
  card._config = { entities: { solar_power: "sensor.plant_pv", pv_to_load_power: "sensor.one_flow" }, source_policy: { mapping_id: "source-test", metrics: { pv_power: { entity_id: "sensor.plant_pv", selection_origin: "automatic", verification: "verified", display_decision: "normal", reason: "Source meaning verified" } } } };
  // The one observed directional operand cannot stand in for a complete sum.
  card._hass = { states: { "sensor.plant_pv": { state: "0", attributes: { unit_of_measurement: "kW" } }, "sensor.one_flow": { state: "7", attributes: { unit_of_measurement: "kW" } } } };
  let nodes = card._nodeDisplays(card._flowDisplays());
  assert.equal(nodes.solar.available, true);
  assert.equal(nodes.solar.numericValue, 0);
  assert.equal(nodes.home.available, false);
  card._config.source_policy.metrics.pv_power.verification = "unverified";
  card._config.source_policy.metrics.pv_power.display_decision = "withheld";
  nodes = card._nodeDisplays(card._flowDisplays());
  assert.equal(nodes.solar.available, false);
});

test("complete flows convert compatible power units and never infer a legacy battery sign", () => {
  const card = new FlowCard();
  card._config = { entities: { solar_power: "sensor.pv", battery_power: "sensor.legacy_battery" } };
  card._hass = { states: { "sensor.legacy_battery": { state: "4", attributes: { unit_of_measurement: "kW" } } } };
  const display = (numericValue, unit) => ({ numericValue, unit, available: true, numericValid: true });
  const flows = { pv_to_load_power: display(500, "W"), pv_to_grid_power: display(0, "kW"), pv_to_battery_power: display(0.5, "kW"), grid_to_load_power: display(NaN, "kW"), battery_to_load_power: display(NaN, "kW") };
  const nodes = card._nodeDisplays(flows);
  assert.equal(nodes.solar.numericValue, 1000);
  assert.equal(nodes.solar.unit, "W");
  assert.equal(nodes.battery.available, false);
  card._config.automatic_entities = { solar_power: "sensor.automatic" };
  const manualMissing = card._nodeDisplays(flows);
  assert.equal(manualMissing.solar.available, false, "missing manual source must remain authoritative");
});

test("non-numeric manual values stay unavailable without creating a zero", () => {
  const card = new FlowCard();
  card._config = { entities: { solar_power: "sensor.manual" }, automatic_entities: { solar_power: "sensor.automatic" } };
  for (const value of ["unknown", "unavailable", "none", "", "12 garbage", "0x10"]) {
    card._hass = { states: { "sensor.manual": { state: value, attributes: { unit_of_measurement: "kW" } } } };
    const nodes = card._nodeDisplays(card._flowDisplays());
    assert.equal(nodes.solar.available, false, value);
    assert.equal(nodes.solar.formatted, "--", value);
  }
});

test("live flow node values use manually selected sources before directional sums", () => {
  const card = new FlowCard();
  card._config = { entities: {
    solar_power: "sensor.manual_solar",
    load_power: "sensor.manual_load",
    grid_power: "sensor.manual_grid",
    battery_power: "sensor.manual_battery",
  }, automatic_entities: {
    solar_power: "sensor.automatic_solar",
    load_power: "sensor.automatic_load",
    grid_power: "sensor.automatic_grid",
    battery_power: "sensor.automatic_battery",
  } };
  card._hass = { states: {
    "sensor.manual_solar": { state: "7.5", attributes: { unit_of_measurement: "kW" } },
    "sensor.manual_load": { state: "3.8", attributes: { unit_of_measurement: "kW" } },
    "sensor.manual_grid": { state: "-1.2", attributes: { unit_of_measurement: "kW" } },
    "sensor.manual_battery": { state: "2.1", attributes: { unit_of_measurement: "kW" } },
  } };
  const route = (value) => ({ numericValue: value, unit: "kW", formatted: `${value} kW`, available: true });
  const nodes = card._nodeDisplays({
    pv_to_load_power: route(1), pv_to_grid_power: route(2), pv_to_battery_power: route(3),
    grid_to_load_power: route(4), battery_to_load_power: route(5),
  });
  assert.equal(nodes.solar.numericValue, 7.5);
  assert.equal(nodes.home.numericValue, 3.8);
  assert.equal(nodes.grid.numericValue, -1.2);
  assert.equal(nodes.battery.numericValue, 2.1);
});

test("live flow node values fall back to directional sums when direct sources are unavailable", () => {
  const card = new FlowCard();
  card._config = { entities: {} };
  card._hass = { states: {} };
  const route = (value) => ({ numericValue: value, unit: "kW", formatted: `${value} kW`, available: true });
  const nodes = card._nodeDisplays({
    pv_to_load_power: route(1), pv_to_grid_power: route(2), pv_to_battery_power: route(3),
    grid_to_load_power: route(4), battery_to_load_power: route(5),
  });
  assert.equal(nodes.solar.numericValue, 6);
  assert.equal(nodes.home.numericValue, 10);
  assert.equal(nodes.grid.numericValue, 2);
  assert.equal(nodes.battery.numericValue, 2);
});

test("live flow preserves directional sums while configured sources remain automatic", () => {
  const card = new FlowCard();
  card._config = {
    entities: { solar_power: "sensor.automatic_solar" },
    automatic_entities: { solar_power: "sensor.automatic_solar" },
  };
  card._hass = { states: { "sensor.automatic_solar": { state: "9.9", attributes: { unit_of_measurement: "kW" } } } };
  const route = (value) => ({ numericValue: value, unit: "kW", formatted: `${value} kW`, available: true });
  const nodes = card._nodeDisplays({
    pv_to_load_power: route(1), pv_to_grid_power: route(2), pv_to_battery_power: route(3),
    grid_to_load_power: route(4), battery_to_load_power: route(5),
  });
  assert.equal(nodes.solar.numericValue, 6);
});

function createBatteryFlowCard(compact = false, values = {}, units = {}) {
  const card = new FlowCard();
  const readings = {
    solar_power: 3.8, load_power: 5.17, grid_power: -0.43, battery_power: -1.8,
    battery_soc: 100, pv_to_load_power: 3.37, pv_to_grid_power: 0.43,
    pv_to_battery_power: 0, grid_to_load_power: 0, battery_to_load_power: 1.8,
    ...values,
  };
  const entities = Object.fromEntries(Object.keys(readings).map((key) => [key, `sensor.${key}`]));
  card._isCompact = () => compact;
  card.setConfig({ entities, automatic_entities: { ...entities } });
  card.hass = {
    locale: { language: "en-US" },
    states: Object.fromEntries(Object.entries(readings).filter(([, value]) => value !== null).map(([key, value]) => [
      entities[key], { state: String(value), attributes: { unit_of_measurement: units[key] ?? (key === "battery_soc" ? "%" : "kW") } },
    ])),
  };
  return card;
}

function batteryDirection(card) {
  return card._batteryDirectionStatus(card._flowDisplays());
}

test("issue 15 discharge and charging retain battery identity in both layouts", () => {
  for (const compact of [false, true]) {
    const card = createBatteryFlowCard(compact);
    let html = card.shadowRoot.innerHTML;
    assert.match(html, /aria-label="Battery power: 1\.80 kW; Discharging"/);
    assert.match(html, />Battery power<\/text>/);
    assert.match(html, />Discharging<\/text>/);
    assert.match(html, /data-node="home" data-chip="power"[\s\S]*?>5\.17 kW<\/text>/);
    assert.match(html, /data-node="battery" data-chip="soc"[\s\S]*?>100 %<\/text>/);
    assert.doesNotMatch(html, /class="route-pill pill-solar-battery"/);
    assert.doesNotMatch(html, /class="route-pill pill-battery-home"/);
    assert.match(html, /edge-active edge-battery-home[^>]*opacity:0\.96/);
    assert.match(html, /edge-active edge-solar-battery[^>]*opacity:0;/);
    const layout = card._layout(compact);
    assert.ok(layout.nodes.battery.powerChip.x < layout.nodes.battery.x - layout.radius);

    card._hass.states["sensor.pv_to_battery_power"].state = "1.8";
    card._hass.states["sensor.battery_to_load_power"].state = "0";
    card.hass = card._hass;
    html = card.shadowRoot.innerHTML;
    assert.match(html, /aria-label="Battery power: -1\.80 kW; Charging"/);
    assert.doesNotMatch(html, /class="route-pill pill-solar-battery"/);
    assert.match(html, /edge-active edge-solar-battery[^>]*opacity:0\.96/);
  }
});

test("battery direction covers boundaries, simultaneous flows, and invalid live inputs", () => {
  const cases = [
    [0, 0, "Idle"], [0.01, 0, "Idle"], [0, 0.01, "Idle"], [0.01, 0.01, "Idle"],
    [0, 0.011, "Discharging"], [0.011, 0, "Charging"],
    [1, 2, "Discharging"], [2, 1, "Charging"], [1, 1, "Balanced flows"],
    [1, 1.005, "Balanced flows"], [null, 0, "Direction unavailable"],
    [0, null, "Direction unavailable"], ["unknown", 0, "Direction unavailable"],
    [0, "unavailable", "Direction unavailable"], ["NaN", 0, "Direction unavailable"],
    ["Infinity", 0, "Direction unavailable"], ["", 0, "Direction unavailable"],
    ["1.8invalid", 0, "Direction unavailable"], [-1, 0, "Direction unavailable"],
  ];
  for (const [charge, discharge, expected] of cases) {
    const card = createBatteryFlowCard(false, { pv_to_battery_power: charge, battery_to_load_power: discharge });
    assert.equal(batteryDirection(card), expected, `${charge}/${discharge}`);
  }
  for (const unit of ["", "W", "kWh", "%"]) {
    assert.equal(batteryDirection(createBatteryFlowCard(false, {}, { pv_to_battery_power: unit })), "Direction unavailable");
  }
  for (const unit of ["W", "kW", "MW"]) {
    assert.equal(batteryDirection(createBatteryFlowCard(false, {}, { pv_to_battery_power: unit, battery_to_load_power: unit })), "Discharging");
  }
});

test("battery labels retain distinct manual and simultaneous route values", () => {
  const card = createBatteryFlowCard();
  card._config.entities.battery_power = "sensor.manual_battery";
  card._hass.states["sensor.manual_battery"] = { state: "-4.2", attributes: { unit_of_measurement: "kW" } };
  card.hass = card._hass;
  assert.match(card.shadowRoot.innerHTML, /aria-label="Battery power: -4\.20 kW; Discharging"/);
  assert.match(card.shadowRoot.innerHTML, /class="route-pill pill-battery-home"/);
  card._hass.states["sensor.manual_battery"].state = "unavailable";
  card.hass = card._hass;
  assert.match(card.shadowRoot.innerHTML, /class="route-pill pill-battery-home"/);
  const simultaneous = createBatteryFlowCard(false, { pv_to_battery_power: 1, battery_to_load_power: 2 });
  assert.match(simultaneous.shadowRoot.innerHTML, /class="route-pill pill-solar-battery"/);
  assert.match(simultaneous.shadowRoot.innerHTML, /class="route-pill pill-battery-home"/);
  for (const opposite of [null, "unknown"]) {
    assert.match(createBatteryFlowCard(false, { pv_to_battery_power: opposite }).shadowRoot.innerHTML, /class="route-pill pill-battery-home"/);
  }
  assert.match(createBatteryFlowCard(false, {}, { pv_to_battery_power: "W" }).shadowRoot.innerHTML, /class="route-pill pill-battery-home"/);
});

test("battery label suppression uses strict tolerance and retains unavailable nodes", () => {
  const card = createBatteryFlowCard();
  const layout = card._layout(false);
  const power = (value, available = true) => ({ numericValue: value, unit: "kW", available, formatted: `${value} kW` });
  const flow = power(0.02);
  const flows = { pv_to_battery_power: power(0), battery_to_load_power: flow };
  const label = (node) => card._renderEdgeLabel("battery_to_load_power", layout.edges.battery_to_load_power, flow, { battery: node }, flows);
  assert.equal(label(power(0.025)), "");
  assert.notEqual(label(power(0.01)), "");
  assert.notEqual(label(power(0.02, false)), "");
  flows.pv_to_battery_power = power(0.01);
  assert.equal(label(power(0.02)), "");
  flows.pv_to_battery_power = power(0.011);
  assert.notEqual(label(power(0.02)), "");
});

test("battery caption and direction localize, fall back, and escape custom text", () => {
  const card = createBatteryFlowCard();
  for (const [locale, caption, direction] of [
    ["en-US", "Battery power", "Discharging"], ["de-DE", "Batterieleistung", "Entlädt"],
    ["sv-SE", "Batterieffekt", "Urladdar"], ["es-ES", "Potencia de batería", "Descargando"],
    ["fr-FR", "Battery power", "Discharging"],
  ]) {
    card._hass.locale.language = locale;
    card.hass = card._hass;
    assert.ok(card.shadowRoot.innerHTML.includes(`>${caption}</text>`));
    assert.ok(card.shadowRoot.innerHTML.includes(`>${direction}</text>`));
  }
  card._config.labels = { battery_power_caption: "<Custom battery>", battery_status_discharging: "Custom discharge" };
  card.hass = card._hass;
  assert.match(card.shadowRoot.innerHTML, /&lt;Custom battery&gt;/);
  assert.match(card.shadowRoot.innerHTML, /Custom discharge/);
});

test("source mapping card renders automatic and manual status without changing defaults", () => {
  const card = new SourceCard();
  card.setConfig({ schema_version: 1, mapping_id: "target", defaults: { p13112: "sensor.auto" }, overrides: {}, metrics: [{ key: "p13112", group: "today_energy", label: "Solar production today", default: "sensor.auto", selected: "sensor.auto", status: "automatic", candidates: [] }], labels: { groups: { today_energy: "Today's energy" } } });
  card.hass = { user: { is_admin: true }, states: { "sensor.auto": { state: "42.8", attributes: { unit_of_measurement: "kWh", friendly_name: "Automatic solar" } } } };
  assert.match(card.shadowRoot.innerHTML, /Automatic/);
  assert.match(card.shadowRoot.innerHTML, /42\.8 kWh/);
  assert.equal(card._config.defaults.p13112, "sensor.auto");

  card._config.overrides.p13112 = "sensor.manual";
  card._hass.states["sensor.manual"] = { state: "41.9", attributes: { unit_of_measurement: "kWh", friendly_name: "Manual solar" } };
  card._render();
  assert.match(card.shadowRoot.innerHTML, /Manual/);
  assert.match(card.shadowRoot.innerHTML, /Manual solar/);
  assert.equal(card._config.defaults.p13112, "sensor.auto");
});

test("source mapping card JSON pointers update only declared bindings", () => {
  const card = new SourceCard();
  const dashboard = { views: [{ cards: [{ entity: "sensor.auto" }, { entity: "sensor.other" }] }] };
  assert.equal(card._getPointer(dashboard, "/views/0/cards/0/entity"), "sensor.auto");
  card._setPointer(dashboard, "/views/0/cards/0/entity", "sensor.manual");
  assert.equal(dashboard.views[0].cards[0].entity, "sensor.manual");
  assert.equal(dashboard.views[0].cards[1].entity, "sensor.other");
});

test("source mapping card exposes warning text and admin-only controls", () => {
  const card = new SourceCard();
  card.setConfig({ schema_version: 1, mapping_id: "target", defaults: { p13116: "sensor.direct", p13112: "sensor.production" }, overrides: {}, metrics: [{ key: "p13112", group: "today_energy", label: "Solar production", default: "sensor.production", candidates: [] }, { key: "p13116", group: "today_energy", label: "Direct solar consumption", default: "sensor.direct", validation: { schema_version: 2, rules: [{ type: "not_materially_greater_than", metric: "p13112", relative_tolerance: 0.05, absolute_tolerance: 0.1 }] }, candidates: [] }], labels: { groups: { today_energy: "Today's energy" }, readonly: "Administrators only" } });
  card.hass = { user: { is_admin: false }, states: { "sensor.production": { state: "42.8", attributes: { unit_of_measurement: "kWh" } }, "sensor.direct": { state: "52.6", attributes: { unit_of_measurement: "kWh" } } } };
  assert.match(card.shadowRoot.innerHTML, /Needs review/);
  assert.match(card.shadowRoot.innerHTML, /exceeds solar production/);
  assert.match(card.shadowRoot.innerHTML, /Administrators only/);
  assert.match(card.shadowRoot.innerHTML, /disabled/);
});

function sourceSaveFixture() {
  const config = {
    schema_version: 1,
    matcher_version: 5,
    mapping_id: "source-test",
    dashboard_url_path: "gosungrow",
    defaults: { p13112: "sensor.auto" },
    overrides: {},
    candidates: { p13112: [{ entity_id: "sensor.auto", confidence: "high", reason: "Current automatic match", recommended: true, verification: "verified", compatibility: "compatible" }, { entity_id: "sensor.manual", confidence: "medium", reason: "Compatible same-plant source", recommended: true, verification: "verified", compatibility: "compatible" }] },
    bindings: { p13112: ["/views/0/cards/0/entity", "/views/0/cards/1/entity", "/views/0/cards/0/source_policy/metrics/p13112/entity_id", "/views/0/cards/1/source_policy/metrics/p13112/entity_id"] },
    metrics: [{ key: "p13112", group: "today_energy", label: "Solar production", default: "sensor.auto", confidence: "high", ...verifiedPolicy({ production: "sensor.auto" }).metrics.p13112, validation: { schema_version: 2, rules: [] } }],
    labels: { groups: { today_energy: "Today's energy" } },
  };
  const dashboard = { views: [{ cards: [{ entity: "sensor.auto", source_policy: verifiedPolicy({ production: "sensor.auto" }) }, { entity: "sensor.auto", source_policy: verifiedPolicy({ production: "sensor.auto" }) }, { entity: "sensor.auto" }] }, { cards: [{ type: "custom:gosungrow-source-mapping-card-v1", ...structuredClone(config) }] }] };
  return { config, dashboard };
}

function transactionalWS(dashboard, onSave = () => {}) {
  let current = structuredClone(dashboard);
  return async (request) => {
    if (request.type === "lovelace/config") return structuredClone(current);
    if (request.type === "lovelace/config/save") {
      current = structuredClone(request.config);
      onSave(structuredClone(current));
      return undefined;
    }
    throw new Error(`Unexpected websocket request: ${request.type}`);
  };
}

test("source mapping selection remains pending until the explicit save action", () => {
  const card = new SourceCard();
  const { config } = sourceSaveFixture();
  card.setConfig(config);
  card._pendingEntity = "sensor.manual";
  assert.equal(card._selected(card._metrics()[0]), "sensor.auto");
  assert.equal(card._config.overrides.p13112, undefined);
  assert.match(card._dialog(card._metrics()[0]), /data-commit="p13112"/);
});

test("source mapping save mutates exactly its bindings and commits only after websocket success", async () => {
  const card = new SourceCard();
  const { config, dashboard } = sourceSaveFixture();
  let saved;
  const events = [];
  card.setConfig(config);
  card.dispatchEvent = (event) => { events.push(event.type); return true; };
  card.hass = { user: { is_admin: true }, states: { "sensor.auto": { state: "4", attributes: { unit_of_measurement: "kWh" } }, "sensor.manual": { state: "5", attributes: { unit_of_measurement: "kWh" } } }, callWS: transactionalWS(dashboard, (config) => { saved = config; }) };
  await card._save("p13112", "sensor.manual");
  await Promise.resolve();
  assert.equal(saved.views[0].cards[0].entity, "sensor.manual");
  assert.equal(saved.views[0].cards[1].entity, "sensor.manual");
  assert.equal(saved.views[0].cards[2].entity, "sensor.auto");
  assert.equal(saved.views[1].cards[0].overrides.p13112, "sensor.manual");
  assert.equal(dashboard.views[0].cards[0].entity, "sensor.auto");
  assert.equal(card._config.overrides.p13112, "sensor.manual");
  assert.equal(card._config.metrics[0].confidence, "medium");
  assert.deepEqual(events, ["hass-notification", "config-refresh"]);
});

test("source mapping reset restores automatic and removes the override transactionally", async () => {
  const card = new SourceCard();
  const { config, dashboard } = sourceSaveFixture();
  config.overrides.p13112 = "sensor.manual";
  Object.assign(config.metrics[0], { entity_id: "sensor.manual", selection_origin: "manual" });
  dashboard.views[0].cards[0].entity = "sensor.manual";
  dashboard.views[0].cards[1].entity = "sensor.manual";
  dashboard.views[1].cards[0].overrides.p13112 = "sensor.manual";
  Object.assign(dashboard.views[1].cards[0].metrics[0], { entity_id: "sensor.manual", selection_origin: "manual" });
  for (const display of dashboard.views[0].cards.slice(0, 2)) Object.assign(display.source_policy.metrics.p13112, { entity_id: "sensor.manual", selection_origin: "manual" });
  let saved;
  card.setConfig(config);
  card.hass = { user: { is_admin: true }, states: { "sensor.auto": { state: "4", attributes: { unit_of_measurement: "kWh" } }, "sensor.manual": { state: "5", attributes: { unit_of_measurement: "kWh" } } }, callWS: transactionalWS(dashboard, (config) => { saved = config; }) };
  await card._save("p13112", null);
  assert.equal(saved.views[0].cards[0].entity, "sensor.auto");
  assert.equal(saved.views[0].cards[1].entity, "sensor.auto");
  assert.equal(saved.views[1].cards[0].overrides.p13112, undefined);
  assert.equal(card._config.overrides.p13112, undefined);
  assert.equal(card._config.metrics[0].confidence, "high");
});

test("source mapping rejects stale and non-candidate saves without local mutation", async () => {
  for (const mode of ["stale", "non-candidate"]) {
    const card = new SourceCard();
    const { config, dashboard } = sourceSaveFixture();
    if (mode === "stale") dashboard.views[0].cards[0].entity = "sensor.changed";
    card.setConfig(config);
    let saveCalls = 0;
    card.hass = { user: { is_admin: true }, states: {}, callWS: async (request) => { if (request.type === "lovelace/config/save") saveCalls += 1; return structuredClone(dashboard); } };
    await card._save("p13112", mode === "non-candidate" ? "sensor.injected" : "sensor.manual");
    assert.equal(saveCalls, 0, mode);
    assert.equal(JSON.stringify(card._config.overrides), "{}", mode);
  }
});

test("source mapping rejects stale defaults, overrides, and bindings", async () => {
  for (const field of ["defaults", "overrides", "bindings"]) {
    const card = new SourceCard();
    const { config, dashboard } = sourceSaveFixture();
    const mapping = dashboard.views[1].cards[0];
    if (field === "defaults") mapping.defaults.p13112 = "sensor.changed";
    if (field === "overrides") mapping.overrides.p13112 = "sensor.changed";
    if (field === "bindings") mapping.bindings.p13112 = ["/views/0/cards/99/entity"];
    card.setConfig(config);
    let saveCalls = 0;
    card.hass = { user: { is_admin: true }, states: {}, callWS: async (request) => { if (request.type === "lovelace/config/save") saveCalls += 1; return structuredClone(dashboard); } };
    await card._save("p13112", "sensor.manual");
    assert.equal(saveCalls, 0, field);
    assert.equal(JSON.stringify(card._config.overrides), "{}", field);
  }
});

test("source mapping read-only mode never calls the websocket API", async () => {
  const card = new SourceCard();
  const { config } = sourceSaveFixture();
  card.setConfig(config);
  let calls = 0;
  card.hass = { user: { is_admin: false }, states: {}, callWS: async () => { calls += 1; } };
  await card._save("p13112", "sensor.manual");
  assert.equal(calls, 0);
  assert.equal(JSON.stringify(card._config.overrides), "{}");
});

test("source mapping warned selection requires explicit confirmation", async () => {
  const card = new SourceCard();
  const { config } = sourceSaveFixture();
  card.setConfig(config);
  let calls = 0;
  card.hass = { user: { is_admin: true }, states: {}, callWS: async () => { calls += 1; } };
  sandbox.window.confirm = () => false;
  await card._save("p13112", "sensor.manual");
  sandbox.window.confirm = () => true;
  assert.equal(calls, 0);
  assert.equal(JSON.stringify(card._config.overrides), "{}");
});

test("source mapping Escape closes the dialog and restores the configured focus target", async () => {
  const card = new SourceCard();
  let prevented = false;
  let focused = false;
  card._activeMetric = { key: "p13112" };
  card._pendingEntity = "sensor.manual";
  card._lastFocus = { focus: () => { focused = true; } };
  card._trapDialogKeys({ key: "Escape", preventDefault: () => { prevented = true; } });
  await Promise.resolve();
  assert.equal(prevented, true);
  assert.equal(card._activeMetric, null);
  assert.equal(card._pendingEntity, null);
  assert.equal(focused, true);
});

test("source mapping authorization and save failures keep local state unchanged", async () => {
  for (const failureAt of ["read", "save"]) {
    const card = new SourceCard();
    const { config, dashboard } = sourceSaveFixture();
    card.setConfig(config);
    card.hass = { user: { is_admin: true }, states: { "sensor.manual": { state: "5", attributes: { unit_of_measurement: "kWh" } } }, callWS: async (request) => { if (failureAt === "read" || request.type === "lovelace/config/save") throw new Error("Not authorized"); return structuredClone(dashboard); } };
    await card._save("p13112", "sensor.manual");
    assert.equal(JSON.stringify(card._config.overrides), "{}", failureAt);
    assert.match(card._notice, /Not authorized/);
  }
});

test("source mapping candidate values and warnings follow live Home Assistant state", () => {
  const card = new SourceCard();
  card.setConfig({ schema_version: 1, mapping_id: "target", defaults: { p13112: "sensor.production", p13116: "sensor.direct" }, overrides: {}, metrics: [{ key: "p13112", group: "today_energy", default: "sensor.production", candidates: [] }, { key: "p13116", group: "today_energy", default: "sensor.direct", candidates: [{ entity_id: "sensor.direct", reason: "Compatible" }], validation: { schema_version: 2, rules: [{ type: "not_materially_greater_than", metric: "p13112", relative_tolerance: 0.05, absolute_tolerance: 0.1 }] } }], labels: { groups: { today_energy: "Today" } } });
  card.hass = { user: { is_admin: true }, states: { "sensor.production": { state: "42.8", attributes: { unit_of_measurement: "kWh" } }, "sensor.direct": { state: "52.6", attributes: { unit_of_measurement: "kWh", friendly_name: "Live direct solar" } } } };
  assert.equal(card._status(card._metric("p13116")), "needs_review");
  assert.match(card._candidate(card._metric("p13116"), card._metric("p13116").candidates[0]), /Live direct solar/);
  assert.match(card._candidate(card._metric("p13116"), card._metric("p13116").candidates[0]), /52\.6/);
  card._hass.states["sensor.direct"].state = "40";
  assert.equal(card._status(card._metric("p13116")), "automatic");
  delete card._hass.states["sensor.direct"];
  assert.equal(card._status(card._metric("p13116")), "unavailable");
});

test("source mapping candidate labels remove shared prefixes while preserving full accessible identity", () => {
  const card = new SourceCard();
  const candidates = [
    { entity_id: "sensor.gosungrow_5072099_14_1_1_overview_pv_power" },
    { entity_id: "sensor.gosungrow_5072099_14_1_1_overview_total_dc_power" },
  ];
  card.setConfig({ schema_version: 1, mapping_id: "target", defaults: { pv: candidates[0].entity_id }, candidates: { pv: candidates }, metrics: [{ key: "pv", group: "live_power", default: candidates[0].entity_id }], labels: { groups: { live_power: "Live" } } });
  card.hass = { user: { is_admin: true }, states: {
    [candidates[0].entity_id]: { state: "1.2", attributes: { friendly_name: "GoSungrow 5072099 roof inverter overview PV power", unit_of_measurement: "kW" } },
    [candidates[1].entity_id]: { state: "1.1", attributes: { friendly_name: "GoSungrow 5072099 roof inverter overview total DC power", unit_of_measurement: "kW" } },
  } };
  const html = card._candidate(card._metrics()[0], candidates[1], candidates);
  assert.match(html, />total DC power<\/strong>/);
  assert.match(html, /title="GoSungrow 5072099 roof inverter overview total DC power"/);
  assert.match(html, /aria-label="GoSungrow 5072099 roof inverter overview total DC power; sensor\.gosungrow_/);
  assert.doesNotMatch(html, />GoSungrow 5072099 roof inverter overview total DC power<\/strong>/);
});

test("source mapping rows remove shared prefixes while preserving full accessible identity", () => {
  const card = new SourceCard();
  card.setConfig({
    schema_version: 1,
    mapping_id: "target",
    defaults: { pv: "sensor.roof_pv_power", load: "sensor.roof_load_power" },
    metrics: [
      { key: "pv", group: "live_power", label: "Solar power", default: "sensor.roof_pv_power" },
      { key: "load", group: "live_power", label: "Home power", default: "sensor.roof_load_power" },
    ],
    labels: { groups: { live_power: "Live" } },
  });
  card.hass = { user: { is_admin: true }, states: {
    "sensor.roof_pv_power": { state: "4.2", attributes: { friendly_name: "GoSungrow 5072099 roof inverter overview PV power", unit_of_measurement: "kW" } },
    "sensor.roof_load_power": { state: "1.4", attributes: { friendly_name: "GoSungrow 5072099 roof inverter overview Load power", unit_of_measurement: "kW" } },
  } };
  const html = card._row(card._metrics()[0]);
  assert.match(html, />PV power<\/div>/);
  assert.match(html, /title="GoSungrow 5072099 roof inverter overview PV power · sensor\.roof_pv_power"/);
  assert.match(html, /aria-label="GoSungrow 5072099 roof inverter overview PV power; sensor\.roof_pv_power"/);
  assert.doesNotMatch(html, />GoSungrow 5072099 roof inverter overview PV power<\/div>/);
});

test("source mapping verifies the persisted dashboard before changing local state", async () => {
  const card = new SourceCard();
  const { config, dashboard } = sourceSaveFixture();
  card.setConfig(config);
  card.hass = { user: { is_admin: true }, states: { "sensor.manual": { state: "5", attributes: { unit_of_measurement: "kWh" } } }, callWS: async (request) => request.type === "lovelace/config" ? structuredClone(dashboard) : undefined };
  await card._save("p13112", "sensor.manual");
  assert.equal(card._config.overrides.p13112, undefined);
  assert.match(card._notice, /Could not save/);
});

test("source mapping requests a native Lovelace configuration refresh", () => {
  const card = new SourceCard();
  const events = [];
  card.dispatchEvent = (event) => { events.push(event); return true; };
  card._refreshDashboardView();
  assert.equal(events.length, 1);
  assert.equal(events[0].type, "config-refresh");
  assert.equal(events[0].bubbles, true);
  assert.equal(events[0].composed, true);
});

test("source mapping stale warning follows live timestamps and clears after an update", () => {
  const card = new SourceCard();
  card.setConfig({ schema_version: 1, mapping_id: "target", defaults: { pv_power: "sensor.power" }, overrides: {}, metrics: [{ key: "pv_power", group: "live_power", default: "sensor.power", validation: { schema_version: 2, rules: [{ type: "freshness", max_age_seconds: 1800 }] } }], labels: { groups: { live_power: "Live power" } } });
  card.hass = { user: { is_admin: true }, states: { "sensor.power": { state: "3.2", last_updated: new Date(Date.now() - 3600000).toISOString(), attributes: { unit_of_measurement: "kW" } } } };
  assert.equal(card._status(card._metric("pv_power")), "needs_review");
  assert.match(card._warning(card._metric("pv_power")), /not updated recently/);
  card._hass.states["sensor.power"].last_updated = new Date().toISOString();
  assert.equal(card._status(card._metric("pv_power")), "automatic");
});

function maxRow(date, max) {
  return { start: `${date}T00:00:00.000Z`, max };
}

function timestampRow(date, sum) {
  return { start: new Date(`${date}T00:00:00.000Z`).getTime(), sum };
}

function chartFor(card, period, rows = []) {
  const cache = card._parseStatistics({ [entityID]: rows }, [entityID], period);
  card._statsCache[card._cacheKey(period)] = cache;
  return card._chartDisplay(period, card._metricDefinitions());
}

function values(chart) {
  return Array.from(chart.buckets, (bucket) => bucket.values[entityID]);
}

function bucketKeys(chart) {
  return Array.from(chart.buckets, (bucket) => bucket.key);
}

test("fresh install shows one live bucket for day, month, and year", () => {
  const card = createCard("2026-05-16T12:00:00.000Z", 7);

  for (const period of ["day", "month", "year"]) {
    const chart = card._chartDisplay(period, card._metricDefinitions());
    assert.equal(chart.buckets.length, 1);
    assert.deepEqual(values(chart), [7]);
    assert.equal(card._headlineStatValue(period, entityID).value, 7);
  }
});

test("daily bucket labels follow Home Assistant language date ordering", () => {
  const date = new Date(2026, 6, 21, 12);
  const card = createCard("2026-07-21T12:00:00.000Z", 7);

  card._hass.locale = { language: "en-US", date_format: "language" };
  assert.equal(card._bucketLabel("day", date), "07/21");

  card._hass.locale = { language: "en-GB", date_format: "language" };
  assert.equal(card._bucketLabel("day", date), "21/07");
});

test("explicit Home Assistant date preferences override language ordering", () => {
  const date = new Date(2026, 6, 21, 12);
  const card = createCard("2026-07-21T12:00:00.000Z", 7);

  for (const [language, dateFormat, expected] of [
    ["en-US", "DMY", "21/07"],
    ["en-GB", "MDY", "07/21"],
    ["en-GB", "YMD", "07/21"],
  ]) {
    card._hass.locale = { language, date_format: dateFormat };
    assert.equal(card._bucketLabel("day", date), expected);
  }
});

test("system, missing, and invalid locale preferences have deterministic fallbacks", () => {
  const date = new Date(2026, 6, 21, 12);
  const card = createCard("2026-07-21T12:00:00.000Z", 7);

  card._hass.locale = { language: "en-GB", date_format: "system" };
  assert.equal(
    card._bucketLabel("day", date),
    new Intl.DateTimeFormat(undefined, { day: "2-digit", month: "2-digit" }).format(date),
  );

  card._hass.locale = { language: "en-US" };
  assert.equal(card._bucketLabel("day", date), "07/21");

  card._hass.locale = { language: "invalid_locale", date_format: "DMY" };
  assert.equal(card._bucketLabel("day", date), "21/07");
});

test("month and year bucket labels remain localized", () => {
  const date = new Date(2026, 6, 21, 12);
  const card = createCard("2026-07-21T12:00:00.000Z", 7);
  card._hass.locale = { language: "de-DE", date_format: "DMY" };

  assert.equal(card._bucketLabel("month", date), new Intl.DateTimeFormat("de-DE", { month: "short" }).format(date));
  assert.equal(card._bucketLabel("year", date), new Intl.DateTimeFormat("de-DE", { year: "2-digit" }).format(date));
});

test("cached chart buckets relabel after a date preference change without refetching statistics", () => {
  const card = createCard("2026-05-17T12:00:00.000Z", 3);
  card._hass.locale = { language: "en-US", date_format: "language" };
  const initial = chartFor(card, "day", [stateRow("2026-05-16", 7)]);
  assert.deepEqual(Array.from(initial.buckets, (bucket) => bucket.label), ["05/16", "05/17"]);

  let requests = 0;
  card.hass = {
    ...card._hass,
    locale: { language: "en-US", date_format: "DMY" },
    callWS: () => {
      requests += 1;
      return Promise.resolve({});
    },
  };

  const relabeled = card._chartDisplay("day", card._metricDefinitions());
  assert.deepEqual(Array.from(relabeled.buckets, (bucket) => bucket.label), ["16/05", "17/05"]);
  assert.equal(requests, 0);
});

test("month and year combine completed recorder days with today's live value", () => {
  const card = createCard("2026-05-16T12:00:00.000Z", 3);
  const rows = [
    stateRow("2026-05-14", 10),
    stateRow("2026-05-15", 14),
    stateRow("2026-05-16", 100),
  ];

  assert.deepEqual(values(chartFor(card, "month", rows)), [27]);
  assert.equal(card._headlineStatValue("month", entityID).value, 27);

  assert.deepEqual(values(chartFor(card, "year", rows)), [27]);
  assert.equal(card._headlineStatValue("year", entityID).value, 27);
});

test("current day recorder rows are ignored and replaced by live daily state", () => {
  const card = createCard("2026-05-16T12:00:00.000Z", 3);
  const rows = [
    stateRow("2026-05-15", 10),
    stateRow("2026-05-16", 100),
  ];

  assert.deepEqual(values(chartFor(card, "day", rows)), [10, 3]);
  assert.equal(card._headlineStatValue("day", entityID).value, 3);
});

test("second day keeps the first completed recorder day when no baseline exists", () => {
  const card = createCard("2026-05-17T12:00:00.000Z", 3);
  const rows = [
    stateRow("2026-05-16", 7),
    stateRow("2026-05-17", 100),
  ];

  const dayChart = chartFor(card, "day", rows);
  assert.deepEqual(bucketKeys(dayChart), ["2026-05-16", "2026-05-17"]);
  assert.deepEqual(values(dayChart), [7, 3]);

  const monthChart = chartFor(card, "month", rows);
  assert.deepEqual(bucketKeys(monthChart), ["2026-05"]);
  assert.deepEqual(values(monthChart), [10]);
  assert.equal(card._headlineStatValue("month", entityID).value, 10);

  const yearChart = chartFor(card, "year", rows);
  assert.deepEqual(bucketKeys(yearChart), ["2026"]);
  assert.deepEqual(values(yearChart), [10]);
  assert.equal(card._headlineStatValue("year", entityID).value, 10);
});

test("recorder rows with Home Assistant millisecond timestamps are parsed", () => {
  const card = createCard("2026-05-17T12:00:00.000Z", 3);
  const rows = [
    timestampRow("2026-05-16", 7),
    timestampRow("2026-05-17", 100),
  ];

  const dayChart = chartFor(card, "day", rows);
  assert.deepEqual(bucketKeys(dayChart), ["2026-05-16", "2026-05-17"]);
  assert.deepEqual(values(dayChart), [7, 3]);

  const monthChart = chartFor(card, "month", rows);
  assert.deepEqual(values(monthChart), [10]);
});

test("daily recorder max is preferred for accumulating daily sensors", () => {
  const card = createCard("2026-05-17T12:00:00.000Z", 4);
  const rows = [
    { ...maxRow("2026-05-16", 23), state: 22.8, sum: 5.5 },
    { ...maxRow("2026-05-17", 99), state: 99, sum: 99 },
  ];

  const dayChart = chartFor(card, "day", rows);
  assert.deepEqual(bucketKeys(dayChart), ["2026-05-16", "2026-05-17"]);
  assert.deepEqual(values(dayChart), [23, 4]);

  const monthChart = chartFor(card, "month", rows);
  assert.deepEqual(values(monthChart), [27]);
  assert.equal(card._headlineStatValue("month", entityID).value, 27);
});

test("month boundary keeps previous month total and starts current month from live day", () => {
  const card = createCard("2026-06-01T12:00:00.000Z", 2);
  const rows = [
    stateRow("2026-05-29", 0),
    stateRow("2026-05-30", 4),
    stateRow("2026-05-31", 9),
    stateRow("2026-06-01", 99),
  ];

  const chart = chartFor(card, "month", rows);
  assert.deepEqual(bucketKeys(chart), ["2026-05", "2026-06"]);
  assert.deepEqual(values(chart), [13, 2]);
  assert.equal(card._headlineStatValue("month", entityID).value, 2);
});

test("year boundary keeps previous year total and starts current year from live day", () => {
  const card = createCard("2027-01-01T12:00:00.000Z", 2);
  const rows = [
    stateRow("2026-12-30", 0),
    stateRow("2026-12-31", 5),
    stateRow("2027-01-01", 99),
  ];

  const chart = chartFor(card, "year", rows);
  assert.deepEqual(bucketKeys(chart), ["2026", "2027"]);
  assert.deepEqual(values(chart), [5, 2]);
  assert.equal(card._headlineStatValue("year", entityID).value, 2);
});

test("missing live state and no recorder rows leaves the chart empty", () => {
  const card = createCard("2026-05-16T12:00:00.000Z", NaN);
  const chart = card._chartDisplay("month", card._metricDefinitions());

  assert.equal(chart.buckets.length, 0);
  assert.equal(chart.series.length, 0);
  assert.equal(card._statisticsStatus("month"), "Statistics unavailable");
});

test("recorder requests use daily statistics for every view", () => {
  const card = createCard("2026-05-16T12:00:00.000Z", 3);
  let request;
  card._period = "year";
  card._hass.callWS = (nextRequest) => {
    request = nextRequest;
    return Promise.resolve({});
  };

  card._ensureStatistics();

  assert.equal(request.period, "day");
  assert.deepEqual(Array.from(request.types), ["max", "state", "sum"]);
});

test("rendered bars expose bucket metadata and keyboard focus", () => {
  const card = createCard("2026-05-16T12:00:00.000Z", 7);
  card._hass.locale = { language: "en-US", date_format: "DMY" };
  const chart = card._chartDisplay("day", card._metricDefinitions());
  const markup = card._renderChart(chart);

  assert.match(markup, /data-bucket-index="0"/);
  assert.match(markup, /data-series-key="production"/);
  assert.match(markup, /tabindex="0"/);
  assert.match(markup, /role="img"/);
  assert.match(markup, /aria-label="16\/05, Production, 7\.00 kWh"/);
  assert.match(markup, />16\/05<\/text>/);
  assert.match(markup, /class="chart-tooltip hidden"/);
});

test("tooltip heading uses the same localized bucket label as the axis", () => {
  const card = createCard("2026-05-16T12:00:00.000Z", 7);
  card._hass.locale = { language: "en-US", date_format: "DMY" };
  const chart = card._chartDisplay("day", card._metricDefinitions());
  const tooltip = {
    innerHTML: "",
    classList: { remove() {} },
    getBoundingClientRect: () => ({ width: 120, height: 60 }),
    style: {},
  };
  const wrapper = {
    getBoundingClientRect: () => ({ left: 0, top: 0, width: 720, height: 230 }),
  };
  card.shadowRoot = {
    querySelector: (selector) => (selector === ".chart-tooltip" ? tooltip : selector === ".chart-wrap" ? wrapper : null),
  };

  card._showChartTooltip({ clientX: 100, clientY: 100 }, chart, 0);

  assert.match(tooltip.innerHTML, /class="tooltip-title">16\/05<\/div>/);
});

test("tooltip rows include all configured metrics with color, label, and formatted value", () => {
  const card = createMultiMetricCard("2026-05-16T12:00:00.000Z", {
    production: 12.34,
    consumption: 8.9,
    to_grid: 1.2,
    from_grid: 0.4,
    to_battery: 2.5,
    from_battery: 3.6,
  });
  const chart = card._chartDisplay("day", card._metricDefinitions());
  const rows = card._tooltipRows(chart, 0);

  assert.equal(rows.length, 6);
  assert.deepEqual(
    Array.from(rows, (row) => row.label),
    ["Production", "Consumption", "To Grid", "From Grid", "To Battery", "From Battery"],
  );
  assert.deepEqual(
    Array.from(rows, (row) => row.color),
    ["#f59e0b", "#38bdf8", "#8b5cf6", "#cbd5e1", "#ec4899", "#14b8a6"],
  );
  assert.deepEqual(
    Array.from(rows, (row) => row.display),
    ["12.3 kWh", "8.90 kWh", "1.20 kWh", "0.40 kWh", "2.50 kWh", "3.60 kWh"],
  );
});

test("tooltip rows omit missing bucket values", () => {
  const card = createMultiMetricCard("2026-05-16T12:00:00.000Z", {
    production: 12.34,
    to_grid: 1.2,
  });
  const chart = card._chartDisplay("day", card._metricDefinitions());
  const rows = card._tooltipRows(chart, 0);

  assert.deepEqual(
    Array.from(rows, (row) => row.label),
    ["Production", "To Grid"],
  );
});
