package cmd

import "testing"

func issue19SemanticFixture() (haDashboardTarget, []haState) {
	target := haDashboardTarget{
		PsID: "1498605", PsKey: "1498605_11_0_0",
		PlantDevices: []dashboardPlantDevice{
			{PsKey: "1498605_1_1_1", DeviceType: 1},
			{PsKey: "1498605_7_1_1", DeviceType: 7},
			{PsKey: "1498605_11_0_0", DeviceType: 11},
		},
	}
	energy := func(entity, value string) haState {
		return haState{EntityID: entity, RegistryUniqueID: entity, State: value, Attributes: map[string]any{"unit_of_measurement": "kWh", "state_class": "total_increasing"}}
	}
	states := []haState{
		{EntityID: "sensor.gosungrow_1498605_7_1_1_phase_b_active_power", RegistryUniqueID: "sensor.gosungrow_1498605_7_1_1_phase_b_active_power", State: "0", Attributes: map[string]any{"unit_of_measurement": "kW"}},
		{EntityID: "sensor.gosungrow_1498605_1_1_1_total_active_power", RegistryUniqueID: "sensor.gosungrow_1498605_1_1_1_total_active_power", State: "1.2", Attributes: map[string]any{"unit_of_measurement": "kW"}},
		energy("sensor.gosungrow_1498605_1_1_1_daily_generation", "5.0"),
		energy("sensor.gosungrow_1498605_11_0_0_feed_in_energy_today", "3.6"),
		energy("sensor.gosungrow_1498605_11_0_0_energy_purchased_today", "6.4"),
		energy("sensor.gosungrow_1498605_11_0_0_daily_home_consumption", "7.1"),
		{EntityID: "sensor.gosungrow_1498605_11_0_0_total_energy", State: "27.036", Attributes: map[string]any{"unit_of_measurement": "MWh", "state_class": "total_increasing"}},
	}
	return target, states
}

func TestDashboardCanonicalMatcherUsesRegistryUniqueIDAfterEntityRename(t *testing.T) {
	target, _ := issue19SemanticFixture()
	states := []haState{{
		EntityID: "sensor.meine_solaranlage_heute", RegistryUniqueID: "gosungrow_1498605_11_0_0_p13112", State: "12.4",
		Attributes: map[string]any{"unit_of_measurement": "kWh", "friendly_name": "Ertrag heute", "state_class": "total_increasing"},
	}}
	match := dashboardSemanticRecommendation(target, "p13112", states, true)
	if !match.Confident || match.Entity != states[0].EntityID || match.Candidates[0].PointID != "p13112" {
		t.Fatalf("renamed canonical entity was not matched through unique_id: %#v", match)
	}
}

