package cmd

import (
	"sort"
	"strings"
	"time"
	"unicode"
)

const dashboardSemanticMatcherVersion = 3

type dashboardSemanticSourceSpec struct {
	Identifier     string
	CanonicalPoint string
	Provenance     string
	Period         string
	Scope          string
	Score          int
}

type dashboardSemanticContract struct {
	metric          string
	aliases         []string
	requiredMeaning []string
	forbidden       []string
	allowedRoles    map[string]bool
	daily           bool
	sources         []dashboardSemanticSourceSpec
}

type dashboardSemanticMatch struct {
	Entity     string
	Score      int
	Reason     string
	Confident  bool
	Candidates []dashboardMetricCandidate
}

func dashboardSemanticContractFor(metric string) (dashboardSemanticContract, bool) {
	native := func(identifier, canonical, scope string, score int) dashboardSemanticSourceSpec {
		return dashboardSemanticSourceSpec{Identifier: identifier, CanonicalPoint: canonical, Provenance: "native", Period: "day", Scope: scope, Score: score}
	}
	verifiedAlias := func(identifier, canonical, scope string, score int) dashboardSemanticSourceSpec {
		return dashboardSemanticSourceSpec{Identifier: identifier, CanonicalPoint: canonical, Provenance: "verified_alias", Period: "day", Scope: scope, Score: score}
	}
	contracts := map[string]dashboardSemanticContract{
		"pv_power": {
			metric: "pv_power", aliases: []string{"plant_active_power", "inverter_active_power", "pv_active_power"},
			requiredMeaning: []string{"pv", "solar", "generation", "active_power"},
			forbidden:       []string{"phase_a", "phase_b", "phase_c"},
			allowedRoles:    map[string]bool{"inverter": true, "ess": true, "plant": true},
		},
		"p13112": {
			metric: "p13112", daily: true,
			sources: []dashboardSemanticSourceSpec{
				native("p13112", "p13112", "plant", 1000), native("p83022", "p13112", "plant", 980),
				native("p83009", "p13112", "inverter", 920), native("p1", "p13112", "inverter", 900),
				verifiedAlias("daily_generation", "p13112", "inverter", 880), verifiedAlias("generation_today", "p13112", "inverter", 870),
			},
			forbidden: []string{"p13122", "p13173", "feed_in", "feedin", "export", "import", "purchased", "theoretical"},
		},
		"p13116": {
			metric: "p13116", daily: true,
			sources:   []dashboardSemanticSourceSpec{native("p13116", "p13116", "plant", 1000), native("p83097", "p13116", "plant", 980)},
			forbidden: []string{"pv_consumption_energy", "pv_to_load_energy"},
		},
		"p13173": {
			metric: "p13173", daily: true,
			sources: []dashboardSemanticSourceSpec{
				native("p13173", "p13173", "plant", 1000), native("p83119", "p13173", "plant", 980),
				verifiedAlias("feed_in_energy_today", "p13173", "plant", 950), verifiedAlias("export_energy_today", "p13173", "plant", 940),
				verifiedAlias("pv_to_grid_energy", "p13173", "plant", 930),
			},
			forbidden: []string{"import", "purchased", "p83102"},
		},
		"p13147": {
			metric: "p13147", daily: true,
			sources: []dashboardSemanticSourceSpec{
				native("p13147", "p13147", "plant", 1000), native("p83102", "p13147", "plant", 980),
				verifiedAlias("energy_purchased_today", "p13147", "plant", 950), verifiedAlias("import_energy_today", "p13147", "plant", 940),
				verifiedAlias("grid_to_load_energy", "p13147", "plant", 930),
			},
			forbidden: []string{"feed_in", "feedin", "export", "p13173", "p83119"},
		},
		"p13199": {
			metric: "p13199", daily: true,
			sources: []dashboardSemanticSourceSpec{
				native("p13199", "p13199", "plant", 1000),
				verifiedAlias("daily_total_energy", "p13199", "plant", 960),
				verifiedAlias("total_daily_energy", "p13199", "plant", 950),
				verifiedAlias("total_load_energy", "p13199", "plant", 940),
				verifiedAlias("daily_load_consumption", "p13199", "plant", 930),
				verifiedAlias("daily_load_energy_consumption", "p13199", "plant", 920),
				verifiedAlias("daily_home_consumption", "p13199", "plant", 910),
			},
			forbidden: []string{"feed_in", "feedin", "export", "import", "purchased", "battery", "generation", "total_energy"},
		},
	}
	contract, ok := contracts[strings.ToLower(strings.TrimSpace(metric))]
	return contract, ok
}

