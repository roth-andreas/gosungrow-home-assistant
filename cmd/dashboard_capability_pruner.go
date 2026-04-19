package cmd

import "strings"

func pruneDashboardForMissingBattery(config map[string]any, targets []haDashboardTarget, stateEntityIDs []string) map[string]any {
	if len(config) == 0 || len(targets) == 0 || len(stateEntityIDs) == 0 {
		return config
	}

	singleTarget := len(targets) == 1
	batteryByTarget := make(map[string]bool, len(targets))
	anyMissingBattery := false
	for _, target := range targets {
		psKey := strings.ToLower(strings.TrimSpace(target.PsKey))
		hasBattery := dashboardTargetHasBattery(target, stateEntityIDs, singleTarget)
		batteryByTarget[psKey] = hasBattery
		if !hasBattery {
			anyMissingBattery = true
		}
	}
	if !anyMissingBattery {
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
		hasBattery := batteryByTarget[strings.ToLower(strings.TrimSpace(target.PsKey))]
		if hasBattery {
			prunedViews = append(prunedViews, rawView)
			continue
		}

		prunedView, viewChanged := pruneBatteryFromView(viewMap)
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

func dashboardTargetHasBattery(target haDashboardTarget, stateEntityIDs []string, singleTarget bool) bool {
	for _, entityID := range stateEntityIDs {
		candidate := strings.ToLower(strings.TrimSpace(entityID))
		if !strings.HasPrefix(candidate, "sensor.") || !strings.Contains(candidate, "gosungrow") {
			continue
		}

		_, ok := dashboardEntityPlantAffinity(candidate, target, singleTarget)
		if !ok {
			continue
		}

		if isBatteryEntityReference(candidate) {
			return true
		}

		tokens := dashboardTokenSet(candidate)
		if _, hasStorage := tokens["es"]; hasStorage {
			if _, hasPower := tokens["power"]; hasPower {
				return true
			}
			if _, hasEnergy := tokens["energy"]; hasEnergy {
				return true
			}
		}
	}

	return false
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

func pruneBatteryFromView(view map[string]any) (map[string]any, bool) {
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

			prunedSection, keepSection, sectionChanged := pruneBatteryFromSection(sectionMap)
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
		prunedCards, cardsChanged := pruneBatteryFromCards(rawCards)
		if cardsChanged {
			changed = true
		}
		ret["cards"] = prunedCards
	}

	return ret, changed
}

func pruneBatteryFromSection(section map[string]any) (map[string]any, bool, bool) {
	ret := make(map[string]any, len(section))
	for key, value := range section {
		ret[key] = value
	}

	rawCards, ok := section["cards"].([]any)
	if !ok {
		return ret, true, false
	}

	prunedCards, changed := pruneBatteryFromCards(rawCards)
	ret["cards"] = prunedCards
	if !hasNonHeadingCards(prunedCards) {
		return ret, false, true
	}

	return ret, true, changed
}

func pruneBatteryFromCards(cards []any) ([]any, bool) {
	ret := make([]any, 0, len(cards))
	changed := false
	for _, rawCard := range cards {
		cardMap, ok := rawCard.(map[string]any)
		if !ok {
			ret = append(ret, rawCard)
			continue
		}

		prunedCard, keepCard, cardChanged := pruneBatteryFromCard(cardMap)
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

func pruneBatteryFromCard(card map[string]any) (map[string]any, bool, bool) {
	ret := make(map[string]any, len(card))
	for key, value := range card {
		ret[key] = value
	}

	changed := false
	cardType := strings.ToLower(strings.TrimSpace(stringValue(card["type"])))

	if entity, ok := card["entity"].(string); ok && isBatteryEntityReference(entity) {
		return ret, false, true
	}

	if entitiesMap, ok := card["entities"].(map[string]any); ok {
		filtered := make(map[string]any, len(entitiesMap))
		for key, value := range entitiesMap {
			entity, isString := value.(string)
			if isBatteryFlowEntityKey(key) || (isString && isBatteryEntityReference(entity)) {
				changed = true
				continue
			}
			filtered[key] = value
		}
		ret["entities"] = filtered
	}

	if entitiesList, ok := card["entities"].([]any); ok {
		filteredEntities := make([]any, 0, len(entitiesList))
		for _, entry := range entitiesList {
			if isBatteryEntityListEntry(entry) {
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
		filteredNested, nestedChanged := pruneBatteryFromCards(nestedCards)
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

func isBatteryEntityListEntry(entry any) bool {
	switch typed := entry.(type) {
	case map[string]any:
		entity, ok := typed["entity"].(string)
		return ok && isBatteryEntityReference(entity)
	case string:
		return isBatteryEntityReference(typed)
	default:
		return false
	}
}

func isBatteryFlowEntityKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "battery_power", "battery_soc", "pv_to_battery_power", "battery_to_load_power", "battery_to_home_power", "battery_to_heatpump_power":
		return true
	default:
		return false
	}
}

func isBatteryEntityReference(entityID string) bool {
	candidate := strings.ToLower(strings.TrimSpace(entityID))
	if candidate == "" {
		return false
	}

	if strings.Contains(candidate, "_battery_") ||
		strings.Contains(candidate, "battery_to_") ||
		strings.Contains(candidate, "_to_battery_") ||
		strings.HasSuffix(candidate, "_battery") ||
		strings.HasSuffix(candidate, "_soc") ||
		strings.Contains(candidate, "_soc_") ||
		strings.Contains(candidate, "_p13141") ||
		strings.Contains(candidate, "_p13142") ||
		strings.Contains(candidate, "_p13174") ||
		strings.Contains(candidate, "_p13150") ||
		strings.Contains(candidate, "_p13126") {
		return true
	}

	tokens := dashboardTokenSet(candidate)
	if _, ok := tokens["battery"]; ok {
		return true
	}
	if _, ok := tokens["soc"]; ok {
		return true
	}
	return false
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