func TestDashboardCanonicalMatcherRejectsDirectionAndLifetimeConflicts(t *testing.T) {
	target, _ := issue19SemanticFixture()
	states := []haState{
		{EntityID: "sensor.export", RegistryUniqueID: "gosungrow_1498605_11_0_0_p13173", State: "4", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
		{EntityID: "sensor.lifetime", RegistryUniqueID: "gosungrow_1498605_11_0_0_total_energy", State: "1234", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
	}
	if match := dashboardSemanticRecommendation(target, "p13112", states, true); match.Entity != "" {
		t.Fatalf("production selected export or lifetime energy: %#v", match)
	}
}

func TestDashboardCanonicalDirectSolarAcceptsOnlyNativePoints(t *testing.T) {
	target, _ := issue19SemanticFixture()
	states := []haState{
		{EntityID: "sensor.calculated", RegistryUniqueID: "gosungrow_1498605_11_0_0_pv_consumption_energy", State: "9", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
		{EntityID: "sensor.native", RegistryUniqueID: "gosungrow_1498605_11_0_0_p83097", State: "8", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
	}
	match := dashboardSemanticRecommendation(target, "p13116", states, true)
	if !match.Confident || match.Entity != "sensor.native" || len(match.Candidates) != 1 {
		t.Fatalf("unexpected direct-solar match: %#v", match)
	}
}

func TestDashboardCanonicalProductionDoesNotSilentlyChooseOneOfMultipleInverters(t *testing.T) {
	target, _ := issue19SemanticFixture()
	target.PlantDevices = append(target.PlantDevices, dashboardPlantDevice{PsKey: "1498605_1_2_1", DeviceType: 1})
	states := []haState{{EntityID: "sensor.one", RegistryUniqueID: "gosungrow_1498605_1_1_1_p1", State: "6", Attributes: map[string]any{"unit_of_measurement": "kWh"}}}
	match := dashboardSemanticRecommendation(target, "p13112", states, true)
	if match.Confident || match.Entity != "sensor.one" {
		t.Fatalf("multi-inverter candidate should be review-only: %#v", match)
	}
}

func TestDashboardCanonicalMatcherFallsBackConservativelyWithoutRegistry(t *testing.T) {
	target, _ := issue19SemanticFixture()
	states := []haState{{EntityID: "sensor.gosungrow_1498605_11_0_0_p13112", State: "5", Attributes: map[string]any{"unit_of_measurement": "kWh"}}}
	match := dashboardSemanticRecommendation(target, "p13112", states, true)
	if match.Confident || len(match.Candidates) != 1 || match.Candidates[0].Compatibility != "unverified" {
		t.Fatalf("registry-less match was not conservative: %#v", match)
	}
}

func TestDashboardCanonicalMatcherSupportsCommunicationTargetAndIsolatesPlants(t *testing.T) {
	target := haDashboardTarget{
		PsID: "1610907", PsKey: "1610907_22_247_1",
		PlantDevices: []dashboardPlantDevice{{PsKey: "1610907_22_247_1", DeviceType: 22}},
	}
	states := []haState{
		{EntityID: "sensor.target_load", RegistryUniqueID: "gosungrow_1610907_22_247_1_p13199", State: "8", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
		{EntityID: "sensor.other_load", RegistryUniqueID: "gosungrow_9999999_22_247_1_p13199", State: "99", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
	}
	match := dashboardSemanticRecommendation(target, "p13199", states, false)
	if !match.Confident || match.Entity != "sensor.target_load" || len(match.Candidates) != 1 {
		t.Fatalf("communication target was not isolated: %#v", match)
	}
}

func TestDashboardCanonicalMatcherPrefersPlantAggregateOverSingleInverter(t *testing.T) {
	target, _ := issue19SemanticFixture()
	states := []haState{
		{EntityID: "sensor.inverter", RegistryUniqueID: "gosungrow_1498605_1_1_1_p1", State: "5", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
		{EntityID: "sensor.plant", RegistryUniqueID: "gosungrow_1498605_11_0_0_p83022", State: "5", Attributes: map[string]any{"unit_of_measurement": "kWh"}},
	}
	match := dashboardSemanticRecommendation(target, "p13112", states, true)
	if !match.Confident || match.Entity != "sensor.plant" {
		t.Fatalf("plant aggregate was not preferred: %#v", match)
	}
}

func TestDashboardSemanticMatcherResolvesIssue19Meanings(t *testing.T) {
	target, states := issue19SemanticFixture()
	want := map[string]string{
		"pv_power": "sensor.gosungrow_1498605_1_1_1_total_active_power",
		"p13112":   "sensor.gosungrow_1498605_1_1_1_daily_generation",
		"p13173":   "sensor.gosungrow_1498605_11_0_0_feed_in_energy_today",
		"p13147":   "sensor.gosungrow_1498605_11_0_0_energy_purchased_today",
		"p13199":   "sensor.gosungrow_1498605_11_0_0_daily_home_consumption",
	}
	for metric, entity := range want {
		match := dashboardSemanticRecommendation(target, metric, states, true)
		if !match.Confident || match.Entity != entity {
			t.Errorf("%s: got entity %q confident=%v, want %q", metric, match.Entity, match.Confident, entity)
		}
	}
}

func TestDashboardSemanticMatcherDoesNotGuessMissingDailyConsumption(t *testing.T) {
	target, states := issue19SemanticFixture()
	states = states[:len(states)-2]
	states = append(states, haState{EntityID: "sensor.gosungrow_1498605_11_0_0_total_energy", State: "27.036", Attributes: map[string]any{"unit_of_measurement": "MWh"}})
	match := dashboardSemanticRecommendation(target, "p13199", states, true)
	if match.Entity != "" || match.Confident {
		t.Fatalf("lifetime total was accepted as daily consumption: %#v", match)
	}
}
