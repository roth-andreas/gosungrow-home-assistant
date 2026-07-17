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
  CustomEvent: class CustomEvent {},
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
  },
  navigator: { language: "en-US" },
  window: { customCards: [], confirm: () => true, location: { reload: () => {} } },
};

vm.createContext(sandbox);
vm.runInContext(fs.readFileSync(assetPath, "utf8"), sandbox, { filename: assetPath });

const SummaryCard = registry.get("gosungrow-energy-summary-card-v1");
const SourceCard = registry.get("gosungrow-source-mapping-card-v1");
const entityID = "sensor.gosungrow_daily_production";
const metricEntities = {
  production: "sensor.gosungrow_daily_production",
  consumption: "sensor.gosungrow_daily_consumption",
  to_grid: "sensor.gosungrow_daily_to_grid",
  from_grid: "sensor.gosungrow_daily_from_grid",
  to_battery: "sensor.gosungrow_daily_to_battery",
  from_battery: "sensor.gosungrow_daily_from_battery",
};

function createCard(now, liveValue) {
  const card = new SummaryCard();
  card._config = {
    buckets: { day: 14, month: 12, year: 5 },
    entities: { production: entityID },
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
  card.setConfig({ schema_version: 1, mapping_id: "target", defaults: { p13116: "sensor.direct", p13112: "sensor.production" }, overrides: {}, metrics: [{ key: "p13112", group: "today_energy", label: "Solar production", default: "sensor.production", candidates: [] }, { key: "p13116", group: "today_energy", label: "Direct solar consumption", default: "sensor.direct", validation: { schema_version: 1, rules: [{ type: "not_materially_greater_than", metric: "p13112", relative_tolerance: 0.05, absolute_tolerance: 0.1 }] }, candidates: [] }], labels: { groups: { today_energy: "Today's energy" }, readonly: "Administrators only" } });
  card.hass = { user: { is_admin: false }, states: { "sensor.production": { state: "42.8", attributes: { unit_of_measurement: "kWh" } }, "sensor.direct": { state: "52.6", attributes: { unit_of_measurement: "kWh" } } } };
  assert.match(card.shadowRoot.innerHTML, /Needs review/);
  assert.match(card.shadowRoot.innerHTML, /exceeds solar production/);
  assert.match(card.shadowRoot.innerHTML, /Administrators only/);
  assert.match(card.shadowRoot.innerHTML, /disabled/);
});

function sourceSaveFixture() {
  const config = {
    schema_version: 1,
    mapping_id: "source-test",
    dashboard_url_path: "gosungrow",
    defaults: { p13112: "sensor.auto" },
    overrides: {},
    candidates: { p13112: [{ entity_id: "sensor.auto", confidence: "high", reason: "Current automatic match", recommended: true }, { entity_id: "sensor.manual", confidence: "medium", reason: "Compatible same-plant source", recommended: true }] },
    bindings: { p13112: ["/views/0/cards/0/entity", "/views/0/cards/1/entity"] },
    metrics: [{ key: "p13112", group: "today_energy", label: "Solar production", default: "sensor.auto", confidence: "high", reason: "Current automatic match" }],
    labels: { groups: { today_energy: "Today's energy" } },
  };
  const dashboard = { views: [{ cards: [{ entity: "sensor.auto" }, { entity: "sensor.auto" }, { entity: "sensor.auto" }] }, { cards: [{ type: "custom:gosungrow-source-mapping-card-v1", ...structuredClone(config) }] }] };
  return { config, dashboard };
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
  card.setConfig(config);
  card.hass = { user: { is_admin: true }, states: { "sensor.auto": { state: "4", attributes: {} }, "sensor.manual": { state: "5", attributes: {} } }, callWS: async (request) => request.type === "lovelace/config" ? structuredClone(dashboard) : (saved = request.config) };
  await card._save("p13112", "sensor.manual");
  assert.equal(saved.views[0].cards[0].entity, "sensor.manual");
  assert.equal(saved.views[0].cards[1].entity, "sensor.manual");
  assert.equal(saved.views[0].cards[2].entity, "sensor.auto");
  assert.equal(saved.views[1].cards[0].overrides.p13112, "sensor.manual");
  assert.equal(dashboard.views[0].cards[0].entity, "sensor.auto");
  assert.equal(card._config.overrides.p13112, "sensor.manual");
  assert.equal(card._config.metrics[0].confidence, "medium");
});

