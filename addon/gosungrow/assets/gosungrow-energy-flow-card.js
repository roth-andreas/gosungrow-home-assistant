class GoSungrowEnergyFlowCard extends HTMLElement {
  static getStubConfig() {
    return {
      type: "custom:gosungrow-energy-flow-card",
      title: "Live Energy Flow",
      entities: {
        solar_power: "sensor.example_solar_power",
        load_power: "sensor.example_load_power",
        battery_power: "sensor.example_battery_power",
        battery_soc: "sensor.example_battery_soc",
        grid_power: "sensor.example_grid_power",
        pv_to_load_power: "sensor.example_pv_to_load_power",
        pv_to_battery_power: "sensor.example_pv_to_battery_power",
        pv_to_grid_power: "sensor.example_pv_to_grid_power",
        battery_to_load_power: "sensor.example_battery_to_load_power",
        grid_to_load_power: "sensor.example_grid_to_load_power",
      },
    };
  }

  setConfig(config) {
    if (!config || !config.entities) {
      throw new Error("Missing required config.entities");
    }
    this._config = config;
    if (!this.shadowRoot) {
      this.attachShadow({ mode: "open" });
    }
    this._render();
  }

  set hass(hass) {
    this._hass = hass;
    this._render();
  }

  getCardSize() {
    return 6;
  }

  _render() {
    if (!this.shadowRoot || !this._config) {
      return;
    }

    const layout = this._layout();
    const nodes = this._nodeDisplays();
    const flows = this._flowDisplays();

    const edgeMarkup = Object.entries(layout.edges)
      .map(([key, edge]) => this._renderEdge(edge, flows[key]))
      .join("");

    const edgeLabels = Object.entries(layout.edges)
      .map(([key, edge]) => this._renderEdgeLabel(edge, flows[key]))
      .join("");

    const nodeMarkup = Object.entries(layout.nodes)
      .map(([key, node]) => this._renderNode(key, node, nodes))
      .join("");

    this.shadowRoot.innerHTML = `
      <style>
        :host {
          display: block;
        }

        ha-card {
          overflow: hidden;
        }

        .title {
          padding: 18px 20px 0;
          font-size: 1rem;
          font-weight: 600;
          color: var(--primary-text-color);
        }

        .shell {
          padding: 10px 14px 14px;
        }

        svg {
          display: block;
          width: 100%;
          height: auto;
        }

        .stage {
          border-radius: 20px;
          overflow: hidden;
          background:
            radial-gradient(circle at 50% 16%, rgba(255,255,255,0.05), transparent 34%),
            linear-gradient(180deg, rgba(255,255,255,0.025), rgba(255,255,255,0.01)),
            var(--card-background-color, #1f1f1f);
          box-shadow: inset 0 1px 0 rgba(255,255,255,0.04);
        }

        .edge-base {
          fill: none;
          stroke: rgba(148, 163, 184, 0.22);
          stroke-width: 5;
          stroke-linecap: round;
        }

        .edge-active {
          fill: none;
          stroke-linecap: round;
          opacity: 0.95;
          transition: stroke-width 180ms ease, opacity 180ms ease;
        }

        .edge-solar-home {
          stroke: #f59e0b;
        }

        .edge-solar-grid {
          stroke: #8b5cf6;
        }

        .edge-solar-battery {
          stroke: #ec4899;
        }

        .edge-battery-home {
          stroke: #14b8a6;
        }

        .edge-grid-home {
          stroke: #cbd5e1;
        }

        .node-ring {
          fill: rgba(15, 23, 42, 0.18);
          stroke-width: 3;
        }

        .node-fill {
          fill: rgba(15, 23, 42, 0.55);
        }

        .node-button {
          cursor: pointer;
        }

        .node-button[role="presentation"] {
          cursor: default;
        }

        .node-hit {
          fill: transparent;
        }

        .solar-ring {
          stroke: #f59e0b;
        }

        .home-ring {
          stroke: #f59e0b;
        }

        .battery-ring {
          stroke: #ec4899;
        }

        .grid-ring {
          stroke: #60a5fa;
        }

        .node-icon {
          fill: none;
          stroke: rgba(255,255,255,0.92);
          stroke-width: 2.4;
          stroke-linecap: round;
          stroke-linejoin: round;
        }

        .node-value {
          fill: var(--primary-text-color);
          font-size: 16px;
          font-weight: 700;
          text-anchor: middle;
        }

        .node-subvalue {
          font-size: 12px;
          font-weight: 700;
          text-anchor: middle;
        }

        .battery-soc {
          fill: #5eead4;
        }

        .node-label {
          fill: var(--secondary-text-color);
          font-size: 14px;
          font-weight: 500;
          text-anchor: middle;
        }

        .route-pill rect {
          stroke: rgba(255,255,255,0.08);
          stroke-width: 1;
        }

        .route-pill text {
          font-size: 11px;
          font-weight: 700;
          text-anchor: middle;
          dominant-baseline: middle;
        }

        .route-pill.inactive {
          opacity: 0.38;
        }

        .pill-solar-home rect {
          fill: rgba(245, 158, 11, 0.18);
        }

        .pill-solar-home text {
          fill: #fbbf24;
        }

        .pill-solar-grid rect {
          fill: rgba(139, 92, 246, 0.18);
        }

        .pill-solar-grid text {
          fill: #c4b5fd;
        }

        .pill-solar-battery rect {
          fill: rgba(236, 72, 153, 0.18);
        }

        .pill-solar-battery text {
          fill: #f9a8d4;
        }

        .pill-battery-home rect {
          fill: rgba(20, 184, 166, 0.18);
        }

        .pill-battery-home text {
          fill: #99f6e4;
        }

        .pill-grid-home rect {
          fill: rgba(148, 163, 184, 0.16);
        }

        .pill-grid-home text {
          fill: #e2e8f0;
        }
      </style>
      <ha-card>
        ${this._config.title ? `<div class="title">${this._escape(this._config.title)}</div>` : ""}
        <div class="shell">
          <div class="stage">
            <svg viewBox="0 0 ${layout.width} ${layout.height}" preserveAspectRatio="xMidYMid meet">
              ${edgeMarkup}
              ${edgeLabels}
              ${nodeMarkup}
            </svg>
          </div>
        </div>
      </ha-card>
    `;

    this.shadowRoot.querySelectorAll(".node-button[data-entity]").forEach((node) => {
      node.addEventListener("click", () => {
        const entityId = node.getAttribute("data-entity");
        if (!entityId) {
          return;
        }
        this._fire("hass-more-info", { entityId });
      });
    });
  }

  _layout() {
    return {
      width: 1000,
      height: 590,
      radius: 58,
      nodes: {
        solar: { x: 500, y: 105, label: "PV", ringClass: "solar-ring" },
        grid: { x: 260, y: 290, label: "Grid", ringClass: "grid-ring" },
        home: { x: 740, y: 290, label: "Home", ringClass: "home-ring" },
        battery: { x: 500, y: 465, label: "Battery", ringClass: "battery-ring" },
      },
      edges: {
        pv_to_grid_power: {
          path: "M456 154 C393 176 330 210 294 246",
          labelX: 394,
          labelY: 202,
          edgeClass: "edge-solar-grid",
          pillClass: "pill-solar-grid",
        },
        pv_to_load_power: {
          path: "M544 154 C607 176 670 210 706 246",
          labelX: 606,
          labelY: 202,
          edgeClass: "edge-solar-home",
          pillClass: "pill-solar-home",
        },
        pv_to_battery_power: {
          path: "M500 164 C500 230 500 300 500 404",
          labelX: 500,
          labelY: 248,
          edgeClass: "edge-solar-battery",
          pillClass: "pill-solar-battery",
        },
        grid_to_load_power: {
          path: "M324 290 C415 290 585 290 676 290",
          labelX: 500,
          labelY: 322,
          edgeClass: "edge-grid-home",
          pillClass: "pill-grid-home",
        },
        battery_to_load_power: {
          path: "M544 423 C584 392 648 346 700 314",
          labelX: 627,
          labelY: 382,
          edgeClass: "edge-battery-home",
          pillClass: "pill-battery-home",
        },
      },
    };
  }

  _nodeDisplays() {
    return {
      solar: this._entityDisplay("solar_power"),
      grid: this._entityDisplay("grid_power"),
      home: this._entityDisplay("load_power"),
      battery: this._entityDisplay("battery_power"),
      batterySoc: this._entityDisplay("battery_soc"),
    };
  }

  _flowDisplays() {
    return {
      pv_to_grid_power: this._entityDisplay("pv_to_grid_power"),
      pv_to_load_power: this._entityDisplay("pv_to_load_power"),
      pv_to_battery_power: this._entityDisplay("pv_to_battery_power"),
      grid_to_load_power: this._entityDisplay("grid_to_load_power"),
      battery_to_load_power: this._entityDisplay("battery_to_load_power"),
    };
  }

  _renderEdge(edge, display) {
    const magnitude = Math.abs(display.numericValue);
    const active = magnitude > 0.01;
    const width = active ? 4 + Math.min(magnitude, 6) * 1.6 : 3.5;
    const opacity = active ? 1 : 0.14;

    return `
      <path class="edge-base" d="${edge.path}"></path>
      <path class="edge-active ${edge.edgeClass}" d="${edge.path}" style="stroke-width:${width};opacity:${opacity};"></path>
    `;
  }

  _renderEdgeLabel(edge, display) {
    const width = Math.max(70, display.formatted.length * 7.2);
    const active = Math.abs(display.numericValue) > 0.01;
    return `
      <g class="route-pill ${edge.pillClass}${active ? "" : " inactive"}" transform="translate(${edge.labelX} ${edge.labelY})">
        <rect x="${-width / 2}" y="-12" width="${width}" height="24" rx="12"></rect>
        <text x="0" y="1">${this._escape(display.formatted)}</text>
      </g>
    `;
  }

  _renderNode(key, node, displays) {
    const radius = this._layout().radius;
    const entityId = this._entityIdForNode(key);
    const display = displays[key];
    const labelY = node.y + radius + 24;
    const iconMarkup = this._renderIcon(key, node.x, node.y - 12);
    const valueY = key === "battery" ? node.y + 18 : node.y + 20;
    const batterySoc = displays.batterySoc;
    return `
      <g class="node-button" ${entityId ? `data-entity="${this._escape(entityId)}"` : `role="presentation"`}>
        <circle class="node-hit" cx="${node.x}" cy="${node.y}" r="${radius + 18}"></circle>
        <circle class="node-ring ${node.ringClass}" cx="${node.x}" cy="${node.y}" r="${radius}"></circle>
        <circle class="node-fill" cx="${node.x}" cy="${node.y}" r="${radius - 2}"></circle>
        ${iconMarkup}
        <text class="node-value" x="${node.x}" y="${valueY}">${this._escape(display.formatted)}</text>
        ${key === "battery" ? `<text class="node-subvalue battery-soc" x="${node.x}" y="${node.y + 36}">${this._escape(batterySoc.formatted)}</text>` : ""}
        <text class="node-label" x="${node.x}" y="${labelY}">${this._escape(node.label)}</text>
      </g>
    `;
  }

  _renderIcon(key, x, y) {
    switch (key) {
      case "solar":
        return `
          <g class="node-icon" transform="translate(${x} ${y})">
            <circle cx="0" cy="-14" r="5"></circle>
            <line x1="-10" y1="-14" x2="-15" y2="-14"></line>
            <line x1="10" y1="-14" x2="15" y2="-14"></line>
            <line x1="0" y1="-24" x2="0" y2="-19"></line>
            <path d="M-18 12 L-10 -8 H10 L18 12 Z"></path>
            <line x1="-10" y1="-1" x2="10" y2="-1"></line>
            <line x1="-14" y1="5" x2="14" y2="5"></line>
            <line x1="-7" y1="-8" x2="-7" y2="11"></line>
            <line x1="0" y1="-8" x2="0" y2="11"></line>
            <line x1="7" y1="-8" x2="7" y2="11"></line>
          </g>
        `;
      case "home":
        return `
          <g class="node-icon" transform="translate(${x} ${y})">
            <path d="M-18 1 L0 -14 L18 1"></path>
            <path d="M-13 1 V18 H13 V1"></path>
            <path d="M2 -4 L-3 7 H3 L-2 18"></path>
          </g>
        `;
      case "battery":
        return `
          <g class="node-icon" transform="translate(${x} ${y})">
            <rect x="-12" y="-18" width="24" height="36" rx="4"></rect>
            <rect x="-4" y="-24" width="8" height="6" rx="2"></rect>
            <rect x="-5" y="-4" width="10" height="14" rx="2"></rect>
          </g>
        `;
      case "grid":
        return `
          <g class="node-icon" transform="translate(${x} ${y})">
            <path d="M0 -22 L-12 18"></path>
            <path d="M0 -22 L12 18"></path>
            <path d="M-8 -6 H8"></path>
            <path d="M-10 6 H10"></path>
            <path d="M-4 18 H4"></path>
            <path d="M-12 18 H12"></path>
            <path d="M-9 0 L0 -10 L9 0"></path>
          </g>
        `;
      default:
        return "";
    }
  }

  _entityIdForNode(key) {
    const mapping = {
      solar: "solar_power",
      grid: "grid_power",
      home: "load_power",
      battery: "battery_power",
    };
    return this._config?.entities?.[mapping[key]] || "";
  }

  _entityDisplay(key) {
    const entityId = this._config?.entities?.[key];
    const stateObj = entityId && this._hass ? this._hass.states[entityId] : null;
    const unit = stateObj?.attributes?.unit_of_measurement || "";

    if (!stateObj) {
      return { formatted: "--", numericValue: 0 };
    }

    const numericValue = Number.parseFloat(stateObj.state);
    if (!Number.isFinite(numericValue)) {
      return { formatted: stateObj.state, numericValue: 0 };
    }

    return {
      formatted: this._formatNumber(numericValue, unit),
      numericValue,
    };
  }

  _formatNumber(value, unit) {
    if (unit === "%") {
      return `${new Intl.NumberFormat(this._locale(), {
        minimumFractionDigits: 0,
        maximumFractionDigits: 0,
      }).format(value)} ${unit}`;
    }

    return `${new Intl.NumberFormat(this._locale(), {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(value)}${unit ? ` ${unit}` : ""}`;
  }

  _locale() {
    return this._hass?.locale?.language || navigator.language || "en-US";
  }

  _escape(value) {
    return String(value)
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");
  }

  _fire(type, detail) {
    this.dispatchEvent(
      new CustomEvent(type, {
        detail,
        bubbles: true,
        composed: true,
      }),
    );
  }
}

customElements.define("gosungrow-energy-flow-card", GoSungrowEnergyFlowCard);

window.customCards = window.customCards || [];
window.customCards.push({
  type: "gosungrow-energy-flow-card",
  name: "GoSungrow Energy Flow Card",
  description: "Custom Sungrow energy flow card with Energy dashboard-inspired layout.",
});
