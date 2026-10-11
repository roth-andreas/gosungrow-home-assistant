package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/queryDeviceList"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api"
)

func dailyDashboardTemplate(t *testing.T, targets []haDashboardTarget) map[string]any {
	t.Helper()
	config, err := renderDashboardConfig("../addon/gosungrow/assets/home-assistant-sungrow-flow.yaml", "Synthetic Plant", targets, defaultDashboardLocaleBundle)
	if err != nil {
		t.Fatal(err)
	}
	return config
}

func dailyDashboardTarget() haDashboardTarget {
	return haDashboardTarget{PsID: "100", PsKey: "100_11_0_0", ViewTitle: "Roof", ViewPath: "roof", DeviceType: 11}
}

func dailyTrace(t *testing.T, report dashboardRemapReport, key, metric string) dashboardMetricTrace {
	t.Helper()
	for _, trace := range report.Traces {
		if trace.TargetPsKey == key && trace.Metric == metric {
			return trace
		}
	}
	t.Fatalf("missing final trace target=%s metric=%s", key, metric)
	return dashboardMetricTrace{}
}

func assertDailyBindings(t *testing.T, config map[string]any, target haDashboardTarget, metric, entity string) []any {
	t.Helper()
	card := findDashboardSourceMappingCard(config, dashboardSourceMappingID(target))
	if card == nil {
		t.Fatal("source card missing")
	}
	paths, _ := card["bindings"].(map[string]any)[metric].([]any)
	if len(paths) == 0 {
		t.Fatalf("source %s has no bindings", metric)
	}
	for _, raw := range paths {
		got, ok := getDashboardJSONPointer(config, stringValue(raw))
		if !ok || got != entity {
			t.Fatalf("binding %s=%v, want %s", raw, got, entity)
		}
	}
	return paths
}

// Decode real API point rows and project them through the production MQTT
// discovery/state serializers. The registry IDs come from those messages,
// while Home Assistant's displayed entity names are deliberately unrelated.
func decodedDailyDashboardStates(t *testing.T, load string) []haState {
	t.Helper()
	ep := queryDeviceList.Init(api.Web{})
	ep.Request.PsId.SetString("100")
	response := fmt.Sprintf(`{"result_data":{"pageList":[{"ps_id":"100","ps_key":"100_11_0_0","device_type":11,"point_data":[{"point_id":"p13112","point_name":"Daily Production","value":12.4,"unit":"kWh"},{"point_id":"p13199","point_name":"Daily Home Consumption","value":%s,"unit":"kWh"},{"point_id":"p13173","point_name":"Daily Export","value":3.6,"unit":"kWh"},{"point_id":"p13147","point_name":"Daily Import","value":1.8,"unit":"kWh"},{"point_id":"p83097","point_name":"Direct Solar","value":8.8,"unit":"kWh"}]}]}}`, load)
	if err := json.Unmarshal([]byte(response), &ep.Response); err != nil {
		t.Fatal(err)
	}
	data := ep.GetData()
	data.ProcessMap()
	transport := &pvTestTransport{}
	command := pvCommand(t, transport)
	if _, err := command.Client.SetDeviceConfig("GoSungrow", "100_11_0_0", "100_11_0_0", "Synthetic Plant", "Plant", "Sungrow", "Roof"); err != nil {
		t.Fatal(err)
	}
	_ = capturePVCommandLogs(t, func() {
		if err := command.Update("queryDeviceList", data, false); err != nil {
			t.Fatal(err)
		}
	})
	names := map[string]string{"p13112": "sensor.ertrag_heute", "p13199": "sensor.house_today", "p13173": "sensor.export_today", "p13147": "sensor.import_today", "p83097": "sensor.direct_solar"}
	byTopic := make(map[string]map[string]any)
	for _, publication := range transport.publications {
		byTopic[publication.topic] = publication.payload
	}
	states := []haState{dashboardTestState("sensor.gosungrow_virtual_100_pv_power", "1.2", "kW")}
	for _, publication := range transport.publications {
		uid := stringValue(publication.payload["unique_id"])
		for point, entity := range names {
			if !dashboardSemanticTerminalIdentifier(uid, point) {
				continue
			}
			payload := byTopic[stringValue(publication.payload["state_topic"])]
			states = append(states, haState{EntityID: entity, RegistryUniqueID: uid, State: stringValue(payload["value"]), Attributes: map[string]any{"unit_of_measurement": publication.payload["unit_of_measurement"], "friendly_name": "Translated display name"}})
		}
	}
	if len(states) != 6 {
		t.Fatalf("expected five real daily discovery/state projections, got %#v", states)
	}
	return states
}

