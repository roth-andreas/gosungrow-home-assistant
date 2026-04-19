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
	stateEntityIDs := []string{
		"sensor.gosungrow_100_sungrow_gosungrow_pv_information_total_active_power",
		"sensor.gosungrow_100_sungrow_gosungrow_load_information_total_active_power",
		"sensor.gosungrow_100_sungrow_gosungrow_grid_information_total_active_power",
	}

	pruned := pruneDashboardForMissingBattery(config, targets, stateEntityIDs)
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
	stateEntityIDs := []string{
		"sensor.gosungrow_100_sungrow_gosungrow_battery_information_battery_soc",
		"sensor.gosungrow_200_sungrow_gosungrow_pv_information_total_active_power",
	}

	pruned := pruneDashboardForMissingBattery(config, targets, stateEntityIDs)
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
