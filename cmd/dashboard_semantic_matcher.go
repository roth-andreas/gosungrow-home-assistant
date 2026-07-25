package cmd

import (
	"sort"
	"strings"
	"time"
)

const dashboardSemanticMatcherVersion = 2

type dashboardSemanticContract struct {
	metric          string
	aliases         []string
	requiredMeaning []string
	forbidden       []string
	allowedRoles    map[string]bool
	daily           bool
}

type dashboardSemanticMatch struct {
	Entity     string
	Score      int
	Reason     string
	Confident  bool
	Candidates []dashboardMetricCandidate
}

func dashboardSemanticContractFor(metric string) (dashboardSemanticContract, bool) {
	contracts := map[string]dashboardSemanticContract{
		"pv_power": {
			metric: "pv_power", aliases: []string{"plant_active_power", "inverter_active_power", "pv_active_power"},
			requiredMeaning: []string{"pv", "solar", "generation", "active_power"},
			forbidden:       []string{"phase_a", "phase_b", "phase_c"},
			allowedRoles:    map[string]bool{"inverter": true, "ess": true, "plant": true},
		},
		"p13112": {
			metric: "p13112", aliases: []string{"daily_generation", "generation_today", "daily_generation_energy"},
			requiredMeaning: []string{"generation", "production", "yield"},
			forbidden:       []string{"feed_in", "feedin", "export", "import", "purchased"},
			allowedRoles:    map[string]bool{"inverter": true, "ess": true, "plant": true}, daily: true,
		},
		"p13173": {
			metric: "p13173", aliases: []string{"feed_in_energy_today", "daily_feed_in_energy", "export_energy_today"},
			requiredMeaning: []string{"feed_in", "feedin", "export"},
			forbidden:       []string{"import", "purchased"},
			allowedRoles:    map[string]bool{"meter": true, "ess": true, "plant": true}, daily: true,
		},
		"p13147": {
			metric: "p13147", aliases: []string{"energy_purchased_today", "purchased_energy_today", "import_energy_today"},
			requiredMeaning: []string{"import", "purchased"},
			forbidden:       []string{"feed_in", "feedin", "export"},
			allowedRoles:    map[string]bool{"meter": true, "ess": true, "plant": true}, daily: true,
		},
		"p13199": {
			metric: "p13199", aliases: []string{"home_consumption_today", "daily_home_consumption", "daily_load_energy", "load_energy_today"},
			requiredMeaning: []string{"consumption", "load", "home", "house", "use"},
			forbidden:       []string{"feed_in", "feedin", "export", "import", "purchased", "battery"},
			allowedRoles:    map[string]bool{"ess": true, "plant": true}, daily: true,
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
	profile := dashboardSemanticProfile(metric, contract)
	values := make([]dashboardMetricCandidate, 0)
	for _, state := range states {
		entity, score, reason, compatible := dashboardScoreMetricCandidate(target, metric, profile, state, singleTarget)
		if !compatible || entity == "" || !dashboardSemanticCandidateAllowed(target, contract, state) {
			continue
		}
		score += 300
		values = append(values, dashboardMetricCandidate{
			Entity: entity, Metric: metric, Score: score, State: state.State, Unit: dashboardStateUnit(state),
			Source: dashboardMetricSourceCategory(target, metric, entity), Reason: reason,
		})
	}
	sort.SliceStable(values, func(i, j int) bool {
		if values[i].Score == values[j].Score {
			return len(values[i].Entity) < len(values[j].Entity)
		}
		return values[i].Score > values[j].Score
	})
	if len(values) == 0 {
		return dashboardSemanticMatch{}
	}
	confident := true
	if len(values) > 1 && values[0].Score-values[1].Score < 20 && !dashboardEquivalentDuplicate(values[0], values[1]) {
		confident = false
	}
	return dashboardSemanticMatch{
		Entity: values[0].Entity, Score: values[0].Score, Confident: confident, Candidates: values,
		Reason: "Semantic match: compatible device role, direction and measurement period",
	}
}

func dashboardSemanticCandidateAllowed(target haDashboardTarget, contract dashboardSemanticContract, state haState) bool {
	entity := strings.ToLower(strings.TrimSpace(state.EntityID))
	role := dashboardCandidateDeviceRole(target, entity)
	if len(contract.allowedRoles) > 0 && !contract.allowedRoles[role] {
		return false
	}
	for _, forbidden := range contract.forbidden {
		if dashboardSemanticContains(entity, forbidden) {
			return false
		}
	}
	meaning := false
	for _, required := range contract.requiredMeaning {
		if dashboardSemanticContains(entity, required) {
			meaning = true
			break
		}
	}
	if !meaning {
		return false
	}
	if contract.daily && !dashboardCandidateIsDaily(state) {
		return false
	}
	return true
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
	entity := strings.ToLower(strings.TrimSpace(state.EntityID))
	for _, marker := range []string{"_today", "today_", "_daily", "daily_", "_p13112", "_p13147", "_p13173", "_p13199"} {
		if strings.Contains(entity, marker) {
			return true
		}
	}
	if state.Attributes == nil {
		return false
	}
	reset := strings.TrimSpace(stringValue(state.Attributes["last_reset"]))
	if reset == "" {
		return false
	}
	when, err := time.Parse(time.RFC3339, reset)
	if err != nil {
		return false
	}
	age := time.Since(when)
	return age >= -6*time.Hour && age <= 48*time.Hour
}

func dashboardSemanticContains(entity, phrase string) bool {
	normalize := func(value string) string {
		value = strings.ToLower(strings.TrimSpace(value))
		return strings.NewReplacer(".", "_", "-", "_", " ", "_", "/", "_").Replace(value)
	}
	entity = "_" + strings.Trim(normalize(entity), "_") + "_"
	phrase = strings.Trim(normalize(phrase), "_")
	return phrase != "" && strings.Contains(entity, "_"+phrase+"_")
}

func dashboardEquivalentDuplicate(a, b dashboardMetricCandidate) bool {
	normalize := func(value string) string {
		value = strings.ToLower(strings.TrimSpace(value))
		value = strings.TrimSuffix(value, "_2")
		return value
	}
	return normalize(a.Entity) == normalize(b.Entity) && a.State == b.State && a.Unit == b.Unit
}

func dashboardManualCandidateCompatible(target haDashboardTarget, metric string, state haState, singleTarget bool) bool {
	entity := strings.ToLower(strings.TrimSpace(state.EntityID))
	if !strings.HasPrefix(entity, "sensor.") || !strings.Contains(entity, "gosungrow") {
		return false
	}
	if _, ok := dashboardEntityPlantAffinity(entity, target, singleTarget); !ok {
		return false
	}
	return dashboardMetricStateRejectionReason(state, dashboardMetricProfileFor(metric)) == ""
}
