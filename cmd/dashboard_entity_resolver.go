package cmd

import (
	"math"
	"strconv"
	"strings"
)

type dashboardEntityRef struct {
	Entity string
	Target haDashboardTarget
	Metric string
}

type dashboardMetricProfile struct {
	Aliases         []string
	TokenGroups     [][]string
	ForbiddenTokens []string
	Kind            string
}

const (
	dashboardMetricKindPower   = "power"
	dashboardMetricKindEnergy  = "energy"
	dashboardMetricKindPercent = "percent"
)

func remapDashboardEntities(config map[string]any, targets []haDashboardTarget, states []haState) map[string]any {
	if len(config) == 0 || len(targets) == 0 || len(states) == 0 {
		return config
	}

	refs := collectLegacyDashboardEntityRefs(config, targets)
	if len(refs) == 0 {
		return config
	}

	stateByID := make(map[string]haState, len(states))
	for _, state := range states {
		entityID := strings.ToLower(strings.TrimSpace(state.EntityID))
		if entityID == "" {
			continue
		}
		stateByID[entityID] = state
	}

	replacements := make(map[string]string)
	singleTarget := len(targets) == 1
	for _, ref := range refs {
		resolved := resolveDashboardEntityRef(ref, states, stateByID, singleTarget)
		if resolved == "" || strings.EqualFold(resolved, ref.Entity) {
			continue
		}
		replacements[ref.Entity] = resolved
	}
	if len(replacements) == 0 {
		return config
	}

	remapped, ok := replaceDashboardEntityStrings(config, replacements).(map[string]any)
	if !ok {
		return config
	}
	return remapped
}

func collectLegacyDashboardEntityRefs(value any, targets []haDashboardTarget) []dashboardEntityRef {
	refs := make([]dashboardEntityRef, 0)
	seen := make(map[string]struct{})

	var walk func(any)
	walk = func(current any) {
		switch typed := current.(type) {
		case map[string]any:
			for _, entry := range typed {
				walk(entry)
			}
		case []any:
			for _, entry := range typed {
				walk(entry)
			}
		case string:
			ref, ok := parseLegacyDashboardEntityRef(typed, targets)
			if !ok {
				return
			}
			if _, exists := seen[ref.Entity]; exists {
				return
			}
			seen[ref.Entity] = struct{}{}
			refs = append(refs, ref)
		}
	}

	walk(value)
	return refs
}

func parseLegacyDashboardEntityRef(entity string, targets []haDashboardTarget) (dashboardEntityRef, bool) {
	normalized := strings.ToLower(strings.TrimSpace(entity))
	for _, target := range targets {
		psKey := strings.ToLower(strings.TrimSpace(target.PsKey))
		psID := strings.ToLower(strings.TrimSpace(target.PsID))
		prefixes := []string{
			"sensor.gosungrow_virtual_" + psKey + "_",
			"sensor.gosungrow_virtual_" + psID + "_",
		}
		for _, prefix := range prefixes {
			if !strings.HasPrefix(normalized, prefix) {
				continue
			}
			metric := strings.TrimPrefix(normalized, prefix)
			if metric == "" {
				return dashboardEntityRef{}, false
			}
			return dashboardEntityRef{
				Entity: entity,
				Target: target,
				Metric: metric,
			}, true
		}
	}
	return dashboardEntityRef{}, false
}

