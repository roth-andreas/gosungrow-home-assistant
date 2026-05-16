const ENTITY_IDS = {
  solarPower: "sensor.preview_pv_power",
  loadPower: "sensor.preview_load_power",
  batteryPower: "sensor.preview_battery_power",
  batterySoc: "sensor.preview_battery_soc",
  gridPower: "sensor.preview_grid_power",
  pvToLoadPower: "sensor.preview_pv_to_load_power",
  pvToBatteryPower: "sensor.preview_pv_to_battery_power",
  pvToGridPower: "sensor.preview_pv_to_grid_power",
  batteryToLoadPower: "sensor.preview_battery_to_load_power",
  gridToLoadPower: "sensor.preview_grid_to_load_power",
  dailyPvYield: "sensor.preview_daily_pv_yield",
  dailyPvToLoad: "sensor.preview_daily_pv_to_load",
  dailyPvToBattery: "sensor.preview_daily_pv_to_battery",
  dailyFeedIn: "sensor.preview_daily_feed_in",
  dailyGridImport: "sensor.preview_daily_grid_import",
};

const SERIES_COLORS = {
  PV: "#4f7dff",
  Load: "#f6c343",
  Grid: "#ff6f59",
  Battery: "#62d2c5",
  "Battery SOC": "#4f7dff",
  "PV Yield": "#4f7dff",
  "PV To Load": "#f6c343",
  "PV To Battery": "#ff6f59",
  "Feed-in": "#70d8cb",
  "Grid Import": "#9b5de5",
  "PV To Grid": "#9b5de5",
  "Battery To Load": "#62d2c5",
  "Grid To Load": "#d1d5db",
};

