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
    if (!this._config || !this.shadowRoot) {
      return;
    }

    const compact = this.offsetWidth > 0 && this.offsetWidth < 760;
    const layout = compact ? this._compactLayout() : this._wideLayout();
    const nodeValues = this._buildNodeValues();
    const flowValues = this._buildFlowValues();

    const edgeMarkup = Object.entries(layout.edges)
      .map(([key, edge]) => this._renderEdge(key, edge, flowValues[key]))
      .join("");

    const nodeMarkup = Object.entries(layout.nodes)
      .map(([key, node]) => this._renderNode(key, node, nodeValues[key], nodeValues.batterySoc, compact))
      .join("");

    const labelMarkup = Object.entries(layout.edges)
      .map(([key, edge]) => this._renderEdgeLabel(key, edge, flowValues[key]))
      .join("");

    this.shadowRoot.innerHTML = `
      <style>
        :host {
          display: block;
        }

        ha-card {
          overflow: hidden;
        }

        .card-header {
          padding: 18px 20px 0;
          font-size: 1.15rem;
          font-weight: 600;
          color: var(--primary-text-color);
        }

        .stage {
          position: relative;
          margin: 14px 16px 18px;
          aspect-ratio: ${layout.width} / ${layout.height};
          border-radius: 22px;
          overflow: hidden;
          background:
            radial-gradient(circle at 50% 16%, rgba(255,255,255,0.08), transparent 38%),
            linear-gradient(180deg, rgba(255,255,255,0.04), rgba(255,255,255,0.01)),
            var(--card-background-color, #1f1f1f);
          box-shadow: inset 0 1px 0 rgba(255,255,255,0.04);
        }

        .links {
          position: absolute;
          inset: 0;
          width: 100%;
          height: 100%;
        }

        .edge-base {
          fill: none;
          stroke: rgba(148, 163, 184, 0.26);
          stroke-width: 11;
          stroke-linecap: round;
        }

        .edge-active {
          fill: none;
          stroke-linecap: round;
          transition: stroke-width 180ms ease, opacity 180ms ease;
          filter: drop-shadow(0 0 10px rgba(255,255,255,0.06));
        }

        .node {
          position: absolute;
          transform: translate(-50%, -50%);
          background: transparent;
          border: 0;
          padding: 0;
          cursor: pointer;
          font: inherit;
          color: inherit;
        }

        .node[disabled] {
          cursor: default;
        }

        .node-shell {
          width: ${compact ? 116 : 128}px;
          height: ${compact ? 116 : 128}px;
          border-radius: 999px;
          border: 4px solid currentColor;
          background: rgba(15, 23, 42, 0.10);
          backdrop-filter: blur(2px);
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          gap: 8px;
          box-shadow:
            0 18px 40px rgba(15, 23, 42, 0.16),
            inset 0 1px 0 rgba(255,255,255,0.08);
        }

        .node.compact .node-shell {
          width: 104px;
          height: 104px;
        }

        .node ha-icon {
          --mdc-icon-size: ${compact ? 34 : 38}px;
        }

        .node-value {
          font-size: ${compact ? "1rem" : "1.05rem"};
          line-height: 1;
          font-weight: 700;
          letter-spacing: -0.02em;
        }

        .node-label {
          margin-top: 10px;
          text-align: center;
          font-size: ${compact ? "0.86rem" : "0.92rem"};
          font-weight: 500;
          color: var(--secondary-text-color);
        }

        .soc-pill {
          position: absolute;
          left: 50%;
          top: calc(100% - 14px);
          transform: translate(-50%, 0);
          min-width: 70px;
          padding: 7px 10px;
          border-radius: 999px;
          background: rgba(204, 251, 241, 0.96);
          border: 1px solid rgba(15, 118, 110, 0.22);
          color: #134e4a;
          font-size: 0.88rem;
          font-weight: 700;
          text-align: center;
          box-shadow: 0 8px 20px rgba(20, 184, 166, 0.18);
        }

        .edge-label {
          position: absolute;
          transform: translate(-50%, -50%);
          min-width: 74px;
          padding: 7px 10px;
          border-radius: 999px;
          text-align: center;
          font-size: ${compact ? "0.76rem" : "0.82rem"};
          font-weight: 700;
          line-height: 1;
          letter-spacing: -0.01em;
          border: 1px solid rgba(255,255,255,0.08);
          box-shadow: 0 10px 24px rgba(15, 23, 42, 0.14);
          backdrop-filter: blur(8px);
          transition: opacity 180ms ease;
        }

        .edge-label.inactive {
          opacity: 0.42;
        }

        .solar {
          color: #f59e0b;
        }

        .home {
          color: #3b82f6;
        }

        .battery {
          color: #14b8a6;
        }

        .grid {
          color: #64748b;
        }

        .edge-solar {
          stroke: #f59e0b;
        }

        .edge-battery {
          stroke: #14b8a6;
        }

        .edge-grid {
          stroke: #64748b;
        }

        .label-solar {
          background: rgba(245, 158, 11, 0.16);
          color: #a16207;
        }

        .label-battery {
          background: rgba(20, 184, 166, 0.16);
          color: #0f766e;
        }

        .label-grid {
          background: rgba(100, 116, 139, 0.16);
          color: #475569;
        }

        .label-home {
          background: rgba(59, 130, 246, 0.16);
          color: #1d4ed8;
        }

        @media (prefers-color-scheme: dark) {
          .stage {
            background:
              radial-gradient(circle at 50% 12%, rgba(255,255,255,0.06), transparent 34%),
              linear-gradient(180deg, rgba(255,255,255,0.03), rgba(255,255,255,0.01)),
              var(--card-background-color, #111827);
          }

          .label-solar {
            color: #fbbf24;
          }

          .label-battery {
            color: #5eead4;
          }

          .label-grid {
            color: #cbd5e1;
          }

          .label-home {
            color: #93c5fd;
          }
        }
      </style>
      <ha-card>
        ${this._config.title ? `<div class="card-header">${this._escape(this._config.title)}</div>` : ""}
        <div class="stage">
          <svg class="links" viewBox="0 0 ${layout.width} ${layout.height}" preserveAspectRatio="xMidYMid meet">
            ${edgeMarkup}
          </svg>
          ${labelMarkup}
          ${nodeMarkup}
        </div>
      </ha-card>
    `;

    this.shadowRoot.querySelectorAll(".node[data-entity]").forEach((node) => {
      node.addEventListener("click", (event) => {
        const entityId = event.currentTarget.getAttribute("data-entity");
        if (!entityId) {
          return;
        }
        this._fire("hass-more-info", { entityId });
      });
    });
  }

  _wideLayout() {
    return {
      width: 1000,
      height: 640,
      nodes: {
        solar: { x: 500, y: 120, label: "PV" },
        home: { x: 500, y: 312, label: "Home" },
        battery: { x: 278, y: 510, label: "Battery" },
        grid: { x: 722, y: 510, label: "Grid" },
      },
      edges: {
        pv_to_load_power: {
          path: "M500 184 C500 216 500 248 500 280",
          labelX: 500,
          labelY: 228,
          colorClass: "edge-solar",
          labelClass: "label-solar",
        },
        pv_to_battery_power: {
          path: "M452 164 C388 198 326 290 294 432",
          labelX: 380,
          labelY: 244,
          colorClass: "edge-solar",
          labelClass: "label-solar",
        },
        pv_to_grid_power: {
          path: "M548 164 C612 198 674 290 706 432",
          labelX: 620,
          labelY: 244,
          colorClass: "edge-solar",
          labelClass: "label-solar",
        },
        battery_to_load_power: {
          path: "M334 462 C390 404 430 360 464 334",
          labelX: 388,
          labelY: 402,
          colorClass: "edge-battery",
          labelClass: "label-battery",
        },
        grid_to_load_power: {
          path: "M666 462 C610 404 570 360 536 334",
          labelX: 612,
          labelY: 402,
          colorClass: "edge-grid",
          labelClass: "label-grid",
        },
      },
    };
  }

  _compactLayout() {
    return {
      width: 760,
      height: 760,
      nodes: {
        solar: { x: 380, y: 120, label: "PV" },
        home: { x: 380, y: 330, label: "Home" },
        battery: { x: 182, y: 574, label: "Battery" },
        grid: { x: 578, y: 574, label: "Grid" },
      },
      edges: {
        pv_to_load_power: {
          path: "M380 188 C380 228 380 258 380 296",
          labelX: 380,
          labelY: 240,
          colorClass: "edge-solar",
          labelClass: "label-solar",
        },
        pv_to_battery_power: {
          path: "M332 172 C260 214 210 318 194 480",
          labelX: 266,
          labelY: 276,
          colorClass: "edge-solar",
          labelClass: "label-solar",
        },
        pv_to_grid_power: {
          path: "M428 172 C500 214 550 318 566 480",
          labelX: 494,
          labelY: 276,
          colorClass: "edge-solar",
          labelClass: "label-solar",
        },
        battery_to_load_power: {
          path: "M238 528 C282 464 324 400 350 354",
          labelX: 288,
          labelY: 448,
          colorClass: "edge-battery",
          labelClass: "label-battery",
        },
        grid_to_load_power: {
          path: "M522 528 C478 464 436 400 410 354",
          labelX: 472,
          labelY: 448,
          colorClass: "edge-grid",
          labelClass: "label-grid",
        },
      },
    };
  }

  _buildNodeValues() {
    return {
      solar: this._entityDisplay("solar_power", "mdi:solar-power-variant"),
      home: this._entityDisplay("load_power", "mdi:home-lightning-bolt-outline"),
      battery: this._entityDisplay("battery_power", "mdi:battery-high"),
      grid: this._entityDisplay("grid_power", "mdi:transmission-tower"),
      batterySoc: this._entityDisplay("battery_soc"),
    };
  }

  _buildFlowValues() {
    return {
      pv_to_load_power: this._entityDisplay("pv_to_load_power"),
      pv_to_battery_power: this._entityDisplay("pv_to_battery_power"),
      pv_to_grid_power: this._entityDisplay("pv_to_grid_power"),
      battery_to_load_power: this._entityDisplay("battery_to_load_power"),
      grid_to_load_power: this._entityDisplay("grid_to_load_power"),
    };
  }

  _renderEdge(key, edge, display) {
    const active = Math.abs(display.numericValue) > 0.01;
    const strokeWidth = active ? this._scaledStroke(display.numericValue) : 8;
    const opacity = active ? 1 : 0.18;
    return `
      <path class="edge-base" d="${edge.path}"></path>
      <path class="edge-active ${edge.colorClass}" d="${edge.path}" style="stroke-width:${strokeWidth};opacity:${opacity};"></path>
    `;
  }

  _renderEdgeLabel(_key, edge, display) {
    const active = Math.abs(display.numericValue) > 0.01;
    return `
      <div
        class="edge-label ${edge.labelClass}${active ? "" : " inactive"}"
        style="left:${edge.labelX}px;top:${edge.labelY}px;"
      >
        ${this._escape(display.formatted)}
      </div>
    `;
  }

  _renderNode(key, node, display, batterySoc, compact) {
    const icon = this._escape(display.icon || "mdi:flash");
    const entityId = this._entityIdForKey(key);
    const isBattery = key === "battery";

    return `
      <button
        class="node ${key}${compact ? " compact" : ""}"
        style="left:${node.x}px;top:${node.y}px;"
        ${entityId ? `data-entity="${this._escape(entityId)}"` : "disabled"}
      >
        <div class="node-shell">
          <ha-icon icon="${icon}"></ha-icon>
          <div class="node-value">${this._escape(display.formatted)}</div>
        </div>
        <div class="node-label">${this._escape(node.label)}</div>
        ${isBattery ? `<div class="soc-pill">${this._escape(batterySoc.formatted)}</div>` : ""}
      </button>
    `;
  }

  _entityIdForKey(key) {
    const mapping = {
      solar: "solar_power",
      home: "load_power",
      battery: "battery_power",
      grid: "grid_power",
    };
    const entityKey = mapping[key] || key;
    return this._config?.entities?.[entityKey] || "";
  }

  _entityDisplay(key, fallbackIcon = "") {
    const entityId = this._config?.entities?.[key];
    const stateObj = entityId && this._hass ? this._hass.states[entityId] : null;
    const unit = stateObj?.attributes?.unit_of_measurement || "";
    const icon = fallbackIcon || stateObj?.attributes?.icon || "mdi:flash";

    if (!stateObj) {
      return {
        formatted: "--",
        numericValue: 0,
        icon,
      };
    }

    const numericValue = Number.parseFloat(stateObj.state);
    if (!Number.isFinite(numericValue)) {
      return {
        formatted: stateObj.state,
        numericValue: 0,
        icon,
      };
    }

    return {
      formatted: this._formatNumber(numericValue, unit),
      numericValue,
      icon,
    };
  }

  _scaledStroke(value) {
    const magnitude = Math.min(Math.abs(value), 6);
    return 7 + magnitude * 3;
  }

  _formatNumber(value, unit) {
    if (unit === "%") {
      return `${new Intl.NumberFormat(this._locale(), {
        minimumFractionDigits: 0,
        maximumFractionDigits: 0,
      }).format(value)} ${unit}`;
    }

    const digits = Math.abs(value) >= 10 ? 1 : 2;
    return `${new Intl.NumberFormat(this._locale(), {
      minimumFractionDigits: digits,
      maximumFractionDigits: digits,
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
  description: "Custom Sungrow energy flow card with power routes and battery SOC.",
});
