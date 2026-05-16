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
      };
      return this.shadowRoot;
    }
  },
  Intl,
  Math,
  Number,
  Object,
  Promise,
  String,
  URLSearchParams,
  customElements: {
    define: (name, elementClass) => registry.set(name, elementClass),
  },
  navigator: { language: "en-US" },
  window: { customCards: [] },
};

vm.createContext(sandbox);
vm.runInContext(fs.readFileSync(assetPath, "utf8"), sandbox, { filename: assetPath });

const SummaryCard = registry.get("gosungrow-energy-summary-card-v1");
const entityID = "sensor.gosungrow_daily_production";

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

function row(date, sum) {
  return { start: `${date}T00:00:00.000Z`, sum };
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
    row("2026-05-14", 10),
    row("2026-05-15", 14),
    row("2026-05-16", 100),
  ];

  assert.deepEqual(values(chartFor(card, "month", rows)), [7]);
  assert.equal(card._headlineStatValue("month", entityID).value, 7);

  assert.deepEqual(values(chartFor(card, "year", rows)), [7]);
  assert.equal(card._headlineStatValue("year", entityID).value, 7);
});

test("current day recorder rows are ignored and replaced by live daily state", () => {
  const card = createCard("2026-05-16T12:00:00.000Z", 3);
  const rows = [
    row("2026-05-15", 10),
    row("2026-05-16", 100),
  ];

  assert.deepEqual(values(chartFor(card, "day", rows)), [3]);
  assert.equal(card._headlineStatValue("day", entityID).value, 3);
});

test("month boundary keeps previous month total and starts current month from live day", () => {
  const card = createCard("2026-06-01T12:00:00.000Z", 2);
  const rows = [
    row("2026-05-29", 0),
    row("2026-05-30", 4),
    row("2026-05-31", 9),
    row("2026-06-01", 99),
  ];

  const chart = chartFor(card, "month", rows);
  assert.deepEqual(bucketKeys(chart), ["2026-05", "2026-06"]);
  assert.deepEqual(values(chart), [9, 2]);
  assert.equal(card._headlineStatValue("month", entityID).value, 2);
});

test("year boundary keeps previous year total and starts current year from live day", () => {
  const card = createCard("2027-01-01T12:00:00.000Z", 2);
  const rows = [
    row("2026-12-30", 0),
    row("2026-12-31", 5),
    row("2027-01-01", 99),
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
});