func TestDecodedDailySourcesSurviveCompleteDashboardComposition(t *testing.T) {
	for _, load := range []string{"12.4", "0"} {
		t.Run(load, func(t *testing.T) {
			target := dailyDashboardTarget()
			states := decodedDailyDashboardStates(t, load)
			config, _, report := composeDashboardSources(dailyDashboardTemplate(t, []haDashboardTarget{target}), nil, nil, []haDashboardTarget{target}, states, "gosungrow-flow", defaultDashboardLocaleBundle)
			card := findDashboardSourceMappingCard(config, dashboardSourceMappingID(target))
			for metric, entity := range map[string]string{"p13112": "sensor.ertrag_heute", "p13199": "sensor.house_today", "p13173": "sensor.export_today", "p13147": "sensor.import_today", "p13116": "sensor.direct_solar"} {
				if anyMapToStringMap(card["defaults"])[metric] != entity {
					t.Fatalf("%s lost its canonical source: %#v", metric, card["defaults"])
				}
				paths := assertDailyBindings(t, config, target, metric, entity)
				trace := dailyTrace(t, report, target.PsKey, metric)
				if trace.Source != "canonical" || !trace.Resolution.Match.Confident || trace.BindingCount != len(paths) {
					t.Fatalf("trace disagrees with composed dashboard: %#v", trace)
				}
				found := false
				for _, raw := range card["candidates"].(map[string]any)[metric].([]any) {
					found = found || stringValue(raw.(map[string]any)["entity_id"]) == entity
				}
				if !found {
					t.Fatalf("renamed canonical choice missing from selector: %s", metric)
				}
			}
			if anyMapToStringMap(card["defaults"])["pv_power"] != states[0].EntityID {
				t.Fatal("unrelated live PV mapping changed")
			}
		})
	}
}

func TestDailyDashboardRejectsLegacyFallbackThroughComposition(t *testing.T) {
	target := dailyDashboardTarget()
	for _, tc := range []struct{ metric, suffix, unit, value string }{{"p13199", "total_energy", "MWh", "27.036"}, {"p13112", "daily_feed_in_energy_pv", "kWh", "3.6"}} {
		t.Run(tc.metric, func(t *testing.T) {
			state := dashboardTestState("sensor.gosungrow_100_"+tc.suffix, tc.value, tc.unit)
			config, _, report := composeDashboardSources(dailyDashboardTemplate(t, []haDashboardTarget{target}), nil, nil, []haDashboardTarget{target}, []haState{state}, "gosungrow-flow", defaultDashboardLocaleBundle)
			card := findDashboardSourceMappingCard(config, dashboardSourceMappingID(target))
			if anyMapToStringMap(card["defaults"])[tc.metric] != "" {
				t.Fatalf("unsafe daily default: %#v", card["defaults"])
			}
			trace := dailyTrace(t, report, target.PsKey, tc.metric)
			if trace.Resolved != "" || trace.BindingCount != 0 || trace.Source != "unavailable" {
				t.Fatalf("unsafe daily source survived pruning: %#v", trace)
			}
		})
	}
}