func resolveDashboardEntityRef(ref dashboardEntityRef, states []haState, stateByID map[string]haState, singleTarget bool) string {
	psKey := strings.ToLower(strings.TrimSpace(ref.Target.PsKey))
	psID := strings.ToLower(strings.TrimSpace(ref.Target.PsID))
	metric := strings.ToLower(strings.TrimSpace(ref.Metric))
	profile := dashboardMetricProfileFor(metric)

	legacy := []string{
		"sensor.gosungrow_virtual_" + psKey + "_" + metric,
		"sensor.gosungrow_virtual_" + psID + "_" + metric,
	}
	for _, candidate := range legacy {
		state, ok := stateByID[candidate]
		if ok && dashboardStateMatchesMetricKind(state, profile) {
			return candidate
		}
	}

	bestCandidate := ""
	bestScore := -1
	for _, state := range states {
		candidate := strings.ToLower(strings.TrimSpace(state.EntityID))
		if !strings.HasPrefix(candidate, "sensor.") {
			continue
		}
		if !strings.Contains(candidate, "gosungrow") {
			continue
		}

		affinityScore, ok := dashboardEntityPlantAffinity(candidate, ref.Target, singleTarget)
		if !ok {
			continue
		}

		suffixScore, hasSuffixMatch := dashboardMetricSuffixScore(candidate, metric, profile)
		tokenScore, hasTokenMatch := dashboardMetricTokenScore(candidate, profile)
		if !hasSuffixMatch && !hasTokenMatch {
			continue
		}
		stateScore, ok := dashboardMetricStateScore(state, profile)
		if !ok {
			continue
		}

		forbiddenPenalty := 0
		if hasTokenMatch {
			for _, forbidden := range profile.ForbiddenTokens {
				if dashboardTokenExists(candidate, forbidden) {
					forbiddenPenalty += 28
				}
			}
		}

		score := affinityScore + suffixScore + tokenScore + stateScore - forbiddenPenalty
		if strings.HasSuffix(candidate, "_active") {
			score -= 4
		}
		if strings.Contains(candidate, "_virtual_") {
			score += 16
		}
		if strings.Contains(candidate, "_information_") {
			score += 6
		}
		if strings.HasPrefix(candidate, "sensor.gosungrow_") {
			score += 8
		}

		if !hasSuffixMatch && hasTokenMatch {
			score -= 24
		}

		if bestCandidate == "" || score > bestScore || (score == bestScore && len(candidate) < len(bestCandidate)) {
			bestCandidate = candidate
			bestScore = score
		}
	}

	return bestCandidate
}

func dashboardMetricStateScore(state haState, profile dashboardMetricProfile) (int, bool) {
	if !dashboardStateMatchesMetricKind(state, profile) {
		return 0, false
	}
	if strings.TrimSpace(profile.Kind) == "" {
		return 0, true
	}
	unit := dashboardStateUnit(state)
	if unit == "" {
		return -18, true
	}
	return 28, true
}

func dashboardStateMatchesMetricKind(state haState, profile dashboardMetricProfile) bool {
	kind := strings.TrimSpace(profile.Kind)
	if kind == "" {
		return true
	}
	if !dashboardStateHasUsableNumericValue(state) {
		return false
	}

	unit := dashboardStateUnit(state)
	if unit == "" {
		return true
	}

	switch kind {
	case dashboardMetricKindPower:
		return isDashboardPowerUnit(unit)
	case dashboardMetricKindEnergy:
		return isDashboardEnergyUnit(unit)
	case dashboardMetricKindPercent:
		return strings.TrimSpace(unit) == "%"
	default:
		return true
	}
}

func dashboardStateHasUsableNumericValue(state haState) bool {
	text := strings.ToLower(strings.TrimSpace(state.State))
	switch text {
	case "", "unknown", "unavailable", "none", "null":
		return false
	}
	value, err := strconv.ParseFloat(text, 64)
	return err == nil && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func dashboardStateUnit(state haState) string {
	if state.Attributes == nil {
		return ""
	}
	unit, _ := state.Attributes["unit_of_measurement"].(string)
	return strings.TrimSpace(unit)
}

func isDashboardPowerUnit(unit string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(unit), " ", ""))
	switch normalized {
	case "w", "kw", "mw":
		return true
	default:
		return false
	}
}

func isDashboardEnergyUnit(unit string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(unit), " ", ""))
	switch normalized {
	case "wh", "kwh", "mwh":
		return true
	default:
		return false
	}
}

func dashboardEntityPlantAffinity(candidate string, target haDashboardTarget, singleTarget bool) (int, bool) {
	psKey := strings.ToLower(strings.TrimSpace(target.PsKey))
	psID := strings.ToLower(strings.TrimSpace(target.PsID))

	score := 0
	if psKey != "" && dashboardEntityContainsIdentifier(candidate, psKey) {
		score += 120
	}
	if psID != "" && dashboardEntityContainsIdentifier(candidate, psID) {
		score += 90
	}
	if score > 0 {
		return score, true
	}
	if !singleTarget {
		return 0, false
	}
	return 10, true
}

