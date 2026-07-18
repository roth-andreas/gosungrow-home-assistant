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
  dailyConsumption: "sensor.preview_daily_consumption",
  dailyBatteryDischarge: "sensor.preview_daily_battery_discharge",
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
    sourceState: params.get("sourceState") || "automatic",
    theme: params.get("theme") || "dark",
    language: params.get("language") || "en",
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
  document.getElementById("source-state").value = state.sourceState;
  document.getElementById("theme").value = state.theme;
  document.getElementById("language").value = state.language;

  document.getElementById("apply").addEventListener("click", () => {
    const next = new URLSearchParams(window.location.search);
    next.set("scenario", document.getElementById("scenario").value);
    next.set("view", document.getElementById("view").value);
    next.set("device", document.getElementById("device").value);
    next.set("sourceState", document.getElementById("source-state").value);
    next.set("theme", document.getElementById("theme").value);
    next.set("language", document.getElementById("language").value);
    next.set("chrome", state.chrome ? "1" : "0");
    window.location.search = next.toString();
  });
}

function render(state) {
  document.documentElement.dataset.theme = state.theme;
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

  if (state.view === "aggregates") {
    const aggregates = document.createElement("div");
    aggregates.innerHTML = `
    <div class="summary-preview-card">
      <div id="summary-mount"></div>
    </div>
    `;
    root.appendChild(aggregates);
    mountSummaryCard(root.querySelector("#summary-mount"), scenario);
    return root;
  }

  if (state.view === "sources") {
    const sources = document.createElement("div");
    sources.innerHTML = `<div id="sources-mount"></div>`;
    root.appendChild(sources);
    mountSourcesCard(root.querySelector("#sources-mount"), scenario, state.sourceState, state.language);
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

function mountSourcesCard(container, scenario, sourceState, language = "en") {
  const card = document.createElement("gosungrow-source-mapping-card-v1");
  const issue = sourceState === "warning";
  const manual = sourceState === "manual" || sourceState === "saved";
  const missing = sourceState === "missing";
  const empty = sourceState === "empty";
  const longPrefix = ["multi", "dialog", "save_error"].includes(sourceState)
    ? "GoSungrow 5072099_14_1_1 - SH6.0RT(COM1-001)_001_001 - "
    : "";
  const translations = {
    en: { title: "Data Sources", subtitle: "Review automatic matches or choose a dashboard override.", automatic: "Automatic", manual: "Manual", review: "Needs review", unavailable: "Unavailable", configure: "Configure", recommended: "Recommended", other: "Other compatible entities", search: "Search entities", use: "Use this source", reset: "Reset to automatic", cancel: "Cancel", saved: "Data source saved.", readonly: "Only Home Assistant administrators can change data sources.", groups: { live_power: "Live power", today_energy: "Today's energy", battery: "Battery", energy_summary: "Energy summary" }, production: "Solar production today", direct: "Direct solar consumption" },
    de: { title: "Datenquellen", subtitle: "Automatische Zuordnungen prüfen oder eine Dashboard-Quelle auswählen.", automatic: "Automatisch", manual: "Manuell", review: "Prüfung nötig", unavailable: "Nicht verfügbar", configure: "Konfigurieren", recommended: "Empfohlen", other: "Weitere passende Entitäten", search: "Entitäten suchen", use: "Diese Quelle verwenden", reset: "Auf Automatik zurücksetzen", cancel: "Abbrechen", saved: "Datenquelle gespeichert.", readonly: "Nur Home-Assistant-Administratoren können Datenquellen ändern.", groups: { live_power: "Aktuelle Leistung", today_energy: "Heutige Energie", battery: "Batterie", energy_summary: "Energieübersicht" }, production: "Solarerzeugung heute", direct: "Direkter Solarverbrauch" },
    sv: { title: "Datakällor", subtitle: "Granska automatiska matchningar eller välj en källa för instrumentpanelen.", automatic: "Automatisk", manual: "Manuell", review: "Behöver granskas", unavailable: "Inte tillgänglig", configure: "Konfigurera", recommended: "Rekommenderad", other: "Andra kompatibla entiteter", search: "Sök entiteter", use: "Använd denna källa", reset: "Återställ till automatisk", cancel: "Avbryt", saved: "Datakällan sparades.", readonly: "Endast Home Assistant-administratörer kan ändra datakällor.", groups: { live_power: "Aktuell effekt", today_energy: "Dagens energi", battery: "Batteri", energy_summary: "Energisammanfattning" }, production: "Solproduktion idag", direct: "Direkt solförbrukning" },
  };
  const text = translations[language] || translations.en;
  const liveStates = {};
  const metric = (key, group, icon, label, entity, state, unit) => {
    const candidates = empty ? [] : [
      { entity_id: entity, point_id: key, score: 260, confidence: "high", reason: "Current automatic match", recommended: true },
      { entity_id: `${entity}_alternative`, device: "Plant total", point_id: key, score: 248, confidence: "high", reason: "Compatible unit and same plant", recommended: true },
      { entity_id: `${entity}_device_2`, device: "Device 2", point_id: key, score: 196, confidence: "medium", reason: "Compatible device-level sensor", recommended: false },
    ];
    liveStates[entity] = { state: String(state), attributes: { unit_of_measurement: unit, friendly_name: `${longPrefix}${label}` } };
    liveStates[`${entity}_alternative`] = { state: String(Math.max(0, Number(state) - 1.2)), attributes: { unit_of_measurement: unit, friendly_name: `${longPrefix}${label} – Plant total` } };
    liveStates[`${entity}_device_2`] = { state: String(Math.max(0, Number(state) / 2)), attributes: { unit_of_measurement: unit, friendly_name: `${longPrefix}${label} – Device 2` } };
    return { key, group, icon, label, default: entity, confidence: "high", reason: "Current automatic match", candidates };
  };
  const production = 42.8;
  const direct = issue ? 52.6 : 30.5;
  const metrics = [
    metric("pv_power", "live_power", "mdi:solar-power", "Solar power", ENTITY_IDS.solarPower, 3.2, "kW"),
    metric("load_power", "live_power", "mdi:home-lightning-bolt-outline", "Home consumption power", ENTITY_IDS.loadPower, 0.9, "kW"),
    metric("grid_power", "live_power", "mdi:transmission-tower", "Grid power", ENTITY_IDS.gridPower, -0.2, "kW"),
    metric("p13112", "today_energy", "mdi:white-balance-sunny", text.production, ENTITY_IDS.dailyPvYield, production, "kWh"),
    metric("p13116", "today_energy", "mdi:home-lightning-bolt-outline", text.direct, ENTITY_IDS.dailyPvToLoad, direct, "kWh"),
    metric("p13174", "today_energy", "mdi:battery-charging-medium", "Solar energy to battery", ENTITY_IDS.dailyPvToBattery, 12.2, "kWh"),
    metric("p13141", "battery", "mdi:battery-medium", "Battery state of charge", ENTITY_IDS.batterySoc, 75, "%"),
    metric("battery_power", "battery", "mdi:battery-charging", "Battery power", ENTITY_IDS.batteryPower, -2.1, "kW"),
    metric("p13199", "energy_summary", "mdi:home-lightning-bolt", "Home consumption today", ENTITY_IDS.dailyConsumption, 31.2, "kWh"),
    metric("p13147", "energy_summary", "mdi:download-network-outline", "Grid import today", ENTITY_IDS.dailyGridImport, 0.2, "kWh"),
  ];
  metrics.find((item) => item.key === "p13116").validation = { schema_version: 1, rules: [{ type: "not_materially_greater_than", metric: "p13112", relative_tolerance: 0.05, absolute_tolerance: 0.1 }] };
  const overrides = manual ? { p13116: `${ENTITY_IDS.dailyPvToLoad}_alternative` } : missing ? { p13116: "sensor.preview_missing_manual" } : {};
  if (manual || missing) { const selectedMetric = metrics.find((item) => item.key === "p13116"); selectedMetric.confidence = "manual"; selectedMetric.reason = "Current manual selection"; }
  const defaults = Object.fromEntries(metrics.map((item) => [item.key, item.default]));
  const candidates = Object.fromEntries(metrics.map((item) => [item.key, item.candidates]));
  metrics.forEach((item) => delete item.candidates);
  const bindings = Object.fromEntries(metrics.map((item, index) => [item.key, [`/views/0/cards/${index}/entity`]]));
  const labels = { title: text.title, subtitle: text.subtitle, automatic: text.automatic, manual: text.manual, needs_review: text.review, unavailable: text.unavailable, configure: text.configure, recommended: text.recommended, other: text.other, search: text.search, use_source: text.use, reset: text.reset, cancel: text.cancel, saved: text.saved, readonly: text.readonly,
    source_unavailable_warning: language === "de" ? "Die ausgewählte Entität ist nicht verfügbar oder nicht numerisch." : language === "sv" ? "Den valda entiteten är inte tillgänglig eller saknar numeriskt värde." : "The selected entity is unavailable or non-numeric.",
    source_physical_warning: language === "de" ? "Der ausgewählte Wert ({value}) überschreitet die Solarerzeugung ({reference}). Bitte Quelle prüfen." : language === "sv" ? "Det valda värdet ({value}) överstiger solproduktionen ({reference}). Granska källan." : "Selected value ({value}) exceeds solar production ({reference}). Review this source.",
    source_save_error: language === "de" ? "Datenquelle konnte nicht gespeichert werden." : language === "sv" ? "Datakällan kunde inte sparas." : "Could not save the data source.", confidence_high: language === "de" ? "Hohe Zuverlässigkeit" : language === "sv" ? "Hög säkerhet" : "High confidence", confidence_medium: language === "de" ? "Mittlere Zuverlässigkeit" : language === "sv" ? "Medelhög säkerhet" : "Medium confidence", confidence_low: language === "de" ? "Niedrige Zuverlässigkeit" : language === "sv" ? "Låg säkerhet" : "Low confidence", confidence_manual: language === "de" ? "Vom Benutzer gewählt" : language === "sv" ? "Användarvald" : "User selected", groups: text.groups };
  const config = { type: "custom:gosungrow-source-mapping-card-v1", schema_version: 1, mapping_id: "preview", dashboard_url_path: "gosungrow-flow", defaults, overrides, candidates, bindings, metrics, labels };
  card.setConfig(config);
  const hass = buildHass(scenario);
  hass.user = { is_admin: sourceState !== "readonly" };
  hass.locale = { language };
  Object.assign(hass.states, liveStates);
  if (missing) delete hass.states["sensor.preview_missing_manual"];
  const dashboard = { views: [{ cards: metrics.map((item) => ({ entity: overrides[item.key] || item.default })) }, { cards: [{ ...structuredClone(config) }] }] };
  hass.callWS = async (request) => {
    if (sourceState === "save_error" && request.type === "lovelace/config/save") throw new Error("Preview: Home Assistant rejected the save.");
    if (request.type === "lovelace/config") return structuredClone(dashboard);
    if (request.type === "lovelace/config/save") { Object.assign(dashboard, structuredClone(request.config)); return {}; }
    return {};
  };
  card.hass = hass;
  container.appendChild(card);
  if (sourceState === "saved") { card._notice = text.saved; card._render(); }
  if (["dialog", "save_error", "empty"].includes(sourceState)) {
    card._activeMetric = card._metrics().find((item) => item.key === "p13116");
    card._pendingEntity = sourceState === "save_error" ? `${ENTITY_IDS.dailyPvToLoad}_alternative` : card._selected(card._activeMetric);
    if (sourceState === "save_error") card._notice = language === "de" ? "Vorschau: Home Assistant hat das Speichern abgelehnt." : language === "sv" ? "Förhandsvisning: Home Assistant avvisade sparningen." : "Preview: Home Assistant rejected the save.";
    card._render();
  }
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

function mountSummaryCard(container, scenario) {
  const card = document.createElement("gosungrow-energy-summary-card-v1");
  card.setConfig({
    title: "Energy Summary",
    labels: {
      title: "Energy Summary",
      period_day: "Day",
      period_month: "Month",
      period_year: "Year",
      unavailable: "Unavailable",
      statistics_unavailable: "Statistics unavailable",
      name_production: "Production",
      name_consumption: "Consumption",
      name_to_grid: "To Grid",
      name_from_grid: "From Grid",
      name_to_battery: "To Battery",
      name_from_battery: "From Battery",
    },
    entities: {
      production: ENTITY_IDS.dailyPvYield,
      consumption: ENTITY_IDS.dailyConsumption,
      to_grid: ENTITY_IDS.dailyFeedIn,
      from_grid: ENTITY_IDS.dailyGridImport,
      to_battery: ENTITY_IDS.dailyPvToBattery,
      from_battery: ENTITY_IDS.dailyBatteryDischarge,
    },
  });
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
  const aggregates = aggregateValues(scenario);

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
      [ENTITY_IDS.dailyConsumption]: stateObj(aggregates.day.consumption, "kWh"),
      [ENTITY_IDS.dailyBatteryDischarge]: stateObj(aggregates.day.from_battery, "kWh"),
    },
    callWS: (request) => {
      if (request?.type !== "recorder/statistics_during_period") {
        return Promise.resolve({});
      }
      return Promise.resolve(buildStatisticsResponse(request, aggregates));
    },
  };
}

function aggregateValues(scenario) {
  const day = {
    production: scenario.daily.pvYield,
    consumption: scenario.daily.pvToLoad + scenario.daily.gridImport + scenario.flows.batteryToLoad * 2.6,
    to_grid: scenario.daily.feedIn,
    from_grid: scenario.daily.gridImport,
    to_battery: scenario.daily.pvToBattery,
    from_battery: scenario.flows.batteryToLoad * 2.6,
  };
  return {
    day,
    month: scaleAggregate(day, 18.4),
    year: scaleAggregate(day, 214.6),
  };
}

function scaleAggregate(values, factor) {
  return Object.fromEntries(Object.entries(values).map(([key, value]) => [key, Number((value * factor).toFixed(2))]));
}

function buildStatisticsResponse(request, aggregates) {
  const metricEntities = {
    production: ENTITY_IDS.dailyPvYield,
    consumption: ENTITY_IDS.dailyConsumption,
    to_grid: ENTITY_IDS.dailyFeedIn,
    from_grid: ENTITY_IDS.dailyGridImport,
    to_battery: ENTITY_IDS.dailyPvToBattery,
    from_battery: ENTITY_IDS.dailyBatteryDischarge,
  };
  const start = new Date(request.start_time);
  const end = new Date(request.end_time);
  const bucketStarts = request.period === "day" ? dailyBucketStarts(start, end) : monthlyBucketStarts(start, end);
  const response = {};

  Object.entries(metricEntities).forEach(([metric, entityID]) => {
    response[entityID] = bucketStarts.map((date, index) => {
      const value = bucketValue(metric, date, index, request.period, aggregates);
      return {
        start: date.toISOString(),
        sum: Number(value.toFixed(2)),
      };
    });
  });

  return response;
}

function dailyBucketStarts(start, end) {
  const current = new Date(start);
  current.setHours(0, 0, 0, 0);
  const limit = new Date(end);
  const output = [];
  while (current <= limit) {
    output.push(new Date(current));
    current.setDate(current.getDate() + 1);
  }
  return output;
}

function monthlyBucketStarts(start, end) {
  const current = new Date(start);
  current.setDate(1);
  current.setHours(0, 0, 0, 0);
  const limit = new Date(end);
  const output = [];
  while (current <= limit) {
    output.push(new Date(current));
    current.setMonth(current.getMonth() + 1);
  }
  return output;
}

function bucketValue(metric, date, index, period, aggregates) {
  const base = period === "day" ? aggregates.day[metric] : aggregates.month[metric];
  const seasonal = period === "day"
    ? 0.72 + ((index % 7) * 0.08)
    : 0.52 + Math.max(0.15, Math.sin(((date.getMonth() + 1) / 12) * Math.PI)) * 0.72;
  const flowBias = {
    production: 1,
    consumption: 0.9,
    to_grid: 0.75,
    from_grid: 1.12,
    to_battery: 0.82,
    from_battery: 0.72,
  }[metric] || 1;
  return Math.max(0, base * seasonal * flowBias);
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
