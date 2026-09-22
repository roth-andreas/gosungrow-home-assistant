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
    const services = await socket.call({ type: "get_services" });
    if (services.lovelace?.reload_resources) {
      await socket.call({ type: "call_service", domain: "lovelace", service: "reload_resources" });
    }
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
    page.on("pageerror", (error) => pageErrors.push(error.message));
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
    assert.deepEqual(pageErrors, [], `browser page errors: ${pageErrors.join(" | ")}`);
  } finally {
    await browser.close();
  }
}

await waitForHomeAssistant();
await verifyResource();
const token = await finishOnboarding();
await provisionDashboard(token);
await verifyBrowser();
console.log("Home Assistant managed-dashboard browser smoke passed");
