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
	stateEntityIDs := []string{
		"sensor.gosungrow_1610907_sungrow_gosungrow_load_information_load_power",
		"sensor.gosungrow_1610907_sungrow_gosungrow_pv_information_pv_power",
	}

	remapped := remapDashboardEntities(config, targets, stateEntityIDs)
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
	stateEntityIDs := []string{
		"sensor.gosungrow_something_p13141",
	}

	remapped := remapDashboardEntities(config, targets, stateEntityIDs)
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
	stateEntityIDs := []string{
		"sensor.gosungrow_200_sungrow_gosungrow_p13112",
		"sensor.gosungrow_100_sungrow_gosungrow_p13112",
	}

	remapped := remapDashboardEntities(config, targets, stateEntityIDs)
	views := remapped["views"].([]any)
	cards := views[0].(map[string]any)["cards"].([]any)
	if got := cards[0].(map[string]any)["entity"]; got != "sensor.gosungrow_100_sungrow_gosungrow_p13112" {
		t.Fatalf("expected best matching plant entity, got %v", got)
	}
}
