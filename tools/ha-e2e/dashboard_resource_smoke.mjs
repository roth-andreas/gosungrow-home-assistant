import assert from "node:assert/strict";
import { chromium } from "playwright";

const baseURL = (process.env.HA_E2E_URL || "http://127.0.0.1:8123").replace(/\/$/, "");
const resourceURL = process.env.HA_E2E_RESOURCE_URL;
const expectedHash = process.env.HA_E2E_RESOURCE_HASH;
const clientID = `${baseURL}/`;

assert(resourceURL, "HA_E2E_RESOURCE_URL is required");
assert.match(expectedHash || "", /^[0-9a-f]{64}$/, "HA_E2E_RESOURCE_HASH must be a full lowercase SHA-256 hash");

async function waitForHomeAssistant() {
  const deadline = Date.now() + 180_000;
  while (Date.now() < deadline) {
    try {
      const response = await fetch(`${baseURL}/api/onboarding`);
      if (response.ok) return;
    } catch {}
    await new Promise((resolve) => setTimeout(resolve, 2_000));
  }
  throw new Error("Home Assistant did not become ready within 180 seconds");
}

async function request(path, { token, body, form } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  let requestBody;
  if (form) {
    headers["Content-Type"] = "application/x-www-form-urlencoded";
    requestBody = new URLSearchParams(form);
  } else if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    requestBody = JSON.stringify(body);
  }
  const response = await fetch(`${baseURL}${path}`, { method: "POST", headers, body: requestBody });
  const text = await response.text();
  if (!response.ok) throw new Error(`${path} returned ${response.status}: ${text.slice(0, 500)}`);
  return text ? JSON.parse(text) : {};
}

async function finishOnboarding() {
  const status = await (await fetch(`${baseURL}/api/onboarding`)).json();
  const completed = new Set(status.filter((entry) => entry.done).map((entry) => entry.step));
  if (completed.has("user")) {
    throw new Error("The disposable Home Assistant fixture unexpectedly already contains a user");
  }
  const user = await request("/api/onboarding/users", {
    body: { name: "GoSungrow CI", username: "gosungrow-ci", password: "gosungrow-ci-password", client_id: clientID, language: "en" },
  });
  const tokens = await request("/auth/token", {
    form: { grant_type: "authorization_code", code: user.auth_code, client_id: clientID },
  });
  const token = tokens.access_token;
  await request("/api/onboarding/core_config", { token, body: {} });
  await request("/api/onboarding/analytics", { token, body: {} });
  await request("/api/onboarding/integration", {
    token,
    body: { client_id: clientID, redirect_uri: `${baseURL}/?auth_callback=1` },
  });
  return token;
}

async function websocketClient(token) {
  const endpoint = baseURL.replace(/^http/, "ws") + "/api/websocket";
  const socket = new WebSocket(endpoint);
  let nextID = 0;
  const pending = new Map();
  await new Promise((resolve, reject) => {
    socket.addEventListener("error", () => reject(new Error("Home Assistant websocket connection failed")), { once: true });
    socket.addEventListener("message", (event) => {
      const message = JSON.parse(event.data);
      if (message.type === "auth_required") socket.send(JSON.stringify({ type: "auth", access_token: token }));
      else if (message.type === "auth_ok") resolve();
      else if (message.type === "auth_invalid") reject(new Error(`Home Assistant websocket auth failed: ${message.message}`));
    });
  });
  socket.addEventListener("message", (event) => {
    const message = JSON.parse(event.data);
    const waiter = pending.get(message.id);
    if (!waiter) return;
    pending.delete(message.id);
    if (message.success) waiter.resolve(message.result);
    else waiter.reject(new Error(`${message.error?.code || "unknown"}: ${message.error?.message || "websocket request failed"}`));
  });
  return {
    call(payload) {
      const id = ++nextID;
      return new Promise((resolve, reject) => {
        pending.set(id, { resolve, reject });
        socket.send(JSON.stringify({ id, ...payload }));
      });
    },
    close() { socket.close(); },
  };
}

