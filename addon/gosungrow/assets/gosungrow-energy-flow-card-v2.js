class GoSungrowEnergyFlowCard extends HTMLElement {
  static getStubConfig() {
    return {
      type: "custom:gosungrow-energy-flow-card-v2",
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
    return 4;
  }

  getGridOptions() {
    return {
      rows: 6,
      columns: "full",
      min_rows: 5,
      min_columns: 6,
    };
  }

  _render() {
    if (!this.shadowRoot || !this._config) {
      return;
    }

    const layout = this._layout(this._isCompact());
    const nodes = this._nodeDisplays();
    const flows = this._flowDisplays();

    const edgeMarkup = Object.entries(layout.edges)
      .map(([key, edge]) => this._renderEdge(edge, flows[key]))
      .join("");

    const edgeLabels = Object.entries(layout.edges)
      .map(([key, edge]) => this._renderEdgeLabel(edge, flows[key]))
      .join("");

    const nodeMarkup = Object.entries(layout.nodes)
      .map(([key, node]) => this._renderNode(key, node, nodes, layout))
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
          padding: 10px 12px 0;
          font-size: 0.94rem;
          font-weight: 600;
          color: var(--primary-text-color);
        }

        .shell {
          padding: 10px 8px 8px;
        }

        svg {
          display: block;
          width: 100%;
          height: auto;
        }

        .stage {
          border-radius: 16px;
          overflow: hidden;
          background:
            radial-gradient(circle at 50% 16%, rgba(255,255,255,0.04), transparent 32%),
            linear-gradient(180deg, rgba(255,255,255,0.025), rgba(255,255,255,0.01)),
            var(--card-background-color, #1f1f1f);
          box-shadow: inset 0 1px 0 rgba(255,255,255,0.04);
        }

        .edge-base {
          fill: none;
          stroke: rgba(148, 163, 184, 0.18);
          stroke-width: 4;
          stroke-linecap: round;
        }

        .edge-active {
          fill: none;
          stroke-linecap: round;
          opacity: 0.95;
          transition: stroke-width 180ms ease, opacity 180ms ease;
        }

        .edge-dot {
          opacity: 0;
        }

        .edge-dot.active {
          opacity: 1;
        }

        .edge-dot circle {
          filter: drop-shadow(0 0 5px rgba(255,255,255,0.18));
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
          stroke-width: 2.2;
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
          stroke-width: 2.1;
          stroke-linecap: round;
          stroke-linejoin: round;
        }

        .node-label {
          fill: var(--secondary-text-color);
          font-size: 11px;
          font-weight: 450;
          text-anchor: middle;
        }

        .node-chip rect {
          stroke-width: 1;
          filter: drop-shadow(0 1px 3px rgba(0,0,0,0.18));
        }

        .node-chip text {
          font-size: 12px;
          font-weight: 700;
          text-anchor: middle;
          dominant-baseline: middle;
          font-variant-numeric: tabular-nums;
        }

        .node-chip-solar rect,
        .node-chip-home rect {
          fill: rgba(245, 158, 11, 0.18);
          stroke: rgba(245, 158, 11, 0.42);
        }

        .node-chip-solar text,
        .node-chip-home text {
          fill: #fbbf24;
        }

        .node-chip-grid rect {
          fill: rgba(96, 165, 250, 0.16);
          stroke: rgba(96, 165, 250, 0.38);
        }

        .node-chip-grid text {
          fill: #93c5fd;
        }

        .node-chip-battery rect {
          fill: rgba(236, 72, 153, 0.16);
          stroke: rgba(236, 72, 153, 0.38);
        }

        .node-chip-battery text {
          fill: #f9a8d4;
        }

        .node-chip-soc rect {
          fill: rgba(45, 212, 191, 0.16);
          stroke: rgba(45, 212, 191, 0.38);
        }

        .node-chip-soc text {
          fill: #99f6e4;
        }

        .route-pill rect {
          stroke: rgba(255,255,255,0.08);
          stroke-width: 1;
        }

        .route-pill text {
          font-size: 9px;
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

  _isCompact() {
    const width = Math.max(this.clientWidth || 0, this.getBoundingClientRect?.().width || 0, window.innerWidth || 0);
    return width > 0 && width < 700;
  }

  _layout(compact) {
    if (compact) {
      return {
        width: 760,
        height: 430,
        radius: 44,
        nodes: {
          solar: { x: 380, y: 76, label: "PV", ringClass: "solar-ring", labelY: 140, powerChip: { x: 380, y: 16, className: "node-chip-solar" } },
          grid: { x: 194, y: 206, label: "Grid", ringClass: "grid-ring", labelY: 270, powerChip: { x: 116, y: 206, className: "node-chip-grid" } },
          home: { x: 566, y: 206, label: "Home", ringClass: "home-ring", labelY: 270, powerChip: { x: 644, y: 206, className: "node-chip-home" } },
          battery: { x: 380, y: 332, label: "Battery", ringClass: "battery-ring", labelY: 420, powerChip: { x: 300, y: 332, className: "node-chip-battery" }, socChip: { x: 380, y: 388, className: "node-chip-soc" } },
        },
        edges: {
          pv_to_grid_power: {
            path: "M350 110 C314 126 266 152 222 184",
            labelX: 292,
            labelY: 146,
            edgeClass: "edge-solar-grid",
            pillClass: "pill-solar-grid",
            dotDur: "4.6s",
          },
          pv_to_load_power: {
            path: "M410 110 C446 126 494 152 538 184",
            labelX: 468,
            labelY: 146,
            edgeClass: "edge-solar-home",
            pillClass: "pill-solar-home",
            dotDur: "4.2s",
          },
          pv_to_battery_power: {
            path: "M380 120 C380 170 380 222 380 286",
            labelX: 380,
            labelY: 204,
            edgeClass: "edge-solar-battery",
            pillClass: "pill-solar-battery",
            dotDur: "4.8s",
          },
          grid_to_load_power: {
            path: "M246 206 C320 204 440 204 514 206",
            labelX: 380,
            labelY: 236,
            edgeClass: "edge-grid-home",
            pillClass: "pill-grid-home",
            dotDur: "4.4s",
          },
          battery_to_load_power: {
            path: "M412 304 C448 278 490 246 536 216",
            labelX: 470,
            labelY: 276,
            edgeClass: "edge-battery-home",
            pillClass: "pill-battery-home",
            dotDur: "4.9s",
          },
        },
      };
    }

    return {
      width: 940,
      height: 320,
      radius: 38,
      nodes: {
        solar: { x: 470, y: 58, label: "PV", ringClass: "solar-ring", labelY: 120, powerChip: { x: 470, y: 18, className: "node-chip-solar" } },
        grid: { x: 272, y: 152, label: "Grid", ringClass: "grid-ring", labelY: 214, powerChip: { x: 196, y: 152, className: "node-chip-grid" } },
        home: { x: 668, y: 152, label: "Home", ringClass: "home-ring", labelY: 214, powerChip: { x: 744, y: 152, className: "node-chip-home" } },
        battery: { x: 470, y: 220, label: "Battery", ringClass: "battery-ring", labelY: 308, powerChip: { x: 394, y: 220, className: "node-chip-battery" }, socChip: { x: 470, y: 276, className: "node-chip-soc" } },
      },
      edges: {
        pv_to_grid_power: {
          path: "M442 88 C398 102 350 122 306 146",
          labelX: 378,
          labelY: 116,
          edgeClass: "edge-solar-grid",
          pillClass: "pill-solar-grid",
          dotDur: "4.6s",
        },
        pv_to_load_power: {
          path: "M498 88 C542 102 590 122 634 146",
          labelX: 562,
          labelY: 116,
          edgeClass: "edge-solar-home",
          pillClass: "pill-solar-home",
          dotDur: "4.2s",
        },
        pv_to_battery_power: {
          path: "M470 96 C470 132 470 170 470 208",
          labelX: 470,
          labelY: 148,
          edgeClass: "edge-solar-battery",
          pillClass: "pill-solar-battery",
          dotDur: "4.8s",
        },
        grid_to_load_power: {
          path: "M316 152 C392 150 548 150 624 152",
          labelX: 470,
          labelY: 180,
          edgeClass: "edge-grid-home",
          pillClass: "pill-grid-home",
          dotDur: "4.4s",
        },
        battery_to_load_power: {
          path: "M500 192 C544 176 586 164 630 156",
          labelX: 570,
          labelY: 194,
          edgeClass: "edge-battery-home",
          pillClass: "pill-battery-home",
          dotDur: "4.9s",
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
    const width = active ? 3.4 + Math.min(magnitude, 6) * 1.2 : 2.4;
    const opacity = active ? 0.96 : 0.12;
    const color = this._edgeColor(edge.edgeClass);

    return `
      <path class="edge-base" d="${edge.path}"></path>
      <path class="edge-active ${edge.edgeClass}" d="${edge.path}" style="stroke-width:${width};opacity:${opacity};"></path>
      <g class="edge-dot${active ? " active" : ""}">
        <circle r="${Math.max(3, width * 0.52)}" fill="${color}">
          <animateMotion dur="${edge.dotDur || "4.5s"}" repeatCount="indefinite" rotate="auto" path="${edge.path}" keyPoints="0;1" keyTimes="0;1"></animateMotion>
        </circle>
      </g>
    `;
  }

  _renderEdgeLabel(edge, display) {
    const active = Math.abs(display.numericValue) > 0.01;
    if (!active) {
      return "";
    }
    const width = Math.max(62, display.formatted.length * 6.8);
    return `
      <g class="route-pill ${edge.pillClass}" transform="translate(${edge.labelX} ${edge.labelY})">
        <rect x="${-width / 2}" y="-11" width="${width}" height="22" rx="11"></rect>
        <text x="0" y="1">${this._escape(display.formatted)}</text>
      </g>
    `;
  }

  _renderNode(key, node, displays, layout) {
    const radius = layout.radius;
    const entityId = this._entityIdForNode(key);
    const iconLayout = this._iconLayout(key, node);
    const iconMarkup = this._renderIcon(key, iconLayout.x, iconLayout.y, iconLayout.scale);
    const batterySoc = displays.batterySoc;
    const chips = [
      this._renderNodeChip(node.powerChip, displays[key].formatted),
      key === "battery" ? this._renderNodeChip(node.socChip, batterySoc.formatted) : "",
    ].join("");
    return `
      <g class="node-button" ${entityId ? `data-entity="${this._escape(entityId)}"` : `role="presentation"`}>
        <circle class="node-hit" cx="${node.x}" cy="${node.y}" r="${radius + 18}"></circle>
        <circle class="node-ring ${node.ringClass}" cx="${node.x}" cy="${node.y}" r="${radius}"></circle>
        <circle class="node-fill" cx="${node.x}" cy="${node.y}" r="${radius - 2}"></circle>
        ${iconMarkup}
        <text class="node-label" x="${node.x}" y="${node.labelY}">${this._escape(node.label)}</text>
      </g>
      ${chips}
    `;
  }

  _renderNodeChip(chip, text) {
    if (!chip || !text) {
      return "";
    }
    const width = Math.max(72, text.length * 7.2);
    return `
      <g class="node-chip ${chip.className}" transform="translate(${chip.x} ${chip.y})">
        <rect x="${-width / 2}" y="-13" width="${width}" height="26" rx="13"></rect>
        <text x="0" y="1">${this._escape(text)}</text>
      </g>
    `;
  }

  _iconLayout(key, node) {
    switch (key) {
      case "solar":
        return { x: node.x, y: node.y - 6, scale: 0.95 };
      case "home":
        return { x: node.x, y: node.y - 5, scale: 0.94 };
      case "battery":
        return { x: node.x, y: node.y - 6, scale: 0.92 };
      case "grid":
        return { x: node.x, y: node.y - 6, scale: 0.94 };
      default:
        return { x: node.x, y: node.y - 6, scale: 0.92 };
    }
  }

  _renderIcon(key, x, y, scale = 1) {
    const transform = `translate(${x} ${y}) scale(${scale})`;
    switch (key) {
      case "solar":
        return `
          <g class="node-icon" transform="${transform}">
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
          <g class="node-icon" transform="${transform}">
            <path d="M-18 1 L0 -14 L18 1"></path>
            <path d="M-13 1 V18 H13 V1"></path>
            <path d="M2 -4 L-3 7 H3 L-2 18"></path>
          </g>
        `;
      case "battery":
        return `
          <g class="node-icon" transform="${transform}">
            <rect x="-12" y="-18" width="24" height="36" rx="4"></rect>
            <rect x="-4" y="-24" width="8" height="6" rx="2"></rect>
            <rect x="-5" y="-4" width="10" height="14" rx="2"></rect>
          </g>
        `;
      case "grid":
        return `
          <g class="node-icon" transform="${transform}">
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

  _edgeColor(edgeClass) {
    switch (edgeClass) {
      case "edge-solar-home":
        return "#f59e0b";
      case "edge-solar-grid":
        return "#8b5cf6";
      case "edge-solar-battery":
        return "#ec4899";
      case "edge-battery-home":
        return "#14b8a6";
      case "edge-grid-home":
        return "#cbd5e1";
      default:
        return "#ffffff";
    }
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

customElements.define("gosungrow-energy-flow-card-v2", GoSungrowEnergyFlowCard);

window.customCards = window.customCards || [];
window.customCards.push({
  type: "gosungrow-energy-flow-card-v2",
  name: "GoSungrow Energy Flow Card v2",
  description: "Custom Sungrow energy flow card with Energy dashboard-inspired layout.",
});
