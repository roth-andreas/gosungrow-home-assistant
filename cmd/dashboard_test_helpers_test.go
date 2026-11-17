package cmd

import "strings"

func dashboardTestState(entityID string, state string, unit string) haState {
	attrs := map[string]any{}
	if unit != "" {
		attrs["unit_of_measurement"] = unit
	}
	uniqueID := entityID
	if strings.HasPrefix(entityID, "sensor.gosungrow_virtual_") && strings.HasSuffix(entityID, "_pv_power") {
		key := strings.TrimSuffix(strings.TrimPrefix(entityID, "sensor.gosungrow_virtual_"), "_pv_power")
		uniqueID = "GoSungrow-virtual-" + key + "-pv_power"
	}
	return haState{
		EntityID:         entityID,
		RegistryUniqueID: uniqueID,
		State:            state,
		Attributes:       attrs,
	}
}

func dashboardTestCanonicalState(entityID, registryID, state, unit string) haState {
	ret := dashboardTestState(entityID, state, unit)
	ret.RegistryUniqueID = registryID
	return ret
}