const SCENARIOS = {
  export_high: {
    label: "High Export",
    flows: { pvToLoad: 0.2, pvToBattery: 0.0, pvToGrid: 2.8, batteryToLoad: 0.0, gridToLoad: 0.0, batterySoc: 100 },
    daily: { pvYield: 9.0, pvToLoad: 1.4, pvToBattery: 3.5, feedIn: 4.1, gridImport: 0.0 },
    charts: {
      powerBalance: [
        series("PV", [2.4, 2.6, 2.8, 3.0, 3.0, 3.0, 2.8, 2.3, 1.5, 0.6, 0.0, 0.0, 0.0, 0.1, 0.1, 0.1, 0.0, 0.0, 0.1, 0.3, 1.2, 2.4, 2.7, 2.9]),
        series("Load", [0.1, 0.1, 0.1, 0.3, 1.4, 0.2, 0.2, 0.2, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.2, 0.2, 2.0, 0.2, 0.2]),
        series("Grid", [-2.3, -2.6, -2.7, -2.8, -2.5, -2.2, -1.8, -1.0, -0.3, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, -0.6, -2.2, -2.6, -2.8]),
        series("Battery", [0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0]),
      ],
      battery: [series("Battery SOC", [100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100])],
      dailyEnergy: [
        series("PV Yield", cumulative([0.3, 0.6, 0.8, 1.2, 1.2, 1.1, 0.9, 0.7, 0.6, 0.4, 0.3, 0.2, 0, 0, 0, 0, 0, 0, 0.2, 0.4, 0.7, 0.9, 0.8, 0.6])),
        series("PV To Load", cumulative([0.05, 0.08, 0.1, 0.1, 0.09, 0.08, 0.07, 0.06, 0.05, 0.05, 0.05, 0.05, 0, 0, 0, 0, 0, 0, 0.05, 0.05, 0.08, 0.1, 0.12, 0.1])),
        series("PV To Battery", cumulative([0.2, 0.25, 0.3, 0.38, 0.38, 0.35, 0.3, 0.25, 0.2, 0.18, 0.16, 0.12, 0, 0, 0, 0, 0, 0, 0.06, 0.08, 0.1, 0.12, 0.12, 0.08])),
        series("Feed-in", cumulative([0.08, 0.14, 0.18, 0.2, 0.2, 0.18, 0.16, 0.14, 0.12, 0.1, 0.08, 0.08, 0, 0, 0, 0, 0, 0, 0.02, 0.04, 0.08, 0.12, 0.18, 0.22])),
        series("Grid Import", cumulative([0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0])),
      ],
      routeDetail: [
        series("PV To Load", [0.08, 0.1, 0.12, 0.14, 0.12, 0.1, 0.09, 0.08, 0.07, 0.06, 0.05, 0.05, 0, 0, 0, 0, 0, 0, 0.04, 0.06, 0.08, 0.1, 0.18, 0.2]),
        series("PV To Battery", [0.2, 0.26, 0.3, 0.34, 0.34, 0.32, 0.28, 0.24, 0.2, 0.18, 0.15, 0.12, 0, 0, 0, 0, 0, 0, 0.04, 0.06, 0.08, 0.1, 0.1, 0.08]),
        series("PV To Grid", [2.0, 2.2, 2.3, 2.4, 2.5, 2.4, 2.2, 1.9, 1.4, 0.6, 0.1, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.02, 0.08, 0.6, 2.1, 2.5, 2.8]),
        series("Battery To Load", [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]),
        series("Grid To Load", [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]),
      ],
    },
  },
  battery_charge: {
    label: "Battery Charging",
    flows: { pvToLoad: 0.9, pvToBattery: 2.1, pvToGrid: 0.2, batteryToLoad: 0.0, gridToLoad: 0.0, batterySoc: 46 },
    daily: { pvYield: 7.0, pvToLoad: 1.4, pvToBattery: 3.5, feedIn: 2.1, gridImport: 0.0 },
    charts: {
      powerBalance: [
        series("PV", [1.0, 1.2, 1.6, 2.1, 2.4, 2.6, 2.8, 2.9, 3.0, 2.8, 2.4, 1.9, 1.0, 0.4, 0.0, 0.0, 0.0, 0.0, 0.2, 0.6, 1.4, 2.1, 2.4, 2.6]),
        series("Load", [0.2, 0.2, 0.3, 0.4, 1.0, 0.2, 0.2, 0.2, 0.15, 0.15, 0.15, 0.15, 0.15, 0.15, 0.1, 0.1, 0.1, 0.1, 0.12, 0.18, 0.25, 0.35, 0.45, 0.5]),
        series("Grid", [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]),
        series("Battery", [-0.8, -1.0, -1.3, -1.6, -1.8, -2.0, -2.2, -2.3, -2.4, -2.2, -1.8, -1.5, -0.8, -0.2, 0, 0, 0, 0, -0.1, -0.3, -0.6, -1.0, -1.5, -1.8]),
      ],
      battery: [series("Battery SOC", [10, 12, 16, 22, 28, 35, 42, 48, 53, 58, 62, 66, 68, 68, 68, 68, 69, 70, 72, 76, 82, 88, 94, 96])],
      dailyEnergy: [
        series("PV Yield", cumulative([0.2, 0.25, 0.3, 0.38, 0.46, 0.5, 0.55, 0.55, 0.55, 0.5, 0.45, 0.4, 0.15, 0.05, 0, 0, 0, 0, 0.08, 0.12, 0.18, 0.22, 0.24, 0.22])),
        series("PV To Load", cumulative([0.08, 0.09, 0.1, 0.12, 0.12, 0.12, 0.12, 0.12, 0.1, 0.08, 0.06, 0.06, 0.04, 0.02, 0, 0, 0, 0, 0.02, 0.04, 0.06, 0.08, 0.09, 0.1])),
        series("PV To Battery", cumulative([0.1, 0.14, 0.18, 0.2, 0.24, 0.26, 0.28, 0.28, 0.3, 0.28, 0.24, 0.2, 0.08, 0.02, 0, 0, 0, 0, 0.02, 0.04, 0.06, 0.08, 0.1, 0.1])),
        series("Feed-in", cumulative([0.02, 0.02, 0.02, 0.03, 0.04, 0.05, 0.06, 0.06, 0.06, 0.05, 0.04, 0.04, 0.01, 0, 0, 0, 0, 0, 0, 0.01, 0.01, 0.02, 0.03, 0.03])),
        series("Grid Import", cumulative([0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0])),
      ],
      routeDetail: [
        series("PV To Load", [0.2, 0.25, 0.3, 0.35, 0.45, 0.6, 0.75, 0.9, 1.0, 0.9, 0.8, 0.7, 0.3, 0.1, 0, 0, 0, 0, 0.08, 0.12, 0.2, 0.4, 0.6, 0.9]),
        series("PV To Battery", [0.6, 0.8, 1.0, 1.2, 1.4, 1.7, 1.9, 2.1, 2.1, 2.0, 1.8, 1.6, 0.8, 0.2, 0, 0, 0, 0, 0.02, 0.08, 0.16, 0.3, 0.5, 0.8]),
        series("PV To Grid", [0.1, 0.12, 0.14, 0.16, 0.18, 0.2, 0.22, 0.2, 0.18, 0.16, 0.14, 0.12, 0.06, 0.02, 0, 0, 0, 0, 0, 0.02, 0.04, 0.06, 0.08, 0.1]),
        series("Battery To Load", [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]),
        series("Grid To Load", [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]),
      ],
    },
  },
  evening_discharge: {
    label: "Evening Discharge",
    flows: { pvToLoad: 0.0, pvToBattery: 0.0, pvToGrid: 0.0, batteryToLoad: 1.6, gridToLoad: 0.3, batterySoc: 32 },
    daily: { pvYield: 4.4, pvToLoad: 0.9, pvToBattery: 1.1, feedIn: 0.0, gridImport: 0.0 },
    charts: {
      powerBalance: [
        series("PV", [0.2, 0.1, 0.05, 0.02, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.05, 0.1, 0.2, 0.3]),
        series("Load", [1.5, 1.7, 1.8, 1.9, 2.1, 2.2, 2.0, 1.8, 1.6, 1.4, 1.2, 1.0, 0.9, 0.8, 0.8, 0.9, 1.0, 1.0, 1.1, 1.2, 1.4, 1.6, 1.8, 1.9]),
        series("Grid", [0.3, 0.3, 0.2, 0.2, 0.3, 0.4, 0.2, 0.1, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.1, 0.2, 0.3]),
        series("Battery", [1.0, 1.2, 1.4, 1.6, 1.8, 1.8, 1.8, 1.7, 1.6, 1.4, 1.2, 1.0, 0.9, 0.8, 0.8, 0.9, 1.0, 1.0, 1.1, 1.2, 1.35, 1.5, 1.6, 1.6]),
      ],
      battery: [series("Battery SOC", [82, 80, 78, 76, 73, 70, 66, 62, 58, 54, 50, 46, 42, 39, 36, 34, 32, 31, 30, 30, 31, 32, 32, 32])],
      dailyEnergy: [
        series("PV Yield", cumulative([0.05, 0.06, 0.08, 0.1, 0.12, 0.14, 0.16, 0.18, 0.16, 0.14, 0.12, 0.1, 0.02, 0, 0, 0, 0, 0, 0, 0, 0.02, 0.04, 0.05, 0.05])),
        series("PV To Load", cumulative([0.03, 0.04, 0.05, 0.06, 0.08, 0.1, 0.1, 0.1, 0.08, 0.08, 0.06, 0.05, 0.01, 0, 0, 0, 0, 0, 0, 0, 0.01, 0.02, 0.03, 0.03])),
        series("PV To Battery", cumulative([0.02, 0.02, 0.03, 0.04, 0.04, 0.04, 0.05, 0.06, 0.06, 0.05, 0.04, 0.03, 0.01, 0, 0, 0, 0, 0, 0, 0, 0.01, 0.02, 0.02, 0.02])),
        series("Feed-in", cumulative([0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0])),
        series("Grid Import", cumulative([0.02, 0.02, 0.02, 0.03, 0.04, 0.04, 0.04, 0.04, 0.03, 0.03, 0.02, 0.02, 0.01, 0.01, 0.01, 0.01, 0.01, 0.01, 0.01, 0.01, 0.02, 0.02, 0.02, 0.02])),
      ],
      routeDetail: [
        series("PV To Load", [0.15, 0.12, 0.1, 0.08, 0.06, 0.04, 0.02, 0.01, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0.01, 0.02, 0.04]),
        series("PV To Battery", [0.05, 0.04, 0.04, 0.03, 0.03, 0.02, 0.02, 0.01, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0.01, 0.02]),
        series("PV To Grid", [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]),
        series("Battery To Load", [1.2, 1.4, 1.6, 1.8, 1.9, 1.8, 1.7, 1.6, 1.5, 1.3, 1.1, 0.9, 0.8, 0.7, 0.7, 0.8, 0.9, 0.9, 1.0, 1.1, 1.2, 1.4, 1.5, 1.6]),
        series("Grid To Load", [0.2, 0.3, 0.2, 0.1, 0.2, 0.4, 0.3, 0.2, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.2, 0.2, 0.3, 0.3]),
      ],
    },
  },
  grid_import: {
    label: "Grid Import",
    flows: { pvToLoad: 0.4, pvToBattery: 0.0, pvToGrid: 0.0, batteryToLoad: 0.0, gridToLoad: 2.4, batterySoc: 18 },
    daily: { pvYield: 1.8, pvToLoad: 0.5, pvToBattery: 0.0, feedIn: 0.0, gridImport: 8.4 },
    charts: {
      powerBalance: [
        series("PV", [0.0, 0.0, 0.0, 0.0, 0.1, 0.2, 0.4, 0.6, 0.4, 0.2, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0]),
        series("Load", [2.2, 2.3, 2.5, 2.6, 2.8, 2.8, 2.8, 2.9, 2.8, 2.8, 2.6, 2.5, 2.4, 2.4, 2.3, 2.3, 2.2, 2.2, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7]),
        series("Grid", [2.2, 2.3, 2.5, 2.6, 2.7, 2.6, 2.4, 2.3, 2.4, 2.6, 2.6, 2.5, 2.4, 2.4, 2.3, 2.3, 2.2, 2.2, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7]),
        series("Battery", [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]),
      ],
      battery: [series("Battery SOC", [22, 22, 21, 20, 19, 19, 18, 18, 18, 18, 18, 18, 18, 18, 18, 18, 18, 18, 18, 18, 18, 18, 18, 18])],
      dailyEnergy: [
        series("PV Yield", cumulative([0, 0, 0, 0.02, 0.04, 0.08, 0.12, 0.16, 0.12, 0.08, 0.04, 0.02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0])),
        series("PV To Load", cumulative([0, 0, 0, 0.02, 0.04, 0.06, 0.08, 0.08, 0.06, 0.05, 0.04, 0.02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0])),
        series("PV To Battery", cumulative([0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0])),
        series("Feed-in", cumulative([0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0])),
        series("Grid Import", cumulative([0.1, 0.14, 0.18, 0.2, 0.24, 0.28, 0.32, 0.36, 0.34, 0.32, 0.28, 0.26, 0.24, 0.24, 0.22, 0.22, 0.22, 0.22, 0.22, 0.22, 0.24, 0.26, 0.28, 0.3])),
      ],
      routeDetail: [
        series("PV To Load", [0, 0, 0, 0.08, 0.1, 0.15, 0.2, 0.4, 0.35, 0.3, 0.2, 0.08, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]),
        series("PV To Battery", [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]),
        series("PV To Grid", [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]),
        series("Battery To Load", [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]),
        series("Grid To Load", [2.2, 2.3, 2.5, 2.6, 2.8, 2.8, 2.8, 2.4, 2.4, 2.6, 2.6, 2.5, 2.4, 2.4, 2.3, 2.3, 2.2, 2.2, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7]),
      ],
    },
  },
};