func TestDailyDashboardRejectsUnverifiedInvalidAndAmbiguousSources(t *testing.T) {
	target := dailyDashboardTarget()
	base := dashboardTestCanonicalState("sensor.gosungrow_virtual_100_11_0_0_p13199", "gosungrow_virtual_100_11_0_0_p13199", "12.4", "kWh")
	for _, tc := range []struct {
		name   string
		mutate func(*haState) []haState
	}{
		{"no registry", func(s *haState) []haState { s.RegistryUniqueID = ""; return nil }},
		{"partial registry", func(s *haState) []haState {
			s.RegistryUniqueID = ""
			return []haState{dashboardTestState("sensor.unrelated", "1", "kW")}
		}},
		{"conflicting identity", func(s *haState) []haState { s.RegistryUniqueID = "gosungrow_100_p13173"; return nil }},
		{"foreign plant despite local entity name", func(s *haState) []haState { s.RegistryUniqueID = "GoSungrow-virtual-200_11_0_0-p13199"; return nil }},
		{"wrong unit", func(s *haState) []haState { s.Attributes = map[string]any{"unit_of_measurement": "kW"}; return nil }},
		{"unknown", func(s *haState) []haState { s.State = "unknown"; return nil }},
		{"non finite", func(s *haState) []haState { s.State = "NaN"; return nil }},
		{"stale", func(s *haState) []haState {
			s.LastUpdated = time.Now().Add(-48 * time.Hour).Format(time.RFC3339)
			return nil
		}},
		{"future", func(s *haState) []haState {
			s.LastUpdated = time.Now().Add(30 * time.Minute).Format(time.RFC3339)
			return nil
		}},
		{"ambiguous", func(s *haState) []haState {
			return []haState{dashboardTestCanonicalState("sensor.other_home", "gosungrow_getpsdetail_100_p13199", "7", "kWh")}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := base
			states := append(tc.mutate(&state), state)
			config, _, report := composeDashboardSources(dailyDashboardTemplate(t, []haDashboardTarget{target}), nil, nil, []haDashboardTarget{target}, states, "gosungrow-flow", defaultDashboardLocaleBundle)
			trace := dailyTrace(t, report, target.PsKey, "p13199")
			if trace.Resolved != "" || trace.BindingCount != 0 || trace.Resolution.Match.Confident {
				t.Fatalf("invalid/unverified daily source selected: %#v", trace)
			}
			if anyMapToStringMap(findDashboardSourceMappingCard(config, dashboardSourceMappingID(target))["defaults"])["p13199"] != "" {
				t.Fatal("legacy exact name bypassed canonical rejection")
			}
		})
	}
}

func TestDailyDashboardPreservesPinnedAndAcceptedUnavailableSources(t *testing.T) {
	target := dailyDashboardTarget()
	mappingID := dashboardSourceMappingID(target)
	old := "sensor.gosungrow_100_total_energy"
	for _, mode := range []string{"pinned with recommendation", "missing manual", "missing manual without automatic", "stale manual"} {
		t.Run(mode, func(t *testing.T) {
			currentCard := map[string]any{"type": dashboardSourceMappingCardType, "schema_version": 1, "mapping_id": mappingID, "pinned_defaults": map[string]any{"p13199": old}}
			current := map[string]any{"views": []any{map[string]any{"cards": []any{currentCard}}}}
			states := []haState{dashboardTestState(old, "27.036", "MWh")}
			persisted := make(map[string]map[string]string)
			want, source := old, "pinned"
			if mode == "pinned with recommendation" {
				states = append(states, dashboardTestCanonicalState("sensor.home_today", "gosungrow_100_p13199", "12.4", "kWh"))
			} else {
				want, source = "sensor.accepted_home", "manual"
				currentCard["overrides"] = map[string]any{"p13199": want}
				persisted[mappingID] = map[string]string{"p13199": want}
				if mode == "missing manual without automatic" {
					delete(currentCard, "pinned_defaults")
				}
				if mode == "stale manual" {
					state := dashboardTestCanonicalState(want, "gosungrow_100_p13199", "12.4", "kWh")
					state.LastUpdated = time.Now().Add(-48 * time.Hour).Format(time.RFC3339)
					states = append(states, state)
				}
			}
			config, accepted, report := composeDashboardSources(dailyDashboardTemplate(t, []haDashboardTarget{target}), current, persisted, []haDashboardTarget{target}, states, "gosungrow-flow", defaultDashboardLocaleBundle)
			assertDailyBindings(t, config, target, "p13199", want)
			trace := dailyTrace(t, report, target.PsKey, "p13199")
			if trace.Source != source || !trace.Resolution.NeedsReview {
				t.Fatalf("saved source identity/review state lost: %#v", trace)
			}
			if source == "manual" && accepted[mappingID]["p13199"] != want {
				t.Fatalf("accepted missing/stale source was dropped: %#v", accepted)
			}
			if mode == "pinned with recommendation" && anyMapToStringMap(findDashboardSourceMappingCard(config, mappingID)["recommendations"])["p13199"] != "sensor.home_today" {
				t.Fatal("safer source was not offered for explicit adoption")
			}
		})
	}
}

