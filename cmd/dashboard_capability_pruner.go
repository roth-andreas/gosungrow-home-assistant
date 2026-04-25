package cmd

import "strings"

func pruneDashboardForUnavailableMetrics(config map[string]any, targets []haDashboardTarget, states []haState) map[string]any {
	if len(config) == 0 || len(targets) == 0 || len(states) == 0 {
		return config
	}

	unsupportedByTarget := dashboardUnsupportedMetricsByTarget(config, targets, states)
	if len(unsupportedByTarget) == 0 {
		return config
	}

	rawViews, ok := config["views"].([]any)
	if !ok || len(rawViews) == 0 {
		return config
	}

	changed := false
	prunedViews := make([]any, 0, len(rawViews))
	for _, rawView := range rawViews {
		viewMap, ok := rawView.(map[string]any)
		if !ok {
			prunedViews = append(prunedViews, rawView)
			continue
		}

		target, hasTarget := dashboardTargetForView(viewMap, targets)
		if !hasTarget {
			prunedViews = append(prunedViews, rawView)
			continue
		}

		unsupported := unsupportedByTarget[strings.ToLower(strings.TrimSpace(target.PsKey))]
		if len(unsupported) == 0 {
			prunedViews = append(prunedViews, rawView)
			continue
		}

		prunedView, viewChanged := pruneUnsupportedMetricsFromView(viewMap, unsupported)
		if viewChanged {
			changed = true
		}
		prunedViews = append(prunedViews, prunedView)
	}

	if !changed {
		return config
	}

	ret := make(map[string]any, len(config))
	for key, value := range config {
		ret[key] = value
	}
	ret["views"] = prunedViews
	return ret
}

func pruneDashboardForMissingBattery(config map[string]any, targets []haDashboardTarget, states []haState) map[string]any {
	return pruneDashboardForUnavailableMetrics(config, targets, states)
}

func dashboardUnsupportedMetricsByTarget(config map[string]any, targets []haDashboardTarget, states []haState) map[string]map[string]struct{} {
	refs := collectLegacyDashboardEntityRefs(config, targets)
	if len(refs) == 0 {
		return nil
	}

	metricsByTarget := make(map[string]map[string]struct{}, len(targets))
	for _, ref := range refs {
		psKey := strings.ToLower(strings.TrimSpace(ref.Target.PsKey))
		if psKey == "" {
			continue
		}
		if metricsByTarget[psKey] == nil {
			metricsByTarget[psKey] = make(map[string]struct{})
		}
		metricsByTarget[psKey][strings.ToLower(strings.TrimSpace(ref.Metric))] = struct{}{}
	}

	if len(metricsByTarget) == 0 {
		return nil
	}

	singleTarget := len(targets) == 1
	stateByID := dashboardStateByEntityID(states)
	ret := make(map[string]map[string]struct{})
	for _, target := range targets {
		psKey := strings.ToLower(strings.TrimSpace(target.PsKey))
		metrics := metricsByTarget[psKey]
		if len(metrics) == 0 {
			continue
		}

		hasBattery := dashboardTargetHasBattery(target, states, singleTarget)
		unsupported := make(map[string]struct{})
		for metric := range metrics {
			if isBatteryDashboardMetric(metric) && !hasBattery {
				unsupported[metric] = struct{}{}
				continue
			}
			if resolveDashboardMetricEntity(target, metric, states, stateByID, singleTarget) == "" {
				unsupported[metric] = struct{}{}
			}
		}

		if len(unsupported) > 0 {
			ret[psKey] = unsupported
		}
	}

	if len(ret) == 0 {
		return nil
	}
	return ret
}

func dashboardTargetHasBattery(target haDashboardTarget, states []haState, singleTarget bool) bool {
	for _, state := range states {
		candidate := strings.ToLower(strings.TrimSpace(state.EntityID))
		if !strings.HasPrefix(candidate, "sensor.") || !strings.Contains(candidate, "gosungrow") {
			continue
		}

		_, ok := dashboardEntityPlantAffinity(candidate, target, singleTarget)
		if !ok {
			continue
		}

		if isBatteryCapabilityState(candidate, state) {
			return true
		}
	}

	return false
}

