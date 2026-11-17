package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/roth-andreas/gosungrow-home-assistant/cmdHassio"
)

// Source meaning and current availability are independent: a missing reading
// does not turn a known daily counter into an unknown or lifetime counter.
type dashboardSourcePolicy struct {
	EntityID        string `json:"entity_id"`
	SelectionOrigin string `json:"selection_origin"`
	Verification    string `json:"verification"`
	DisplayDecision string `json:"display_decision"`
	Reason          string `json:"reason"`
}

func dashboardPlantPVUniqueID(target haDashboardTarget) string {
	return cmdHassio.JoinStringsForId("GoSungrow", "virtual."+strings.TrimSpace(target.PsID)+".pv_power")
}

func dashboardPlantPVRecommendation(target haDashboardTarget, states []haState, now time.Time) dashboardSemanticMatch {
	registryAvailable := dashboardRegistryMetadataAvailable(states)
	expectedID := dashboardPlantPVUniqueID(target)
	defaultEntity := "sensor.gosungrow_virtual_" + strings.ToLower(strings.TrimSpace(target.PsID)) + "_pv_power"
	var candidates []dashboardMetricCandidate
	for _, state := range states {
		canonical := strings.EqualFold(state.RegistryUniqueID, expectedID)
		if !registryAvailable {
			canonical = strings.EqualFold(state.EntityID, defaultEntity)
		}
		if !canonical || !strings.HasPrefix(strings.ToLower(state.EntityID), "sensor.") ||
			dashboardMetricStateRejectionReason(state, dashboardMetricProfileFor("pv_power")) != "" ||
			!dashboardSourceStateRecent(state, "pv_power", now) || (!registryAvailable && dashboardPVForbidden(state, "pv_power")) {
			continue
		}
		compatibility := "compatible"
		if !registryAvailable {
			compatibility = "unverified"
		}
		candidates = append(candidates, dashboardMetricCandidate{
			Entity: state.EntityID, Metric: "pv_power", Score: 9999, State: state.State,
			Unit: dashboardStateUnit(state), Source: "plant-aggregate", Scope: "plant",
			Compatibility: compatibility, Reason: "Canonical plant PV aggregate",
		})
	}
	dashboardSortSemanticCandidates(candidates)
	if len(candidates) == 0 {
		return dashboardSemanticMatch{}
	}
	confident := registryAvailable
	for _, candidate := range candidates[1:] {
		if !dashboardEquivalentDuplicate(candidates[0], candidate) {
			confident = false
		}
	}
	reason := "Canonical plant PV aggregate"
	if !registryAvailable {
		reason = "Canonical registry verification unavailable"
	} else if !confident {
		reason = "Ambiguous canonical plant PV entities"
	}
	return dashboardSemanticMatch{Entity: candidates[0].Entity, Score: 9999, Reason: reason, Confident: confident, Candidates: candidates}
}

