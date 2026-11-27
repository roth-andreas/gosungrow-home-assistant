package cmd

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud"
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
		{EntityID: "sensor.gosungrow_virtual_" + psKey + "_p13112", RegistryUniqueID: "gosungrow_" + psKey + "_p13112", State: "42.8", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
		{EntityID: "sensor.gosungrow_virtual_" + psKey + "_p13116", RegistryUniqueID: "gosungrow_" + psKey + "_p13116", State: "30.5", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
	}
	traces := []dashboardMetricTrace{
		{Metric: "p13112", TargetPsKey: psKey, Resolved: states[0].EntityID},
		{Metric: "p13116", TargetPsKey: psKey, Resolved: states[1].EntityID},
	}

	result, overrides, sourceErr := applyDashboardSourceMappings(config, nil, nil, []haDashboardTarget{{PsKey: psKey, ViewPath: "roof"}}, states, traces, "gosungrow-flow", defaultDashboardLocaleBundle)
	if sourceErr != nil {
		t.Fatal(sourceErr)
	}
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

func TestDashboardSourceMappingPinsExistingChoiceAndOffersSemanticUpgrade(t *testing.T) {
	target, states := issue19SemanticFixture()
	placeholder := "sensor.gosungrow_template_pv_power"
	oldEntity := states[0].EntityID
	betterEntity := states[1].EntityID
	makeConfig := func() map[string]any {
		return map[string]any{"views": []any{
			map[string]any{"path": "overview", "cards": []any{map[string]any{"type": "tile", "entity": placeholder}}},
			map[string]any{"path": "data-sources", "cards": []any{map[string]any{"type": dashboardSourceMappingCardType, "schema_version": 1, "mapping_id": target.PsKey}}},
		}}
	}
	trace := []dashboardMetricTrace{{Metric: "pv_power", TargetPsKey: target.PsKey, Placeholder: placeholder, Resolved: oldEntity}}
	mappingID := dashboardSourceMappingID(target)
	current := map[string]any{"views": []any{map[string]any{"cards": []any{map[string]any{
		"type": dashboardSourceMappingCardType, "schema_version": 1, "mapping_id": mappingID,
		"matcher_version": 2, "defaults": map[string]any{"pv_power": oldEntity}, "pinned_defaults": map[string]any{"pv_power": oldEntity},
	}}}}}

	existing, _, sourceErr := applyDashboardSourceMappings(makeConfig(), current, nil, []haDashboardTarget{target}, states, trace, "gosungrow", defaultDashboardLocaleBundle)
	if sourceErr != nil {
		t.Fatal(sourceErr)
	}
	existingViews := existing["views"].([]any)
	display := existingViews[0].(map[string]any)["cards"].([]any)[0].(map[string]any)
	if got, _ := getDashboardJSONPointer(display, "/source_policy/native_template/entity"); got != oldEntity {
		t.Fatalf("existing automatic choice changed without consent: got %v want %s", got, oldEntity)
	}
	if display["type"] != "vertical-stack" {
		t.Fatal("unverified pinned PV source was rendered without review")
	}
	card := findDashboardSourceMappingCard(existing, mappingID)
	if got := intValue(card["matcher_version"]); got != dashboardSemanticMatcherVersion {
		t.Fatalf("matcher-v2 card was not migrated in place: %d", got)
	}
	if got := anyMapToStringMap(card["recommendations"])["pv_power"]; got != betterEntity {
		t.Fatalf("semantic upgrade was not offered: got %q want %q", got, betterEntity)
	}
	metric := card["metrics"].([]any)[0].(map[string]any)
	if needsReview, _ := metric["needs_review"].(bool); !needsReview {
		t.Fatalf("existing questionable source was not marked for review: %#v", metric)
	}

	fresh, _, sourceErr := applyDashboardSourceMappings(makeConfig(), nil, nil, []haDashboardTarget{target}, states, trace, "gosungrow", defaultDashboardLocaleBundle)
	if sourceErr != nil {
		t.Fatal(sourceErr)
	}
	freshViews := fresh["views"].([]any)
	if got := freshViews[0].(map[string]any)["cards"].([]any)[0].(map[string]any)["entity"]; got != betterEntity {
		t.Fatalf("new dashboard did not use confident semantic match: got %v want %s", got, betterEntity)
	}
}

func TestDashboardSourceMappingOffersCanonicalPlantPVUpgrade(t *testing.T) {
	target := haDashboardTarget{PsID: "100", PsKey: "100_55_1_1", PlantDevices: []dashboardPlantDevice{
		{PsKey: "100_55_1_1", DeviceType: 55}, {PsKey: "100_55_1_2", DeviceType: 55},
	}}
	oldEntity := "sensor.gosungrow_100_55_1_1_microinverter1_total_dc_power"
	plantEntity := "sensor.gosungrow_virtual_100_pv_power"
	placeholder := "sensor.gosungrow_template_pv_power"
	makeConfig := func() map[string]any {
		return map[string]any{"views": []any{
			map[string]any{"path": "overview", "cards": []any{map[string]any{"type": "tile", "entity": placeholder}}},
			map[string]any{"path": "data-sources", "cards": []any{map[string]any{"type": dashboardSourceMappingCardType, "schema_version": 1, "mapping_id": target.PsKey}}},
		}}
	}
	states := []haState{
		dashboardTestState(oldEntity, "1", "kW"),
		dashboardTestState(plantEntity, "3", "kW"),
	}
	trace := []dashboardMetricTrace{{Metric: "pv_power", TargetPsKey: target.PsKey, Placeholder: placeholder, Resolved: oldEntity}}
	mappingID := dashboardSourceMappingID(target)
	current := map[string]any{"views": []any{map[string]any{"cards": []any{map[string]any{
		"type": dashboardSourceMappingCardType, "schema_version": 1, "mapping_id": mappingID,
		"defaults": map[string]any{"pv_power": oldEntity}, "pinned_defaults": map[string]any{"pv_power": oldEntity},
	}}}}}

	existing, _, sourceErr := applyDashboardSourceMappings(makeConfig(), current, nil, []haDashboardTarget{target}, states, trace, "gosungrow", defaultDashboardLocaleBundle)
	if sourceErr != nil {
		t.Fatal(sourceErr)
	}
	card := findDashboardSourceMappingCard(existing, mappingID)
	if got := anyMapToStringMap(card["defaults"])["pv_power"]; got != oldEntity {
		t.Fatalf("pinned source = %q, want %q", got, oldEntity)
	}
	if got := anyMapToStringMap(card["recommendations"])["pv_power"]; got != plantEntity {
		t.Fatalf("plant recommendation = %q, want %q", got, plantEntity)
	}

	fresh, _, sourceErr := applyDashboardSourceMappings(makeConfig(), nil, nil, []haDashboardTarget{target}, states, trace, "gosungrow", defaultDashboardLocaleBundle)
	if sourceErr != nil {
		t.Fatal(sourceErr)
	}
	if got := fresh["views"].([]any)[0].(map[string]any)["cards"].([]any)[0].(map[string]any)["entity"]; got != plantEntity {
		t.Fatalf("fresh pv source = %v, want %q", got, plantEntity)
	}
}

func TestDashboardSourceMappingSeparatesMetricsThatPreviouslySharedEntity(t *testing.T) {
	target, states := issue19SemanticFixture()
	exportPlaceholder := "sensor.gosungrow_template_grid_export_today"
	importPlaceholder := "sensor.gosungrow_template_grid_import_today"
	oldShared := "sensor.gosungrow_1498605_11_0_0_feed_in_energy_today"
	config := map[string]any{"views": []any{
		map[string]any{"path": "overview", "cards": []any{
			map[string]any{"type": "tile", "entity": exportPlaceholder},
			map[string]any{"type": "tile", "entity": importPlaceholder},
		}},
		map[string]any{"path": "data-sources", "cards": []any{map[string]any{"type": dashboardSourceMappingCardType, "schema_version": 1, "mapping_id": target.PsKey}}},
	}}
	traces := []dashboardMetricTrace{
		{Metric: "p13173", TargetPsKey: target.PsKey, Placeholder: exportPlaceholder, Resolved: oldShared},
		{Metric: "p13147", TargetPsKey: target.PsKey, Placeholder: importPlaceholder, Resolved: oldShared},
	}
	result, _, sourceErr := applyDashboardSourceMappings(config, nil, nil, []haDashboardTarget{target}, states, traces, "gosungrow", defaultDashboardLocaleBundle)
	if sourceErr != nil {
		t.Fatal(sourceErr)
	}
	cards := result["views"].([]any)[0].(map[string]any)["cards"].([]any)
	if got := cards[0].(map[string]any)["entity"]; got != "sensor.gosungrow_1498605_11_0_0_feed_in_energy_today" {
		t.Fatalf("export source changed incorrectly: %v", got)
	}
	if got := cards[1].(map[string]any)["entity"]; got != "sensor.gosungrow_1498605_11_0_0_energy_purchased_today" {
		t.Fatalf("import source was not separated from export: %v", got)
	}
	card := findDashboardSourceMappingCard(result, dashboardSourceMappingID(target))
	bindings := card["bindings"].(map[string]any)
	if !reflect.DeepEqual(bindings["p13173"], []any{"/views/0/cards/0/source_policy/metrics/p13173/entity_id", "/views/0/cards/0/source_policy/native_template/entity", "/views/0/cards/0/entity"}) || !reflect.DeepEqual(bindings["p13147"], []any{"/views/0/cards/1/source_policy/metrics/p13147/entity_id", "/views/0/cards/1/source_policy/native_template/entity", "/views/0/cards/1/entity"}) {
		t.Fatalf("metric bindings were conflated: %#v", bindings)
	}
}

func TestDashboardSourceMappingBindsLiveFlowEntityMap(t *testing.T) {
	psKey := "100_14_1_1"
	target := haDashboardTarget{PsID: "100", PsKey: psKey}
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
	states := []haState{{EntityID: defaultEntity, RegistryUniqueID: dashboardPlantPVUniqueID(target), State: "4.2", Attributes: map[string]any{"unit_of_measurement": "kW"}}}
	traces := []dashboardMetricTrace{{Metric: "pv_power", TargetPsKey: psKey, Resolved: defaultEntity}}
	result, _, sourceErr := applyDashboardSourceMappings(config, nil, nil, []haDashboardTarget{target}, states, traces, "gosungrow", defaultDashboardLocaleBundle)
	if sourceErr != nil {
		t.Fatal(sourceErr)
	}
	card := findDashboardSourceMappingCard(result, dashboardSourceMappingID(target))
	paths := card["bindings"].(map[string]any)["pv_power"].([]any)
	want := []any{"/views/0/sections/0/cards/0/entities/solar_power", "/views/0/sections/0/cards/0/source_policy/metrics/pv_power/entity_id"}
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
	result, overrides, sourceErr := applyDashboardSourceMappings(sourceMappingTestConfig(target.PsKey), nil, nil, []haDashboardTarget{target}, nil, nil, "gosungrow", defaultDashboardLocaleBundle)
	if sourceErr != nil {
		t.Fatal(sourceErr)
	}
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

	result, overrides, sourceErr := applyDashboardSourceMappings(config, current, nil, []haDashboardTarget{{PsKey: psKey}}, states, traces, "gosungrow-flow", defaultDashboardLocaleBundle)
	if sourceErr != nil {
		t.Fatal(sourceErr)
	}
	if got := overrides[dashboardSourceMappingID(haDashboardTarget{PsKey: psKey})]["p13116"]; got != overrideEntity {
		t.Fatalf("override not preserved: %q", got)
	}
	views := result["views"].([]any)
	cards := views[0].(map[string]any)["cards"].([]any)
	if got, _ := getDashboardJSONPointer(cards[1], "/source_policy/native_template/entity"); got != overrideEntity {
		t.Fatalf("override not applied: %v", got)
	}
	card := findDashboardSourceMappingCard(result, dashboardSourceMappingID(haDashboardTarget{PsKey: psKey}))
	paths := card["bindings"].(map[string]any)["p13116"].([]any)
	if len(paths) != 3 {
		t.Fatalf("expected rendered, template, and policy bindings, got %#v", paths)
	}
	for _, raw := range paths {
		if value, ok := getDashboardJSONPointer(result, raw.(string)); !ok || value != overrideEntity {
			t.Fatalf("override binding was not updated: %v", raw)
		}
	}
}

func TestDashboardFallbackKeepsReleasedMappingAcrossProjections(t *testing.T) {
	const (
		psKey     = "100_14_1_1"
		released  = "source-fee8a2cdfe22"
		metric    = "p13116"
		automatic = "sensor.gosungrow_virtual_100_14_1_1_p13116"
		manual    = "sensor.gosungrow_inverter_100_14_1_1_p13116"
	)
	current := sourceMappingTestConfig(psKey)
	card := findDashboardSourceMappingCard(current, psKey)
	card["mapping_id"] = released
	card["pinned_defaults"] = map[string]any{metric: automatic}
	card["overrides"] = map[string]any{metric: manual}
	states := []haState{
		{EntityID: automatic, State: "52.6", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
		{EntityID: manual, State: "30.5", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
	}
	traces := []dashboardMetricTrace{{Metric: metric, TargetPsKey: psKey, Resolved: automatic}}
	trees := iSolarCloud.PsTrees{"100": dashboardTestTree("100", []string{psKey}, 14)}
	fallback := &dashboardDiscovery{plants: []string{"100"}, denied: map[string]bool{"100": true}, inventories: trees}
	recovery := &dashboardDiscovery{plants: []string{"100"}, trees: trees}
	state := &haDashboardState{TargetPsKeys: []string{psKey}}
	for _, attempt := range []*dashboardDiscovery{fallback, fallback, recovery} {
		selection := attempt.selectTargets(state)
		if selection.reason != "" || len(selection.targets) != 1 {
			t.Fatalf("saved target unavailable: %+v", selection)
		}
		if got := dashboardSourceMappingID(selection.targets[0]); got != released {
			t.Fatalf("mapping ID changed: got %s, want %s", got, released)
		}
		enhanced, overrides, err := applyDashboardSourceMappings(sourceMappingTestConfig(psKey), current,
			state.SourceOverrides, selection.targets, states, traces, "gosungrow-flow", defaultDashboardLocaleBundle)
		if err != nil {
			t.Fatal(err)
		}
		native, err := nativeDashboardFallback(enhanced, defaultDashboardLocaleBundle)
		if err != nil {
			t.Fatal(err)
		}
		for _, projection := range []map[string]any{enhanced, native} {
			if extractDashboardSourceDefaults(projection)[released][metric] != automatic ||
				extractDashboardSourceOverrides(projection)[released][metric] != manual {
				t.Fatalf("projection lost source choices for %s", released)
			}
		}
		current = native
		state = &haDashboardState{TargetPsKeys: []string{psKey},
			InventoryTargetPlants: selection.inventoryPlants, SourceOverrides: overrides}
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
	result, _, sourceErr := applyDashboardSourceMappings(config, nil, nil, []haDashboardTarget{{PsKey: psKey}}, states, traces, "gosungrow-flow", defaultDashboardLocaleBundle)
	if sourceErr != nil {
		t.Fatal(sourceErr)
	}
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

func TestDashboardSourceCandidatesExcludeAmbiguousDailyEnergyAliases(t *testing.T) {
	target, states := issue19SemanticFixture()
	states = append(states,
		haState{EntityID: "sensor.wrong_export", RegistryUniqueID: "gosungrow_1498605_11_0_0_p13173", State: "3", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
		haState{EntityID: "sensor.calculated_direct", RegistryUniqueID: "gosungrow_1498605_11_0_0_pv_consumption_energy", State: "4", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
	)
	production := dashboardSourceCandidates(target, "p13112", "", states, true)
	for _, raw := range production {
		if stringValue(raw.(map[string]any)["entity_id"]) == "sensor.wrong_export" {
			t.Fatalf("feed-in was offered as production: %#v", production)
		}
	}
	direct := dashboardSourceCandidates(target, "p13116", "", states, true)
	for _, raw := range direct {
		if stringValue(raw.(map[string]any)["entity_id"]) == "sensor.calculated_direct" {
			t.Fatalf("calculated direct solar was offered: %#v", direct)
		}
	}
}

func TestDashboardSourceOverrideValidationRejectsInjectedDirectionConflict(t *testing.T) {
	target, states := issue19SemanticFixture()
	production := states[2].EntityID
	export := states[3].EntityID
	got := validateDashboardSourceOverrides("source-test", map[string]string{"p13112": production}, map[string]string{"p13112": export}, nil, target, true, states)
	if len(got) != 0 {
		t.Fatalf("injected feed-in override was accepted as production: %#v", got)
	}
}

func TestDashboardSourceMappingShowsUnavailableWithoutNativeDirectSolar(t *testing.T) {
	target := haDashboardTarget{PsID: "100", PsKey: "100_14_1_1"}
	calculated := "sensor.gosungrow_100_14_1_1_pv_consumption_energy"
	config := sourceMappingTestConfig(target.PsKey)
	states := []haState{{EntityID: calculated, RegistryUniqueID: calculated, State: "12", Attributes: map[string]any{"unit_of_measurement": "kWh"}}}
	traces := []dashboardMetricTrace{{Metric: "p13116", TargetPsKey: target.PsKey, Resolved: calculated}}
	result, _, sourceErr := applyDashboardSourceMappings(config, nil, nil, []haDashboardTarget{target}, states, traces, "gosungrow", defaultDashboardLocaleBundle)
	if sourceErr != nil {
		t.Fatal(sourceErr)
	}
	card := findDashboardSourceMappingCard(result, dashboardSourceMappingID(target))
	if got := anyMapToStringMap(card["defaults"])["p13116"]; got != "" {
		t.Fatalf("calculated source remained a new default: %q", got)
	}
	metrics := card["metrics"].([]any)
	metric := metrics[0].(map[string]any)
	if unavailable, _ := metric["native_unavailable"].(bool); !unavailable {
		t.Fatalf("native-unavailable state missing: %#v", metric)
	}
	if got := card["candidates"].(map[string]any)["p13116"].([]any); len(got) != 0 {
		t.Fatalf("calculated candidates leaked into selector: %#v", got)
	}
}

func TestDashboardSourceMappingPreservesLegacyCalculatedDefaultWithBlockingWarning(t *testing.T) {
	target := haDashboardTarget{PsID: "100", PsKey: "100_14_1_1"}
	calculated := "sensor.gosungrow_100_14_1_1_pv_consumption_energy"
	config := sourceMappingTestConfig(target.PsKey)
	current := sourceMappingTestConfig(target.PsKey)
	currentCard := findDashboardSourceMappingCard(current, target.PsKey)
	currentCard["mapping_id"] = dashboardSourceMappingID(target)
	currentCard["pinned_defaults"] = map[string]any{"p13116": calculated}
	states := []haState{{EntityID: calculated, RegistryUniqueID: calculated, State: "12", Attributes: map[string]any{"unit_of_measurement": "kWh"}}}
	traces := []dashboardMetricTrace{{Metric: "p13116", TargetPsKey: target.PsKey, Resolved: calculated}}
	result, _, sourceErr := applyDashboardSourceMappings(config, current, nil, []haDashboardTarget{target}, states, traces, "gosungrow", defaultDashboardLocaleBundle)
	if sourceErr != nil {
		t.Fatal(sourceErr)
	}
	card := findDashboardSourceMappingCard(result, dashboardSourceMappingID(target))
	if got := anyMapToStringMap(card["defaults"])["p13116"]; got != calculated {
		t.Fatalf("legacy source was not preserved: %q", got)
	}
	metric := card["metrics"].([]any)[0].(map[string]any)
	if unsupported, _ := metric["unsupported_calculated"].(bool); !unsupported {
		t.Fatalf("legacy source lacks blocking warning: %#v", metric)
	}
	candidates := card["candidates"].(map[string]any)["p13116"].([]any)
	if selectable, exists := candidates[0].(map[string]any)["selectable"]; !exists || selectable != false {
		t.Fatalf("legacy calculated source remained selectable: %#v", candidates)
	}
}

func TestDashboardSourceMappingPreservesLegacyCalculatedManualOverrideWithBlockingWarning(t *testing.T) {
	target := haDashboardTarget{PsID: "100", PsKey: "100_14_1_1"}
	automatic := "sensor.gosungrow_100_14_1_1_p83097"
	calculated := "sensor.gosungrow_100_14_1_1_pv_to_load_energy"
	config := sourceMappingTestConfig(target.PsKey)
	current := sourceMappingTestConfig(target.PsKey)
	currentCard := findDashboardSourceMappingCard(current, target.PsKey)
	currentCard["mapping_id"] = dashboardSourceMappingID(target)
	currentCard["pinned_defaults"] = map[string]any{"p13116": automatic}
	currentCard["overrides"] = map[string]any{"p13116": calculated}
	states := []haState{
		{EntityID: automatic, RegistryUniqueID: automatic, State: "8", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
		{EntityID: calculated, RegistryUniqueID: calculated, State: "9", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
	}
	traces := []dashboardMetricTrace{{Metric: "p13116", TargetPsKey: target.PsKey, Resolved: automatic}}
	persisted := map[string]map[string]string{dashboardSourceMappingID(target): {"p13116": calculated}}
	result, accepted, sourceErr := applyDashboardSourceMappings(config, current, persisted, []haDashboardTarget{target}, states, traces, "gosungrow", defaultDashboardLocaleBundle)
	if sourceErr != nil {
		t.Fatal(sourceErr)
	}
	if accepted[dashboardSourceMappingID(target)]["p13116"] != calculated {
		t.Fatalf("legacy manual override was not preserved: %#v", accepted)
	}
	card := findDashboardSourceMappingCard(result, dashboardSourceMappingID(target))
	metric := card["metrics"].([]any)[0].(map[string]any)
	if unsupported, _ := metric["unsupported_calculated"].(bool); !unsupported {
		t.Fatalf("manual legacy source lacks blocking warning: %#v", metric)
	}
	candidates := card["candidates"].(map[string]any)["p13116"].([]any)
	found := false
	for _, raw := range candidates {
		candidate := raw.(map[string]any)
		if stringValue(candidate["entity_id"]) == calculated {
			found = candidate["selectable"] == false && candidate["compatibility"] == "unsupported"
		}
	}
	if !found {
		t.Fatalf("manual legacy candidate was not blocked: %#v", candidates)
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
		map[string]any{"path": "roof-trends", "cards": []any{map[string]any{"type": "history-graph", "entities": []any{shared}}}},
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

	result, accepted, sourceErr := applyDashboardSourceMappings(config, currentConfig, nil, []haDashboardTarget{targetA, targetB}, states, traces, "gosungrow", defaultDashboardLocaleBundle)
	if sourceErr != nil {
		t.Fatal(sourceErr)
	}
	if accepted[dashboardSourceMappingID(targetA)]["p13116"] != override {
		t.Fatalf("override was not accepted: %#v", accepted)
	}
	views := result["views"].([]any)
	if got, _ := getDashboardJSONPointer(views[0], "/cards/0/source_policy/native_template/entity"); got != override {
		t.Fatalf("target A not updated: %v", got)
	}
	if got, _ := getDashboardJSONPointer(views[1], "/cards/0/source_policy/native_template/entities/0"); got != override {
		t.Fatalf("target A repeated view not updated: %v", got)
	}
	if got, _ := getDashboardJSONPointer(views[3], "/cards/0/source_policy/native_template/entity"); got != "" {
		t.Fatalf("target B was incorrectly updated: %v", got)
	}
	bindings := findDashboardSourceMappingCard(result, dashboardSourceMappingID(targetA))["bindings"].(map[string]any)["p13116"].([]any)
	if len(bindings) != 6 {
		t.Fatalf("missing repeated native bindings: %#v", bindings)
	}
	for _, raw := range bindings {
		path := raw.(string)
		if !(strings.HasPrefix(path, "/views/0/") || strings.HasPrefix(path, "/views/1/")) {
			t.Fatalf("bindings escaped target A: %#v", bindings)
		}
		if value, ok := getDashboardJSONPointer(result, path); !ok || value != override {
			t.Fatalf("target A binding was not updated: %s", path)
		}
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
	result, accepted, sourceErr := applyDashboardSourceMappings(config, current, map[string]map[string]string{mappingID: {"p13112": "sensor.old_manual_p13112"}}, []haDashboardTarget{{PsKey: psKey}}, []haState{{EntityID: defaultEntity, State: "4", Attributes: map[string]any{"unit_of_measurement": "kWh"}}}, []dashboardMetricTrace{{Metric: "p13112", TargetPsKey: psKey, Resolved: defaultEntity}}, "gosungrow", defaultDashboardLocaleBundle)
	if sourceErr != nil {
		t.Fatal(sourceErr)
	}
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
	adoptedRaw, err := deepCopyJSONValue(automatic)
	if err != nil {
		t.Fatal(err)
	}
	adopted := adoptedRaw.(map[string]any)
	adoptedCard := findDashboardSourceMappingCard(adopted, "target")
	adoptedCard["defaults"] = map[string]any{"p13112": "sensor.safer_auto"}
	adoptedCard["pinned_defaults"] = map[string]any{"p13112": "sensor.safer_auto"}
	adoptedCard["matcher_version"] = dashboardSemanticMatcherVersion
	adoptedCard["metrics"].([]any)[0].(map[string]any)["default"] = "sensor.safer_auto"
	adopted["views"].([]any)[0].(map[string]any)["cards"].([]any)[0].(map[string]any)["entity"] = "sensor.safer_auto"
	adoptedHash, err := hashDashboardStructure(adopted)
	if err != nil {
		t.Fatal(err)
	}
	if autoHash != adoptedHash {
		t.Fatalf("adopted automatic source changed structure hash: %s != %s", autoHash, adoptedHash)
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
		if got := candidate["recommended"]; got != false {
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
