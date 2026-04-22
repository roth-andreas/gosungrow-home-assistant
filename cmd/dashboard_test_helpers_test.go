package cmd

func dashboardTestState(entityID string, state string, unit string) haState {
	attrs := map[string]any{}
	if unit != "" {
		attrs["unit_of_measurement"] = unit
	}
	return haState{
		EntityID:   entityID,
		State:      state,
		Attributes: attrs,
	}
}