async function provisionDashboard(token) {
  const socket = await websocketClient(token);
  try {
    await socket.call({ type: "lovelace/resources/create", url: resourceURL, res_type: "module" });
    const resources = await socket.call({ type: "lovelace/resources/list" });
    assert(resources.some((resource) => resource.url === resourceURL && (resource.type || resource.res_type) === "module"), "Home Assistant must re-read the exact registered module");
    await socket.call({
      type: "lovelace/dashboards/create",
      url_path: "gosungrow-flow",
      title: "GoSungrow Flow",
      icon: "mdi:solar-power",
      show_in_sidebar: true,
      require_admin: false,
    });
    await socket.call({
      type: "lovelace/config/save",
      url_path: "gosungrow-flow",
      config: {
        title: "GoSungrow Flow",
        views: [{
          title: "Overview",
          path: "overview",
          cards: [
            { type: "custom:gosungrow-energy-flow-card-v2", entities: {} },
            { type: "custom:gosungrow-energy-summary-card-v1", entities: {} },
            { type: "custom:gosungrow-source-mapping-card-v1", schema_version: 1, mapping_id: "ci", dashboard_url_path: "gosungrow-flow", metrics: [], candidates: {}, defaults: {}, overrides: {}, bindings: {}, labels: {} },
          ],
        }],
      },
    });
    const saved = await socket.call({ type: "lovelace/config", url_path: "gosungrow-flow", force: true });
    assert.equal(saved.views?.[0]?.cards?.length, 3, "Home Assistant must re-read the saved managed dashboard");
  } finally {
    socket.close();
  }
}

async function verifyResource() {
  const response = await fetch(`${baseURL}${resourceURL}`);
  assert.equal(response.status, 200, "registered module must be fetchable");
  assert.match(response.headers.get("content-type") || "", /(?:text|application)\/(?:javascript|ecmascript)/i, "registered module must use JavaScript MIME");
  const bytes = new Uint8Array(await response.arrayBuffer());
  const digest = Buffer.from(await crypto.subtle.digest("SHA-256", bytes)).toString("hex");
  assert.equal(digest, expectedHash, "registered module bytes must match the expected hash");
}

