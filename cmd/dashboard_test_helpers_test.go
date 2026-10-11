package cmd

func dashboardTestState(entityID string, state string, unit string) haState {
	attrs := map[string]any{}
	if unit != "" {
		attrs["unit_of_measurement"] = unit
	}
	return haState{
		EntityID:         entityID,
		RegistryUniqueID: entityID,
		State:            state,
		Attributes:       attrs,
	}
}

func dashboardTestCanonicalState(entityID, registryID, state, unit string) haState {
	ret := dashboardTestState(entityID, state, unit)
	ret.RegistryUniqueID = registryID
	return ret
}