func isBatteryCapabilityState(candidate string, state haState) bool {
	if !dashboardStateHasUsableNumericValue(state) {
		return false
	}

	unit := dashboardStateUnit(state)
	if isBatterySocEntityReference(candidate) && (unit == "" || strings.TrimSpace(unit) == "%") {
		return true
	}
	if isBatteryPowerCapabilityEntityReference(candidate) && (unit == "" || isDashboardPowerUnit(unit)) {
		return true
	}
	return false
}

func isBatterySocEntityReference(entityID string) bool {
	candidate := strings.ToLower(strings.TrimSpace(entityID))
	return strings.HasSuffix(candidate, "_soc") ||
		strings.Contains(candidate, "_soc_") ||
		strings.Contains(candidate, "_p13141") ||
		strings.Contains(candidate, "_p13142") ||
		strings.Contains(candidate, "_p83129") ||
		strings.Contains(candidate, "_p83252") ||
		strings.Contains(candidate, "_battery_level") ||
		strings.Contains(candidate, "_battery_charge_percent")
}

func isBatteryPowerCapabilityEntityReference(entityID string) bool {
	candidate := strings.ToLower(strings.TrimSpace(entityID))
	return strings.Contains(candidate, "_battery_power") ||
		strings.Contains(candidate, "_battery_charge_power") ||
		strings.Contains(candidate, "_battery_discharge_power") ||
		strings.Contains(candidate, "_es_power") ||
		strings.Contains(candidate, "_p13126") ||
		strings.Contains(candidate, "_p13150") ||
		strings.Contains(candidate, "_p83081") ||
		strings.Contains(candidate, "_p83128") ||
		strings.Contains(candidate, "optical_storage")
}

func isBatteryDashboardMetric(metric string) bool {
	switch strings.ToLower(strings.TrimSpace(metric)) {
	case "battery_power", "p13141", "p13174", "pv_to_battery_power", "battery_to_load_power":
		return true
	default:
		return false
	}
}

func dashboardTargetForView(view map[string]any, targets []haDashboardTarget) (haDashboardTarget, bool) {
	if len(targets) == 1 {
		return targets[0], true
	}

	scores := make(map[string]int, len(targets))
	for _, target := range targets {
		scores[target.PsKey] = 0
	}

	var walk func(any)
	walk = func(value any) {
		switch typed := value.(type) {
		case map[string]any:
			for _, entry := range typed {
				walk(entry)
			}
		case []any:
			for _, entry := range typed {
				walk(entry)
			}
		case string:
			candidate := strings.ToLower(strings.TrimSpace(typed))
			for _, target := range targets {
				psKey := strings.ToLower(strings.TrimSpace(target.PsKey))
				psID := strings.ToLower(strings.TrimSpace(target.PsID))
				if psKey != "" && strings.Contains(candidate, "gosungrow_virtual_"+psKey+"_") {
					scores[target.PsKey] += 3
				}
				if psID != "" && strings.Contains(candidate, "gosungrow_virtual_"+psID+"_") {
					scores[target.PsKey]++
				}
			}
		}
	}

	walk(view)

	bestKey := ""
	bestScore := 0
	for psKey, score := range scores {
		if score > bestScore {
			bestKey = psKey
			bestScore = score
		}
	}
	if bestKey == "" {
		return haDashboardTarget{}, false
	}
	for _, target := range targets {
		if target.PsKey == bestKey {
			return target, true
		}
	}
	return haDashboardTarget{}, false
}

func pruneUnsupportedMetricsFromView(view map[string]any, unsupported map[string]struct{}) (map[string]any, bool) {
	ret := make(map[string]any, len(view))
	for key, value := range view {
		ret[key] = value
	}

	changed := false
	if rawSections, ok := view["sections"].([]any); ok {
		prunedSections := make([]any, 0, len(rawSections))
		for _, rawSection := range rawSections {
			sectionMap, ok := rawSection.(map[string]any)
			if !ok {
				prunedSections = append(prunedSections, rawSection)
				continue
			}

			prunedSection, keepSection, sectionChanged := pruneUnsupportedMetricsFromSection(sectionMap, unsupported)
			if sectionChanged {
				changed = true
			}
			if keepSection {
				prunedSections = append(prunedSections, prunedSection)
			} else {
				changed = true
			}
		}
		ret["sections"] = prunedSections
	}

	if rawCards, ok := view["cards"].([]any); ok {
		prunedCards, cardsChanged := pruneUnsupportedMetricsFromCards(rawCards, unsupported)
		if cardsChanged {
			changed = true
		}
		ret["cards"] = prunedCards
	}

	return ret, changed
}