func dashboardSemanticProfile(metric string, contract dashboardSemanticContract) dashboardMetricProfile {
	profile := dashboardMetricProfileFor(metric)
	profile.Aliases = append(append([]string{}, contract.aliases...), profile.Aliases...)
	return profile
}

func dashboardSemanticRecommendation(target haDashboardTarget, metric string, states []haState, singleTarget bool) dashboardSemanticMatch {
	contract, ok := dashboardSemanticContractFor(metric)
	if !ok {
		return dashboardSemanticMatch{}
	}
	if len(contract.sources) > 0 {
		return dashboardCanonicalRecommendation(target, metric, contract, states, singleTarget)
	}
	return dashboardLegacySemanticRecommendation(target, metric, contract, states, singleTarget)
}

func dashboardCanonicalRecommendation(target haDashboardTarget, metric string, contract dashboardSemanticContract, states []haState, singleTarget bool) dashboardSemanticMatch {
	profile := dashboardMetricProfileFor(metric)
	values := make([]dashboardMetricCandidate, 0)
	registryAvailable := dashboardRegistryMetadataAvailable(states)
	for _, state := range states {
		if dashboardMetricStateRejectionReason(state, profile) != "" || !dashboardSourceStateRecent(state, metric, time.Now()) || !dashboardSemanticStateMatchesTargetStable(target, state, singleTarget, registryAvailable) {
			continue
		}
		var source dashboardSemanticSourceSpec
		var ok bool
		if registryAvailable {
			source, ok = dashboardSemanticSource(state.RegistryUniqueID, contract.sources)
		} else {
			source, ok = dashboardSemanticSourceState(state, contract.sources)
		}
		if !ok {
			continue
		}
		role := dashboardCandidateDeviceRoleState(target, state)
		scope := source.Scope
		if scope == "" {
			scope = role
		}
		score := source.Score
		if scope == "plant" {
			score += 80
		} else if role == "inverter" {
			score += 30
		}
		compatibility := "compatible"
		reason := "Canonical Sungrow point with compatible daily semantics"
		if !registryAvailable {
			compatibility = "unverified"
			reason = "Canonical-looking entity; registry metadata unavailable, review before selecting"
		}
		values = append(values, dashboardMetricCandidate{
			Entity: strings.TrimSpace(state.EntityID), Metric: metric, Score: score, State: state.State,
			Unit: dashboardStateUnit(state), Source: "canonical", Reason: reason,
			PointID: source.CanonicalPoint, Provenance: source.Provenance, Period: source.Period, Scope: scope, Role: role, Compatibility: compatibility,
		})
	}
	dashboardSortSemanticCandidates(values)
	if len(values) == 0 {
		return dashboardSemanticMatch{}
	}
	confident := true
	if len(values) > 1 && values[0].Score-values[1].Score < 20 && !dashboardEquivalentDuplicate(values[0], values[1]) {
		confident = false
	}
	if metric == "p13112" && values[0].Scope == "inverter" && dashboardTargetInverterCount(target) != 1 {
		confident = false
	}
	if !registryAvailable {
		confident = false
	}
	return dashboardSemanticMatch{Entity: values[0].Entity, Score: values[0].Score, Confident: confident, Candidates: values, Reason: values[0].Reason}
}

func dashboardLegacySemanticRecommendation(target haDashboardTarget, metric string, contract dashboardSemanticContract, states []haState, singleTarget bool) dashboardSemanticMatch {
	profile := dashboardSemanticProfile(metric, contract)
	values := make([]dashboardMetricCandidate, 0)
	for _, state := range states {
		entity, score, reason, compatible := dashboardScoreMetricCandidate(target, metric, profile, state, singleTarget)
		if !compatible || entity == "" || !dashboardSemanticCandidateAllowed(target, contract, state) {
			continue
		}
		score += 300
		values = append(values, dashboardMetricCandidate{Entity: entity, Metric: metric, Score: score, State: state.State, Unit: dashboardStateUnit(state), Source: dashboardMetricSourceCategory(target, metric, entity), Reason: reason})
	}
	dashboardSortSemanticCandidates(values)
	if len(values) == 0 {
		return dashboardSemanticMatch{}
	}
	confident := len(values) == 1 || values[0].Score-values[1].Score >= 20 || dashboardEquivalentDuplicate(values[0], values[1])
	return dashboardSemanticMatch{Entity: values[0].Entity, Score: values[0].Score, Confident: confident, Candidates: values, Reason: "Semantic match: compatible device role, direction and measurement period"}
}

