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

    const compact = this._isCompact();
    this._compact = compact;
    const layout = this._layout(compact);
    const flows = this._flowDisplays();
    const nodes = this._nodeDisplays(flows);

    const edgeMarkup = Object.entries(layout.edges)
      .map(([key, edge]) => this._renderEdge(key, edge, flows[key]))
      .join("");

    const edgeLabels = Object.entries(layout.edges)
      .map(([key, edge]) => this._renderEdgeLabel(key, edge, flows[key], nodes))
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
          display: block;
          overflow: hidden;
          background: var(--card-background-color, #1f1f1f);
          border-radius: var(--ha-card-border-radius, 12px);
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

        .shell.compact {
          padding: 8px 2px 6px;
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
          stroke: rgba(255,255,255,0.28);
          stroke-width: 1.2;
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
        <div class="shell${compact ? " compact" : ""}">
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
        width: 700,
        height: 560,
        radius: 40,
        nodes: {
          solar: { x: 350, y: 122, label: this._label("node_pv", "PV"), ringClass: "solar-ring", labelY: 182, powerChip: { x: 350, y: 40, className: "node-chip-solar" } },
          grid: { x: 154, y: 286, label: this._label("node_grid", "Grid"), ringClass: "grid-ring", labelY: 350, powerChip: { x: 94, y: 286, className: "node-chip-grid" } },
          home: { x: 546, y: 286, label: this._label("node_home", "Home"), ringClass: "home-ring", labelY: 350, powerChip: { x: 606, y: 286, className: "node-chip-home" } },
          battery: { x: 350, y: 442, label: this._label("node_battery", "Battery"), ringClass: "battery-ring", labelY: 516, powerChip: { x: 350, y: 366, className: "node-chip-battery" }, socChip: { x: 350, y: 492, className: "node-chip-soc" } },
        },
        edges: {
          pv_to_grid_power: {
            path: "M323 145 Q264 195 182 263",
            labelX: 272,
            labelY: 216,
            edgeClass: "edge-solar-grid",
            pillClass: "pill-solar-grid",
            dotDur: "4.6s",
          },
          pv_to_load_power: {
            path: "M377 145 Q436 195 518 263",
            labelX: 428,
            labelY: 216,
            edgeClass: "edge-solar-home",
            pillClass: "pill-solar-home",
            dotDur: "4.2s",
          },
          pv_to_battery_power: {
            path: "M350 164 C350 236 350 306 350 396",
            labelX: 350,
            labelY: 286,
            edgeClass: "edge-solar-battery",
            pillClass: "pill-solar-battery",
            dotDur: "4.8s",
            hideLabelWhenNodeReflectsFlow: { node: "battery", sign: -1 },
          },
          grid_to_load_power: {
            path: "M194 286 L506 286",
            labelX: 350,
            labelY: 258,
            edgeClass: "edge-grid-home",
            pillClass: "pill-grid-home",
            dotDur: "4.4s",
          },
          battery_to_load_power: {
            path: "M378 420 Q447 356 518 308",
            labelX: 454,
            labelY: 352,
            edgeClass: "edge-battery-home",
            pillClass: "pill-battery-home",
            dotDur: "4.9s",
          },
        },
      };
    }

    return {
      width: 940,
      height: 340,
      radius: 38,
      nodes: {
          solar: { x: 470, y: 74, label: this._label("node_pv", "PV"), ringClass: "solar-ring", labelY: 136, powerChip: { x: 470, y: 20, className: "node-chip-solar" } },
          grid: { x: 190, y: 164, label: this._label("node_grid", "Grid"), ringClass: "grid-ring", labelY: 226, powerChip: { x: 104, y: 164, className: "node-chip-grid" } },
          home: { x: 750, y: 164, label: this._label("node_home", "Home"), ringClass: "home-ring", labelY: 226, powerChip: { x: 836, y: 164, className: "node-chip-home" } },
          battery: { x: 470, y: 244, label: this._label("node_battery", "Battery"), ringClass: "battery-ring", labelY: 328, powerChip: { x: 470, y: 180, className: "node-chip-battery" }, socChip: { x: 470, y: 300, className: "node-chip-soc" } },
      },
      edges: {
        pv_to_grid_power: {
          path: "M439 88 Q351 104 221 150",
          labelX: 350,
          labelY: 128,
          edgeClass: "edge-solar-grid",
          pillClass: "pill-solar-grid",
          dotDur: "4.6s",
        },
        pv_to_load_power: {
          path: "M501 88 Q589 104 719 150",
          labelX: 590,
          labelY: 128,
          edgeClass: "edge-solar-home",
          pillClass: "pill-solar-home",
          dotDur: "4.2s",
        },
        pv_to_battery_power: {
          path: "M470 112 C470 150 470 190 470 226",
          labelX: 470,
          labelY: 160,
          edgeClass: "edge-solar-battery",
          pillClass: "pill-solar-battery",
          dotDur: "4.8s",
          hideLabelWhenNodeReflectsFlow: { node: "battery", sign: -1 },
        },
        grid_to_load_power: {
          path: "M228 164 L712 164",
          labelX: 470,
          labelY: 168,
          edgeClass: "edge-grid-home",
          pillClass: "pill-grid-home",
          dotDur: "4.4s",
        },
        battery_to_load_power: {
          path: "M501 231 Q610 190 719 178",
          labelX: 604,
          labelY: 208,
          edgeClass: "edge-battery-home",
          pillClass: "pill-battery-home",
          dotDur: "4.9s",
        },
      },
    };
  }

  _nodeDisplays(flows) {
    const direct = {
      solar: this._entityDisplay("solar_power"),
      grid: this._entityDisplay("grid_power"),
      home: this._entityDisplay("load_power"),
      battery: this._entityDisplay("battery_power"),
      batterySoc: this._entityDisplay("battery_soc"),
    };

    const flowUnit =
      flows.pv_to_load_power.unit ||
      flows.pv_to_grid_power.unit ||
      flows.pv_to_battery_power.unit ||
      flows.grid_to_load_power.unit ||
      flows.battery_to_load_power.unit ||
      direct.solar.unit ||
      direct.home.unit ||
      direct.grid.unit ||
      direct.battery.unit;

    const computed = {
      solar: this._displayFromValue(
        flows.pv_to_load_power.numericValue + flows.pv_to_grid_power.numericValue + flows.pv_to_battery_power.numericValue,
        flowUnit,
        direct.solar,
        this._hasAnyDisplay([flows.pv_to_load_power, flows.pv_to_grid_power, flows.pv_to_battery_power]),
      ),
      grid: this._displayFromValue(
        flows.grid_to_load_power.numericValue - flows.pv_to_grid_power.numericValue,
        flowUnit,
        direct.grid,
        this._hasAnyDisplay([flows.grid_to_load_power, flows.pv_to_grid_power]),
      ),
      home: this._displayFromValue(
        flows.pv_to_load_power.numericValue + flows.grid_to_load_power.numericValue + flows.battery_to_load_power.numericValue,
        flowUnit,
        direct.home,
        this._hasAnyDisplay([flows.pv_to_load_power, flows.grid_to_load_power, flows.battery_to_load_power]),
      ),
      battery: this._displayFromValue(
        flows.battery_to_load_power.numericValue - flows.pv_to_battery_power.numericValue,
        flowUnit,
        direct.battery,
        this._hasAnyDisplay([flows.battery_to_load_power, flows.pv_to_battery_power]),
      ),
    };

    return {
      solar: this._manualNodeDisplay("solar_power", direct.solar, computed.solar),
      grid: this._manualNodeDisplay("grid_power", direct.grid, computed.grid),
      home: this._manualNodeDisplay("load_power", direct.home, computed.home),
      battery: this._manualNodeDisplay("battery_power", direct.battery, computed.battery),
      batterySoc: direct.batterySoc,
    };
  }

  _manualNodeDisplay(key, direct, automaticDisplay) {
    const configured = this._config?.entities?.[key];
    const automatic = this._config?.automatic_entities?.[key];
    const manuallySelected = configured && automatic && configured.toLowerCase() !== automatic.toLowerCase();
    return manuallySelected ? direct : automaticDisplay;
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

  _renderEdge(key, edge, display) {
    const magnitude = Math.abs(display.numericValue);
    const active = magnitude > 0.01;
    const width = active ? 3.4 + Math.min(magnitude, 6) * 1.2 : 0;
    const opacity = active ? 0.96 : 0;
    const color = this._edgeColor(edge.edgeClass);

    return `
      <path class="edge-base" data-edge="${key}" d="${edge.path}"></path>
      <path class="edge-active ${edge.edgeClass}" data-edge="${key}" d="${edge.path}" style="stroke-width:${width};opacity:${opacity};"></path>
      <g class="edge-dot${active ? " active" : ""}" data-edge="${key}">
        <circle r="${Math.max(4.5, width * 0.9)}" fill="${color}">
          <animateMotion dur="${edge.dotDur || "4.5s"}" repeatCount="indefinite" rotate="auto" path="${edge.path}" keyPoints="0;1" keyTimes="0;1"></animateMotion>
        </circle>
      </g>
    `;
  }

  _renderEdgeLabel(key, edge, display, nodes) {
    if (this._compact) {
      return "";
    }
    const active = Math.abs(display.numericValue) > 0.01;
    if (!active) {
      return "";
    }
    if (this._isRedundantEdgeLabel(edge, display, nodes)) {
      return "";
    }
    const width = Math.max(62, display.formatted.length * 6.8);
    return `
      <g class="route-pill ${edge.pillClass}" data-edge="${key}" transform="translate(${edge.labelX} ${edge.labelY})">
        <rect x="${-width / 2}" y="-11" width="${width}" height="22" rx="11"></rect>
        <text x="0" y="1">${this._escape(display.formatted)}</text>
      </g>
    `;
  }

  _isRedundantEdgeLabel(edge, display, nodes) {
    const rule = edge.hideLabelWhenNodeReflectsFlow;
    if (!rule || !nodes) {
      return false;
    }

    const nodeDisplay = nodes[rule.node];
    if (!nodeDisplay?.available) {
      return false;
    }

    const sign = Number.isFinite(rule.sign) ? rule.sign : 1;
    const expectedNodeValue = display.numericValue * sign;
    return Math.abs(nodeDisplay.numericValue - expectedNodeValue) < 0.01;
  }

  _renderNode(key, node, displays, layout) {
    const radius = layout.radius;
    const entityId = this._entityIdForNode(key);
    const iconLayout = this._iconLayout(key, node);
    const iconMarkup = this._renderIcon(key, iconLayout.x, iconLayout.y, iconLayout.scale);
    const batterySoc = displays.batterySoc;
    const chips = [
      this._renderNodeChip(key, "power", node.powerChip, displays[key].formatted),
      key === "battery" ? this._renderNodeChip(key, "soc", node.socChip, batterySoc.formatted) : "",
    ].join("");
    return `
      <g class="node-button" data-node="${key}" ${entityId ? `data-entity="${this._escape(entityId)}"` : `role="presentation"`}>
        <circle class="node-hit" cx="${node.x}" cy="${node.y}" r="${radius + 18}"></circle>
        <circle class="node-ring ${node.ringClass}" cx="${node.x}" cy="${node.y}" r="${radius}"></circle>
        <circle class="node-fill" cx="${node.x}" cy="${node.y}" r="${radius - 2}"></circle>
        ${iconMarkup}
        <text class="node-label" data-node-label="${key}" x="${node.x}" y="${node.labelY}">${this._escape(node.label)}</text>
      </g>
      ${chips}
    `;
  }

  _renderNodeChip(nodeKey, chipType, chip, text) {
    if (!chip || !text) {
      return "";
    }
    const width = Math.max(72, text.length * 7.2);
    return `
      <g class="node-chip ${chip.className}" data-node="${nodeKey}" data-chip="${chipType}" transform="translate(${chip.x} ${chip.y})">
        <rect x="${-width / 2}" y="-13" width="${width}" height="26" rx="13"></rect>
        <text x="0" y="1">${this._escape(text)}</text>
      </g>
    `;
  }

  _iconLayout(key, node) {
    switch (key) {
      case "solar":
        return { x: node.x, y: node.y + 6, scale: 0.95 };
      case "home":
        return { x: node.x, y: node.y - 2, scale: 0.94 };
      case "battery":
        return { x: node.x, y: node.y + 3, scale: 0.92 };
      case "grid":
        return { x: node.x, y: node.y + 2, scale: 0.94 };
      default:
        return { x: node.x, y: node.y, scale: 0.92 };
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
      return { formatted: "--", numericValue: 0, unit: "", available: false };
    }

    const numericValue = Number.parseFloat(stateObj.state);
    if (!Number.isFinite(numericValue)) {
      return { formatted: stateObj.state, numericValue: 0, unit, available: true };
    }

    return {
      formatted: this._formatNumber(numericValue, unit),
      numericValue,
      unit,
      available: true,
    };
  }

  _hasAnyDisplay(displays) {
    return displays.some((display) => display && display.available);
  }

  _displayFromValue(value, unit, fallback, useComputed) {
    if (!useComputed) {
      return fallback;
    }

    const normalized = Math.abs(value) < 0.005 ? 0 : value;
    return {
      formatted: this._formatNumber(normalized, unit || fallback?.unit || ""),
      numericValue: normalized,
      unit: unit || fallback?.unit || "",
      available: true,
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

  _label(key, fallback) {
    const configLabel = this._config?.labels?.[key];
    if (typeof configLabel === "string" && configLabel.trim() !== "") {
      return configLabel;
    }

    const locale = this._locale().toLowerCase().replaceAll("_", "-");
    const locales = this._labelsByLocale();
    const candidates = [locale];
    if (locale.includes("-")) {
      candidates.push(locale.split("-")[0]);
    }
    candidates.push("en");

    for (const candidate of candidates) {
      const table = locales[candidate];
      if (!table) {
        continue;
      }
      const localized = table[key];
      if (typeof localized === "string" && localized.trim() !== "") {
        return localized;
      }
    }
    return fallback;
  }

  _labelsByLocale() {
    return {
      en: {
        node_pv: "PV",
        node_grid: "Grid",
        node_home: "Home",
        node_battery: "Battery",
      },
      sv: {
        node_pv: "PV",
        node_grid: "Nät",
        node_home: "Hem",
        node_battery: "Batteri",
      },
      de: {
        node_pv: "PV",
        node_grid: "Netz",
        node_home: "Haus",
        node_battery: "Batterie",
      },
    };
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

class GoSungrowEnergySummaryCard extends HTMLElement {
  static getStubConfig() {
    return {
      type: "custom:gosungrow-energy-summary-card-v1",
      title: "Energy Summary",
      buckets: { day: 14, month: 12, year: 5 },
      entities: {
        production: "sensor.example_pv_yield",
        consumption: "sensor.example_consumption",
        to_grid: "sensor.example_to_grid",
        from_grid: "sensor.example_from_grid",
        to_battery: "sensor.example_to_battery",
        from_battery: "sensor.example_from_battery",
      },
    };
  }

  constructor() {
    super();
    this._period = "day";
    this._statsCache = {};
    this._pendingKey = "";
  }

  setConfig(config) {
    if (!config || !config.entities) {
      throw new Error("Missing required config.entities");
    }
    this._config = config;
    this._labels = config.labels || {};
    if (!this.shadowRoot) {
      this.attachShadow({ mode: "open" });
    }
    this._render();
  }

  set hass(hass) {
    this._hass = hass;
    this._render();
    this._ensureStatistics();
  }

  getCardSize() {
    return 6;
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

    const period = this._period || "day";
    const values = this._metricDefinitions().map((metric) => this._metricDisplay(metric, period));
    const chart = this._chartDisplay(period, values);
    const title = this._label("title", this._config.title || "Energy Summary");
    const status = this._statisticsStatus(period);

    this.shadowRoot.innerHTML = `
      <style>
        :host {
          display: block;
        }

        ha-card {
          display: block;
          overflow: hidden;
          background: var(--card-background-color, #1f1f1f);
          border-radius: var(--ha-card-border-radius, 12px);
        }

        .summary {
          padding: 14px 16px 16px;
        }

        .topbar {
          display: flex;
          align-items: center;
          justify-content: space-between;
          gap: 12px;
          margin-bottom: 14px;
        }

        .title {
          min-width: 0;
          color: var(--primary-text-color);
          font-size: 0.96rem;
          font-weight: 650;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }

        .periods {
          display: inline-grid;
          grid-template-columns: repeat(3, minmax(54px, 1fr));
          flex: 0 0 auto;
          padding: 3px;
          border-radius: 9px;
          background: rgba(148, 163, 184, 0.12);
          border: 1px solid rgba(148, 163, 184, 0.14);
        }

        button {
          appearance: none;
          min-width: 0;
          height: 30px;
          padding: 0 10px;
          color: var(--secondary-text-color);
          background: transparent;
          border: 0;
          border-radius: 7px;
          font: inherit;
          font-size: 0.82rem;
          font-weight: 650;
          cursor: pointer;
        }

        button.active {
          color: var(--primary-text-color);
          background: var(--card-background-color, rgba(255,255,255,0.08));
          box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
        }

        .metrics {
          display: grid;
          grid-template-columns: repeat(3, minmax(0, 1fr));
          gap: 10px;
        }

        .metric {
          min-width: 0;
          padding: 12px;
          border-radius: 8px;
          background: rgba(148, 163, 184, 0.09);
          border: 1px solid rgba(148, 163, 184, 0.1);
        }

        .metric-name {
          display: flex;
          align-items: center;
          gap: 7px;
          min-width: 0;
          color: var(--secondary-text-color);
          font-size: 0.78rem;
          font-weight: 600;
        }

        .dot {
          width: 8px;
          height: 8px;
          border-radius: 50%;
          flex: 0 0 auto;
        }

        .metric-value {
          margin-top: 8px;
          color: var(--primary-text-color);
          font-size: 1.22rem;
          font-weight: 750;
          line-height: 1.1;
          font-variant-numeric: tabular-nums;
          white-space: nowrap;
        }

        .metric-value.unavailable {
          color: var(--secondary-text-color);
          font-size: 0.9rem;
          font-weight: 650;
          white-space: normal;
        }

        .status {
          margin-top: 10px;
          color: var(--secondary-text-color);
          font-size: 0.78rem;
        }

        .chart-wrap {
          position: relative;
          margin-top: 14px;
          padding: 12px 10px 10px;
          border-radius: 8px;
          background: rgba(15, 23, 42, 0.16);
          border: 1px solid rgba(148, 163, 184, 0.1);
        }

        .chart {
          display: block;
          width: 100%;
          height: auto;
          min-height: 210px;
        }

        .axis,
        .chart-empty {
          fill: var(--secondary-text-color);
          font-size: 11px;
          font-weight: 600;
        }

        .gridline {
          stroke: rgba(148, 163, 184, 0.16);
          stroke-width: 1;
        }

        .bar {
          opacity: 0.9;
          transition: filter 120ms ease, opacity 120ms ease;
        }

        .bar[data-bucket-index] {
          cursor: pointer;
        }

        .bar[data-bucket-index]:hover,
        .bar[data-bucket-index]:focus-visible {
          filter: brightness(1.14);
          opacity: 1;
          outline: none;
        }

        .bar.empty {
          opacity: 0;
        }

        .chart-tooltip {
          position: absolute;
          z-index: 2;
          min-width: 190px;
          max-width: min(260px, calc(100% - 16px));
          padding: 10px 11px;
          color: var(--primary-text-color);
          background: var(--card-background-color, rgba(17, 24, 39, 0.96));
          border: 1px solid rgba(148, 163, 184, 0.2);
          border-radius: 8px;
          box-shadow: 0 10px 28px rgba(0, 0, 0, 0.32);
          pointer-events: none;
          transform: translate3d(0, 0, 0);
        }

        .chart-tooltip.hidden {
          display: none;
        }

        .tooltip-title {
          margin-bottom: 7px;
          font-size: 0.78rem;
          font-weight: 750;
        }

        .tooltip-row {
          display: grid;
          grid-template-columns: 8px minmax(0, 1fr) auto;
          align-items: center;
          gap: 7px;
          min-width: 0;
          font-size: 0.76rem;
          line-height: 1.45;
        }

        .tooltip-row .dot {
          width: 8px;
          height: 8px;
        }

        .tooltip-name {
          min-width: 0;
          overflow: hidden;
          color: var(--secondary-text-color);
          text-overflow: ellipsis;
          white-space: nowrap;
        }

        .tooltip-value {
          color: var(--primary-text-color);
          font-variant-numeric: tabular-nums;
          font-weight: 700;
          white-space: nowrap;
        }

        .legend {
          display: flex;
          flex-wrap: wrap;
          gap: 8px 14px;
          margin-top: 8px;
        }

        .legend-item {
          display: inline-flex;
          align-items: center;
          gap: 6px;
          min-width: 0;
          color: var(--secondary-text-color);
          font-size: 0.76rem;
          font-weight: 600;
        }

        @media (max-width: 720px) {
          .topbar {
            align-items: stretch;
            flex-direction: column;
          }

          .periods {
            width: 100%;
          }

          .metrics {
            grid-template-columns: repeat(2, minmax(0, 1fr));
          }

          .chart {
            min-height: 190px;
          }
        }
      </style>
      <ha-card>
        <div class="summary">
          <div class="topbar">
            <div class="title">${this._escape(title)}</div>
            <div class="periods" role="tablist" aria-label="${this._escape(title)}">
              ${this._renderPeriodButton("day", period)}
              ${this._renderPeriodButton("month", period)}
              ${this._renderPeriodButton("year", period)}
            </div>
          </div>
          <div class="metrics">
            ${values.map((metric) => this._renderMetric(metric)).join("")}
          </div>
          ${this._renderChart(chart)}
          ${status ? `<div class="status">${this._escape(status)}</div>` : ""}
        </div>
      </ha-card>
    `;

    this.shadowRoot.querySelectorAll("button[data-period]").forEach((button) => {
      button.addEventListener("click", () => {
        const nextPeriod = button.getAttribute("data-period");
        if (!nextPeriod || nextPeriod === this._period) {
          return;
        }
        this._period = nextPeriod;
        this._render();
        this._ensureStatistics();
      });
    });
    this._bindChartTooltip(chart);
  }

  _renderPeriodButton(period, activePeriod) {
    const label = this._label(`period_${period}`, period);
    const active = period === activePeriod;
    return `
      <button type="button" data-period="${this._escape(period)}" class="${active ? "active" : ""}" role="tab" aria-selected="${active ? "true" : "false"}">
        ${this._escape(label)}
      </button>
    `;
  }

  _renderMetric(metric) {
    const valueClass = metric.available ? "metric-value" : "metric-value unavailable";
    return `
      <div class="metric">
        <div class="metric-name">
          <span class="dot" style="background:${metric.color};"></span>
          <span>${this._escape(metric.label)}</span>
        </div>
        <div class="${valueClass}">${this._escape(metric.display)}</div>
      </div>
    `;
  }

  _renderChart(chart) {
    const width = 720;
    const height = 230;
    const margin = { top: 16, right: 12, bottom: 34, left: 44 };
    const plotWidth = width - margin.left - margin.right;
    const plotHeight = height - margin.top - margin.bottom;

    if (!chart || chart.series.length === 0 || chart.buckets.length === 0) {
      return `
        <div class="chart-wrap">
          <svg class="chart" viewBox="0 0 ${width} ${height}" preserveAspectRatio="none">
            <text class="chart-empty" x="${width / 2}" y="${height / 2}" text-anchor="middle">${this._escape(this._label("no_statistics", "No statistics yet"))}</text>
          </svg>
        </div>
      `;
    }

    const values = chart.series.flatMap((series) => series.values).filter((value) => Number.isFinite(value));
    const max = Math.max(...values, 1);
    const yMax = this._niceMax(max);
    const ticks = [0, 0.25, 0.5, 0.75, 1].map((ratio) => yMax * ratio);
    const grid = ticks
      .map((tick) => {
        const y = margin.top + plotHeight - (tick / yMax) * plotHeight;
        return `
          <line class="gridline" x1="${margin.left}" y1="${y.toFixed(1)}" x2="${width - margin.right}" y2="${y.toFixed(1)}"></line>
          <text class="axis" x="${margin.left - 8}" y="${(y + 4).toFixed(1)}" text-anchor="end">${this._escape(this._formatAxis(tick))}</text>
        `;
      })
      .join("");

    const groupWidth = plotWidth / Math.max(chart.buckets.length, 1);
    const groupGap = Math.min(10, groupWidth * 0.18);
    const barGap = 2;
    const barWidth = Math.max(2, (groupWidth - groupGap - barGap * (chart.series.length - 1)) / chart.series.length);
    const bars = chart.buckets
      .map((bucket, bucketIndex) => {
        const groupX = margin.left + bucketIndex * groupWidth + groupGap / 2;
        return chart.series
          .map((series, seriesIndex) => {
            const value = series.values[bucketIndex];
            if (!Number.isFinite(value)) {
              return `<rect class="bar empty" x="0" y="0" width="0" height="0"></rect>`;
            }
            const heightValue = Math.max(0, (value / yMax) * plotHeight);
            const x = groupX + seriesIndex * (barWidth + barGap);
            const y = margin.top + plotHeight - heightValue;
            const label = `${bucket.label}, ${series.label}, ${this._formatEnergy(value, this._dayValue(this._entity(series.key)).unit || "kWh")}`;
            return `<rect class="bar" data-bucket-index="${bucketIndex}" data-series-key="${this._escape(series.key)}" tabindex="0" role="img" aria-label="${this._escape(label)}" x="${x.toFixed(1)}" y="${y.toFixed(1)}" width="${barWidth.toFixed(1)}" height="${heightValue.toFixed(1)}" rx="2" fill="${series.color}"></rect>`;
          })
          .join("");
      })
      .join("");

    const labels = chart.buckets
      .map((bucket, index) => {
        const x = margin.left + index * groupWidth + groupWidth / 2;
        return `<text class="axis" x="${x.toFixed(1)}" y="${height - 10}" text-anchor="middle">${this._escape(bucket.label)}</text>`;
      })
      .join("");

    const legend = chart.series
      .map((series) => `
        <span class="legend-item">
          <span class="dot" style="background:${series.color};"></span>
          <span>${this._escape(series.label)}</span>
        </span>
      `)
      .join("");

    return `
      <div class="chart-wrap">
        <svg class="chart" viewBox="0 0 ${width} ${height}" preserveAspectRatio="none">
          ${grid}
          ${bars}
          ${labels}
        </svg>
        <div class="chart-tooltip hidden" role="tooltip"></div>
        <div class="legend">${legend}</div>
      </div>
    `;
  }

  _bindChartTooltip(chart) {
    if (!this.shadowRoot) {
      return;
    }
    const tooltip = this.shadowRoot.querySelector(".chart-tooltip");
    const bars = this.shadowRoot.querySelectorAll(".bar[data-bucket-index]");
    if (!tooltip || !bars.length) {
      return;
    }

    bars.forEach((bar) => {
      const bucketIndex = Number.parseInt(bar.getAttribute("data-bucket-index"), 10);
      if (!Number.isFinite(bucketIndex)) {
        return;
      }
      bar.addEventListener("pointerenter", (event) => this._showChartTooltip(event, chart, bucketIndex));
      bar.addEventListener("pointermove", (event) => this._showChartTooltip(event, chart, bucketIndex));
      bar.addEventListener("pointerleave", () => this._hideChartTooltip());
      bar.addEventListener("focus", (event) => this._showChartTooltip(event, chart, bucketIndex));
      bar.addEventListener("blur", () => this._hideChartTooltip());
    });
  }

  _tooltipRows(chart, bucketIndex) {
    const bucket = chart?.buckets?.[bucketIndex];
    if (!bucket) {
      return [];
    }
    return chart.series
      .map((series) => {
        const value = series.values?.[bucketIndex];
        if (!Number.isFinite(value)) {
          return null;
        }
        const entityID = this._entity(series.key);
        return {
          key: series.key,
          label: series.label,
          color: series.color,
          value,
          display: this._formatEnergy(value, this._dayValue(entityID).unit || "kWh"),
        };
      })
      .filter(Boolean);
  }

  _showChartTooltip(event, chart, bucketIndex) {
    const tooltip = this.shadowRoot?.querySelector(".chart-tooltip");
    const wrapper = this.shadowRoot?.querySelector(".chart-wrap");
    const bucket = chart?.buckets?.[bucketIndex];
    const rows = this._tooltipRows(chart, bucketIndex);
    if (!tooltip || !wrapper || !bucket || rows.length === 0) {
      return;
    }

    tooltip.innerHTML = `
      <div class="tooltip-title">${this._escape(bucket.label)}</div>
      ${rows
        .map((row) => `
          <div class="tooltip-row">
            <span class="dot" style="background:${row.color};"></span>
            <span class="tooltip-name">${this._escape(row.label)}</span>
            <span class="tooltip-value">${this._escape(row.display)}</span>
          </div>
        `)
        .join("")}
    `;
    tooltip.classList.remove("hidden");

    const wrapperRect = wrapper.getBoundingClientRect();
    const tooltipRect = tooltip.getBoundingClientRect();
    const targetRect = event.currentTarget?.getBoundingClientRect?.();
    const pointerX = Number.isFinite(event.clientX)
      ? event.clientX - wrapperRect.left
      : targetRect
        ? targetRect.left + targetRect.width / 2 - wrapperRect.left
        : wrapperRect.width / 2;
    const pointerY = Number.isFinite(event.clientY)
      ? event.clientY - wrapperRect.top
      : targetRect
        ? targetRect.top - wrapperRect.top
        : wrapperRect.height / 2;
    const padding = 8;
    const left = Math.min(Math.max(pointerX + 12, padding), Math.max(padding, wrapperRect.width - tooltipRect.width - padding));
    const top = Math.min(Math.max(pointerY - tooltipRect.height - 10, padding), Math.max(padding, wrapperRect.height - tooltipRect.height - padding));
    tooltip.style.left = `${left}px`;
    tooltip.style.top = `${top}px`;
  }

  _hideChartTooltip() {
    const tooltip = this.shadowRoot?.querySelector(".chart-tooltip");
    if (!tooltip) {
      return;
    }
    tooltip.classList.add("hidden");
  }

  _metricDefinitions() {
    return [
      { key: "production", label: this._label("name_production", "Production"), color: "#f59e0b" },
      { key: "consumption", label: this._label("name_consumption", "Consumption"), color: "#38bdf8" },
      { key: "to_grid", label: this._label("name_to_grid", "To Grid"), color: "#8b5cf6" },
      { key: "from_grid", label: this._label("name_from_grid", "From Grid"), color: "#cbd5e1" },
      { key: "to_battery", label: this._label("name_to_battery", "To Battery"), color: "#ec4899" },
      { key: "from_battery", label: this._label("name_from_battery", "From Battery"), color: "#14b8a6" },
    ].filter((metric) => this._entity(metric.key));
  }

  _metricDisplay(metric, period) {
    const entityID = this._entity(metric.key);
    const source = period === "day" ? this._dayValue(entityID) : this._headlineStatValue(period, entityID);
    const available = Number.isFinite(source.value);
    return {
      ...metric,
      available,
      display: available ? this._formatEnergy(source.value, source.unit) : this._label("unavailable", "Unavailable"),
    };
  }

  _entity(key) {
    const value = this._config?.entities?.[key];
    return typeof value === "string" ? value.trim() : "";
  }

  _dayValue(entityID) {
    const state = this._hass?.states?.[entityID];
    if (!state) {
      return { value: NaN, unit: "kWh" };
    }
    return {
      value: Number.parseFloat(state.state),
      unit: state.attributes?.unit_of_measurement || "kWh",
    };
  }

  _headlineStatValue(period, entityID) {
    const cache = this._statsCache[this._cacheKey(period)] || {};
    const entry = cache.headline?.[entityID];
    if (!entry) {
      return this._dayValue(entityID);
    }
    return entry;
  }

  _chartDisplay(period, metrics) {
    const cache = this._statsCache[this._cacheKey(period)];
    const buckets = this._populatedBuckets(cache?.buckets || []);
    const series = metrics
      .map((metric) => {
        const entityID = this._entity(metric.key);
        const values = buckets.map((bucket) => bucket.values?.[entityID]);
        if (!values.some((value) => Number.isFinite(value))) {
          return null;
        }
        return {
          key: metric.key,
          label: metric.label,
          color: metric.color,
          values,
        };
      })
      .filter(Boolean);
    if (series.length === 0) {
      return this._liveChartDisplay(period, metrics);
    }
    return { buckets, series };
  }

  _liveChartDisplay(period, metrics) {
    const now = this._now();
    const bucket = {
      key: this._bucketKey(period, now),
      label: this._bucketLabel(period, now),
      values: {},
    };
    const series = metrics
      .map((metric) => {
        const entityID = this._entity(metric.key);
        const source = this._dayValue(entityID);
        if (!Number.isFinite(source.value)) {
          return null;
        }
        bucket.values[entityID] = source.value;
        return {
          key: metric.key,
          label: metric.label,
          color: metric.color,
          values: [source.value],
        };
      })
      .filter(Boolean);
    return { buckets: series.length > 0 ? [bucket] : [], series };
  }

  _populatedBuckets(buckets) {
    return buckets.filter((bucket) => {
      const values = Object.values(bucket.values || {});
      return values.some((value) => Number.isFinite(value));
    });
  }

  _statisticsStatus(period) {
    const key = this._cacheKey(period);
    if (this._pendingKey === key) {
      return "";
    }
    if (this._hasLiveMetricValues()) {
      return "";
    }
    if (this._statsCache[key]) {
      return "";
    }
    return this._label("statistics_unavailable", "Statistics unavailable");
  }

  _hasLiveMetricValues() {
    return this._metricDefinitions().some((metric) => {
      const entityID = this._entity(metric.key);
      return Number.isFinite(this._dayValue(entityID).value);
    });
  }

  _ensureStatistics() {
    if (!this._hass || typeof this._hass.callWS !== "function") {
      return;
    }
    const key = this._cacheKey(this._period);
    if (this._statsCache[key] || this._pendingKey === key) {
      return;
    }

    const range = this._statisticsRange(this._period);
    const entityIDs = this._metricDefinitions().map((metric) => this._entity(metric.key)).filter(Boolean);
    if (entityIDs.length === 0) {
      return;
    }

    this._pendingKey = key;
    this._hass
      .callWS({
        type: "recorder/statistics_during_period",
        start_time: range.start.toISOString(),
        end_time: range.end.toISOString(),
        statistic_ids: entityIDs,
        period: "day",
        types: ["max", "state", "sum"],
      })
      .then((response) => {
        this._statsCache[key] = this._parseStatistics(response, entityIDs, this._period);
      })
      .catch(() => {
        this._statsCache[key] = {};
      })
      .finally(() => {
        if (this._pendingKey === key) {
          this._pendingKey = "";
        }
        this._render();
      });
  }

  _periodRange(period) {
    const now = this._now();
    const start = new Date(now);
    const count = this._bucketCount(period);
    if (period === "day") {
      start.setDate(start.getDate() - count + 1);
    } else if (period === "month") {
      start.setDate(1);
      start.setMonth(start.getMonth() - count + 1);
    } else {
      start.setMonth(0, 1);
      start.setFullYear(start.getFullYear() - count + 1);
    }
    start.setHours(0, 0, 0, 0);
    return { start, end: now };
  }

  _statisticsRange(period) {
    const range = this._periodRange(period);
    const start = new Date(range.start);
    start.setDate(start.getDate() - 1);
    return { start, end: range.end };
  }

  _cacheKey(period) {
    const range = this._periodRange(period);
    return `${period}:${this._bucketCount(period)}:${range.start.toISOString().slice(0, 10)}`;
  }

  _bucketCount(period) {
    const defaults = { day: 14, month: 12, year: 5 };
    const configured = Number.parseInt(this._config?.buckets?.[period], 10);
    if (Number.isFinite(configured) && configured > 0 && configured <= 60) {
      return configured;
    }
    return defaults[period] || 12;
  }

  _parseStatistics(response, entityIDs, period) {
    const buckets = this._emptyBuckets(period);
    const bucketByKey = Object.fromEntries(buckets.map((bucket) => [bucket.key, bucket]));
    const headline = {};
    const todayKey = this._bucketKey("day", this._now());

    entityIDs.forEach((entityID) => {
      const rows = Array.isArray(response?.[entityID]) ? response[entityID] : [];
      const numericRows = rows
        .map((row) => ({
          start: row?.start,
          max: Number.parseFloat(row?.max),
          sum: Number.parseFloat(row?.sum),
          state: Number.parseFloat(row?.state),
        }))
        .filter((row) => this._isValidStatisticStart(row.start) && (Number.isFinite(row.max) || Number.isFinite(row.state) || Number.isFinite(row.sum)))
        .sort((a, b) => this._timeForStatisticStart(a.start) - this._timeForStatisticStart(b.start));

      if (numericRows.length > 0) {
        const bucketRows = this._bucketRows(numericRows);
        numericRows.forEach((row, rowIndex) => {
          if (this._bucketKeyForStart("day", row.start) === todayKey) {
            return;
          }
          const bucketKey = this._bucketKeyForStart(period, row.start);
          const bucket = bucketByKey[bucketKey];
          if (!bucket) {
            return;
          }
          const value = bucketRows[rowIndex];
          if (!Number.isFinite(value)) {
            return;
          }
          const current = bucket.values[entityID];
          bucket.values[entityID] = Number.isFinite(current) ? current + value : value;
        });
      }

      this._injectLiveValue(period, buckets, entityID);
      const currentBucket = bucketByKey[this._bucketKey(period, this._now())];
      const currentValue = currentBucket?.values?.[entityID];
      if (Number.isFinite(currentValue)) {
        headline[entityID] = {
          value: currentValue,
          unit: this._dayValue(entityID).unit || "kWh",
        };
      }
    });

    return { headline, buckets };
  }

  _injectLiveValue(period, buckets, entityID) {
    const source = this._dayValue(entityID);
    if (!Number.isFinite(source.value)) {
      return;
    }
    const bucketKey = this._bucketKey(period, this._now());
    const bucket = buckets.find((candidate) => candidate.key === bucketKey);
    if (!bucket) {
      return;
    }
    const current = bucket.values[entityID];
    bucket.values[entityID] = Number.isFinite(current) ? current + source.value : source.value;
  }

  _bucketRows(rows) {
    return rows.map((row) => {
      if (Number.isFinite(row.max)) {
        return row.max;
      }
      if (Number.isFinite(row.state)) {
        return row.state;
      }
      return row.sum;
    });
  }

  _emptyBuckets(period) {
    const now = this._now();
    const count = this._bucketCount(period);
    const buckets = [];
    for (let offset = count - 1; offset >= 0; offset -= 1) {
      const date = new Date(now);
      if (period === "day") {
        date.setDate(date.getDate() - offset);
      } else if (period === "month") {
        date.setDate(1);
        date.setMonth(date.getMonth() - offset);
      } else {
        date.setMonth(0, 1);
        date.setFullYear(date.getFullYear() - offset);
      }
      date.setHours(0, 0, 0, 0);
      buckets.push({
        key: this._bucketKey(period, date),
        label: this._bucketLabel(period, date),
        values: {},
      });
    }
    return buckets;
  }

  _bucketKeyForStart(period, value) {
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return "";
    }
    return this._bucketKey(period, date);
  }

  _isValidStatisticStart(value) {
    return Number.isFinite(this._timeForStatisticStart(value));
  }

  _timeForStatisticStart(value) {
    if (typeof value !== "string" && typeof value !== "number") {
      return NaN;
    }
    return new Date(value).getTime();
  }

  _bucketKey(period, date) {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const day = String(date.getDate()).padStart(2, "0");
    if (period === "day") {
      return `${year}-${month}-${day}`;
    }
    if (period === "month") {
      return `${year}-${month}`;
    }
    return String(year);
  }

  _bucketLabel(period, date) {
    const locale = this._locale();
    try {
      if (period === "day") {
        return new Intl.DateTimeFormat(locale, { day: "2-digit", month: "2-digit" }).format(date);
      }
      if (period === "month") {
        return new Intl.DateTimeFormat(locale, { month: "short" }).format(date);
      }
      return new Intl.DateTimeFormat(locale, { year: "2-digit" }).format(date);
    } catch (_) {
      if (period === "day") {
        return `${date.getMonth() + 1}/${date.getDate()}`;
      }
      if (period === "month") {
        return `${date.getMonth() + 1}`;
      }
      return String(date.getFullYear()).slice(-2);
    }
  }

  _formatEnergy(value, unit) {
    const normalizedUnit = unit || "kWh";
    const decimals = Math.abs(value) >= 100 ? 0 : Math.abs(value) >= 10 ? 1 : 2;
    try {
      return `${new Intl.NumberFormat(this._locale(), {
        minimumFractionDigits: decimals,
        maximumFractionDigits: decimals,
      }).format(value)} ${normalizedUnit}`;
    } catch (_) {
      return `${value.toFixed(decimals)} ${normalizedUnit}`;
    }
  }

  _formatAxis(value) {
    if (Math.abs(value) >= 1000) {
      return `${(value / 1000).toFixed(1)}k`;
    }
    if (Math.abs(value) >= 100) {
      return value.toFixed(0);
    }
    if (Math.abs(value) >= 10) {
      return value.toFixed(1);
    }
    return value.toFixed(2);
  }

  _niceMax(value) {
    if (!Number.isFinite(value) || value <= 0) {
      return 1;
    }
    const exponent = Math.floor(Math.log10(value));
    const magnitude = Math.pow(10, exponent);
    const normalized = value / magnitude;
    const nice = normalized <= 1 ? 1 : normalized <= 2 ? 2 : normalized <= 5 ? 5 : 10;
    return nice * magnitude;
  }

  _label(key, fallback) {
    const value = this._labels?.[key];
    if (typeof value === "string" && value.trim() !== "") {
      return value;
    }
    return fallback;
  }

  _locale() {
    return this._hass?.locale?.language || this._hass?.language || navigator.language || "en-US";
  }

  _now() {
    return new Date();
  }

  _escape(value) {
    return String(value)
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");
  }
}

class GoSungrowSourceMappingCard extends HTMLElement {
  static getStubConfig() {
    return { type: "custom:gosungrow-source-mapping-card-v1", schema_version: 1, mapping_id: "preview" };
  }

  constructor() {
    super();
    this.attachShadow({ mode: "open" });
    this._config = {};
    this._hass = null;
    this._activeMetric = null;
    this._search = "";
    this._busy = false;
    this._notice = "";
    this._lastFocus = null;
    this._lastFocusMetric = null;
    this._pendingEntity = null;
  }

  setConfig(config) {
    if (!config || Number(config.schema_version || 1) !== 1) throw new Error("Unsupported GoSungrow source mapping schema");
    this._config = JSON.parse(JSON.stringify(config));
    this._render();
  }

  set hass(hass) {
    this._hass = hass;
    this._render();
  }

  getCardSize() { return 8; }

  getGridOptions() { return { columns: 12, min_columns: 6 }; }

  connectedCallback() { this._render(); }

  _label(key, fallback) { return this._config?.labels?.[key] || fallback; }
  _isAdmin() { return Boolean(this._hass?.user?.is_admin); }
  _metrics() { return Array.isArray(this._config?.metrics) ? this._config.metrics : []; }
  _candidates(metric) { return this._config?.candidates?.[metric.key] || metric.candidates || []; }
  _selected(metric) { return this._config?.overrides?.[metric.key] || metric.selected || metric.default; }
  _state(entityID) { return entityID && this._hass?.states?.[entityID] ? this._hass.states[entityID] : null; }

  _numericState(entityID) {
    const state = this._state(entityID);
    if (!state || ["unknown", "unavailable", ""].includes(String(state.state ?? "").toLowerCase())) return null;
    const value = Number(state.state);
    return Number.isFinite(value) ? value : null;
  }

  _metric(key) { return this._metrics().find((metric) => metric.key === key); }

  _warning(metric, selectedEntity = this._selected(metric)) {
    if (this._numericState(selectedEntity) === null) return this._label("source_unavailable_warning", "The selected entity is unavailable or non-numeric.");
    const validation = metric?.validation;
    if (Number(validation?.schema_version) !== 1 || !Array.isArray(validation?.rules)) return "";
    for (const rule of validation.rules) {
      if (rule?.type === "freshness") {
        const state = this._state(selectedEntity);
        const updated = Date.parse(state?.last_updated || state?.last_changed || "");
        if (Number.isFinite(updated) && (updated > Date.now() + 300000 || Date.now() - updated > Number(rule.max_age_seconds || 0) * 1000)) {
          return this._label("source_stale_warning", "The selected entity has not updated recently.");
        }
        continue;
      }
      if (rule?.type !== "not_materially_greater_than") continue;
      const referenceMetric = this._metric(rule.metric);
      const value = this._numericState(selectedEntity);
      const reference = referenceMetric ? this._numericState(this._selected(referenceMetric)) : null;
      if (value === null || reference === null) continue;
      const relative = Number(rule.relative_tolerance || 0);
      const absolute = Number(rule.absolute_tolerance || 0);
      if (value > reference * (1 + relative) + absolute) {
        return this._label("source_physical_warning", "Selected value ({value}) exceeds solar production ({reference}). Review this source.")
          .replace("{value}", this._formatLiveValue(selectedEntity))
          .replace("{reference}", this._formatLiveValue(this._selected(referenceMetric)));
      }
    }
    return "";
  }

  _formatLiveValue(entityID) {
    const state = this._state(entityID);
    if (!state) return this._label("unavailable", "Unavailable");
    const unit = state.attributes?.unit_of_measurement || "";
    return `${state.state}${unit ? ` ${unit}` : ""}`;
  }

  _displayValue(metric) {
    const state = this._state(this._selected(metric));
    const value = state?.state;
    const unit = state?.attributes?.unit_of_measurement ?? "";
    if (value === undefined || value === null || ["unknown", "unavailable", ""].includes(String(value).toLowerCase())) return this._label("unavailable", "Unavailable");
    return `${value}${unit ? ` ${unit}` : ""}`;
  }

  _status(metric) {
    if (this._numericState(this._selected(metric)) === null) return "unavailable";
    if (this._warning(metric)) return "needs_review";
    if (this._config?.overrides?.[metric.key]) return "manual";
    return "automatic";
  }

  _statusLabel(status) {
    return { automatic: this._label("automatic", "Automatic"), manual: this._label("manual", "Manual"), needs_review: this._label("needs_review", "Needs review"), unavailable: this._label("unavailable", "Unavailable") }[status] || status;
  }

  _confidenceLabel(confidence) {
    return { high: this._label("confidence_high", "High confidence"), medium: this._label("confidence_medium", "Medium confidence"), low: this._label("confidence_low", "Low confidence"), manual: this._label("confidence_manual", "User selected") }[confidence] || this._label("confidence_unknown", "Confidence unavailable");
  }

  _render() {
    if (!this.shadowRoot) return;
    const labels = this._config?.labels || {};
    const groups = labels.groups || {};
    const order = ["live_power", "today_energy", "battery", "energy_summary"];
    const sections = order.map((group) => {
      const metrics = this._metrics().filter((metric) => metric.group === group);
      if (!metrics.length) return "";
      return `<section class="source-group"><h3>${this._escape(groups[group] || group)}</h3>${metrics.map((metric) => this._row(metric)).join("")}</section>`;
    }).join("");
    this.shadowRoot.innerHTML = `<style>${this._styles()}</style><ha-card>
      <div class="header"><div class="header-icon"><ha-icon icon="mdi:tune-variant"></ha-icon></div><div><h2>${this._escape(this._label("title", "Data Sources"))}</h2><p>${this._escape(this._label("subtitle", "Review automatic matches or choose a dashboard override."))}</p></div></div>
      ${!this._isAdmin() ? `<div class="readonly"><ha-icon icon="mdi:lock-outline"></ha-icon>${this._escape(this._label("readonly", "Only Home Assistant administrators can change data sources."))}</div>` : ""}
      <div class="groups">${sections || `<div class="empty">${this._escape(this._label("unavailable", "Unavailable"))}</div>`}</div>
      ${this._notice ? `<div class="toast" role="status">${this._escape(this._notice)}</div>` : ""}
    </ha-card>${this._activeMetric ? this._dialog(this._activeMetric) : ""}`;
    this._wire();
  }

  _row(metric) {
    const status = this._status(metric);
    const selected = this._selected(metric);
    const state = this._state(selected);
    const friendly = state?.attributes?.friendly_name || selected;
    const warning = this._warning(metric);
    return `<article class="metric-row ${status === "needs_review" ? "warning" : ""}">
      <div class="metric-icon"><ha-icon icon="${this._escape(metric.icon || "mdi:chart-line")}"></ha-icon></div>
      <div class="metric-main"><div class="metric-title-line"><strong>${this._escape(metric.label || metric.key)}</strong><span class="badge ${status}"><span class="badge-dot"></span>${this._escape(this._statusLabel(status))}</span></div>
      <div class="metric-value">${this._escape(this._displayValue(metric))}</div><div class="entity-name" title="${this._escape(selected)}">${this._escape(friendly || selected)}</div><div class="match-reason"><span>${this._escape(this._confidenceLabel(metric.confidence))}</span>${this._escape(metric.reason || this._label("source_compatible", "Compatible source"))}</div>${warning ? `<div class="warning-text"><ha-icon icon="mdi:alert-circle-outline"></ha-icon>${this._escape(warning)}</div>` : ""}</div>
      <button class="configure" data-metric="${this._escape(metric.key)}" ${!this._isAdmin() ? "disabled" : ""} aria-label="${this._escape(this._label("configure", "Configure"))} ${this._escape(metric.label || metric.key)}"><ha-icon icon="mdi:tune"></ha-icon><span>${this._escape(this._label("configure", "Configure"))}</span></button>
    </article>`;
  }

  _dialog(metric) {
    const candidates = this._candidates(metric);
    const query = this._search.trim().toLowerCase();
    const filtered = candidates.filter((candidate) => {
      const friendly = this._state(candidate.entity_id)?.attributes?.friendly_name || "";
      return !query || `${friendly} ${candidate.entity_id} ${candidate.device || ""}`.toLowerCase().includes(query);
    });
    const recommended = filtered.filter((candidate) => candidate.recommended).slice(0, 5);
    const other = filtered.filter((candidate) => !recommended.includes(candidate));
    const warning = this._warning(metric, this._pendingEntity || this._selected(metric));
    const changed = Boolean(this._pendingEntity && this._pendingEntity !== this._selected(metric));
    return `<div class="scrim" data-close="true"><div class="dialog" role="dialog" aria-modal="true" aria-labelledby="source-dialog-title">
      <div class="dialog-head"><div><div class="eyebrow">${this._escape(this._label("configure", "Configure"))}</div><h2 id="source-dialog-title">${this._escape(metric.label || metric.key)}</h2></div><button class="icon-button close" aria-label="${this._escape(this._label("cancel", "Cancel"))}"><ha-icon icon="mdi:close"></ha-icon></button></div>
      ${warning ? `<div class="dialog-warning"><ha-icon icon="mdi:alert-circle-outline"></ha-icon><div><strong>${this._escape(this._label("needs_review", "Needs review"))}</strong><span>${this._escape(warning)}</span></div></div>` : ""}
      <label class="search"><ha-icon icon="mdi:magnify"></ha-icon><input type="search" value="${this._escape(this._search)}" placeholder="${this._escape(this._label("search", "Search entities"))}" aria-label="${this._escape(this._label("search", "Search entities"))}"></label>
      <div class="candidate-scroll"><h3>${this._escape(this._label("recommended", "Recommended"))}</h3>${recommended.map((candidate) => this._candidate(metric, candidate, filtered)).join("") || `<div class="empty">No matching entities</div>`}
      ${other.length ? `<details ${query ? "open" : ""}><summary>${this._escape(this._label("other", "Other compatible entities"))}<span>${other.length}</span></summary>${other.map((candidate) => this._candidate(metric, candidate, filtered)).join("")}</details>` : ""}</div>
      <div class="dialog-actions">${this._config?.overrides?.[metric.key] ? `<button class="reset" data-reset="${this._escape(metric.key)}" ${this._busy ? "disabled" : ""}>${this._escape(this._label("reset", "Reset to automatic"))}</button>` : ""}<button class="cancel">${this._escape(this._label("cancel", "Cancel"))}</button><button class="use-source" data-commit="${this._escape(metric.key)}" ${!changed || this._busy ? "disabled" : ""}>${this._escape(this._label("use_source", "Use this source"))}</button></div>
    </div></div>`;
  }

  _candidate(metric, candidate, peers = this._candidates(metric)) {
    const selected = (this._pendingEntity || this._selected(metric)) === candidate.entity_id;
    const state = this._state(candidate.entity_id);
    const friendly = state?.attributes?.friendly_name || candidate.entity_id;
    const peerNames = peers.map((entry) => this._state(entry.entity_id)?.attributes?.friendly_name || entry.entity_id);
    const peerIDs = peers.map((entry) => entry.entity_id);
    const displayName = this._compactCandidateText(friendly, peerNames, 76);
    const displayID = this._compactCandidateText(candidate.entity_id, peerIDs, 68, true);
    const value = this._numericState(candidate.entity_id) === null ? this._label("unavailable", "Unavailable") : state.state;
    const unit = state?.attributes?.unit_of_measurement || "";
    const accessible = `${friendly}; ${candidate.entity_id}; ${value}${unit ? ` ${unit}` : ""}`;
    return `<button class="candidate ${selected ? "selected" : ""}" data-select="${this._escape(candidate.entity_id)}" data-metric="${this._escape(metric.key)}" aria-label="${this._escape(accessible)}">
      <span class="radio"><span></span></span><span class="candidate-main"><strong title="${this._escape(friendly)}">${this._escape(displayName)}</strong><code title="${this._escape(candidate.entity_id)}">${this._escape(displayID)}</code><small>${this._escape([candidate.device, candidate.point_id].filter(Boolean).join(" · "))}</small><small>${this._escape(candidate.reason || candidate.source || "Compatible entity")}</small></span><span class="candidate-value">${this._escape(value)}<small>${this._escape(unit)}</small></span>
    </button>`;
  }

  _compactCandidateText(value, peers, limit = 72, identifier = false) {
    const full = String(value || "");
    const values = [...new Set((peers || []).map((entry) => String(entry || "")).filter(Boolean))];
    let compact = full;
    if (values.length > 1) {
      const lower = values.map((entry) => entry.toLocaleLowerCase());
      let length = 0;
      while (lower.every((entry) => entry[length] && entry[length] === lower[0][length])) length += 1;
      const separators = identifier ? "_.-/: " : " _.-/:()";
      while (length > 0 && !separators.includes(full[length - 1])) length -= 1;
      if (length >= (identifier ? 10 : 4)) compact = full.slice(length).replace(/^[\s_.\-/:()]+/, "");
    }
    if (!compact) compact = full;
    if (compact.length <= limit) return compact;
    const suffix = compact.slice(-(limit - 1)).replace(/^[^\s_.\-/:()]*[\s_.\-/:()]+/, "");
    return `…${suffix || compact.slice(-(limit - 1))}`;
  }

  _wire() {
    this.shadowRoot.querySelectorAll("button[data-metric]:not([data-select])").forEach((button) => button.addEventListener("click", () => { this._lastFocus = button; this._lastFocusMetric = button.dataset.metric; this._activeMetric = this._metrics().find((metric) => metric.key === button.dataset.metric); this._pendingEntity = this._selected(this._activeMetric); this._search = ""; this._render(); queueMicrotask(() => this.shadowRoot.querySelector(".dialog .close")?.focus()); }));
    this.shadowRoot.querySelector(".close")?.addEventListener("click", () => this._close());
    this.shadowRoot.querySelector(".cancel")?.addEventListener("click", () => this._close());
    this.shadowRoot.querySelector(".scrim")?.addEventListener("click", (event) => { if (event.target?.dataset?.close) this._close(); });
    this.shadowRoot.querySelector("input[type=search]")?.addEventListener("input", (event) => { this._search = event.target.value; this._render(); queueMicrotask(() => { const input = this.shadowRoot.querySelector("input[type=search]"); input?.focus(); input?.setSelectionRange(this._search.length, this._search.length); }); });
    this.shadowRoot.querySelectorAll("[data-select]").forEach((button) => button.addEventListener("click", () => { const entity = button.dataset.select; this._pendingEntity = entity; this._render(); queueMicrotask(() => [...this.shadowRoot.querySelectorAll("[data-select]")].find((entry) => entry.dataset.select === entity)?.focus()); }));
    this.shadowRoot.querySelector("[data-commit]")?.addEventListener("click", (event) => this._save(event.currentTarget.dataset.commit, this._pendingEntity));
    this.shadowRoot.querySelector("[data-reset]")?.addEventListener("click", (event) => this._save(event.currentTarget.dataset.reset, null));
    this.shadowRoot.querySelector(".dialog")?.addEventListener("keydown", (event) => this._trapDialogKeys(event));
  }

  _close() { this._activeMetric = null; this._pendingEntity = null; this._search = ""; this._render(); queueMicrotask(() => ([...this.shadowRoot.querySelectorAll("button[data-metric]:not([data-select])")].find((button) => button.dataset.metric === this._lastFocusMetric) || this._lastFocus)?.focus()); }

  _trapDialogKeys(event) {
    if (event.key === "Escape") { event.preventDefault(); this._close(); return; }
    if (event.key !== "Tab") return;
    const focusable = [...this.shadowRoot.querySelectorAll(".dialog button:not([disabled]), .dialog input, .dialog summary")];
    if (!focusable.length) return;
    const first = focusable[0], last = focusable[focusable.length - 1];
    if (event.shiftKey && this.shadowRoot.activeElement === first) { event.preventDefault(); last.focus(); }
    else if (!event.shiftKey && this.shadowRoot.activeElement === last) { event.preventDefault(); first.focus(); }
  }

  async _save(metricKey, entityID) {
    if (this._busy || !this._isAdmin() || !this._hass?.callWS) return;
    const metric = this._metrics().find((entry) => entry.key === metricKey);
    if (!metric) return;
    const warning = entityID ? this._warning(metric, entityID) : "";
    if (warning && !window.confirm(`${warning}\n\n${this._label("source_confirm_warning", "Use this source anyway?")}`)) return;
    this._busy = true;
    try {
      const dashboard = await this._hass.callWS({ type: "lovelace/config", url_path: this._config.dashboard_url_path, force: true });
      const card = this._findMappingCard(dashboard, this._config.mapping_id);
      const stale = () => new Error(this._label("source_stale", "Dashboard changed; reload and try again."));
      if (!card || Number(card.schema_version) !== 1 || !this._sameMap(card.defaults, this._config.defaults) || !this._sameMap(card.overrides || {}, this._config.overrides || {}) || !this._sameValue(card.bindings, this._config.bindings)) throw stale();
      if (entityID) {
        const cardMetric = Array.isArray(card.metrics) ? card.metrics.find((entry) => entry.key === metricKey) : null;
        const localCandidates = this._candidates(metric);
        const cardCandidates = card.candidates?.[metricKey] || cardMetric?.candidates || [];
        const allowedLocally = localCandidates.some((candidate) => candidate.entity_id === entityID);
        const allowed = allowedLocally && cardCandidates.some((candidate) => candidate.entity_id === entityID);
        if (!allowed) throw new Error(this._label("source_incompatible", "This entity is no longer an available compatible source."));
      }
      const oldEntity = card.overrides?.[metricKey] || card.defaults?.[metricKey];
      const nextEntity = entityID || card.defaults?.[metricKey];
      const paths = card.bindings?.[metricKey] || [];
      for (const path of paths) {
        if (this._getPointer(dashboard, path) !== oldEntity) throw stale();
      }
      const nextDashboard = this._clone(dashboard);
      const nextCard = this._findMappingCard(nextDashboard, this._config.mapping_id);
      for (const path of paths) this._setPointer(nextDashboard, path, nextEntity);
      nextCard.overrides = { ...(nextCard.overrides || {}) };
      if (entityID) nextCard.overrides[metricKey] = entityID; else delete nextCard.overrides[metricKey];
      const nextMetric = Array.isArray(nextCard.metrics) ? nextCard.metrics.find((entry) => entry.key === metricKey) : null;
      const nextCandidate = (nextCard.candidates?.[metricKey] || nextMetric?.candidates || []).find((candidate) => candidate.entity_id === nextEntity);
      if (nextMetric) { nextMetric.confidence = nextCandidate?.confidence; nextMetric.reason = nextCandidate?.reason; }
      await this._hass.callWS({ type: "lovelace/config/save", url_path: this._config.dashboard_url_path, config: nextDashboard });
      const verifiedDashboard = await this._hass.callWS({ type: "lovelace/config", url_path: this._config.dashboard_url_path, force: true });
      const verifiedCard = this._findMappingCard(verifiedDashboard, this._config.mapping_id);
      const persistedEntity = verifiedCard?.overrides?.[metricKey] || verifiedCard?.defaults?.[metricKey];
      if (persistedEntity !== nextEntity || paths.some((path) => this._getPointer(verifiedDashboard, path) !== nextEntity)) {
        throw new Error(this._label("source_save_error", "Could not save the data source. Check your administrator access and connection, then try again."));
      }
      this._config.overrides = { ...(this._config.overrides || {}) };
      if (entityID) this._config.overrides[metricKey] = entityID; else delete this._config.overrides[metricKey];
      const localCandidate = this._candidates(metric).find((candidate) => candidate.entity_id === nextEntity);
      metric.confidence = localCandidate?.confidence;
      metric.reason = localCandidate?.reason;
      this._notice = this._label("saved", "Data source saved.");
      this._activeMetric = null;
      this._pendingEntity = null;
      this._render();
      this.dispatchEvent(new CustomEvent("hass-notification", { bubbles: true, composed: true, detail: { message: this._notice } }));
      queueMicrotask(() => this._refreshDashboardView());
    } catch (error) {
      this._notice = error?.message || this._label("source_save_error", "Could not save the data source. Check your administrator access and connection, then try again.");
      this._render();
    } finally { this._busy = false; }
  }

  _findMappingCard(value, mappingID) {
    if (!value || typeof value !== "object") return null;
    if (!Array.isArray(value) && value.type === "custom:gosungrow-source-mapping-card-v1" && value.mapping_id === mappingID) return value;
    for (const entry of Object.values(value)) { const found = this._findMappingCard(entry, mappingID); if (found) return found; }
    return null;
  }

  _pointerParts(path) { return String(path).split("/").slice(1).map((part) => part.replaceAll("~1", "/").replaceAll("~0", "~")); }
  _getPointer(root, path) { return this._pointerParts(path).reduce((value, key) => value?.[key], root); }
  _setPointer(root, path, next) { const parts = this._pointerParts(path); const key = parts.pop(); const parent = parts.reduce((value, part) => value?.[part], root); if (!parent || key === undefined) throw new Error("Dashboard changed; reload and try again."); parent[key] = next; }
  _clone(value) { return JSON.parse(JSON.stringify(value)); }
  _stable(value) {
    if (Array.isArray(value)) return value.map((entry) => this._stable(entry));
    if (value && typeof value === "object") return Object.fromEntries(Object.keys(value).sort().map((key) => [key, this._stable(value[key])]));
    return value ?? null;
  }
  _sameValue(left, right) { return JSON.stringify(this._stable(left)) === JSON.stringify(this._stable(right)); }
  _sameMap(left, right) {
    const normalize = (value) => Object.fromEntries(Object.entries(value || {}).sort(([a], [b]) => a.localeCompare(b)));
    return this._sameValue(normalize(left), normalize(right));
  }
  _refreshDashboardView() {
    this.dispatchEvent(new CustomEvent("config-refresh", { bubbles: true, composed: true }));
  }
  _escape(value) { return String(value ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;"); }

  _styles() { return `
    ha-card{display:block}
    :host{display:block;color:var(--primary-text-color);font-family:var(--paper-font-body1_-_font-family,system-ui,sans-serif)}ha-card{position:relative;padding:22px;border-radius:18px;overflow:hidden}.header{display:flex;gap:14px;align-items:flex-start;margin-bottom:18px}.header-icon{display:grid;place-items:center;width:44px;height:44px;border-radius:14px;background:color-mix(in srgb,var(--primary-color) 16%,transparent);color:var(--primary-color);flex:0 0 auto}.header h2{font-size:20px;line-height:1.2;margin:1px 0 5px}.header p{margin:0;color:var(--secondary-text-color);font-size:14px}.readonly,.dialog-warning{display:flex;gap:10px;align-items:center;border-radius:12px;padding:11px 13px;margin:0 0 16px;background:color-mix(in srgb,var(--warning-color,#f59e0b) 12%,transparent);color:var(--primary-text-color);font-size:13px}.groups{display:grid;gap:18px}.source-group{border:1px solid var(--divider-color);border-radius:16px;overflow:hidden;background:color-mix(in srgb,var(--card-background-color) 94%,var(--primary-text-color) 6%)}.source-group h3{font-size:13px;letter-spacing:.02em;text-transform:uppercase;color:var(--secondary-text-color);margin:0;padding:14px 16px 10px}.metric-row{display:grid;grid-template-columns:42px minmax(0,1fr) auto;gap:12px;align-items:center;padding:14px 16px;border-top:1px solid var(--divider-color)}.metric-row.warning{background:color-mix(in srgb,var(--warning-color,#f59e0b) 7%,transparent)}.metric-icon{display:grid;place-items:center;width:40px;height:40px;border-radius:12px;background:color-mix(in srgb,var(--primary-color) 11%,transparent);color:var(--primary-color)}.metric-title-line{display:flex;gap:9px;align-items:center;flex-wrap:wrap}.metric-title-line strong{font-size:15px}.metric-value{font-size:18px;font-weight:700;margin-top:4px}.entity-name{font-size:12px;color:var(--secondary-text-color);overflow:hidden;text-overflow:ellipsis;white-space:nowrap;max-width:58ch}.badge{display:inline-flex;align-items:center;gap:5px;border-radius:999px;padding:3px 8px;font-size:11px;font-weight:650;background:var(--secondary-background-color)}.badge-dot{width:6px;height:6px;border-radius:50%;background:var(--secondary-text-color)}.badge.manual{color:var(--primary-color)}.badge.manual .badge-dot{background:var(--primary-color)}.badge.needs_review{color:var(--warning-color,#f59e0b)}.badge.needs_review .badge-dot{background:var(--warning-color,#f59e0b)}.badge.unavailable{color:var(--error-color,#ef4444)}.badge.unavailable .badge-dot{background:var(--error-color,#ef4444)}.warning-text{display:flex;gap:5px;align-items:flex-start;color:var(--warning-color,#f59e0b);font-size:12px;margin-top:6px}.warning-text ha-icon{--mdc-icon-size:16px}.configure,.icon-button,.cancel,.reset{min-height:44px;border:0;border-radius:12px;padding:0 14px;background:var(--secondary-background-color);color:var(--primary-text-color);font:inherit;font-weight:600;cursor:pointer}.configure{display:flex;gap:7px;align-items:center}.configure ha-icon{--mdc-icon-size:18px}.configure:disabled{opacity:.45;cursor:not-allowed}button:focus-visible,input:focus-visible,summary:focus-visible{outline:3px solid color-mix(in srgb,var(--primary-color) 55%,transparent);outline-offset:2px}.scrim{position:fixed;inset:0;z-index:9999;display:grid;place-items:center;padding:20px;background:rgba(0,0,0,.58);backdrop-filter:blur(3px)}.dialog{display:flex;flex-direction:column;width:min(680px,calc(100vw - 32px));max-height:min(780px,calc(100vh - 40px));border-radius:22px;background:var(--card-background-color);box-shadow:0 24px 80px rgba(0,0,0,.36);overflow:hidden}.dialog-head{display:flex;justify-content:space-between;gap:16px;padding:22px 22px 12px}.dialog-head h2{margin:3px 0 0;font-size:22px}.eyebrow{font-size:11px;text-transform:uppercase;letter-spacing:.08em;color:var(--primary-color);font-weight:700}.icon-button{width:44px;padding:0}.dialog-warning{margin:0 22px 14px}.dialog-warning div{display:grid;gap:2px}.dialog-warning span{font-size:13px;color:var(--secondary-text-color)}.search{display:flex;align-items:center;gap:9px;margin:0 22px 12px;border:1px solid var(--divider-color);border-radius:12px;padding:0 12px;min-height:46px;background:var(--secondary-background-color)}.search input{flex:1;min-width:0;border:0;background:transparent;color:var(--primary-text-color);font:inherit;outline:0}.candidate-scroll{overflow:auto;padding:0 22px 16px}.candidate-scroll h3{font-size:13px;color:var(--secondary-text-color);margin:10px 0}.candidate{display:grid;grid-template-columns:22px minmax(0,1fr) auto;align-items:center;gap:11px;width:100%;min-height:70px;text-align:left;border:1px solid var(--divider-color);border-radius:14px;padding:11px 13px;margin:8px 0;background:transparent;color:var(--primary-text-color);cursor:pointer}.candidate:hover,.candidate.selected{border-color:var(--primary-color);background:color-mix(in srgb,var(--primary-color) 8%,transparent)}.radio{display:grid;place-items:center;width:18px;height:18px;border:2px solid var(--secondary-text-color);border-radius:50%}.selected .radio{border-color:var(--primary-color)}.selected .radio span{width:9px;height:9px;border-radius:50%;background:var(--primary-color)}.candidate-main{display:grid;min-width:0;gap:2px}.candidate-main strong,.candidate-main code,.candidate-main small{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.candidate-main code{font-size:11px;color:var(--secondary-text-color)}.candidate-main small{font-size:11px;color:var(--secondary-text-color)}.candidate-value{text-align:right;font-weight:700}.candidate-value small{display:block;color:var(--secondary-text-color);font-weight:400}details summary{display:flex;justify-content:space-between;cursor:pointer;padding:12px 2px;color:var(--secondary-text-color);font-size:13px;font-weight:600}.dialog-actions{display:flex;justify-content:flex-end;gap:9px;padding:14px 22px 20px;border-top:1px solid var(--divider-color)}.reset{margin-right:auto;color:var(--error-color,#ef4444)}.toast{position:sticky;bottom:10px;margin:16px auto 0;width:max-content;max-width:90%;padding:10px 14px;border-radius:999px;background:var(--primary-text-color);color:var(--card-background-color);font-size:13px;box-shadow:0 8px 24px rgba(0,0,0,.22)}.empty{padding:20px;color:var(--secondary-text-color);text-align:center}
    @media(max-width:600px){ha-card{padding:16px;border-radius:16px}.header{margin-bottom:14px}.metric-row{grid-template-columns:38px minmax(0,1fr);padding:13px 12px}.configure{grid-column:2;width:100%;justify-content:center;margin-top:2px}.entity-name{max-width:100%}.scrim{place-items:end center;padding:0}.dialog{width:100%;max-height:92vh;border-radius:22px 22px 0 0}.dialog-head,.candidate-scroll{padding-left:16px;padding-right:16px}.search,.dialog-warning{margin-left:16px;margin-right:16px}.dialog-actions{padding:12px 16px 18px;flex-wrap:wrap}.reset{width:100%;margin:0;order:2}.cancel{margin-left:auto}.candidate{grid-template-columns:20px minmax(0,1fr)}.candidate-value{grid-column:2;text-align:left}.configure span{display:inline}}
    .use-source{min-height:44px;border:0;border-radius:12px;padding:0 16px;background:var(--primary-color);color:var(--text-primary-color,#fff);font:inherit;font-weight:700;cursor:pointer}.use-source:disabled,button:disabled{opacity:.48;cursor:not-allowed}
    .match-reason{display:flex;align-items:center;gap:7px;flex-wrap:wrap;margin-top:5px;color:var(--secondary-text-color);font-size:11px}.match-reason span{border:1px solid var(--divider-color);border-radius:999px;padding:2px 7px;color:var(--primary-text-color);font-weight:650}
    :host{padding:24px}ha-card{max-width:1280px;margin:0 auto;padding:24px}.groups{grid-template-columns:repeat(2,minmax(0,1fr));align-items:start}.candidate-main strong{white-space:normal;display:-webkit-box;-webkit-box-orient:vertical;-webkit-line-clamp:2;line-height:1.3}.candidate-main code{direction:ltr}.candidate-main strong,.candidate-main code{overflow-wrap:anywhere}
    @media(max-width:900px){.groups{grid-template-columns:1fr}}
    @media(max-width:600px){:host{padding:0}ha-card{padding:16px}}
  `; }
}

customElements.define("gosungrow-energy-flow-card-v2", GoSungrowEnergyFlowCard);
customElements.define("gosungrow-energy-summary-card-v1", GoSungrowEnergySummaryCard);
customElements.define("gosungrow-source-mapping-card-v1", GoSungrowSourceMappingCard);

window.customCards = window.customCards || [];
window.customCards.push({
  type: "gosungrow-energy-flow-card-v2",
  name: "GoSungrow Energy Flow Card v2",
  description: "Custom Sungrow energy flow card with Energy dashboard-inspired layout.",
});
window.customCards.push({
  type: "gosungrow-source-mapping-card-v1",
  name: "GoSungrow Data Sources",
  description: "Review automatic dashboard matches and choose persistent source overrides.",
});
window.customCards.push({
  type: "gosungrow-energy-summary-card-v1",
  name: "GoSungrow Energy Summary Card",
  description: "Day, month, and year energy aggregates for GoSungrow sensors.",
});
