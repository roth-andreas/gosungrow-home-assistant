package cmd

import "testing"

func TestPruneDashboardForMissingBatteryPrunesBatteryCardsAndEntities(t *testing.T) {
	config := map[string]any{
		"views": []any{
			map[string]any{
				"sections": []any{
					map[string]any{
						"type": "grid",
						"cards": []any{
							map[string]any{
								"type":    "heading",
								"heading": "Live Flow",
							},
							map[string]any{
								"type": "custom:gosungrow-energy-flow-card-v2",
								"entities": map[string]any{
									"solar_power":           "sensor.gosungrow_virtual_100_14_1_1_pv_power",
									"load_power":            "sensor.gosungrow_virtual_100_14_1_1_load_power",
									"grid_power":            "sensor.gosungrow_virtual_100_14_1_1_grid_power",
									"battery_power":         "sensor.gosungrow_virtual_100_14_1_1_battery_power",
									"battery_soc":           "sensor.gosungrow_virtual_100_14_1_1_p13141",
									"pv_to_battery_power":   "sensor.gosungrow_virtual_100_14_1_1_pv_to_battery_power",
									"battery_to_load_power": "sensor.gosungrow_virtual_100_14_1_1_battery_to_load_power",
								},
							},
						},
					},
					map[string]any{
						"type": "grid",
						"cards": []any{
							map[string]any{
								"type":    "heading",
								"heading": "Battery",
							},
							map[string]any{
								"type":   "tile",
								"entity": "sensor.gosungrow_virtual_100_14_1_1_p13141",
							},
						},
					},
					map[string]any{
						"type": "grid",
						"cards": []any{
							map[string]any{
								"type":    "heading",
								"heading": "Power Balance",
							},
							map[string]any{
								"type": "history-graph",
								"entities": []any{
									map[string]any{"entity": "sensor.gosungrow_virtual_100_14_1_1_pv_power"},
									map[string]any{"entity": "sensor.gosungrow_virtual_100_14_1_1_battery_power"},
								},
							},
						},
					},
				},
			},
		},
	}

	targets := []haDashboardTarget{
		{PsID: "100", PsKey: "100_14_1_1"},
	}
	states := []haState{
		dashboardTestState("sensor.gosungrow_100_sungrow_gosungrow_pv_information_total_active_power", "1.20", "kW"),
		dashboardTestState("sensor.gosungrow_100_sungrow_gosungrow_load_information_total_active_power", "0.70", "kW"),
		dashboardTestState("sensor.gosungrow_100_sungrow_gosungrow_grid_information_total_active_power", "0.10", "kW"),
	}

	pruned := pruneDashboardForMissingBattery(config, targets, states)
	views := pruned["views"].([]any)
	sections := views[0].(map[string]any)["sections"].([]any)

	if len(sections) != 2 {
		t.Fatalf("expected battery-only section to be removed, got %d sections", len(sections))
	}

	firstCards := sections[0].(map[string]any)["cards"].([]any)
	flowEntities := firstCards[1].(map[string]any)["entities"].(map[string]any)
	if _, ok := flowEntities["battery_power"]; ok {
		t.Fatalf("expected battery_power to be removed from flow card entities")
	}
	if _, ok := flowEntities["battery_soc"]; ok {
		t.Fatalf("expected battery_soc to be removed from flow card entities")
	}
	if _, ok := flowEntities["pv_to_battery_power"]; ok {
		t.Fatalf("expected pv_to_battery_power to be removed from flow card entities")
	}
	if _, ok := flowEntities["battery_to_load_power"]; ok {
		t.Fatalf("expected battery_to_load_power to be removed from flow card entities")
	}

	historyCards := sections[1].(map[string]any)["cards"].([]any)
	historyEntities := historyCards[1].(map[string]any)["entities"].([]any)
	if len(historyEntities) != 1 {
		t.Fatalf("expected battery history entity to be pruned, got %d", len(historyEntities))
	}
}