func dashboardEntityContainsIdentifier(candidate string, identifier string) bool {
	if identifier == "" {
		return false
	}
	needle := "_" + identifier + "_"
	if strings.Contains(candidate, needle) {
		return true
	}
	if strings.HasPrefix(candidate, "sensor.gosungrow_"+identifier+"_") {
		return true
	}
	if strings.HasSuffix(candidate, "_"+identifier) {
		return true
	}
	return false
}

func dashboardMetricSuffixScore(candidate string, metric string, profile dashboardMetricProfile) (int, bool) {
	if metric != "" && strings.HasSuffix(candidate, "_"+metric) {
		return 240, true
	}

	for index, alias := range profile.Aliases {
		alias = strings.ToLower(strings.TrimSpace(alias))
		if alias == "" {
			continue
		}
		if strings.HasSuffix(candidate, "_"+alias) {
			return 214 - (index * 3), true
		}
	}

	return 0, false
}

func dashboardMetricTokenScore(candidate string, profile dashboardMetricProfile) (int, bool) {
	if len(profile.TokenGroups) == 0 {
		return 0, false
	}

	tokens := dashboardTokenSet(candidate)
	score := 0
	for _, group := range profile.TokenGroups {
		matched := false
		for _, token := range group {
			token = strings.ToLower(strings.TrimSpace(token))
			if token == "" {
				continue
			}
			if _, ok := tokens[token]; ok {
				matched = true
				score += 30
				break
			}
		}
		if !matched {
			return 0, false
		}
	}

	return score, true
}

func dashboardTokenExists(candidate string, token string) bool {
	token = strings.ToLower(strings.TrimSpace(token))
	if token == "" {
		return false
	}
	if strings.Contains(candidate, "_"+token+"_") || strings.HasSuffix(candidate, "_"+token) || strings.HasPrefix(candidate, token+"_") {
		return true
	}
	_, ok := dashboardTokenSet(candidate)[token]
	return ok
}

func dashboardTokenSet(value string) map[string]struct{} {
	tokens := strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})

	ret := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		ret[token] = struct{}{}
	}
	return ret
}