Object.values(SCENARIOS).forEach((scenario) => {
  scenario.charts.solarAllocation = [
    cloneSeries(findSeries(scenario.charts.routeDetail, "PV To Load")),
    cloneSeries(findSeries(scenario.charts.routeDetail, "PV To Battery")),
    cloneSeries(findSeries(scenario.charts.routeDetail, "PV To Grid")),
  ];
  scenario.charts.loadSources = [
    cloneSeries(findSeries(scenario.charts.routeDetail, "PV To Load")),
    cloneSeries(findSeries(scenario.charts.routeDetail, "Battery To Load")),
    cloneSeries(findSeries(scenario.charts.routeDetail, "Grid To Load")),
  ];
  scenario.charts.gridExchange = [
    cloneSeries(findSeries(scenario.charts.powerBalance, "Grid"), "Grid Net"),
    cloneSeries(findSeries(scenario.charts.routeDetail, "Grid To Load")),
    cloneSeries(findSeries(scenario.charts.routeDetail, "PV To Grid")),
  ];
  scenario.charts.batteryFlow = [
    cloneSeries(findSeries(scenario.charts.powerBalance, "Battery"), "Battery Power"),
    cloneSeries(findSeries(scenario.charts.routeDetail, "PV To Battery"), "Charge"),
    cloneSeries(findSeries(scenario.charts.routeDetail, "Battery To Load"), "Discharge"),
  ];
});