func TestDailyDashboardBindingsRemainSeparateAfterPruning(t *testing.T) {
	target := dailyDashboardTarget()
	shared := "sensor.gosungrow_100_feed_in_energy_today"
	current := map[string]any{"views": []any{map[string]any{"cards": []any{map[string]any{"type": dashboardSourceMappingCardType, "schema_version": 1, "mapping_id": dashboardSourceMappingID(target), "pinned_defaults": map[string]any{"p13112": shared, "p13147": shared}}}}}}
	states := []haState{dashboardTestState(shared, "3.6", "kWh"), dashboardTestCanonicalState("sensor.yield", "gosungrow_100_p13112", "12.4", "kWh"), dashboardTestCanonicalState("sensor.import", "gosungrow_100_p13147", "1.8", "kWh")}
	config, _, _ := composeDashboardSources(dailyDashboardTemplate(t, []haDashboardTarget{target}), current, nil, []haDashboardTarget{target}, states, "gosungrow-flow", defaultDashboardLocaleBundle)
	production := assertDailyBindings(t, config, target, "p13112", shared)
	imports := assertDailyBindings(t, config, target, "p13147", shared)
	seen := make(map[string]bool)
	for _, path := range production {
		seen[stringValue(path)] = true
	}
	for _, path := range imports {
		if seen[stringValue(path)] {
			t.Fatalf("production and import bindings collapsed: %v", path)
		}
	}
}

func TestDailyDashboardDiagnosticsDescribeFinalOutcomeAndBoundCandidates(t *testing.T) {
	target := dailyDashboardTarget()
	states := make([]haState, 0, 25)
	for i := 0; i < 25; i++ {
		states = append(states, dashboardTestCanonicalState(fmt.Sprintf("sensor.home_%02d", i), fmt.Sprintf("gosungrow_snapshot_%02d_100_p13199", i), fmt.Sprint(i), "kWh"))
	}
	_, _, report := composeDashboardSources(dailyDashboardTemplate(t, []haDashboardTarget{target}), nil, nil, []haDashboardTarget{target}, states, "gosungrow-flow", defaultDashboardLocaleBundle)
	trace := dailyTrace(t, report, target.PsKey, "p13199")
	if len(trace.Candidates) != 5 || len(trace.Resolution.Match.Candidates) != 25 {
		t.Fatalf("candidate bounds/count lost: %#v", trace)
	}
	var output bytes.Buffer
	writeDashboardInstallDiagnostics(&output, dashboardInstallDiagnostics{Debug: true, MetricTraces: []dashboardMetricTrace{trace}})
	for _, required := range []string{"source=unavailable", "eligible_candidates=25", "Ambiguous canonical daily candidates", "bindings=0"} {
		if !strings.Contains(output.String(), required) {
			t.Fatalf("missing final diagnostic %q: %s", required, output.String())
		}
	}
	if strings.Count(output.String(), "  candidate ") != 5 {
		t.Fatalf("candidate detail unbounded: %s", output.String())
	}
}