test("source mapping reset restores automatic and removes the override transactionally", async () => {
  const card = new SourceCard();
  const { config, dashboard } = sourceSaveFixture();
  config.overrides.p13112 = "sensor.manual";
  dashboard.views[0].cards[0].entity = "sensor.manual";
  dashboard.views[0].cards[1].entity = "sensor.manual";
  dashboard.views[1].cards[0].overrides.p13112 = "sensor.manual";
  let saved;
  card.setConfig(config);
  card.hass = { user: { is_admin: true }, states: { "sensor.auto": { state: "4", attributes: {} }, "sensor.manual": { state: "5", attributes: {} } }, callWS: async (request) => request.type === "lovelace/config" ? structuredClone(dashboard) : (saved = request.config) };
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
    card.hass = { user: { is_admin: true }, states: {}, callWS: async (request) => { if (failureAt === "read" || request.type === "lovelace/config/save") throw new Error("Not authorized"); return structuredClone(dashboard); } };
    await card._save("p13112", "sensor.manual");
    assert.equal(JSON.stringify(card._config.overrides), "{}", failureAt);
    assert.match(card._notice, /Not authorized/);
  }
});

test("source mapping candidate values and warnings follow live Home Assistant state", () => {
  const card = new SourceCard();
  card.setConfig({ schema_version: 1, mapping_id: "target", defaults: { p13112: "sensor.production", p13116: "sensor.direct" }, overrides: {}, metrics: [{ key: "p13112", group: "today_energy", default: "sensor.production", candidates: [] }, { key: "p13116", group: "today_energy", default: "sensor.direct", candidates: [{ entity_id: "sensor.direct", reason: "Compatible" }], validation: { schema_version: 1, rules: [{ type: "not_materially_greater_than", metric: "p13112", relative_tolerance: 0.05, absolute_tolerance: 0.1 }] } }], labels: { groups: { today_energy: "Today" } } });
  card.hass = { user: { is_admin: true }, states: { "sensor.production": { state: "42.8", attributes: { unit_of_measurement: "kWh" } }, "sensor.direct": { state: "52.6", attributes: { unit_of_measurement: "kWh", friendly_name: "Live direct solar" } } } };
  assert.equal(card._status(card._metric("p13116")), "needs_review");
  assert.match(card._candidate(card._metric("p13116"), card._metric("p13116").candidates[0]), /Live direct solar/);
  assert.match(card._candidate(card._metric("p13116"), card._metric("p13116").candidates[0]), /52\.6/);
  card._hass.states["sensor.direct"].state = "40";
  assert.equal(card._status(card._metric("p13116")), "automatic");
  delete card._hass.states["sensor.direct"];
  assert.equal(card._status(card._metric("p13116")), "unavailable");
});

test("source mapping stale warning follows live timestamps and clears after an update", () => {
  const card = new SourceCard();
  card.setConfig({ schema_version: 1, mapping_id: "target", defaults: { pv_power: "sensor.power" }, overrides: {}, metrics: [{ key: "pv_power", group: "live_power", default: "sensor.power", validation: { schema_version: 1, rules: [{ type: "freshness", max_age_seconds: 1800 }] } }], labels: { groups: { live_power: "Live power" } } });
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
  const chart = card._chartDisplay("day", card._metricDefinitions());
  const markup = card._renderChart(chart);

  assert.match(markup, /data-bucket-index="0"/);
  assert.match(markup, /data-series-key="production"/);
  assert.match(markup, /tabindex="0"/);
  assert.match(markup, /role="img"/);
  assert.match(markup, /aria-label="[^"]*Production[^"]*7\.00 kWh/);
  assert.match(markup, /class="chart-tooltip hidden"/);
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