func TestPruneDashboardForMissingBatteryAppliesPerViewTarget(t *testing.T) {
	config := map[string]any{
		"views": []any{
			map[string]any{
				"path": "with-battery",
				"sections": []any{
					map[string]any{
						"type": "grid",
						"cards": []any{
							map[string]any{
								"type":   "tile",
								"entity": "sensor.gosungrow_virtual_100_14_1_1_p13141",
							},
						},
					},
				},
			},
			map[string]any{
				"path": "without-battery",
				"sections": []any{
					map[string]any{
						"type": "grid",
						"cards": []any{
							map[string]any{
								"type":   "tile",
								"entity": "sensor.gosungrow_virtual_200_14_1_1_p13141",
							},
							map[string]any{
								"type":   "tile",
								"entity": "sensor.gosungrow_virtual_200_14_1_1_pv_power",
							},
						},
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
		dashboardTestState("sensor.gosungrow_100_sungrow_gosungrow_battery_information_battery_soc", "88", "%"),
		dashboardTestState("sensor.gosungrow_200_sungrow_gosungrow_pv_information_total_active_power", "1.20", "kW"),
	}

	pruned := pruneDashboardForMissingBattery(config, targets, states)
	views := pruned["views"].([]any)

	firstCards := views[0].(map[string]any)["sections"].([]any)[0].(map[string]any)["cards"].([]any)
	if len(firstCards) != 1 {
		t.Fatalf("expected battery card to remain for battery-capable view")
	}

	secondCards := views[1].(map[string]any)["sections"].([]any)[0].(map[string]any)["cards"].([]any)
	if len(secondCards) != 1 {
		t.Fatalf("expected battery card to be removed in no-battery view, got %d cards", len(secondCards))
	}
	if got := secondCards[0].(map[string]any)["entity"]; got != "sensor.gosungrow_virtual_200_14_1_1_pv_power" {
		t.Fatalf("unexpected remaining card entity: %v", got)
	}
}

func TestPruneDashboardForMissingBatteryDoesNotTreatPvToBatteryEnergyAsBatteryCapability(t *testing.T) {
	config := map[string]any{
		"views": []any{
			map[string]any{
				"sections": []any{
					map[string]any{
						"type": "grid",
						"cards": []any{
							map[string]any{
								"type": "custom:gosungrow-energy-flow-card-v2",
								"entities": map[string]any{
									"solar_power":           "sensor.gosungrow_virtual_1610907_22_247_1_pv_power",
									"load_power":            "sensor.gosungrow_virtual_1610907_22_247_1_load_power",
									"grid_power":            "sensor.gosungrow_virtual_1610907_22_247_1_grid_power",
									"battery_power":         "sensor.gosungrow_virtual_1610907_22_247_1_battery_power",
									"battery_soc":           "sensor.gosungrow_virtual_1610907_22_247_1_p13141",
									"pv_to_battery_power":   "sensor.gosungrow_virtual_1610907_22_247_1_pv_to_battery_power",
									"battery_to_load_power": "sensor.gosungrow_virtual_1610907_22_247_1_battery_to_load_power",
								},
							},
						},
					},
					map[string]any{
						"type": "grid",
						"cards": []any{
							map[string]any{
								"type":   "tile",
								"entity": "sensor.gosungrow_virtual_1610907_22_247_1_p13174",
							},
						},
					},
					map[string]any{
						"type": "grid",
						"cards": []any{
							map[string]any{
								"type":   "tile",
								"entity": "sensor.gosungrow_virtual_1610907_22_247_1_p13112",
							},
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
		dashboardTestState("sensor.gosungrow_1610907_sungrow_gosungrow_pv_information_total_dc_power", "1.23", "kW"),
		dashboardTestState("sensor.gosungrow_1610907_sungrow_gosungrow_load_information_load_power", "0.72", "kW"),
		dashboardTestState("sensor.gosungrow_1610907_sungrow_gosungrow_grid_information_grid_to_load_power", "0.10", "kW"),
		dashboardTestState("sensor.gosungrow_1610907_sungrow_gosungrow_pv_information_pv_daily_energy", "47.70", "kWh"),
		dashboardTestState("sensor.gosungrow_1610907_sungrow_gosungrow_pv_information_pv_to_battery_energy", "47.70", "kWh"),
	}

	pruned := pruneDashboardForMissingBattery(config, targets, states)
	views := pruned["views"].([]any)
	sections := views[0].(map[string]any)["sections"].([]any)

	if len(sections) != 2 {
		t.Fatalf("expected p13174 battery-only section to be removed, got %d sections", len(sections))
	}

	firstCards := sections[0].(map[string]any)["cards"].([]any)
	flowEntities := firstCards[0].(map[string]any)["entities"].(map[string]any)
	if _, ok := flowEntities["battery_power"]; ok {
		t.Fatalf("expected battery_power to be removed from no-battery flow card")
	}
	if _, ok := flowEntities["battery_soc"]; ok {
		t.Fatalf("expected battery_soc to be removed from no-battery flow card")
	}
	if _, ok := flowEntities["pv_to_battery_power"]; ok {
		t.Fatalf("expected pv_to_battery_power to be removed from no-battery flow card")
	}
}

func TestPruneDashboardForUnavailableMetricsPrunesUnsupportedLegacyDirectionalFlows(t *testing.T) {
	config := map[string]any{
		"views": []any{
			map[string]any{
				"sections": []any{
					map[string]any{
						"type": "grid",
						"cards": []any{
							map[string]any{
								"type": "custom:gosungrow-energy-flow-card-v2",
								"entities": map[string]any{
									"solar_power":        "sensor.gosungrow_virtual_100_11_0_0_pv_power",
									"load_power":         "sensor.gosungrow_virtual_100_11_0_0_load_power",
									"grid_power":         "sensor.gosungrow_virtual_100_11_0_0_grid_power",
									"pv_to_load_power":   "sensor.gosungrow_virtual_100_11_0_0_pv_to_load_power",
									"pv_to_grid_power":   "sensor.gosungrow_virtual_100_11_0_0_pv_to_grid_power",
									"grid_to_load_power": "sensor.gosungrow_virtual_100_11_0_0_grid_to_load_power",
								},
							},
						},
					},
					map[string]any{
						"type": "grid",
						"cards": []any{
							map[string]any{"type": "heading", "heading": "Solar Allocation"},
							map[string]any{
								"type": "history-graph",
								"entities": []any{
									map[string]any{"entity": "sensor.gosungrow_virtual_100_11_0_0_pv_to_load_power"},
									map[string]any{"entity": "sensor.gosungrow_virtual_100_11_0_0_pv_to_grid_power"},
								},
							},
						},
					},
					map[string]any{
						"type": "grid",
						"cards": []any{
							map[string]any{"type": "heading", "heading": "Today"},
							map[string]any{"type": "tile", "entity": "sensor.gosungrow_virtual_100_11_0_0_p13112"},
							map[string]any{"type": "tile", "entity": "sensor.gosungrow_virtual_100_11_0_0_p13173"},
						},
					},
				},
			},
		},
	}

	targets := []haDashboardTarget{
		{PsID: "100", PsKey: "100_11_0_0"},
	}
	states := []haState{
		dashboardTestState("sensor.gosungrow_100_sungrow_gosungrow_plant_information_p83076_map", "3.10", "kW"),
		dashboardTestState("sensor.gosungrow_100_sungrow_gosungrow_plant_information_p83106_map", "1.40", "kW"),
		dashboardTestState("sensor.gosungrow_100_sungrow_gosungrow_plant_information_p83549", "0.30", "kW"),
		dashboardTestState("sensor.gosungrow_100_sungrow_gosungrow_plant_information_p83022y", "18.60", "kWh"),
		dashboardTestState("sensor.gosungrow_100_sungrow_gosungrow_plant_information_p83119_map", "5.10", "kWh"),
	}

	pruned := pruneDashboardForUnavailableMetrics(config, targets, states)
	views := pruned["views"].([]any)
	sections := views[0].(map[string]any)["sections"].([]any)
	if len(sections) != 2 {
		t.Fatalf("expected unsupported directional-flow section to be removed, got %d sections", len(sections))
	}

	flowCards := sections[0].(map[string]any)["cards"].([]any)
	flowEntities := flowCards[0].(map[string]any)["entities"].(map[string]any)
	if _, ok := flowEntities["pv_to_load_power"]; ok {
		t.Fatalf("expected pv_to_load_power to be pruned from legacy flow card")
	}
	if _, ok := flowEntities["pv_to_grid_power"]; ok {
		t.Fatalf("expected pv_to_grid_power to be pruned from legacy flow card")
	}
	if _, ok := flowEntities["grid_to_load_power"]; ok {
		t.Fatalf("expected grid_to_load_power to be pruned from legacy flow card")
	}
	if _, ok := flowEntities["solar_power"]; !ok {
		t.Fatalf("expected direct pv metric to remain on legacy flow card")
	}

	todayCards := sections[1].(map[string]any)["cards"].([]any)
	if len(todayCards) != 3 {
		t.Fatalf("expected supported daily tiles to remain, got %d cards", len(todayCards))
	}
}

func TestPruneDashboardForUnavailableMetricsKeepsLegacyBatteryMetricsWhenSocExists(t *testing.T) {
	config := map[string]any{
		"views": []any{
			map[string]any{
				"sections": []any{
					map[string]any{
						"type": "grid",
						"cards": []any{
							map[string]any{
								"type": "custom:gosungrow-energy-flow-card-v2",
								"entities": map[string]any{
									"battery_power": "sensor.gosungrow_virtual_100_11_0_0_battery_power",
									"battery_soc":   "sensor.gosungrow_virtual_100_11_0_0_p13141",
								},
							},
							map[string]any{
								"type":   "tile",
								"entity": "sensor.gosungrow_virtual_100_11_0_0_p13141",
							},
						},
					},
				},
			},
		},
	}

	targets := []haDashboardTarget{
		{PsID: "100", PsKey: "100_11_0_0"},
	}
	states := []haState{
		dashboardTestState("sensor.gosungrow_100_sungrow_gosungrow_plant_information_p83081_map", "0.45", "kW"),
		dashboardTestState("sensor.gosungrow_100_sungrow_gosungrow_plant_information_p83129", "72", "%"),
	}

	pruned := pruneDashboardForUnavailableMetrics(config, targets, states)
	views := pruned["views"].([]any)
	sections := views[0].(map[string]any)["sections"].([]any)
	cards := sections[0].(map[string]any)["cards"].([]any)
	flowEntities := cards[0].(map[string]any)["entities"].(map[string]any)

	if got := flowEntities["battery_power"]; got != "sensor.gosungrow_virtual_100_11_0_0_battery_power" {
		t.Fatalf("expected legacy battery_power to remain when legacy SOC exists, got %v", got)
	}
	if got := flowEntities["battery_soc"]; got != "sensor.gosungrow_virtual_100_11_0_0_p13141" {
		t.Fatalf("expected legacy battery_soc to remain when legacy SOC exists, got %v", got)
	}
	if got := cards[1].(map[string]any)["entity"]; got != "sensor.gosungrow_virtual_100_11_0_0_p13141" {
		t.Fatalf("expected battery tile to remain when legacy SOC exists, got %v", got)
	}
}