init();

function init() {
  const params = new URLSearchParams(window.location.search);
  const state = {
    scenario: params.get("scenario") || "battery_charge",
    view: params.get("view") || "overview",
    device: params.get("device") || "desktop",
    chrome: params.get("chrome") !== "0",
    inspect: params.get("inspect") === "1",
  };

  populateControls(state);
  render(state);
}

function populateControls(state) {
  const toolbar = document.getElementById("toolbar");
  toolbar.classList.toggle("hidden", !state.chrome);

  const scenarioSelect = document.getElementById("scenario");
  Object.entries(SCENARIOS).forEach(([value, scenario]) => {
    const option = document.createElement("option");
    option.value = value;
    option.textContent = scenario.label;
    scenarioSelect.appendChild(option);
  });

  document.getElementById("scenario").value = state.scenario;
  document.getElementById("view").value = state.view;
  document.getElementById("device").value = state.device;

  document.getElementById("apply").addEventListener("click", () => {
    const next = new URLSearchParams(window.location.search);
    next.set("scenario", document.getElementById("scenario").value);
    next.set("view", document.getElementById("view").value);
    next.set("device", document.getElementById("device").value);
    next.set("chrome", state.chrome ? "1" : "0");
    window.location.search = next.toString();
  });
}

function render(state) {
  const app = document.getElementById("app");
  app.className = `preview-shell ${state.device}`;

  const scenario = SCENARIOS[state.scenario];
  app.innerHTML = "";
  app.appendChild(renderView(state, scenario));

  if (state.inspect) {
    window.setTimeout(() => {
      try {
        writeInspectionReport(state, scenario);
      } catch (error) {
        writeInspectionFailure(state, error);
      }
    }, 200);
  }
}