func dashboardMetricProfileFor(metric string) dashboardMetricProfile {
	switch strings.ToLower(strings.TrimSpace(metric)) {
	case "pv_power":
		return dashboardMetricProfile{
			Aliases:     []string{"pv_power", "pv_power_active", "solar_power", "p13003", "total_dc_power", "dc_power", "total_active_power", "active_power"},
			TokenGroups: [][]string{{"pv", "solar"}, {"power"}},
			Kind:        dashboardMetricKindPower,
		}
	case "load_power":
		return dashboardMetricProfile{
			Aliases:     []string{"load_power", "load_power_active", "p13119", "house_power", "consumption_power", "use_power", "total_load_power", "total_active_power", "active_power"},
			TokenGroups: [][]string{{"load", "home", "house", "consumption", "use"}, {"power"}},
			Kind:        dashboardMetricKindPower,
		}
	case "grid_power":
		return dashboardMetricProfile{
			Aliases:         []string{"grid_power", "grid_power_active", "net_grid_power", "p13149", "p13121", "active_power", "total_active_power", "import_power", "export_power"},
			TokenGroups:     [][]string{{"grid", "import", "export", "feed", "purchased"}, {"power"}},
			ForbiddenTokens: []string{"battery"},
			Kind:            dashboardMetricKindPower,
		}
	case "battery_power":
		return dashboardMetricProfile{
			Aliases:     []string{"battery_power", "battery_power_active", "es_power", "p13126", "p13150", "battery_charge_power", "battery_discharge_power", "charge_power", "discharge_power"},
			TokenGroups: [][]string{{"battery", "soc", "es"}, {"power"}},
			Kind:        dashboardMetricKindPower,
		}
	case "pv_to_load_power":
		return dashboardMetricProfile{
			Aliases:     []string{"pv_to_load_power", "pv_to_load_power_active", "load_from_pv_power", "pv_consumption_power"},
			TokenGroups: [][]string{{"pv", "solar"}, {"load", "home", "house", "consumption", "use"}, {"power"}},
			Kind:        dashboardMetricKindPower,
		}
	case "pv_to_battery_power":
		return dashboardMetricProfile{
			Aliases:     []string{"pv_to_battery_power", "pv_to_battery_power_active", "p13126", "battery_charge_power"},
			TokenGroups: [][]string{{"pv", "solar"}, {"battery", "es"}, {"power"}},
			Kind:        dashboardMetricKindPower,
		}
	case "pv_to_grid_power":
		return dashboardMetricProfile{
			Aliases:     []string{"pv_to_grid_power", "pv_to_grid_power_active", "p13121", "export_power", "feed_in_power"},
			TokenGroups: [][]string{{"pv", "solar"}, {"grid", "export", "feed"}, {"power"}},
			Kind:        dashboardMetricKindPower,
		}
	case "battery_to_load_power":
		return dashboardMetricProfile{
			Aliases:     []string{"battery_to_load_power", "battery_to_load_power_active", "p13150", "battery_discharge_power"},
			TokenGroups: [][]string{{"battery", "es"}, {"load", "home", "house", "consumption", "use"}, {"power"}},
			Kind:        dashboardMetricKindPower,
		}
	case "grid_to_load_power":
		return dashboardMetricProfile{
			Aliases:     []string{"grid_to_load_power", "grid_to_load_power_active", "p13149", "import_power", "purchased_power"},
			TokenGroups: [][]string{{"grid", "import", "purchased"}, {"load", "home", "house", "consumption", "use"}, {"power"}},
			Kind:        dashboardMetricKindPower,
		}
	case "p13112":
		return dashboardMetricProfile{
			Aliases:     []string{"p13112", "pv_daily_energy", "daily_pv_yield", "daily_pv_energy", "pv_yield"},
			TokenGroups: [][]string{{"pv", "solar"}, {"energy", "yield", "production"}},
			Kind:        dashboardMetricKindEnergy,
		}
	case "p13116":
		return dashboardMetricProfile{
			Aliases:     []string{"p13116", "pv_to_load_energy", "pv_consumption_energy"},
			TokenGroups: [][]string{{"pv", "solar"}, {"load", "home", "house", "consumption", "use"}, {"energy"}},
			Kind:        dashboardMetricKindEnergy,
		}
	case "p13174":
		return dashboardMetricProfile{
			Aliases:     []string{"p13174", "pv_to_battery_energy", "battery_charge_energy", "battery_charging_energy_from_pv", "daily_battery_charging_energy_from_pv"},
			TokenGroups: [][]string{{"battery", "es"}, {"energy"}, {"charge", "charging"}},
			Kind:        dashboardMetricKindEnergy,
		}
	case "p13173":
		return dashboardMetricProfile{
			Aliases:     []string{"p13173", "pv_to_grid_energy", "feed_in_energy", "export_energy"},
			TokenGroups: [][]string{{"grid", "export", "feed"}, {"energy"}},
			Kind:        dashboardMetricKindEnergy,
		}
	case "p13147":
		return dashboardMetricProfile{
			Aliases:     []string{"p13147", "grid_to_load_energy", "grid_import_energy", "purchased_energy"},
			TokenGroups: [][]string{{"grid", "import", "purchased"}, {"energy"}},
			Kind:        dashboardMetricKindEnergy,
		}
	case "p13141":
		return dashboardMetricProfile{
			Aliases:     []string{"p13141", "battery_soc", "battery_level", "battery_charge_percent", "soc"},
			TokenGroups: [][]string{{"battery", "soc"}, {"percent", "level", "charge", "soc"}},
			Kind:        dashboardMetricKindPercent,
		}
	default:
		return dashboardMetricProfile{
			Aliases: []string{metric},
		}
	}
}

func replaceDashboardEntityStrings(value any, replacements map[string]string) any {
	switch typed := value.(type) {
	case map[string]any:
		ret := make(map[string]any, len(typed))
		for key, entry := range typed {
			ret[key] = replaceDashboardEntityStrings(entry, replacements)
		}
		return ret
	case []any:
		ret := make([]any, 0, len(typed))
		for _, entry := range typed {
			ret = append(ret, replaceDashboardEntityStrings(entry, replacements))
		}
		return ret
	case string:
		if replacement, ok := replacements[typed]; ok {
			return replacement
		}
		return typed
	default:
		return value
	}
}