func assessDashboardSource(target haDashboardTarget, metric, entity string, manual bool, states []haState, now time.Time) dashboardSourcePolicy {
	policy := dashboardSourcePolicy{EntityID: entity, SelectionOrigin: "automatic", Verification: "unverified", DisplayDecision: "withheld", Reason: "Source meaning could not be verified"}
	if manual {
		policy.SelectionOrigin, policy.DisplayDecision = "manual", "warned"
	}
	if entity == "" {
		policy.DisplayDecision, policy.Reason = "withheld", "No source selected"
		return policy
	}
	state, present := dashboardStateByEntityID(states)[strings.ToLower(entity)]
	if !present {
		policy.Reason = "Selected source not observed"
		return policy
	}
	profile := dashboardMetricProfileFor(metric)
	if dashboardStateUnit(state) == "" {
		policy.Reason = "Source unit not observed"
		return policy
	}
	if dashboardMetricUnitRejectionReason(state, profile) != "" {
		policy.Verification, policy.DisplayDecision, policy.Reason = "incompatible", "withheld", "Source unit conflicts with the metric"
		return policy
	}
	registryAvailable := dashboardRegistryMetadataAvailable(states)
	identity := strings.TrimSpace(state.RegistryUniqueID)
	if registryAvailable && identity != "" && !dashboardCanonicalTargetAffinity(strings.ToLower(identity), target, false) {
		policy.Verification, policy.DisplayDecision, policy.Reason = "incompatible", "withheld", "Source belongs to another target"
		return policy
	}
	verified := false
	if dashboardIsCanonicalDailyMetric(metric) {
		contract, _ := dashboardSemanticContractFor(metric)
		candidate, ok := dashboardCanonicalMeaningCandidate(target, metric, contract, state, registryAvailable, now)
		if ok && registryAvailable && metric == "p13112" && candidate.Scope == "inverter" && dashboardTargetInverterCount(target) != 1 {
			policy.Verification, policy.DisplayDecision, policy.Reason = "incompatible", "withheld", "Canonical source meaning conflicts with this metric"
			return policy
		}
		verified = ok && registryAvailable && candidate.Compatibility == "compatible"
		if !verified && registryAvailable && identity != "" {
			for _, other := range []string{"p13112", "p13116", "p13173", "p13147", "p13199"} {
				otherContract, _ := dashboardSemanticContractFor(other)
				if other != metric {
					if _, ok := dashboardSemanticSource(identity, otherContract.sources); ok {
						policy.Verification, policy.DisplayDecision, policy.Reason = "incompatible", "withheld", "Canonical source meaning conflicts with this metric"
						return policy
					}
				}
			}
		}
		if metric == "p13116" && dashboardIsUnsupportedCalculatedDirectSolar(state) {
			policy.DisplayDecision, policy.Reason = "withheld", "Unsupported calculated direct-solar source"
			return policy
		}
	} else if metric == "pv_power" {
		verified = registryAvailable && strings.EqualFold(identity, dashboardPlantPVUniqueID(target))
		if !verified && identity != "" {
			identityState := state
			identityState.EntityID, identityState.Attributes = "sensor."+identity, nil
			if dashboardPVForbidden(identityState, metric) {
				policy.Verification, policy.DisplayDecision, policy.Reason = "incompatible", "withheld", "Canonical source meaning conflicts with this metric"
				return policy
			}
		}
		if !verified && dashboardTargetInverterCount(target) == 1 {
			contract, _ := dashboardSemanticContractFor(metric)
			verified = dashboardSemanticCandidateAllowed(target, contract, state) && !dashboardPVForbidden(state, metric)
		}
	} else {
		// Other metric profiles retain their existing selection semantics. A
		// scoring result is not enough to verify a canonical daily counter.
		name := strings.ToLower(state.EntityID)
		_, affine := dashboardEntityPlantAffinity(name, target, true)
		_, suffix := dashboardMetricSuffixScore(name, metric, profile, target)
		_, tokens := dashboardMetricTokenScore(name, profile)
		verified = strings.HasPrefix(name, "sensor.") && strings.Contains(name, "gosungrow") && affine && (suffix || tokens)
	}
	if verified {
		policy.Verification, policy.DisplayDecision, policy.Reason = "verified", "normal", "Source meaning verified"
	}
	return policy
}

func (p dashboardSourcePolicy) fields() map[string]any {
	return map[string]any{"entity_id": p.EntityID, "selection_origin": p.SelectionOrigin, "verification": p.Verification, "display_decision": p.DisplayDecision, "reason": p.Reason}
}

func localizeDashboardSourcePolicy(policy dashboardSourcePolicy, locale dashboardLocaleBundle) dashboardSourcePolicy {
	key := map[string]string{
		"Source meaning could not be verified": "source_meaning_unverified", "Source meaning verified": "source_meaning_verified",
		"No source selected": "source_no_source", "Selected source not observed": "source_not_observed",
		"Source unit conflicts with the metric": "source_unit_conflict", "Source belongs to another target": "source_target_conflict",
		"Canonical source meaning conflicts with this metric": "source_meaning_conflict", "Source unit not observed": "source_unit_missing",
		"Unsupported calculated direct-solar source": "source_unsupported_calculated",
	}[policy.Reason]
	policy.Reason = localeText(locale, key, policy.Reason)
	if policy.DisplayDecision != "normal" {
		policy.Reason += ". " + localeText(locale, "source_review", "Review Data Sources.")
	}
	return policy
}