function renderView(state, scenario) {
  const root = document.createElement("div");
  root.innerHTML = `<div class="view-title">${escapeHtml(scenario.label)} Preview</div>`;

  if (state.view === "overview") {
    const overview = document.createElement("div");
    overview.innerHTML = `
      <div class="overview-grid">
        <div class="section-card">
          <div class="section-heading">Live Flow</div>
          <div id="flow-mount"></div>
        </div>
        <div class="section-card">
          <div class="section-heading">Today</div>
          <div class="today-grid">${renderTiles(scenario.daily, scenario.flows.batterySoc)}</div>
        </div>
      </div>
      <div class="charts-grid">
        ${renderChartCard("Power Balance", "kW", scenario.charts.powerBalance)}
        ${renderChartCard("Battery", "%", scenario.charts.battery)}
        ${renderChartCard("Daily Energy", "kWh", scenario.charts.dailyEnergy)}
      </div>
    `;
    root.appendChild(overview);
    mountFlowCard(root.querySelector("#flow-mount"), state.device, scenario);
    return root;
  }

  const trends = document.createElement("div");
  trends.innerHTML = `
    <div class="trends-grid">
      ${renderChartCard("Power Balance", "kW", scenario.charts.powerBalance)}
      ${renderChartCard("Solar Allocation", "kW", scenario.charts.solarAllocation)}
      ${renderChartCard("Load Sources", "kW", scenario.charts.loadSources)}
      ${renderChartCard("Grid Exchange", "kW", scenario.charts.gridExchange)}
      ${renderChartCard("Battery Flow", "kW", scenario.charts.batteryFlow)}
      ${renderChartCard("Battery SOC", "%", scenario.charts.battery)}
      ${renderChartCard("Daily Energy", "kWh", scenario.charts.dailyEnergy)}
    </div>
  `;
  root.appendChild(trends);
  return root;
}

function mountFlowCard(container, device, scenario) {
  const card = document.createElement("gosungrow-energy-flow-card-v2");
  card.setConfig({
    title: "Live Energy Flow",
    entities: {
      solar_power: ENTITY_IDS.solarPower,
      load_power: ENTITY_IDS.loadPower,
      battery_power: ENTITY_IDS.batteryPower,
      battery_soc: ENTITY_IDS.batterySoc,
      grid_power: ENTITY_IDS.gridPower,
      pv_to_load_power: ENTITY_IDS.pvToLoadPower,
      pv_to_battery_power: ENTITY_IDS.pvToBatteryPower,
      pv_to_grid_power: ENTITY_IDS.pvToGridPower,
      battery_to_load_power: ENTITY_IDS.batteryToLoadPower,
      grid_to_load_power: ENTITY_IDS.gridToLoadPower,
    },
  });
  card._isCompact = () => device === "mobile";
  card.hass = buildHass(scenario);
  container.appendChild(card);
}

function writeInspectionReport(state, scenario) {
  const flowCard = document.querySelector("gosungrow-energy-flow-card-v2");
  const report = inspectFlowCard(flowCard, state, scenario);
  const existing = document.getElementById("inspect-report");
  if (existing) {
    existing.remove();
  }
  const script = document.createElement("script");
  script.type = "application/json";
  script.id = "inspect-report";
  script.textContent = JSON.stringify(report, null, 2);
  document.body.appendChild(script);
}

function writeInspectionFailure(state, error) {
  const existing = document.getElementById("inspect-report");
  if (existing) {
    existing.remove();
  }
  const script = document.createElement("script");
  script.type = "application/json";
  script.id = "inspect-report";
  script.textContent = JSON.stringify({
    ok: false,
    scenario: state.scenario,
    device: state.device,
    view: state.view,
    warnings: [],
    errors: [error?.message || String(error)],
  }, null, 2);
  document.body.appendChild(script);
}

