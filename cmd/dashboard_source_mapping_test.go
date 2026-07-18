package cmd

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func sourceMappingTestConfig(psKey string) map[string]any {
	return map[string]any{"views": []any{
		map[string]any{"path": "overview", "cards": []any{
			map[string]any{"type": "tile", "entity": "sensor.gosungrow_virtual_" + psKey + "_p13112"},
			map[string]any{"type": "tile", "entity": "sensor.gosungrow_virtual_" + psKey + "_p13116"},
		}},
		map[string]any{"path": "data-sources", "cards": []any{
			map[string]any{"type": dashboardSourceMappingCardType, "schema_version": 1, "mapping_id": psKey},
		}},
	}}
}

func TestDashboardSourceMappingsKeepAutomaticEntityReferences(t *testing.T) {
	psKey := "100_14_1_1"
	config := sourceMappingTestConfig(psKey)
	states := []haState{
		{EntityID: "sensor.gosungrow_virtual_" + psKey + "_p13112", State: "42.8", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
		{EntityID: "sensor.gosungrow_virtual_" + psKey + "_p13116", State: "30.5", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
	}
	traces := []dashboardMetricTrace{
		{Metric: "p13112", TargetPsKey: psKey, Resolved: states[0].EntityID},
		{Metric: "p13116", TargetPsKey: psKey, Resolved: states[1].EntityID},
	}

	result, overrides := applyDashboardSourceMappings(config, nil, nil, []haDashboardTarget{{PsKey: psKey, ViewPath: "roof"}}, states, traces, "gosungrow-flow", defaultDashboardLocaleBundle)
	if len(overrides) != 0 {
		t.Fatalf("expected no overrides, got %#v", overrides)
	}
	views := result["views"].([]any)
	cards := views[0].(map[string]any)["cards"].([]any)
	if got := cards[0].(map[string]any)["entity"]; got != states[0].EntityID {
		t.Fatalf("automatic production mapping changed: %v", got)
	}
	if got := cards[1].(map[string]any)["entity"]; got != states[1].EntityID {
		t.Fatalf("automatic direct-consumption mapping changed: %v", got)
	}
}

func TestDashboardSourceMappingBindsLiveFlowEntityMap(t *testing.T) {
	psKey := "100_14_1_1"
	defaultEntity := "sensor.gosungrow_virtual_" + psKey + "_pv_power"
	config := map[string]any{
		"views": []any{
			map[string]any{
				"path": "overview",
				"sections": []any{
					map[string]any{
						"cards": []any{
							map[string]any{
								"type":     dashboardEnergyFlowCardType,
								"entities": map[string]any{"solar_power": defaultEntity},
							},
						},
					},
				},
			},
			map[string]any{
				"path": "data-sources",
				"cards": []any{
					map[string]any{"type": dashboardSourceMappingCardType, "schema_version": 1, "mapping_id": psKey},
				},
			},
		},
	}
	states := []haState{{EntityID: defaultEntity, State: "4.2", Attributes: map[string]any{"unit_of_measurement": "kW"}}}
	traces := []dashboardMetricTrace{{Metric: "pv_power", TargetPsKey: psKey, Resolved: defaultEntity}}
	result, _ := applyDashboardSourceMappings(config, nil, nil, []haDashboardTarget{{PsKey: psKey}}, states, traces, "gosungrow", defaultDashboardLocaleBundle)
	card := findDashboardSourceMappingCard(result, dashboardSourceMappingID(haDashboardTarget{PsKey: psKey}))
	paths := card["bindings"].(map[string]any)["pv_power"].([]any)
	want := []any{"/views/0/sections/0/cards/0/entities/solar_power"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("live flow source was not bound: got %#v want %#v", paths, want)
	}
	flowCard := result["views"].([]any)[0].(map[string]any)["sections"].([]any)[0].(map[string]any)["cards"].([]any)[0].(map[string]any)
	automatic := flowCard["automatic_entities"].(map[string]any)
	if got := automatic["solar_power"]; got != defaultEntity {
		t.Fatalf("live flow automatic source metadata missing: %v", got)
	}
}

func TestDashboardSourceMappingHashesIdentifierEvenWithoutResolvedMetrics(t *testing.T) {
	target := haDashboardTarget{PsID: "100", PsKey: "100_14_1_1"}
	result, overrides := applyDashboardSourceMappings(sourceMappingTestConfig(target.PsKey), nil, nil, []haDashboardTarget{target}, nil, nil, "gosungrow", defaultDashboardLocaleBundle)
	if len(overrides) != 0 {
		t.Fatalf("unexpected overrides: %#v", overrides)
	}
	card := findDashboardSourceMappingCard(result, dashboardSourceMappingID(target))
	if card == nil || stringValue(card["mapping_id"]) == target.PsKey {
		t.Fatalf("raw target identifier remained in mapping card: %#v", card)
	}
}

func TestDashboardSourceMappingsPreserveAndApplyOverride(t *testing.T) {
	psKey := "100_14_1_1"
	defaultEntity := "sensor.gosungrow_virtual_" + psKey + "_p13116"
	overrideEntity := "sensor.gosungrow_inverter_" + psKey + "_p13116"
	config := sourceMappingTestConfig(psKey)
	current := sourceMappingTestConfig(psKey)
	currentCard := findDashboardSourceMappingCard(current, psKey)
	currentCard["mapping_id"] = dashboardSourceMappingID(haDashboardTarget{PsKey: psKey})
	currentCard["overrides"] = map[string]any{"p13116": overrideEntity}
	states := []haState{
		{EntityID: defaultEntity, State: "52.6", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
		{EntityID: overrideEntity, State: "30.5", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
	}
	traces := []dashboardMetricTrace{{Metric: "p13116", TargetPsKey: psKey, Resolved: defaultEntity}}

	result, overrides := applyDashboardSourceMappings(config, current, nil, []haDashboardTarget{{PsKey: psKey}}, states, traces, "gosungrow-flow", defaultDashboardLocaleBundle)
	if got := overrides[dashboardSourceMappingID(haDashboardTarget{PsKey: psKey})]["p13116"]; got != overrideEntity {
		t.Fatalf("override not preserved: %q", got)
	}
	views := result["views"].([]any)
	cards := views[0].(map[string]any)["cards"].([]any)
	if got := cards[1].(map[string]any)["entity"]; got != overrideEntity {
		t.Fatalf("override not applied: %v", got)
	}
	card := findDashboardSourceMappingCard(result, dashboardSourceMappingID(haDashboardTarget{PsKey: psKey}))
	paths := card["bindings"].(map[string]any)["p13116"].([]any)
	if len(paths) != 1 {
		t.Fatalf("expected one binding path, got %#v", paths)
	}
}

func TestDashboardSourceMappingFlagsImpossibleDirectSolarConsumption(t *testing.T) {
	psKey := "100_14_1_1"
	production := "sensor.gosungrow_virtual_" + psKey + "_p13112"
	direct := "sensor.gosungrow_virtual_" + psKey + "_p13116"
	config := sourceMappingTestConfig(psKey)
	states := []haState{
		{EntityID: production, State: "42.8", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
		{EntityID: direct, State: "52.6", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
	}
	traces := []dashboardMetricTrace{{Metric: "p13112", TargetPsKey: psKey, Resolved: production}, {Metric: "p13116", TargetPsKey: psKey, Resolved: direct}}
	result, _ := applyDashboardSourceMappings(config, nil, nil, []haDashboardTarget{{PsKey: psKey}}, states, traces, "gosungrow-flow", defaultDashboardLocaleBundle)
	card := findDashboardSourceMappingCard(result, dashboardSourceMappingID(haDashboardTarget{PsKey: psKey}))
	metrics := card["metrics"].([]any)
	found := false
	for _, raw := range metrics {
		metric := raw.(map[string]any)
		if metric["key"] == "p13116" {
			found = true
			validation, _ := metric["validation"].(map[string]any)
			if intValue(validation["schema_version"]) != dashboardSourceValidationSchema {
				t.Fatalf("expected live validation rule, got %#v", metric)
			}
		}
	}
	if !found {
		t.Fatal("direct solar metric missing")
	}
}

func TestExtractDashboardSourceOverridesIgnoresUnknownSchema(t *testing.T) {
	config := sourceMappingTestConfig("target")
	card := findDashboardSourceMappingCard(config, "target")
	card["schema_version"] = 2
	card["overrides"] = map[string]any{"p13112": "sensor.example"}
	if got := extractDashboardSourceOverrides(config); !reflect.DeepEqual(got, map[string]map[string]string{}) {
		t.Fatalf("unexpected future-schema overrides: %#v", got)
	}
}

func TestDashboardSourceMappingsAreIsolatedWhenTargetsShareDefaultEntity(t *testing.T) {
	targetA := haDashboardTarget{PsID: "plant-a", PsKey: "a_14_1_1", ViewPath: "roof"}
	targetB := haDashboardTarget{PsID: "plant-b", PsKey: "b_14_1_1", ViewPath: "garage"}
	shared := "sensor.gosungrow_plant_daily_p13116"
	override := "sensor.gosungrow_a_14_1_1_p13116"
	config := map[string]any{"views": []any{
		map[string]any{"path": "roof-overview", "cards": []any{map[string]any{"type": "tile", "entity": shared}}},
		map[string]any{"path": "roof-trends", "cards": []any{map[string]any{"type": "history-graph", "entity": shared}}},
		map[string]any{"path": "roof-data-sources", "cards": []any{map[string]any{"type": dashboardSourceMappingCardType, "schema_version": 1, "mapping_id": targetA.PsKey}}},
		map[string]any{"path": "garage-overview", "cards": []any{map[string]any{"type": "tile", "entity": shared}}},
		map[string]any{"path": "garage-data-sources", "cards": []any{map[string]any{"type": dashboardSourceMappingCardType, "schema_version": 1, "mapping_id": targetB.PsKey}}},
	}}
	current, err := deepCopyJSONValue(config)
	if err != nil {
		t.Fatal(err)
	}
	currentConfig := current.(map[string]any)
	card := findDashboardSourceMappingCard(currentConfig, targetA.PsKey)
	card["mapping_id"] = dashboardSourceMappingID(targetA)
	card["overrides"] = map[string]any{"p13116": override}
	states := []haState{
		{EntityID: shared, State: "30", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
		{EntityID: override, State: "28", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
	}
	traces := []dashboardMetricTrace{
		{Metric: "p13116", TargetPsKey: targetA.PsKey, Resolved: shared},
		{Metric: "p13116", TargetPsKey: targetB.PsKey, Resolved: shared},
	}

	result, accepted := applyDashboardSourceMappings(config, currentConfig, nil, []haDashboardTarget{targetA, targetB}, states, traces, "gosungrow", defaultDashboardLocaleBundle)
	if accepted[dashboardSourceMappingID(targetA)]["p13116"] != override {
		t.Fatalf("override was not accepted: %#v", accepted)
	}
	views := result["views"].([]any)
	if got := views[0].(map[string]any)["cards"].([]any)[0].(map[string]any)["entity"]; got != override {
		t.Fatalf("target A not updated: %v", got)
	}
	if got := views[1].(map[string]any)["cards"].([]any)[0].(map[string]any)["entity"]; got != override {
		t.Fatalf("target A repeated view not updated: %v", got)
	}
	if got := views[3].(map[string]any)["cards"].([]any)[0].(map[string]any)["entity"]; got != shared {
		t.Fatalf("target B was incorrectly updated: %v", got)
	}
	bindings := findDashboardSourceMappingCard(result, dashboardSourceMappingID(targetA))["bindings"].(map[string]any)["p13116"].([]any)
	if !reflect.DeepEqual(bindings, []any{"/views/0/cards/0/entity", "/views/1/cards/0/entity"}) {
		t.Fatalf("bindings escaped target A: %#v", bindings)
	}
}

func TestDashboardSourceOverrideValidationPreservesOnlyAcceptedMissingEntity(t *testing.T) {
	defaults := map[string]string{"p13116": "sensor.auto_p13116"}
	requested := map[string]string{"p13116": "sensor.missing_p13116", "unknown": "sensor.injected"}
	got := validateDashboardSourceOverrides("source-test", defaults, requested, map[string]string{"p13116": "sensor.missing_p13116"}, haDashboardTarget{}, true, nil)
	if !reflect.DeepEqual(got, map[string]string{"p13116": "sensor.missing_p13116"}) {
		t.Fatalf("accepted missing override was not preserved: %#v", got)
	}
	if injected := validateDashboardSourceOverrides("source-test", defaults, requested, nil, haDashboardTarget{}, true, nil); len(injected) != 0 {
		t.Fatalf("injected override was accepted: %#v", injected)
	}
}

func TestDashboardSourceMissingManualCandidateKeepsAutomaticFirst(t *testing.T) {
	automatic := map[string]any{"entity_id": "sensor.auto", "recommended": true}
	got := ensureDashboardSourceCandidate([]any{automatic}, "p13116", "sensor.missing")
	if len(got) != 2 || stringValue(got[0].(map[string]any)["entity_id"]) != "sensor.auto" || stringValue(got[1].(map[string]any)["entity_id"]) != "sensor.missing" {
		t.Fatalf("automatic choice was not kept first: %#v", got)
	}
}

func TestDashboardSourceResetWinsOverPersistedDiagnosticCopy(t *testing.T) {
	psKey := "100_14_1_1"
	config := sourceMappingTestConfig(psKey)
	currentRaw, err := deepCopyJSONValue(config)
	if err != nil {
		t.Fatal(err)
	}
	current := currentRaw.(map[string]any)
	mappingID := dashboardSourceMappingID(haDashboardTarget{PsKey: psKey})
	findDashboardSourceMappingCard(current, psKey)["mapping_id"] = mappingID
	defaultEntity := "sensor.gosungrow_virtual_" + psKey + "_p13112"
	result, accepted := applyDashboardSourceMappings(config, current, map[string]map[string]string{mappingID: {"p13112": "sensor.old_manual_p13112"}}, []haDashboardTarget{{PsKey: psKey}}, []haState{{EntityID: defaultEntity, State: "4", Attributes: map[string]any{"unit_of_measurement": "kWh"}}}, []dashboardMetricTrace{{Metric: "p13112", TargetPsKey: psKey, Resolved: defaultEntity}}, "gosungrow", defaultDashboardLocaleBundle)
	if len(accepted) != 0 {
		t.Fatalf("persisted diagnostic override defeated UI reset: %#v", accepted)
	}
	if got := findDashboardSourceMappingCard(result, mappingID)["overrides"].(map[string]any); len(got) != 0 {
		t.Fatalf("reset override remained in card: %#v", got)
	}
}

func TestDashboardStructureHashIgnoresSourceOverridesButDetectsUnrelatedEdits(t *testing.T) {
	automatic := sourceMappingTestConfig("target")
	card := findDashboardSourceMappingCard(automatic, "target")
	card["defaults"] = map[string]any{"p13112": "sensor.auto"}
	card["overrides"] = map[string]any{}
	card["bindings"] = map[string]any{"p13112": []any{"/views/0/cards/0/entity"}}
	card["metrics"] = []any{map[string]any{"key": "p13112", "default": "sensor.auto", "status": "automatic", "candidates": []any{map[string]any{"entity_id": "sensor.auto"}}}}
	views := automatic["views"].([]any)
	views[0].(map[string]any)["cards"].([]any)[0].(map[string]any)["entity"] = "sensor.auto"
	autoHash, err := hashDashboardStructure(automatic)
	if err != nil {
		t.Fatal(err)
	}
	manualRaw, err := deepCopyJSONValue(automatic)
	if err != nil {
		t.Fatal(err)
	}
	manual := manualRaw.(map[string]any)
	manualCard := findDashboardSourceMappingCard(manual, "target")
	manualCard["overrides"] = map[string]any{"p13112": "sensor.manual"}
	manualCard["metrics"].([]any)[0].(map[string]any)["status"] = "manual"
	manualCard["metrics"].([]any)[0].(map[string]any)["confidence"] = "manual"
	manualCard["metrics"].([]any)[0].(map[string]any)["reason"] = "Current manual selection"
	manual["views"].([]any)[0].(map[string]any)["cards"].([]any)[0].(map[string]any)["entity"] = "sensor.manual"
	manualHash, err := hashDashboardStructure(manual)
	if err != nil {
		t.Fatal(err)
	}
	if autoHash != manualHash {
		t.Fatalf("override changed structure hash: %s != %s", autoHash, manualHash)
	}
	state := &haDashboardState{DashboardStructureHash: autoHash}
	modified, err := dashboardModifiedOutsideGoSungrow(manual, automatic, state)
	if err != nil || modified {
		t.Fatalf("source-only edit treated as external: modified=%v err=%v", modified, err)
	}
	boundEditRaw, err := deepCopyJSONValue(automatic)
	if err != nil {
		t.Fatal(err)
	}
	boundEdit := boundEditRaw.(map[string]any)
	boundEdit["views"].([]any)[0].(map[string]any)["cards"].([]any)[0].(map[string]any)["entity"] = "sensor.user_edit"
	modified, err = dashboardModifiedOutsideGoSungrow(boundEdit, automatic, state)
	if err != nil || !modified {
		t.Fatalf("untracked bound entity edit was not detected: modified=%v err=%v", modified, err)
	}
	manual["views"].([]any)[0].(map[string]any)["title"] = "User edit"
	modified, err = dashboardModifiedOutsideGoSungrow(manual, automatic, state)
	if err != nil || !modified {
		t.Fatalf("unrelated edit was not detected: modified=%v err=%v", modified, err)
	}
}

func TestDashboardSourceCandidatesAreDeterministicLimitedAndContainNoLiveValues(t *testing.T) {
	target := haDashboardTarget{PsID: "100", PsKey: "100_14_1_1"}
	states := make([]haState, 0, 30)
	for index := 0; index < 30; index++ {
		states = append(states, haState{EntityID: fmt.Sprintf("sensor.gosungrow_100_14_1_1_candidate_%02d_p13116", index), State: fmt.Sprint(index), Attributes: map[string]any{"unit_of_measurement": "kWh", "friendly_name": strings.Repeat("Long name ", 4)}})
	}
	first := dashboardSourceCandidates(target, "p13116", states[0].EntityID, states, true)
	second := dashboardSourceCandidates(target, "p13116", states[0].EntityID, states, true)
	if len(first) != dashboardSourceRecommendedLimit+dashboardSourceAdditionalLimit {
		t.Fatalf("unexpected candidate count: %d", len(first))
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("candidate ordering is not deterministic")
	}
	for index, raw := range first {
		candidate := raw.(map[string]any)
		if _, ok := candidate["state"]; ok {
			t.Fatalf("candidate embeds live state: %#v", candidate)
		}
		if _, ok := candidate["unit"]; ok {
			t.Fatalf("candidate embeds live unit: %#v", candidate)
		}
		if _, ok := candidate["name"]; ok {
			t.Fatalf("candidate embeds live friendly name: %#v", candidate)
		}
		if got := candidate["recommended"]; got != (index < dashboardSourceRecommendedLimit) {
			t.Fatalf("unexpected recommended flag at %d: %v", index, got)
		}
	}
}

func TestDashboardSourceCandidatesExcludeStaleAlternativesButKeepAutomatic(t *testing.T) {
	target := haDashboardTarget{PsKey: "100_14_1_1"}
	stale := time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339Nano)
	automatic := "sensor.gosungrow_100_14_1_1_auto_p13116"
	alternative := "sensor.gosungrow_100_14_1_1_alternative_p13116"
	states := []haState{
		{EntityID: automatic, State: "4", LastUpdated: stale, Attributes: map[string]any{"unit_of_measurement": "kWh"}},
		{EntityID: alternative, State: "3", LastUpdated: stale, Attributes: map[string]any{"unit_of_measurement": "kWh"}},
	}
	got := dashboardSourceCandidates(target, "p13116", automatic, states, true)
	if len(got) != 1 || stringValue(got[0].(map[string]any)["entity_id"]) != automatic || got[0].(map[string]any)["confidence"] != "low" {
		t.Fatalf("unexpected stale candidates: %#v", got)
	}
	requested := map[string]string{"p13116": alternative}
	defaults := map[string]string{"p13116": automatic}
	if injected := validateDashboardSourceOverrides("source-test", defaults, requested, nil, target, true, states); len(injected) != 0 {
		t.Fatalf("stale injected override accepted: %#v", injected)
	}
	if preserved := validateDashboardSourceOverrides("source-test", defaults, requested, requested, target, true, states); preserved["p13116"] != alternative {
		t.Fatalf("accepted stale override not preserved: %#v", preserved)
	}
}