func dashboardSortSemanticCandidates(values []dashboardMetricCandidate) {
	sort.SliceStable(values, func(i, j int) bool {
		if values[i].Score == values[j].Score {
			return values[i].Entity < values[j].Entity
		}
		return values[i].Score > values[j].Score
	})
}

func dashboardSemanticCandidateAllowed(target haDashboardTarget, contract dashboardSemanticContract, state haState) bool {
	identity := dashboardSemanticStateIdentity(state)
	role := dashboardCandidateDeviceRoleState(target, state)
	if len(contract.allowedRoles) > 0 && !contract.allowedRoles[role] {
		return false
	}
	if dashboardSemanticHasForbidden(identity, contract.forbidden) {
		return false
	}
	meaning := false
	for _, required := range contract.requiredMeaning {
		if dashboardSemanticContains(identity, required) {
			meaning = true
			break
		}
	}
	if !meaning {
		return false
	}
	return !contract.daily || dashboardCandidateIsDaily(state)
}

func dashboardCandidateDeviceRoleState(target haDashboardTarget, state haState) string {
	for _, identity := range []string{state.RegistryUniqueID, state.EntityID} {
		if role := dashboardCandidateDeviceRole(target, strings.ToLower(strings.TrimSpace(identity))); role != "unknown" {
			return role
		}
	}
	return "unknown"
}

func dashboardCandidateDeviceRole(target haDashboardTarget, entity string) string {
	for _, device := range target.PlantDevices {
		if dashboardEntityContainsIdentifier(entity, strings.ToLower(strings.TrimSpace(device.PsKey))) {
			return dashboardDeviceRole(device.DeviceType)
		}
	}
	if dashboardEntityContainsIdentifier(entity, strings.ToLower(strings.TrimSpace(target.PsID))) {
		return "plant"
	}
	return "unknown"
}

func dashboardDeviceRole(deviceType int64) string {
	switch deviceType {
	case 1:
		return "inverter"
	case 7:
		return "meter"
	case 14:
		return "ess"
	case 11:
		return "plant"
	case 22:
		return "communication"
	default:
		return "unknown"
	}
}

func dashboardCandidateIsDaily(state haState) bool {
	identity := dashboardSemanticStateIdentity(state)
	for _, marker := range []string{"_today", "today_", "_daily", "daily_", "_p13112", "_p13116", "_p13147", "_p13173", "_p13199"} {
		if strings.Contains(identity, marker) {
			return true
		}
	}
	reset := strings.TrimSpace(stringValue(state.Attributes["last_reset"]))
	when, err := time.Parse(time.RFC3339, reset)
	if reset == "" || err != nil {
		return false
	}
	age := time.Since(when)
	return age >= -6*time.Hour && age <= 48*time.Hour
}

func dashboardSemanticStateIdentity(state haState) string {
	return dashboardSemanticNormalize(strings.TrimSpace(state.RegistryUniqueID) + " " + strings.TrimSpace(state.EntityID))
}