async function verifyBrowser() {
  const browser = await chromium.launch({ headless: true });
  try {
    const page = await browser.newPage();
    const pageErrors = [];
    page.on("pageerror", (error) => pageErrors.push(error.stack || error.message));
    await page.goto(`${baseURL}/gosungrow-flow/overview`, { waitUntil: "networkidle", timeout: 90_000 });
    await page.waitForFunction(() => [
      "gosungrow-energy-flow-card-v2",
      "gosungrow-energy-summary-card-v1",
      "gosungrow-source-mapping-card-v1",
    ].every((name) => Boolean(customElements.get(name))), undefined, { timeout: 60_000 });
    const result = await page.evaluate(() => {
      const visit = (root) => {
        const found = [...root.querySelectorAll("hui-error-card, hui-warning-element")];
        for (const element of root.querySelectorAll("*")) {
          if (element.shadowRoot) found.push(...visit(element.shadowRoot));
        }
        return found;
      };
      return {
        cards: window.customCards.filter((entry) => entry?.type?.startsWith("gosungrow-")).map((entry) => entry.type).sort(),
        errors: visit(document).map((element) => element.textContent?.trim() || element.localName),
      };
    });
    assert.deepEqual(result.cards, [
      "gosungrow-energy-flow-card-v2",
      "gosungrow-energy-summary-card-v1",
      "gosungrow-source-mapping-card-v1",
    ]);
    assert.deepEqual(result.errors, [], `Home Assistant rendered configuration errors: ${result.errors.join(" | ")}`);
    const originalViewport = page.viewportSize();
    for (const width of [940, 480]) {
      await page.setViewportSize({ width, height: originalViewport.height });
      await page.evaluate(async (width) => {
        await document.fonts.ready;
        const chargingLabels = [
          ["en-US", "Battery charging"],
          ["de-DE", "Akku wird geladen"],
          ["sv-SE", "Batteriet laddas"],
          ["es-ES", "Cargando la batería"],
          ["fr-FR", "Battery charging"],
        ];
        const entities = { pv_to_battery_power: "sensor.smoke_charge", battery_to_load_power: "sensor.smoke_discharge", battery_soc: "sensor.smoke_soc" };
        const states = {
          [entities.pv_to_battery_power]: { state: "1.80", attributes: { unit_of_measurement: "kW" } },
          [entities.battery_to_load_power]: { state: "0", attributes: { unit_of_measurement: "kW" } },
          [entities.battery_soc]: { state: "81", attributes: { unit_of_measurement: "%" } },
        };
        const card = document.createElement("gosungrow-energy-flow-card-v2");
        Object.assign(card.style, { position: "fixed", top: "0", left: "0", width: `${width}px` });
        document.body.append(card);
        try {
          const fixtures = chargingLabels.flatMap(([language, charging]) => [
            { language, charging, labels: undefined },
            { language, charging, labels: { battery_status_charging: charging } },
          ]);
          for (const { language, charging, labels } of fixtures) {
            card.setConfig({ entities, labels });
            card.hass = { locale: { language }, states };
            const root = card.shadowRoot;
            await root.querySelector("ha-card").updateComplete;
            const layout = `${width}px, ${labels ? "configured" : "built-in"} ${language} labels`;
            if (root.querySelector(".shell").classList.contains("compact") !== (width < 700)) {
              throw new Error(`Unexpected Live Flow layout at ${layout}`);
            }
            const status = [...root.querySelectorAll("svg text")].find((text) => text.textContent === charging);
            const battery = root.querySelector('.node-button[data-node="battery"]');
            if (!status || !battery.getAttribute("aria-label").endsWith(charging)) {
              throw new Error(`Missing visible or accessible battery charging status at ${layout}`);
            }
            const bounds = status.getBoundingClientRect();
            const canvas = status.ownerSVGElement.getBoundingClientRect();
            const power = root.querySelector('[data-node="battery"][data-chip="power"]').getBoundingClientRect();
            const icon = battery.querySelector(".node-ring").getBoundingClientRect();
            if (bounds.width <= 0 || bounds.left < canvas.left || bounds.bottom > canvas.bottom ||
                bounds.right > icon.left || bounds.top < power.bottom) {
              throw new Error(`Battery charging status clips or overlaps at ${layout} ${JSON.stringify({ bounds, canvas, power, icon })}`);
            }
          }
        } finally {
          card.remove();
        }
      }, width);
    }
    await page.setViewportSize(originalViewport);
    const policyChecks = await page.evaluate(async () => {
      const hass = document.querySelector("home-assistant").hass;
      const findDeep = (root, selector) => {
        const direct = root.querySelector(selector);
        if (direct) return direct;
        for (const element of root.querySelectorAll("*")) if (element.shadowRoot) { const found = findDeep(element.shadowRoot, selector); if (found) return found; }
        return null;
      };
      const mount = findDeep(document, "hui-view") || findDeep(document, "hui-root");
      if (!mount) throw new Error("Home Assistant Lovelace context is missing");
      const check = (condition, message) => { if (!condition) throw new Error(message); };
      const oldEntity = "sensor.gosungrow_e2e_total_energy";
      const replacement = "sensor.gosungrow_e2e_daily_consumption";
      const production = "sensor.gosungrow_e2e_daily_production";
      const fields = (entity_id, verification, display_decision, selection_origin = "automatic") => ({ entity_id, selection_origin, verification, display_decision, reason: display_decision === "normal" ? "Source meaning verified" : "Source withheld. Review Data Sources." });
      const consumption = fields(oldEntity, "unverified", "withheld");
      const produced = fields(production, "verified", "normal");
      const editor = document.createElement("gosungrow-source-mapping-card-v1");
      const mapping = {
        type: "custom:gosungrow-source-mapping-card-v1", schema_version: 1, matcher_version: 5, mapping_id: "ci", dashboard_url_path: "gosungrow-flow",
        defaults: { p13199: oldEntity, p13112: production }, pinned_defaults: { p13199: oldEntity, p13112: production }, overrides: {}, recommendations: { p13199: replacement },
        candidates: { p13199: [{ entity_id: oldEntity, verification: "unverified", compatibility: "unverified", recommended: false, selectable: false }, { entity_id: replacement, verification: "verified", compatibility: "compatible", recommended: true, confidence: "high" }] },
        metrics: [{ key: "p13199", label: "Consumption", group: "energy_summary", default: oldEntity, ...consumption, validation: { schema_version: 2, rules: [] } }, { key: "p13112", label: "Production", group: "today_energy", default: production, ...produced, validation: { schema_version: 2, rules: [] } }],
        bindings: { p13199: [], p13112: [] }, labels: {},
      };
      const tileSource = { mapping_id: "ci", metrics: { p13199: structuredClone(consumption) }, native_template: { type: "tile", entity: oldEntity, name: "Consumption", layout_options: { grid_columns: 6 } }, native_bindings: { p13199: ["/entity"] } };
      const graphSource = { mapping_id: "ci", metrics: { p13199: structuredClone(consumption), p13112: structuredClone(produced) }, native_template: { type: "history-graph", title: "Preserved energy graph", hours_to_show: 24, entities: [{ entity: oldEntity, name: "Consumption" }, { entity: production, name: "Production" }] }, native_bindings: { p13199: ["/entities/0/entity"], p13112: ["/entities/1/entity"] } };
      const nativeCards = [tileSource, graphSource].map((source, i) => {
        const projection = editor._projectNative(source);
        for (const [key, paths] of Object.entries(projection.bindings)) mapping.bindings[key].push(...paths.map((path) => `/views/0/cards/${i}${path}`));
        return projection.card;
      });
      const summaryConfig = { type: "custom:gosungrow-energy-summary-card-v1", entities: { consumption: oldEntity, production }, source_policy: { mapping_id: "ci", metrics: { p13199: structuredClone(consumption), p13112: structuredClone(produced) } } };
      for (const [metric, key] of [["p13199", "consumption"], ["p13112", "production"]]) mapping.bindings[metric].push(`/views/0/cards/2/entities/${key}`, `/views/0/cards/2/source_policy/metrics/${metric}/entity_id`);
      const config = { title: "Policy fixture", views: [{ title: "Overview", path: "overview", cards: [...nativeCards, summaryConfig, mapping] }] };
      await hass.callWS({ type: "lovelace/config/save", url_path: "gosungrow-flow", config });
      editor.setConfig(mapping);
      editor.hass = hass;
      editor._refreshDashboardView = () => {};
      mount.append(editor);
      const summary = document.createElement("gosungrow-energy-summary-card-v1");
      summary.setConfig(summaryConfig);
      summary.hass = hass;
      mount.append(summary);
      check(!summary._metricDisplay(summary._metricDefinitions().find((metric) => metric.key === "consumption"), "day").available, "unverified daily total leaked");
      for (const period of ["day", "month", "year"]) check(!summary._chartDisplay(period, summary._metricDefinitions()).series.some((series) => series.key === "consumption"), "withheld history leaked");
      check(editor.shadowRoot.textContent.includes("Source withheld"), "Data Sources omitted the withholding explanation");
      const helpers = await window.loadCardHelpers();
      const native = helpers.createCardElement(nativeCards[1]);
      native.hass = hass;
      mount.append(native);
      await native.updateComplete;
      check(nativeCards[1].cards[1].entities.length === 1 && nativeCards[1].cards[1].entities[0].entity === production, "withholding removed another graph series");
      await editor._save("p13199", replacement);
      check(editor._config.defaults.p13199 === replacement, `adoption failed: ${editor._notice}`);
      const saved = await hass.callWS({ type: "lovelace/config", url_path: "gosungrow-flow", force: true });
      check(saved.views[0].cards[0].type === "tile" && saved.views[0].cards[0].entity === replacement, "tile did not restore");
      check(saved.views[0].cards[1].type === "history-graph" && saved.views[0].cards[1].entities[1].entity === production, "graph layout or other series changed");
      check(saved.views[0].cards[1].hours_to_show === 24 && saved.views[0].cards[1].title === "Preserved energy graph", "native non-source properties changed");
      for (const path of editor._config.bindings.p13199) check(editor._getPointer(saved, path) === replacement, `stale persisted binding: ${path}`);
      summary.setConfig(saved.views[0].cards[2]);
      summary.hass = hass;
      check(summary._metricDisplay(summary._metricDefinitions().find((metric) => metric.key === "consumption"), "day").available, "adopted daily source stayed hidden");
      const source = structuredClone(saved.views[0].cards[1].source_policy);
      Object.assign(source.metrics.p13199, fields(replacement, "incompatible", "withheld", "manual"));
      const withheldAgain = editor._projectNative(source).card;
      check(withheldAgain.cards[1].entities.length === 1 && withheldAgain.cards[1].entities[0].entity === production, "restored source could not be withheld independently");
      return { nativeTile: true, nativeGraph: true, policySave: true, bindingReadback: true, historyWithheld: true };
    });
    assert(Object.values(policyChecks).every(Boolean), "source-policy browser checks must pass");
    assert.deepEqual(pageErrors, [], `browser page errors: ${pageErrors.join(" | ")}`);
  } finally {
    await browser.close();
  }
}

await waitForHomeAssistant();
await verifyResource();
const token = await finishOnboarding();
for (const [entity, state, unit] of [["sensor.gosungrow_e2e_total_energy", "9.92", "MWh"], ["sensor.gosungrow_e2e_daily_consumption", "7.5", "kWh"], ["sensor.gosungrow_e2e_daily_production", "4", "kWh"]]) {
  await request(`/api/states/${entity}`, { token, body: { state, attributes: { unit_of_measurement: unit, state_class: "total_increasing" } } });
}
await provisionDashboard(token);
await verifyBrowser();
console.log("Home Assistant managed-dashboard browser smoke passed");
