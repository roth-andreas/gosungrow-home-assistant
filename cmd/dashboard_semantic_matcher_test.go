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
		return haState{EntityID: entity, State: value, Attributes: map[string]any{"unit_of_measurement": "kWh", "state_class": "total_increasing"}}
	}
	states := []haState{
		{EntityID: "sensor.gosungrow_1498605_7_1_1_phase_b_active_power", State: "0", Attributes: map[string]any{"unit_of_measurement": "kW"}},
		{EntityID: "sensor.gosungrow_1498605_1_1_1_total_active_power", State: "1.2", Attributes: map[string]any{"unit_of_measurement": "kW"}},
		energy("sensor.gosungrow_1498605_1_1_1_daily_generation", "5.0"),
		energy("sensor.gosungrow_1498605_11_0_0_feed_in_energy_today", "3.6"),
		energy("sensor.gosungrow_1498605_11_0_0_energy_purchased_today", "6.4"),
		energy("sensor.gosungrow_1498605_11_0_0_daily_home_consumption", "7.1"),
		{EntityID: "sensor.gosungrow_1498605_11_0_0_total_energy", State: "27.036", Attributes: map[string]any{"unit_of_measurement": "MWh", "state_class": "total_increasing"}},
	}
	return target, states
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