func dashboardSemanticNormalize(value string) string {
	var b strings.Builder
	lastSeparator := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastSeparator = false
		} else if !lastSeparator {
			b.WriteByte('_')
			lastSeparator = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func dashboardSemanticSource(identity string, sources []dashboardSemanticSourceSpec) (dashboardSemanticSourceSpec, bool) {
	for _, source := range sources {
		if dashboardSemanticTerminalIdentifier(identity, source.Identifier) {
			return source, true
		}
	}
	return dashboardSemanticSourceSpec{}, false
}

func dashboardSemanticSourceState(state haState, sources []dashboardSemanticSourceSpec) (dashboardSemanticSourceSpec, bool) {
	for _, identity := range []string{state.RegistryUniqueID, state.EntityID} {
		if source, ok := dashboardSemanticSource(identity, sources); ok {
			return source, true
		}
	}
	return dashboardSemanticSourceSpec{}, false
}

func dashboardSemanticTerminalIdentifier(identity, identifier string) bool {
	identity = dashboardSemanticNormalize(identity)
	identifier = dashboardSemanticNormalize(identifier)
	return identifier != "" && (strings.HasSuffix(identity, "_"+identifier) || strings.HasSuffix(identity, "_"+identifier+"_2") || identity == identifier)
}

func dashboardSemanticHasForbidden(identity string, forbidden []string) bool {
	for _, value := range forbidden {
		if dashboardSemanticContains(identity, value) {
			return true
		}
	}
	return false
}

func dashboardSemanticContains(entity, phrase string) bool {
	entity = "_" + dashboardSemanticNormalize(entity) + "_"
	phrase = dashboardSemanticNormalize(phrase)
	return phrase != "" && strings.Contains(entity, "_"+phrase+"_")
}

func dashboardSemanticStateMatchesTarget(target haDashboardTarget, state haState, singleTarget bool) bool {
	for _, identity := range []string{strings.ToLower(state.RegistryUniqueID), strings.ToLower(state.EntityID)} {
		if _, ok := dashboardEntityPlantAffinity(identity, target, singleTarget); ok {
			return true
		}
	}
	return false
}

func dashboardSemanticStateMatchesTargetStable(target haDashboardTarget, state haState, singleTarget, registryAvailable bool) bool {
	if !registryAvailable {
		return dashboardSemanticStateMatchesTarget(target, state, singleTarget)
	}
	identity := strings.ToLower(strings.TrimSpace(state.RegistryUniqueID))
	if identity == "" {
		return false
	}
	_, ok := dashboardEntityPlantAffinity(identity, target, singleTarget)
	return ok
}

func dashboardRegistryMetadataAvailable(states []haState) bool {
	for _, state := range states {
		if strings.TrimSpace(state.RegistryUniqueID) != "" {
			return true
		}
	}
	return false
}

func dashboardTargetInverterCount(target haDashboardTarget) int {
	count := 0
	for _, device := range target.PlantDevices {
		if device.DeviceType == 1 {
			count++
		}
	}
	return count
}

func dashboardEquivalentDuplicate(a, b dashboardMetricCandidate) bool {
	normalize := func(value string) string { return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), "_2") }
	return normalize(a.Entity) == normalize(b.Entity) && a.State == b.State && a.Unit == b.Unit
}

func dashboardIsUnsupportedCalculatedDirectSolar(state haState) bool {
	for _, identity := range []string{state.RegistryUniqueID, state.EntityID} {
		if dashboardSemanticTerminalIdentifier(identity, "pv_consumption_energy") || dashboardSemanticTerminalIdentifier(identity, "pv_to_load_energy") {
			return true
		}
	}
	return false
}

func dashboardManualCandidateCompatible(target haDashboardTarget, metric string, state haState, singleTarget bool) bool {
	if metric == "p13116" && dashboardIsUnsupportedCalculatedDirectSolar(state) {
		return false
	}
	identity := strings.ToLower(strings.TrimSpace(state.RegistryUniqueID + " " + state.EntityID))
	if !strings.Contains(identity, "gosungrow") || !dashboardSemanticStateMatchesTarget(target, state, singleTarget) {
		return false
	}
	return dashboardMetricStateRejectionReason(state, dashboardMetricProfileFor(metric)) == ""
}

func dashboardCanonicalCandidateCompatible(target haDashboardTarget, metric string, state haState, singleTarget, registryAvailable bool) bool {
	contract, ok := dashboardSemanticContractFor(metric)
	if !ok || len(contract.sources) == 0 || dashboardMetricStateRejectionReason(state, dashboardMetricProfileFor(metric)) != "" {
		return false
	}
	if !dashboardSemanticStateMatchesTargetStable(target, state, singleTarget, registryAvailable) {
		return false
	}
	identity := state.EntityID
	if registryAvailable {
		identity = state.RegistryUniqueID
	}
	_, ok = dashboardSemanticSource(identity, contract.sources)
	return ok
}