function inspectFlowCard(card, state, scenario) {
  const shadow = card?.shadowRoot;
  const svg = shadow?.querySelector("svg");
  if (!svg) {
    return {
      ok: false,
      scenario: state.scenario,
      device: state.device,
      view: state.view,
      errors: ["SVG not found"],
      warnings: [],
      metrics: {},
    };
  }

  const svgBounds = absoluteBox(svg);
  const nodes = collectNodes(shadow);
  const chips = collectChips(shadow);
  const routePills = collectRoutePills(shadow, svg);
  const warnings = [];

  chips.forEach((chip) => {
    if (outside(chip.box, svgBounds, 0)) {
      warnings.push(`${chip.id} is outside the stage bounds`);
    }
  });

  routePills.forEach((pill) => {
    if (outside(pill.box, svgBounds, 0)) {
      warnings.push(`${pill.id} is outside the stage bounds`);
    }
  });

  Object.values(nodes).forEach((node) => {
    if (outside(node.labelBox, svgBounds, 0)) {
      warnings.push(`${node.id} label is outside the stage bounds`);
    }
    if (node.iconBox) {
      const dx = Math.abs(node.iconCenter.x - node.circleCenter.x);
      const dy = Math.abs(node.iconCenter.y - node.circleCenter.y);
      if (dx > 3 || dy > 3) {
        warnings.push(`${node.id} icon is off-center (dx=${dx.toFixed(1)}, dy=${dy.toFixed(1)})`);
      }
    }
  });

  chips.forEach((chip) => {
    Object.values(nodes).forEach((node) => {
      if (node.id !== chip.node && intersects(chip.box, node.circleBox, 3)) {
        warnings.push(`${chip.id} overlaps ${node.id} circle`);
      }
      if (node.id !== chip.node && intersects(chip.box, node.labelBox, 1)) {
        warnings.push(`${chip.id} overlaps ${node.id} label`);
      }
    });
  });

  routePills.forEach((pill) => {
    Object.values(nodes).forEach((node) => {
      if (intersects(pill.box, node.circleBox, 3)) {
        warnings.push(`${pill.id} overlaps ${node.id} circle`);
      }
      if (intersects(pill.box, node.labelBox, 1)) {
        warnings.push(`${pill.id} overlaps ${node.id} label`);
      }
    });
  });

  for (let index = 0; index < chips.length; index += 1) {
    for (let inner = index + 1; inner < chips.length; inner += 1) {
      if (intersects(chips[index].box, chips[inner].box, 2)) {
        warnings.push(`${chips[index].id} overlaps ${chips[inner].id}`);
      }
    }
  }

  for (let index = 0; index < routePills.length; index += 1) {
    for (let inner = index + 1; inner < routePills.length; inner += 1) {
      if (intersects(routePills[index].box, routePills[inner].box, 2)) {
        warnings.push(`${routePills[index].id} overlaps ${routePills[inner].id}`);
      }
    }
  }

  routePills.forEach((pill) => {
    if (pill.path) {
      const midpoint = pointOnScreen(pill.path, pill.path.getTotalLength() * 0.5);
      const center = centerOf(pill.box);
      const distance = Math.hypot(center.x - midpoint.x, center.y - midpoint.y);
      if (distance > 34) {
        warnings.push(`${pill.id} is too far from its edge midpoint (${distance.toFixed(1)})`);
      }
    }
  });

  return {
    ok: warnings.length === 0,
    scenario: state.scenario,
    device: state.device,
    view: state.view,
    warnings,
    metrics: {
      nodeIcons: Object.values(nodes).map((node) => ({
        id: node.id,
        circleCenter: roundPoint(node.circleCenter),
        iconCenter: node.iconBox ? roundPoint(node.iconCenter) : null,
        iconOffset: node.iconBox
          ? {
              dx: Number((node.iconCenter.x - node.circleCenter.x).toFixed(1)),
              dy: Number((node.iconCenter.y - node.circleCenter.y).toFixed(1)),
            }
          : null,
      })),
      nodeChips: chips.map((chip) => ({ id: chip.id, box: roundBox(chip.box) })),
      routePills: routePills.map((pill) => ({ id: pill.id, box: roundBox(pill.box) })),
    },
  };
}

function collectNodes(shadow) {
  const nodes = {};
  shadow.querySelectorAll(".node-button[data-node]").forEach((group) => {
    const id = group.getAttribute("data-node");
    const circle = group.querySelector(".node-ring");
    const label = group.querySelector(`.node-label[data-node-label="${id}"]`);
    const icon = group.querySelector(".node-icon");
    const circleBox = absoluteBox(circle);
    const iconBox = icon ? absoluteBox(icon) : null;
    nodes[id] = {
      id,
      circleBox,
      circleCenter: centerOf(circleBox),
      iconBox,
      iconCenter: iconBox ? centerOf(iconBox) : null,
      labelBox: absoluteBox(label),
    };
  });
  return nodes;
}

