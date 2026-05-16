package cmd

import (
	"fmt"
	"math"
	"sort"
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

type dashboardEntityRemap struct {
	From   string
	To     string
	Metric string
	Source string
}

type dashboardUnresolvedEntityRef struct {
	Entity string
	Metric string
	Reason string
}

type dashboardMetricCandidate struct {
	Entity string
	Metric string
	Score  int
	State  string
	Unit   string
	Source string
	Reason string
}

type dashboardMetricTrace struct {
	Placeholder string
	Metric      string
	TargetPsKey string
	Resolved    string
	Source      string
	Candidates  []dashboardMetricCandidate
}

type dashboardRemapReport struct {
	TotalRefs  int
	Remapped   []dashboardEntityRemap
	Unresolved []dashboardUnresolvedEntityRef
	Traces     []dashboardMetricTrace
}

const (
	dashboardMetricKindPower   = "power"
	dashboardMetricKindEnergy  = "energy"
	dashboardMetricKindPercent = "percent"
)

func remapDashboardEntities(config map[string]any, targets []haDashboardTarget, states []haState) map[string]any {
	remapped, _ := remapDashboardEntitiesWithReport(config, targets, states)
	return remapped
}

func remapDashboardEntitiesWithReport(config map[string]any, targets []haDashboardTarget, states []haState) (map[string]any, dashboardRemapReport) {
	report := dashboardRemapReport{}
	if len(config) == 0 || len(targets) == 0 {
		return config, report
	}

	refs := collectLegacyDashboardEntityRefs(config, targets)
	report.TotalRefs = len(refs)
	if len(refs) == 0 {
		return config, report
	}
	if len(states) == 0 {
		for _, ref := range refs {
			report.Unresolved = append(report.Unresolved, dashboardUnresolvedEntityRef{
				Entity: ref.Entity,
				Metric: ref.Metric,
				Reason: "no Home Assistant states were available",
			})
		}
		return config, report
	}

	stateByID := dashboardStateByEntityID(states)

	replacements := make(map[string]string)
	singleTarget := len(targets) == 1
	for _, ref := range refs {
		resolved, trace := resolveDashboardEntityRefWithTrace(ref, states, stateByID, singleTarget)
		report.Traces = append(report.Traces, trace)
		if resolved == "" {
			report.Unresolved = append(report.Unresolved, dashboardUnresolvedEntityRef{
				Entity: ref.Entity,
				Metric: ref.Metric,
				Reason: dashboardResolveFailureReason(ref, states, singleTarget),
			})
			continue
		}
		if strings.EqualFold(resolved, ref.Entity) {
			continue
		}
		replacements[ref.Entity] = resolved
		report.Remapped = append(report.Remapped, dashboardEntityRemap{
			From:   ref.Entity,
			To:     resolved,
			Metric: ref.Metric,
			Source: dashboardMetricSourceCategory(ref.Target, ref.Metric, resolved),
		})
	}
	if len(replacements) == 0 {
		return config, report
	}

	remapped, ok := replaceDashboardEntityStrings(config, replacements).(map[string]any)
	if !ok {
		return config, report
	}
	return remapped, report
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
	resolved, _ := resolveDashboardEntityRefWithTrace(ref, states, stateByID, singleTarget)
	return resolved
}

func resolveDashboardEntityRefWithTrace(ref dashboardEntityRef, states []haState, stateByID map[string]haState, singleTarget bool) (string, dashboardMetricTrace) {
	resolved, trace := resolveDashboardMetricEntityWithTrace(ref.Target, ref.Metric, ref.Entity, states, stateByID, singleTarget)
	return resolved, trace
}

func resolveDashboardMetricEntity(target haDashboardTarget, metric string, states []haState, stateByID map[string]haState, singleTarget bool) string {
	resolved, _ := resolveDashboardMetricEntityWithTrace(target, metric, "", states, stateByID, singleTarget)
	return resolved
}

func resolveDashboardMetricEntityWithTrace(target haDashboardTarget, metric string, placeholder string, states []haState, stateByID map[string]haState, singleTarget bool) (string, dashboardMetricTrace) {
	if stateByID == nil {
		stateByID = dashboardStateByEntityID(states)
	}

	psKey := strings.ToLower(strings.TrimSpace(target.PsKey))
	psID := strings.ToLower(strings.TrimSpace(target.PsID))
	metric = strings.ToLower(strings.TrimSpace(metric))
	profile := dashboardMetricProfileFor(metric)
	trace := dashboardMetricTrace{
		Placeholder: placeholder,
		Metric:      metric,
		TargetPsKey: target.PsKey,
		Candidates:  make([]dashboardMetricCandidate, 0),
	}

	legacy := []string{
		"sensor.gosungrow_virtual_" + psKey + "_" + metric,
		"sensor.gosungrow_virtual_" + psID + "_" + metric,
	}
	for _, candidate := range legacy {
		state, ok := stateByID[candidate]
		if ok && dashboardStateMatchesMetricKind(state, profile) {
			trace.Resolved = candidate
			trace.Source = dashboardMetricSourceCategory(target, metric, candidate)
			trace.Candidates = appendDashboardMetricCandidate(trace.Candidates, dashboardMetricCandidate{
				Entity: candidate,
				Metric: metric,
				Score:  9999,
				State:  state.State,
				Unit:   dashboardStateUnit(state),
				Source: trace.Source,
				Reason: "exact dashboard virtual entity exists",
			}, 5)
			return candidate, trace
		}
	}

	bestCandidate := ""
	bestScore := -1
	for _, state := range states {
		candidate, score, reason, ok := dashboardScoreMetricCandidate(target, metric, profile, state, singleTarget)
		if candidate == "" {
			if rejected, rejectedOK := dashboardRejectedMetricCandidate(target, metric, profile, state, singleTarget); rejectedOK {
				trace.Candidates = appendDashboardMetricCandidate(trace.Candidates, rejected, 5)
			}
			continue
		}
		trace.Candidates = appendDashboardMetricCandidate(trace.Candidates, dashboardMetricCandidate{
			Entity: candidate,
			Metric: metric,
			Score:  score,
			State:  state.State,
			Unit:   dashboardStateUnit(state),
			Source: dashboardMetricSourceCategory(target, metric, candidate),
			Reason: reason,
		}, 5)
		if !ok {
			continue
		}

		if bestCandidate == "" || score > bestScore || (score == bestScore && len(candidate) < len(bestCandidate)) {
			bestCandidate = candidate
			bestScore = score
		}
	}

	trace.Resolved = bestCandidate
	trace.Source = dashboardMetricSourceCategory(target, metric, bestCandidate)
	return bestCandidate, trace
}

func dashboardRejectedMetricCandidate(target haDashboardTarget, metric string, profile dashboardMetricProfile, state haState, singleTarget bool) (dashboardMetricCandidate, bool) {
	candidate := strings.ToLower(strings.TrimSpace(state.EntityID))
	if !strings.HasPrefix(candidate, "sensor.") || !strings.Contains(candidate, "gosungrow") {
		return dashboardMetricCandidate{}, false
	}

	suffixScore, hasSuffixMatch := dashboardMetricSuffixScore(candidate, metric, profile)
	tokenScore, hasTokenMatch := dashboardMetricTokenScore(candidate, profile)
	if !hasSuffixMatch && !hasTokenMatch {
		return dashboardMetricCandidate{}, false
	}

	if _, ok := dashboardEntityPlantAffinity(candidate, target, singleTarget); !ok {
		return dashboardMetricCandidate{
			Entity: candidate,
			Metric: metric,
			Score:  suffixScore + tokenScore,
			State:  state.State,
			Unit:   dashboardStateUnit(state),
			Source: "wrong-target",
			Reason: "wrong target affinity",
		}, true
	}

	if reason := dashboardMetricStateRejectionReason(state, profile); reason != "" {
		return dashboardMetricCandidate{
			Entity: candidate,
			Metric: metric,
			Score:  suffixScore + tokenScore,
			State:  state.State,
			Unit:   dashboardStateUnit(state),
			Source: dashboardMetricSourceCategory(target, metric, candidate),
			Reason: reason,
		}, true
	}

	return dashboardMetricCandidate{}, false
}

func dashboardScoreMetricCandidate(target haDashboardTarget, metric string, profile dashboardMetricProfile, state haState, singleTarget bool) (string, int, string, bool) {
	candidate := strings.ToLower(strings.TrimSpace(state.EntityID))
	if !strings.HasPrefix(candidate, "sensor.") || !strings.Contains(candidate, "gosungrow") {
		return "", 0, "", false
	}

	affinityScore, ok := dashboardEntityPlantAffinity(candidate, target, singleTarget)
	if !ok {
		return "", 0, "", false
	}

	suffixScore, hasSuffixMatch := dashboardMetricSuffixScore(candidate, metric, profile)
	tokenScore, hasTokenMatch := dashboardMetricTokenScore(candidate, profile)
	if !hasSuffixMatch && !hasTokenMatch {
		return "", 0, "", false
	}

	stateScore, stateOK := dashboardMetricStateScore(state, profile)
	forbiddenPenalty := 0
	if hasTokenMatch {
		for _, forbidden := range profile.ForbiddenTokens {
			if dashboardTokenExists(candidate, forbidden) {
				forbiddenPenalty += 28
			}
		}
	}

	sourcePreferenceScore := dashboardMetricSourcePreferenceScore(metric, candidate)
	score := affinityScore + suffixScore + tokenScore + stateScore + sourcePreferenceScore - forbiddenPenalty
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

	if !stateOK {
		return candidate, score, dashboardMetricStateRejectionReason(state, profile), false
	}
	if forbiddenPenalty > 0 {
		return candidate, score, "usable but penalized by forbidden token", true
	}
	return candidate, score, "usable candidate", true
}

func appendDashboardMetricCandidate(values []dashboardMetricCandidate, value dashboardMetricCandidate, limit int) []dashboardMetricCandidate {
	if value.Entity == "" {
		return values
	}
	for index, entry := range values {
		if entry.Entity == value.Entity {
			if value.Score > entry.Score {
				values[index] = value
			}
			return values
		}
	}
	values = append(values, value)
	sort.SliceStable(values, func(i, j int) bool {
		if values[i].Score == values[j].Score {
			return len(values[i].Entity) < len(values[j].Entity)
		}
		return values[i].Score > values[j].Score
	})
	if len(values) > limit {
		return values[:limit]
	}
	return values
}

func dashboardMetricSourceCategory(target haDashboardTarget, metric string, candidate string) string {
	candidate = strings.ToLower(strings.TrimSpace(candidate))
	metric = strings.ToLower(strings.TrimSpace(metric))
	if candidate == "" {
		return "unresolved"
	}

	psKey := strings.ToLower(strings.TrimSpace(target.PsKey))
	psID := strings.ToLower(strings.TrimSpace(target.PsID))
	if psKey != "" && candidate == "sensor.gosungrow_virtual_"+psKey+"_"+metric {
		return "exact-virtual"
	}
	if psID != "" && candidate == "sensor.gosungrow_virtual_"+psID+"_"+metric {
		return "exact-virtual"
	}

	switch {
	case strings.Contains(candidate, "_p8018") || strings.Contains(candidate, "meter"):
		return "meter-level"
	case dashboardCandidateHasInverterContext(candidate) || strings.Contains(candidate, "inverter"):
		return "inverter-level"
	case strings.Contains(candidate, "plant") || strings.Contains(candidate, "_p83"):
		return "plant-level"
	case strings.Contains(candidate, "_virtual_"):
		return "legacy-virtual"
	case strings.Contains(candidate, "gosungrow"):
		return "legacy-alias"
	default:
		return "unknown"
	}
}

func dashboardMetricSourcePreferenceScore(metric string, candidate string) int {
	metric = strings.ToLower(strings.TrimSpace(metric))
	candidate = strings.ToLower(strings.TrimSpace(candidate))
	switch metric {
	case "pv_power":
		switch {
		case strings.HasSuffix(candidate, "_p24") && dashboardCandidateHasInverterContext(candidate):
			return 44
		case strings.Contains(candidate, "_pv_power") || strings.Contains(candidate, "_solar_power") || strings.Contains(candidate, "_total_dc_power"):
			return 48
		case strings.Contains(candidate, "_p83076") || strings.Contains(candidate, "_p83033") || strings.Contains(candidate, "_p83002"):
			return 18
		}
	case "grid_power":
		switch {
		case strings.Contains(candidate, "_p8018") || strings.Contains(candidate, "_p83032") || strings.Contains(candidate, "_meter_active_power") || strings.Contains(candidate, "_meter_ac_power"):
			return 42
		case strings.Contains(candidate, "_p83549") || strings.Contains(candidate, "_grid_active_power"):
			return 18
		}
	case "p13112":
		switch {
		case strings.HasSuffix(candidate, "_p1") && dashboardCandidateHasInverterContext(candidate):
			return 48
		case strings.Contains(candidate, "_p83009") || strings.Contains(candidate, "_yield_today") || strings.Contains(candidate, "_today_yield"):
			return 48
		case strings.Contains(candidate, "_p83022") || strings.Contains(candidate, "_daily_yield_of_plant"):
			return 18
		case strings.Contains(candidate, "_p83018") || strings.Contains(candidate, "_theoretical"):
			return -60
		}
	}
	return 0
}

func dashboardResolveFailureReason(ref dashboardEntityRef, states []haState, singleTarget bool) string {
	if len(states) == 0 {
		return "no Home Assistant states were available"
	}

	profile := dashboardMetricProfileFor(ref.Metric)
	gosungrowStates := 0
	plantStates := 0
	nameCandidates := 0
	unusableCandidates := 0
	unitRejectedCandidates := 0

	for _, state := range states {
		candidate := strings.ToLower(strings.TrimSpace(state.EntityID))
		if !strings.HasPrefix(candidate, "sensor.") || !strings.Contains(candidate, "gosungrow") {
			continue
		}
		gosungrowStates++

		if _, ok := dashboardEntityPlantAffinity(candidate, ref.Target, singleTarget); !ok {
			continue
		}
		plantStates++

		_, hasSuffixMatch := dashboardMetricSuffixScore(candidate, strings.ToLower(strings.TrimSpace(ref.Metric)), profile)
		_, hasTokenMatch := dashboardMetricTokenScore(candidate, profile)
		if !hasSuffixMatch && !hasTokenMatch {
			continue
		}
		nameCandidates++

		reason := dashboardMetricStateRejectionReason(state, profile)
		if reason == "" {
			continue
		}
		if strings.Contains(reason, "unit") {
			unitRejectedCandidates++
		} else {
			unusableCandidates++
		}
	}

	switch {
	case gosungrowStates == 0:
		return "no GoSungrow sensor entities were found in Home Assistant states"
	case plantStates == 0:
		return fmt.Sprintf("no GoSungrow entities matched ps_id %q or ps_key %q", ref.Target.PsID, ref.Target.PsKey)
	case nameCandidates == 0:
		return fmt.Sprintf("no candidate entity matched metric %q among %d target states", ref.Metric, plantStates)
	case unitRejectedCandidates > 0:
		return fmt.Sprintf("matching candidates existed (%d), but none had a compatible %s unit", nameCandidates, profile.Kind)
	case unusableCandidates > 0:
		return fmt.Sprintf("matching candidates existed (%d), but none had a usable numeric state", nameCandidates)
	default:
		return fmt.Sprintf("no usable candidate entity matched metric %q (%d name candidates, %d unit rejections, %d numeric-state rejections)", ref.Metric, nameCandidates, unitRejectedCandidates, unusableCandidates)
	}
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
	return dashboardMetricStateRejectionReason(state, profile) == ""
}

func dashboardMetricStateRejectionReason(state haState, profile dashboardMetricProfile) string {
	kind := strings.TrimSpace(profile.Kind)
	if kind == "" {
		return ""
	}
	if !dashboardStateHasUsableNumericValue(state) {
		return "state is unavailable or non-numeric"
	}

	unit := dashboardStateUnit(state)
	if unit == "" {
		return ""
	}

	switch kind {
	case dashboardMetricKindPower:
		if !isDashboardPowerUnit(unit) {
			return fmt.Sprintf("unit %q is not compatible with power", unit)
		}
	case dashboardMetricKindEnergy:
		if !isDashboardEnergyUnit(unit) {
			return fmt.Sprintf("unit %q is not compatible with energy", unit)
		}
	case dashboardMetricKindPercent:
		if strings.TrimSpace(unit) != "%" {
			return fmt.Sprintf("unit %q is not compatible with percent", unit)
		}
	}

	return ""
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

func dashboardStateByEntityID(states []haState) map[string]haState {
	stateByID := make(map[string]haState, len(states))
	for _, state := range states {
		entityID := strings.ToLower(strings.TrimSpace(state.EntityID))
		if entityID == "" {
			continue
		}
		stateByID[entityID] = state
	}
	return stateByID
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

func dashboardCandidateHasInverterContext(candidate string) bool {
	candidate = strings.ToLower(strings.TrimSpace(candidate))
	if strings.Contains(candidate, "inverter") {
		return true
	}

	const virtualPrefix = "sensor.gosungrow_virtual_"
	if !strings.HasPrefix(candidate, virtualPrefix) {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(candidate, virtualPrefix), "_")
	return len(parts) >= 5 && parts[1] == "1"
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
		if ((metric == "p13112" && alias == "p1") || (metric == "pv_power" && alias == "p24")) && !dashboardCandidateHasInverterContext(candidate) {
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
			Aliases:     []string{"pv_power", "pv_power_active", "solar_power", "p13003", "p24", "p83076", "p83076_map", "p83033", "p83002", "total_dc_power", "dc_power", "plant_power", "inverter_ac_power", "total_active_power", "active_power"},
			TokenGroups: [][]string{{"pv", "solar"}, {"power"}},
			Kind:        dashboardMetricKindPower,
		}
	case "load_power":
		return dashboardMetricProfile{
			Aliases:     []string{"load_power", "load_power_active", "p13119", "p83106", "p83106_map", "house_power", "consumption_power", "use_power", "total_load_power", "total_active_power", "active_power"},
			TokenGroups: [][]string{{"load", "home", "house", "consumption", "use"}, {"power"}},
			Kind:        dashboardMetricKindPower,
		}
	case "grid_power":
		return dashboardMetricProfile{
			Aliases:         []string{"grid_power", "grid_power_active", "net_grid_power", "p13149", "p13121", "p8018", "p83032", "p83549", "meter_active_power", "meter_ac_power", "grid_active_power", "active_power", "total_active_power", "import_power", "export_power"},
			TokenGroups:     [][]string{{"grid", "meter", "import", "export", "feed", "purchased"}, {"power"}},
			ForbiddenTokens: []string{"battery"},
			Kind:            dashboardMetricKindPower,
		}
	case "battery_power":
		return dashboardMetricProfile{
			Aliases:     []string{"battery_power", "battery_power_active", "es_power", "p13126", "p13150", "p83081", "p83081_map", "p83128", "p83128_map", "battery_charge_power", "battery_discharge_power", "charge_power", "discharge_power", "total_active_power_of_optical_storage"},
			TokenGroups: [][]string{{"battery", "soc", "es", "storage", "optical"}, {"power"}},
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
			Aliases:     []string{"p13112", "p83009", "yield_today", "today_yield", "daily_yield_by_inverter", "p1", "p83022", "p83022y", "pv_daily_energy", "daily_pv_yield", "daily_pv_energy", "pv_yield", "daily_yield_of_plant"},
			TokenGroups: [][]string{{"pv", "solar"}, {"energy", "yield", "production"}},
			Kind:        dashboardMetricKindEnergy,
		}
	case "p13116":
		return dashboardMetricProfile{
			Aliases:     []string{"p13116", "p83097", "p83097_map", "pv_to_load_energy", "pv_consumption_energy", "daily_load_energy_consumption_from_pv"},
			TokenGroups: [][]string{{"pv", "solar"}, {"load", "home", "house", "consumption", "use"}, {"energy"}},
			Kind:        dashboardMetricKindEnergy,
		}
	case "p13174":
		return dashboardMetricProfile{
			Aliases:     []string{"p13174", "p83120", "p83120_map", "p83088", "p83088_map", "pv_to_battery_energy", "battery_charge_energy", "battery_charging_energy_from_pv", "daily_battery_charging_energy_from_pv", "energy_battery_charge", "es_energy"},
			TokenGroups: [][]string{{"battery", "es"}, {"energy"}, {"charge", "charging"}},
			Kind:        dashboardMetricKindEnergy,
		}
	case "p13173":
		return dashboardMetricProfile{
			Aliases:     []string{"p13173", "p83119", "p83119_map", "pv_to_grid_energy", "feed_in_energy", "export_energy", "energy_feed_in"},
			TokenGroups: [][]string{{"grid", "export", "feed"}, {"energy"}},
			Kind:        dashboardMetricKindEnergy,
		}
	case "p13147":
		return dashboardMetricProfile{
			Aliases:     []string{"p13147", "p83102", "p83102_map", "grid_to_load_energy", "grid_import_energy", "purchased_energy", "energy_purchased"},
			TokenGroups: [][]string{{"grid", "import", "purchased"}, {"energy"}},
			Kind:        dashboardMetricKindEnergy,
		}
	case "p13141":
		return dashboardMetricProfile{
			Aliases:     []string{"p13141", "p83129", "p83252", "battery_soc", "battery_level", "battery_charge_percent", "soc"},
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
