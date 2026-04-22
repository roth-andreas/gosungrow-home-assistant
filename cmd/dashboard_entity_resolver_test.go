package cmd

import "testing"

func TestRemapDashboardEntitiesRemapsLegacyVirtualSensorsByMetricAndPlantAffinity(t *testing.T) {
	config := map[string]any{
		"views": []any{
			map[string]any{
				"cards": []any{
					map[string]any{
						"type": "custom:gosungrow-energy-flow-card-v2",
						"entities": map[string]any{
							"load_power": "sensor.gosungrow_virtual_1610907_22_247_1_load_power",
							"pv_power":   "sensor.gosungrow_virtual_1610907_22_247_1_pv_power",
						},
					},
					map[string]any{
						"type":   "tile",
						"entity": "sensor.gosungrow_virtual_1610907_22_247_1_load_power",
					},
				},
			},
		},
	}

	targets := []haDashboardTarget{
		{PsID: "1610907", PsKey: "1610907_22_247_1"},
	}
	states := []haState{
		dashboardTestState("sensor.gosungrow_1610907_sungrow_gosungrow_load_information_load_power", "0.72", "kW"),
		dashboardTestState("sensor.gosungrow_1610907_sungrow_gosungrow_pv_information_pv_power", "1.23", "kW"),
	}

	remapped := remapDashboardEntities(config, targets, states)
	views := remapped["views"].([]any)
	cards := views[0].(map[string]any)["cards"].([]any)
	flowEntities := cards[0].(map[string]any)["entities"].(map[string]any)

	if got := flowEntities["load_power"]; got != "sensor.gosungrow_1610907_sungrow_gosungrow_load_information_load_power" {
		t.Fatalf("unexpected remapped load_power entity: %v", got)
	}
	if got := flowEntities["pv_power"]; got != "sensor.gosungrow_1610907_sungrow_gosungrow_pv_information_pv_power" {
		t.Fatalf("unexpected remapped pv_power entity: %v", got)
	}
	if got := cards[1].(map[string]any)["entity"]; got != "sensor.gosungrow_1610907_sungrow_gosungrow_load_information_load_power" {
		t.Fatalf("unexpected remapped tile entity: %v", got)
	}
}

func TestRemapDashboardEntitiesKeepsLegacyWhenFallbackIsAmbiguousAcrossTargets(t *testing.T) {
	config := map[string]any{
		"views": []any{
			map[string]any{
				"cards": []any{
					map[string]any{
						"type":   "tile",
						"entity": "sensor.gosungrow_virtual_100_14_1_1_p13141",
					},
				},
			},
		},
	}

	targets := []haDashboardTarget{
		{PsID: "100", PsKey: "100_14_1_1"},
		{PsID: "200", PsKey: "200_14_1_1"},
	}
	states := []haState{
		dashboardTestState("sensor.gosungrow_something_p13141", "88", "%"),
	}

	remapped := remapDashboardEntities(config, targets, states)
	views := remapped["views"].([]any)
	cards := views[0].(map[string]any)["cards"].([]any)
	if got := cards[0].(map[string]any)["entity"]; got != "sensor.gosungrow_virtual_100_14_1_1_p13141" {
		t.Fatalf("expected legacy entity to stay unchanged, got %v", got)
	}
}

func TestRemapDashboardEntitiesUsesBestPlantMatchWhenMultipleCandidatesExist(t *testing.T) {
	config := map[string]any{
		"views": []any{
			map[string]any{
				"cards": []any{
					map[string]any{
						"type":   "tile",
						"entity": "sensor.gosungrow_virtual_100_14_1_1_p13112",
					},
				},
			},
		},
	}

	targets := []haDashboardTarget{
		{PsID: "100", PsKey: "100_14_1_1"},
		{PsID: "200", PsKey: "200_14_1_1"},
	}
	states := []haState{
		dashboardTestState("sensor.gosungrow_200_sungrow_gosungrow_p13112", "11.1", "kWh"),
		dashboardTestState("sensor.gosungrow_100_sungrow_gosungrow_p13112", "22.2", "kWh"),
	}

	remapped := remapDashboardEntities(config, targets, states)
	views := remapped["views"].([]any)
	cards := views[0].(map[string]any)["cards"].([]any)
	if got := cards[0].(map[string]any)["entity"]; got != "sensor.gosungrow_100_sungrow_gosungrow_p13112" {
		t.Fatalf("expected best matching plant entity, got %v", got)
	}
}