function collectChips(shadow) {
  return Array.from(shadow.querySelectorAll(".node-chip[data-node][data-chip]")).map((group) => ({
    id: `${group.getAttribute("data-node")}-${group.getAttribute("data-chip")}`,
    node: group.getAttribute("data-node"),
    chipType: group.getAttribute("data-chip"),
    box: absoluteBox(group),
  }));
}

function collectRoutePills(shadow, svg) {
  return Array.from(shadow.querySelectorAll(".route-pill[data-edge]")).map((group) => {
    const edgeId = group.getAttribute("data-edge");
    return {
      id: edgeId,
      edge: edgeId,
      box: absoluteBox(group),
      path: svg.querySelector(`.edge-active[data-edge="${edgeId}"]`),
    };
  });
}

function absoluteBox(element) {
  const rect = element.getBoundingClientRect();
  return {
    x: rect.left,
    y: rect.top,
    width: rect.width,
    height: rect.height,
  };
}

function roundPoint(point) {
  return {
    x: Number(point.x.toFixed(1)),
    y: Number(point.y.toFixed(1)),
  };
}

function pointOnScreen(path, length) {
  const point = path.getPointAtLength(length);
  const matrix = path.getScreenCTM();
  if (!matrix) {
    return { x: point.x, y: point.y };
  }
  return {
    x: matrix.a * point.x + matrix.c * point.y + matrix.e,
    y: matrix.b * point.x + matrix.d * point.y + matrix.f,
  };
}

function intersects(a, b, margin = 0) {
  return !(
    a.x + a.width + margin <= b.x ||
    b.x + b.width + margin <= a.x ||
    a.y + a.height + margin <= b.y ||
    b.y + b.height + margin <= a.y
  );
}

function outside(box, bounds, margin = 0) {
  return (
    box.x < bounds.x + margin ||
    box.y < bounds.y + margin ||
    box.x + box.width > bounds.x + bounds.width - margin ||
    box.y + box.height > bounds.y + bounds.height - margin
  );
}

function centerOf(box) {
  return {
    x: box.x + box.width / 2,
    y: box.y + box.height / 2,
  };
}

function roundBox(box) {
  return {
    x: Number(box.x.toFixed(1)),
    y: Number(box.y.toFixed(1)),
    width: Number(box.width.toFixed(1)),
    height: Number(box.height.toFixed(1)),
  };
}

function buildHass(scenario) {
  const solar = scenario.flows.pvToLoad + scenario.flows.pvToBattery + scenario.flows.pvToGrid;
  const home = scenario.flows.pvToLoad + scenario.flows.gridToLoad + scenario.flows.batteryToLoad;
  const grid = scenario.flows.gridToLoad - scenario.flows.pvToGrid;
  const battery = scenario.flows.batteryToLoad - scenario.flows.pvToBattery;

  return {
    locale: { language: "en-US" },
    states: {
      [ENTITY_IDS.solarPower]: stateObj(solar, "kW"),
      [ENTITY_IDS.loadPower]: stateObj(home, "kW"),
      [ENTITY_IDS.batteryPower]: stateObj(battery, "kW"),
      [ENTITY_IDS.batterySoc]: stateObj(scenario.flows.batterySoc, "%"),
      [ENTITY_IDS.gridPower]: stateObj(grid, "kW"),
      [ENTITY_IDS.pvToLoadPower]: stateObj(scenario.flows.pvToLoad, "kW"),
      [ENTITY_IDS.pvToBatteryPower]: stateObj(scenario.flows.pvToBattery, "kW"),
      [ENTITY_IDS.pvToGridPower]: stateObj(scenario.flows.pvToGrid, "kW"),
      [ENTITY_IDS.batteryToLoadPower]: stateObj(scenario.flows.batteryToLoad, "kW"),
      [ENTITY_IDS.gridToLoadPower]: stateObj(scenario.flows.gridToLoad, "kW"),
      [ENTITY_IDS.dailyPvYield]: stateObj(scenario.daily.pvYield, "kWh"),
      [ENTITY_IDS.dailyPvToLoad]: stateObj(scenario.daily.pvToLoad, "kWh"),
      [ENTITY_IDS.dailyPvToBattery]: stateObj(scenario.daily.pvToBattery, "kWh"),
      [ENTITY_IDS.dailyFeedIn]: stateObj(scenario.daily.feedIn, "kWh"),
      [ENTITY_IDS.dailyGridImport]: stateObj(scenario.daily.gridImport, "kWh"),
    },
  };
}