func TestDailyDashboardCompositionIsolatesOpaqueTargetsAndSavedOverrides(t *testing.T) {
	targets := []haDashboardTarget{
		{PsID: "plant-a", PsKey: "plant-a_11_0_0", ViewTitle: "Roof", ViewPath: "roof", DeviceType: 11},
		{PsID: "plant_a", PsKey: "plant_a_11_0_0", ViewTitle: "Garage", ViewPath: "garage", DeviceType: 11},
	}
	states := []haState{
		dashboardTestCanonicalState("sensor.home_roof", "GoSungrow-virtual-plant-a_11_0_0-p13199", "12.4", "kWh"),
		dashboardTestCanonicalState("sensor.home_garage", "GoSungrow-virtual-plant_a_11_0_0-p13199", "7", "kWh"),
	}
	config, _, report := composeDashboardSources(dailyDashboardTemplate(t, targets), nil, nil, targets, states, "gosungrow-flow", defaultDashboardLocaleBundle)
	for i, target := range targets {
		assertDailyBindings(t, config, target, "p13199", states[i].EntityID)
		if trace := dailyTrace(t, report, target.PsKey, "p13199"); trace.Resolved != states[i].EntityID || len(trace.Resolution.Match.Candidates) != 1 {
			t.Fatalf("wrong plant's point entered resolution: %#v", trace)
		}
	}
	// A saved choice may share an entity with another target. Accepting an
	// available source for one target must still rewrite only its own bindings.
	shared := states[1].EntityID
	current := map[string]any{"views": []any{map[string]any{"cards": []any{
		map[string]any{"type": dashboardSourceMappingCardType, "schema_version": 1, "mapping_id": dashboardSourceMappingID(targets[0]), "pinned_defaults": map[string]any{"p13199": shared}, "overrides": map[string]any{"p13199": states[0].EntityID}},
		map[string]any{"type": dashboardSourceMappingCardType, "schema_version": 1, "mapping_id": dashboardSourceMappingID(targets[1]), "pinned_defaults": map[string]any{"p13199": shared}},
	}}}}
	config, accepted, _ := composeDashboardSources(dailyDashboardTemplate(t, targets), current, nil, targets, states, "gosungrow-flow", defaultDashboardLocaleBundle)
	assertDailyBindings(t, config, targets[0], "p13199", states[0].EntityID)
	assertDailyBindings(t, config, targets[1], "p13199", shared)
	if accepted[dashboardSourceMappingID(targets[0])]["p13199"] != states[0].EntityID {
		t.Fatal("eligible renamed manual source was rejected")
	}
	if _, present := accepted[dashboardSourceMappingID(targets[1])]; present {
		t.Fatal("override crossed targets")
	}
	if invalid := validateDashboardSourceOverrides("source-test", map[string]string{"p13199": ""}, map[string]string{"p13199": states[1].EntityID}, nil, targets[0], false, states); len(invalid) != 0 {
		t.Fatalf("wrong-target daily override accepted: %#v", invalid)
	}
}

func TestDailyDashboardDoesNotChooseOneInverterForMultiInverterYield(t *testing.T) {
	target := dailyDashboardTarget()
	target.PlantDevices = []dashboardPlantDevice{{PsKey: "100_1_1_1", DeviceType: 1}, {PsKey: "100_1_1_2", DeviceType: 1}}
	states := []haState{dashboardTestCanonicalState("sensor.yield", "GoSungrow-virtual-100_1_1_1-p1", "12.4", "kWh")}
	_, _, report := composeDashboardSources(dailyDashboardTemplate(t, []haDashboardTarget{target}), nil, nil, []haDashboardTarget{target}, states, "gosungrow-flow", defaultDashboardLocaleBundle)
	trace := dailyTrace(t, report, target.PsKey, "p13112")
	if trace.Resolved != "" || trace.BindingCount != 0 || trace.Resolution.Match.Confident {
		t.Fatalf("single-inverter yield became a plant default: %#v", trace)
	}
}