func TestRemapDashboardEntitiesUsesSemanticTokenMatchingForModernPowerNames(t *testing.T) {
	config := map[string]any{
		"views": []any{
			map[string]any{
				"cards": []any{
					map[string]any{
						"type": "custom:gosungrow-energy-flow-card-v2",
						"entities": map[string]any{
							"load_power": "sensor.gosungrow_virtual_1610907_22_247_1_load_power",
							"pv_power":   "sensor.gosungrow_virtual_1610907_22_247_1_pv_power",
							"grid_power": "sensor.gosungrow_virtual_1610907_22_247_1_grid_power",
						},
					},
				},
			},
		},
	}

	targets := []haDashboardTarget{
		{PsID: "1610907", PsKey: "1610907_22_247_1"},
	}
	states := []haState{
		dashboardTestState("sensor.gosungrow_1610907_sungrow_gosungrow_load_information_total_active_power", "0.72", "kW"),
		dashboardTestState("sensor.gosungrow_1610907_sungrow_gosungrow_pv_information_total_active_power", "1.23", "kW"),
		dashboardTestState("sensor.gosungrow_1610907_sungrow_gosungrow_grid_information_total_active_power", "0.10", "kW"),
	}

	remapped := remapDashboardEntities(config, targets, states)
	views := remapped["views"].([]any)
	cards := views[0].(map[string]any)["cards"].([]any)
	flowEntities := cards[0].(map[string]any)["entities"].(map[string]any)

	if got := flowEntities["load_power"]; got != "sensor.gosungrow_1610907_sungrow_gosungrow_load_information_total_active_power" {
		t.Fatalf("unexpected remapped load_power entity: %v", got)
	}
	if got := flowEntities["pv_power"]; got != "sensor.gosungrow_1610907_sungrow_gosungrow_pv_information_total_active_power" {
		t.Fatalf("unexpected remapped pv_power entity: %v", got)
	}
	if got := flowEntities["grid_power"]; got != "sensor.gosungrow_1610907_sungrow_gosungrow_grid_information_total_active_power" {
		t.Fatalf("unexpected remapped grid_power entity: %v", got)
	}
}

func TestRemapDashboardEntitiesMapsDailyEnergyPointAliases(t *testing.T) {
	config := map[string]any{
		"views": []any{
			map[string]any{
				"cards": []any{
					map[string]any{
						"type":   "tile",
						"entity": "sensor.gosungrow_virtual_1610907_22_247_1_p13112",
					},
					map[string]any{
						"type":   "tile",
						"entity": "sensor.gosungrow_virtual_1610907_22_247_1_p13147",
					},
				},
			},
		},
	}

	targets := []haDashboardTarget{
		{PsID: "1610907", PsKey: "1610907_22_247_1"},
	}
	states := []haState{
		dashboardTestState("sensor.gosungrow_1610907_sungrow_gosungrow_pv_information_pv_daily_energy", "47.70", "kWh"),
		dashboardTestState("sensor.gosungrow_1610907_sungrow_gosungrow_grid_information_grid_to_load_energy", "2.40", "kWh"),
	}

	remapped := remapDashboardEntities(config, targets, states)
	views := remapped["views"].([]any)
	cards := views[0].(map[string]any)["cards"].([]any)

	if got := cards[0].(map[string]any)["entity"]; got != "sensor.gosungrow_1610907_sungrow_gosungrow_pv_information_pv_daily_energy" {
		t.Fatalf("unexpected remapped p13112 alias: %v", got)
	}
	if got := cards[1].(map[string]any)["entity"]; got != "sensor.gosungrow_1610907_sungrow_gosungrow_grid_information_grid_to_load_energy" {
		t.Fatalf("unexpected remapped p13147 alias: %v", got)
	}
}