function renderTiles(daily, batterySoc) {
  const tiles = [
    { label: "PV Yield", value: `${formatFixed(daily.pvYield)} kWh` },
    { label: "PV To Load", value: `${formatFixed(daily.pvToLoad)} kWh` },
    { label: "PV To Battery", value: `${formatFixed(daily.pvToBattery)} kWh` },
    { label: "Feed-in", value: `${formatFixed(daily.feedIn)} kWh` },
    { label: "Grid Import", value: `${formatFixed(daily.gridImport)} kWh` },
    { label: "Battery SOC", value: `${Math.round(batterySoc)} %` },
  ];

  return tiles
    .map((tile) => `
      <div class="tile-card">
        <div class="tile-label">${escapeHtml(tile.label)}</div>
        <div class="tile-value">${escapeHtml(tile.value)}</div>
      </div>
    `)
    .join("");
}

function renderChartCard(title, unit, seriesList) {
  const { svg, legend } = buildChartSvg(unit, seriesList);
  return `
    <div class="chart-card">
      <div class="chart-title">${escapeHtml(title)}</div>
      ${svg}
      <div class="legend">${legend}</div>
    </div>
  `;
}

function buildChartSvg(unit, seriesList) {
  const width = 520;
  const height = 240;
  const margin = { top: 16, right: 14, bottom: 28, left: 38 };
  const plotWidth = width - margin.left - margin.right;
  const plotHeight = height - margin.top - margin.bottom;
  const values = seriesList.flatMap((seriesDef) => seriesDef.values);
  const min = Math.min(0, ...values);
  const max = Math.max(...values, 1);
  const range = Math.max(max - min, 1);
  const ticks = 4;
  const grid = [];
  const labels = [];

  for (let index = 0; index <= ticks; index += 1) {
    const y = margin.top + (plotHeight / ticks) * index;
    const value = max - (range / ticks) * index;
    grid.push(`<line class="chart-gridline" x1="${margin.left}" y1="${y}" x2="${width - margin.right}" y2="${y}"></line>`);
    labels.push(`<text class="chart-axis" x="${margin.left - 8}" y="${y + 4}" text-anchor="end">${formatAxis(value)}</text>`);
  }

  const seriesMarkup = seriesList
    .map((seriesDef) => {
      const points = seriesDef.values
        .map((value, index) => {
          const x = margin.left + (plotWidth / Math.max(seriesDef.values.length - 1, 1)) * index;
          const y = margin.top + ((max - value) / range) * plotHeight;
          return `${x.toFixed(1)},${y.toFixed(1)}`;
        })
        .join(" ");
      return `<polyline fill="none" stroke="${seriesDef.color}" stroke-width="2.2" stroke-linejoin="round" stroke-linecap="round" points="${points}"></polyline>`;
    })
    .join("");

  const svg = `
    <svg class="chart-frame" viewBox="0 0 ${width} ${height}" preserveAspectRatio="none">
      ${grid.join("")}
      ${labels.join("")}
      <text class="chart-axis" x="${margin.left}" y="12">${escapeHtml(unit)}</text>
      ${seriesMarkup}
    </svg>
  `;

  const legend = seriesList
    .map((seriesDef) => `
      <span class="legend-item">
        <span class="legend-dot" style="background:${seriesDef.color};"></span>
        ${escapeHtml(seriesDef.name)}
      </span>
    `)
    .join("");

  return { svg, legend };
}

function stateObj(value, unit) {
  return {
    state: String(value),
    attributes: {
      unit_of_measurement: unit,
    },
  };
}

function cumulative(values) {
  const output = [];
  let total = 0;
  values.forEach((value) => {
    total += value;
    output.push(Number(total.toFixed(2)));
  });
  return output;
}

function series(name, values) {
  return {
    name,
    values,
    color: SERIES_COLORS[name] || "#ffffff",
  };
}

function cloneSeries(seriesDef, nameOverride) {
  return {
    name: nameOverride || seriesDef.name,
    values: [...seriesDef.values],
    color: SERIES_COLORS[nameOverride || seriesDef.name] || seriesDef.color || "#ffffff",
  };
}

function findSeries(seriesList, name) {
  const found = seriesList.find((seriesDef) => seriesDef.name === name);
  if (!found) {
    throw new Error(`Missing series ${name}`);
  }
  return found;
}

function formatFixed(value) {
  return value.toFixed(2);
}

function formatAxis(value) {
  if (Math.abs(value) >= 10) {
    return value.toFixed(0);
  }
  if (Math.abs(value) >= 1) {
    return value.toFixed(1);
  }
  return value.toFixed(2);
}

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}