// Attach policy at the closest display card, retaining separate ownership when
// two metrics happen to select the same entity. Policy entity references are
// bindings too, so a source transaction cannot leave them pointing at old data.
func attachDashboardSourcePolicies(config map[string]any, mappingID string, bindings map[string]any, policies map[string]dashboardSourcePolicy, locale dashboardLocaleBundle) error {
	nativeCards := make(map[string]map[string]any)
	for metric, policy := range policies {
		paths, _ := bindings[metric].([]any)
		originalPaths := append([]any(nil), paths...)
		seen := make(map[string]bool)
		for _, raw := range originalPaths {
			path := stringValue(raw)
			parts := strings.Split(path, "/")
			var card map[string]any
			cardPath := ""
			for end := 1; end < len(parts); end++ {
				prefix := strings.Join(parts[:end], "/")
				value, ok := getDashboardJSONPointer(config, prefix)
				candidate, isMap := value.(map[string]any)
				if ok && isMap && stringValue(candidate["type"]) != "" && stringValue(candidate["type"]) != dashboardSourceMappingCardType {
					card, cardPath = candidate, prefix
				}
			}
			if card == nil || seen[cardPath] {
				continue
			}
			seen[cardPath] = true
			if strings.HasPrefix(stringValue(card["type"]), "custom:gosungrow-") {
				labels, _ := card["labels"].(map[string]any)
				if labels == nil {
					labels = make(map[string]any)
					card["labels"] = labels
				}
				for _, key := range []string{"source_withheld", "source_manual_unverified", "source_unverified"} {
					labels[key] = localeText(locale, key, defaultDashboardLocaleBundle.Dashboard[key])
				}
				labelKey := "source_metric_" + metric
				labels[labelKey] = localeText(locale, labelKey, defaultDashboardLocaleBundle.Dashboard[labelKey])
			}
			source, _ := card["source_policy"].(map[string]any)
			if source == nil {
				source = map[string]any{"mapping_id": mappingID, "metrics": map[string]any{}}
				card["source_policy"] = source
			}
			metrics, _ := source["metrics"].(map[string]any)
			metrics[metric] = policy.fields()
			if dashboardNativeCardType(stringValue(card["type"])) {
				if _, exists := source["native_template"]; !exists {
					template := make(map[string]any, len(card))
					for key, value := range card {
						if key != "source_policy" {
							template[key] = value
						}
					}
					copied, err := deepCopyJSONValue(template)
					if err != nil {
						return fmt.Errorf("capture native source template: %w", err)
					}
					source["native_template"], source["native_bindings"] = copied, map[string]any{}
				}
				owned := source["native_bindings"].(map[string]any)
				var relative []any
				for _, raw := range originalPaths {
					if strings.HasPrefix(stringValue(raw), cardPath+"/") {
						relative = append(relative, strings.TrimPrefix(stringValue(raw), cardPath))
					}
				}
				owned[metric] = relative
				nativeCards[cardPath] = card
			}
			paths = append(paths, cardPath+"/source_policy/metrics/"+metric+"/entity_id")
		}
		bindings[metric] = paths
	}
	cardPaths := make([]string, 0, len(nativeCards))
	for path := range nativeCards {
		cardPaths = append(cardPaths, path)
	}
	sort.Strings(cardPaths)
	for _, cardPath := range cardPaths {
		card := nativeCards[cardPath]
		source := card["source_policy"].(map[string]any)
		rendered, relative, err := renderDashboardNativeProjection(source)
		if err != nil {
			return err
		}
		for metric, raw := range bindings {
			var kept []any
			for _, entry := range raw.([]any) {
				if !strings.HasPrefix(stringValue(entry), cardPath+"/") {
					kept = append(kept, entry)
				}
			}
			if paths, ok := relative[metric].([]any); ok {
				for _, entry := range paths {
					kept = append(kept, cardPath+stringValue(entry))
				}
			}
			bindings[metric] = kept
		}
		for key := range card {
			delete(card, key)
		}
		for key, value := range rendered {
			card[key] = value
		}
	}
	return nil
}

func dashboardRetainedSourceMapping(card map[string]any) (map[string]any, bool) {
	if stringValue(card["type"]) == dashboardSourceMappingCardType && intValue(card["schema_version"]) == dashboardSourceMappingSchema {
		return card, true
	}
	retained, ok := card["gosungrow_source_mapping"].(map[string]any)
	return retained, ok && intValue(retained["schema_version"]) == dashboardSourceMappingSchema
}

// A verified browser transaction can precede the next diagnostic-state write.
// Recognize that saved manual decision, without treating it as permission to
// render: fresh semantic assessment still controls its display policy.
func acceptedDashboardSourceOverrides(current map[string]any, persisted map[string]map[string]string) map[string]map[string]string {
	accepted := make(map[string]map[string]string, len(persisted))
	for id, choices := range persisted {
		accepted[id] = copyDashboardSourceMap(choices)
	}
	for id, choices := range extractDashboardSourceOverrides(current) {
		if accepted[id] == nil {
			accepted[id] = make(map[string]string)
		}
		for metric, entity := range choices {
			accepted[id][metric] = entity
		}
	}
	return accepted
}