func TestRemapDashboardEntitiesDoesNotMapBatteryEnergyToPvYield(t *testing.T) {
	config := map[string]any{
		"views": []any{
			map[string]any{
				"cards": []any{
					map[string]any{
						"type":   "tile",
						"entity": "sensor.gosungrow_virtual_1610907_22_247_1_p13174",
					},
				},
			},
		},
	}

	targets := []haDashboardTarget{
		{PsID: "1610907", PsKey: "1610907_22_247_1"},
	}
	states := []haState{
		dashboardTestState("sensor.gosungrow_1610907_sungrow_gosungrow_pv_information_pv_daily_energy", "47.70", "kWh"),
	}

	remapped := remapDashboardEntities(config, targets, states)
	views := remapped["views"].([]any)
	cards := views[0].(map[string]any)["cards"].([]any)

	if got := cards[0].(map[string]any)["entity"]; got != "sensor.gosungrow_virtual_1610907_22_247_1_p13174" {
		t.Fatalf("expected p13174 to stay unchanged without a battery-energy candidate, got %v", got)
	}
}

func TestRemapDashboardEntitiesDoesNotMapFeedInEnergyToPvYield(t *testing.T) {
	config := map[string]any{
		"views": []any{
			map[string]any{
				"cards": []any{
					map[string]any{
						"type":   "tile",
						"entity": "sensor.gosungrow_virtual_1610907_22_247_1_p13173",
					},
				},
			},
		},
	}

	targets := []haDashboardTarget{
		{PsID: "1610907", PsKey: "1610907_22_247_1"},
	}
	states := []haState{
		dashboardTestState("sensor.gosungrow_1610907_sungrow_gosungrow_pv_information_pv_daily_energy", "47.70", "kWh"),
	}

	remapped := remapDashboardEntities(config, targets, states)
	views := remapped["views"].([]any)
	cards := views[0].(map[string]any)["cards"].([]any)

	if got := cards[0].(map[string]any)["entity"]; got != "sensor.gosungrow_virtual_1610907_22_247_1_p13173" {
		t.Fatalf("expected p13173 to stay unchanged without a feed-in candidate, got %v", got)
	}
}

func TestRemapDashboardEntitiesSkipsUnknownStatesAndWrongUnits(t *testing.T) {
	config := map[string]any{
		"views": []any{
			map[string]any{
				"cards": []any{
					map[string]any{
						"type": "custom:gosungrow-energy-flow-card-v2",
						"entities": map[string]any{
							"pv_power":              "sensor.gosungrow_virtual_1610907_22_247_1_pv_power",
							"pv_to_battery_power":   "sensor.gosungrow_virtual_1610907_22_247_1_pv_to_battery_power",
							"grid_to_load_power":    "sensor.gosungrow_virtual_1610907_22_247_1_grid_to_load_power",
							"battery_to_load_power": "sensor.gosungrow_virtual_1610907_22_247_1_battery_to_load_power",
						},
					},
				},
			},
		},
	}

	targets := []haDashboardTarget{
		{PsID: "1610907", PsKey: "1610907_22_247_1"},
	}
	states := []haState{
		dashboardTestState("sensor.gosungrow_1610907_sungrow_gosungrow_pv_information_pv_power", "unknown", "kW"),
		dashboardTestState("sensor.gosungrow_1610907_sungrow_gosungrow_pv_information_total_dc_power", "1.23", "kW"),
		dashboardTestState("sensor.gosungrow_1610907_sungrow_gosungrow_pv_information_pv_to_battery_energy", "47.70", "kWh"),
		dashboardTestState("sensor.gosungrow_1610907_sungrow_gosungrow_grid_information_grid_to_load_power", "0.72", "kW"),
	}

	remapped := remapDashboardEntities(config, targets, states)
	views := remapped["views"].([]any)
	cards := views[0].(map[string]any)["cards"].([]any)
	flowEntities := cards[0].(map[string]any)["entities"].(map[string]any)

	if got := flowEntities["pv_power"]; got != "sensor.gosungrow_1610907_sungrow_gosungrow_pv_information_total_dc_power" {
		t.Fatalf("expected usable total_dc_power to replace unknown pv_power, got %v", got)
	}
	if got := flowEntities["pv_to_battery_power"]; got != "sensor.gosungrow_virtual_1610907_22_247_1_pv_to_battery_power" {
		t.Fatalf("expected kWh battery energy not to replace pv_to_battery_power, got %v", got)
	}
	if got := flowEntities["grid_to_load_power"]; got != "sensor.gosungrow_1610907_sungrow_gosungrow_grid_information_grid_to_load_power" {
		t.Fatalf("expected grid_to_load_power to remap to kW candidate, got %v", got)
	}
}