func pruneUnsupportedMetricsFromSection(section map[string]any, unsupported map[string]struct{}) (map[string]any, bool, bool) {
	ret := make(map[string]any, len(section))
	for key, value := range section {
		ret[key] = value
	}

	rawCards, ok := section["cards"].([]any)
	if !ok {
		return ret, true, false
	}

	prunedCards, changed := pruneUnsupportedMetricsFromCards(rawCards, unsupported)
	ret["cards"] = prunedCards
	if !hasNonHeadingCards(prunedCards) {
		return ret, false, true
	}

	return ret, true, changed
}

func pruneUnsupportedMetricsFromCards(cards []any, unsupported map[string]struct{}) ([]any, bool) {
	ret := make([]any, 0, len(cards))
	changed := false
	for _, rawCard := range cards {
		cardMap, ok := rawCard.(map[string]any)
		if !ok {
			ret = append(ret, rawCard)
			continue
		}

		prunedCard, keepCard, cardChanged := pruneUnsupportedMetricsFromCard(cardMap, unsupported)
		if cardChanged {
			changed = true
		}
		if !keepCard {
			changed = true
			continue
		}
		ret = append(ret, prunedCard)
	}

	return ret, changed
}

func pruneUnsupportedMetricsFromCard(card map[string]any, unsupported map[string]struct{}) (map[string]any, bool, bool) {
	ret := make(map[string]any, len(card))
	for key, value := range card {
		ret[key] = value
	}

	changed := false
	cardType := strings.ToLower(strings.TrimSpace(stringValue(card["type"])))

	if entity, ok := card["entity"].(string); ok && isUnsupportedMetricEntityReference(entity, unsupported) {
		return ret, false, true
	}

	if entitiesMap, ok := card["entities"].(map[string]any); ok {
		filtered := make(map[string]any, len(entitiesMap))
		for key, value := range entitiesMap {
			entity, isString := value.(string)
			if isString && isUnsupportedMetricEntityReference(entity, unsupported) {
				changed = true
				continue
			}
			filtered[key] = value
		}
		ret["entities"] = filtered
		if len(filtered) == 0 && cardType != "heading" {
			return ret, false, true
		}
	}

	if entitiesList, ok := card["entities"].([]any); ok {
		filteredEntities := make([]any, 0, len(entitiesList))
		for _, entry := range entitiesList {
			if isUnsupportedMetricEntityListEntry(entry, unsupported) {
				changed = true
				continue
			}
			filteredEntities = append(filteredEntities, entry)
		}
		ret["entities"] = filteredEntities
		if len(filteredEntities) == 0 && cardType != "heading" {
			return ret, false, true
		}
	}

	if nestedCards, ok := card["cards"].([]any); ok {
		filteredNested, nestedChanged := pruneUnsupportedMetricsFromCards(nestedCards, unsupported)
		if nestedChanged {
			changed = true
		}
		ret["cards"] = filteredNested
		if len(filteredNested) == 0 && cardType != "heading" {
			return ret, false, true
		}
	}

	return ret, true, changed
}

func hasNonHeadingCards(cards []any) bool {
	for _, rawCard := range cards {
		cardMap, ok := rawCard.(map[string]any)
		if !ok {
			return true
		}
		cardType := strings.ToLower(strings.TrimSpace(stringValue(cardMap["type"])))
		if cardType != "heading" {
			return true
		}
	}
	return false
}

func isUnsupportedMetricEntityListEntry(entry any, unsupported map[string]struct{}) bool {
	switch typed := entry.(type) {
	case map[string]any:
		entity, ok := typed["entity"].(string)
		return ok && isUnsupportedMetricEntityReference(entity, unsupported)
	case string:
		return isUnsupportedMetricEntityReference(typed, unsupported)
	default:
		return false
	}
}

func isUnsupportedMetricEntityReference(entityID string, unsupported map[string]struct{}) bool {
	if len(unsupported) == 0 {
		return false
	}

	candidate := strings.ToLower(strings.TrimSpace(entityID))
	if candidate == "" || !strings.Contains(candidate, "gosungrow_virtual_") {
		return false
	}

	for metric := range unsupported {
		if strings.HasSuffix(candidate, "_"+metric) {
			return true
		}
	}
	return false
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
