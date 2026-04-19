package cmd

import "strings"

type dashboardEntityRef struct {
	Entity string
	Target haDashboardTarget
	Metric string
}

func remapDashboardEntities(config map[string]any, targets []haDashboardTarget, stateEntityIDs []string) map[string]any {
	if len(config) == 0 || len(targets) == 0 || len(stateEntityIDs) == 0 {
		return config
	}

	refs := collectLegacyDashboardEntityRefs(config, targets)
	if len(refs) == 0 {
		return config
	}

	stateSet := make(map[string]struct{}, len(stateEntityIDs))
	for _, entityID := range stateEntityIDs {
		stateSet[strings.ToLower(strings.TrimSpace(entityID))] = struct{}{}
	}

	replacements := make(map[string]string)
	singleTarget := len(targets) == 1
	for _, ref := range refs {
		resolved := resolveDashboardEntityRef(ref, stateEntityIDs, stateSet, singleTarget)
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

func resolveDashboardEntityRef(ref dashboardEntityRef, stateEntityIDs []string, stateSet map[string]struct{}, singleTarget bool) string {
	psKey := strings.ToLower(strings.TrimSpace(ref.Target.PsKey))
	psID := strings.ToLower(strings.TrimSpace(ref.Target.PsID))
	metric := strings.ToLower(strings.TrimSpace(ref.Metric))

	legacy := []string{
		"sensor.gosungrow_virtual_" + psKey + "_" + metric,
		"sensor.gosungrow_virtual_" + psID + "_" + metric,
	}
	for _, candidate := range legacy {
		if _, ok := stateSet[candidate]; ok {
			return candidate
		}
	}

	bestCandidate := ""
	bestScore := -1
	zeroScoreCandidates := make([]string, 0)
	for _, entityID := range stateEntityIDs {
		candidate := strings.ToLower(strings.TrimSpace(entityID))
		if !strings.HasPrefix(candidate, "sensor.") {
			continue
		}
		if !strings.Contains(candidate, "gosungrow") {
			continue
		}
		if !strings.HasSuffix(candidate, "_"+metric) {
			continue
		}

		affinityScore := 0
		if strings.Contains(candidate, "_"+psKey+"_") {
			affinityScore += 100
		}
		if strings.Contains(candidate, "_"+psID+"_") {
			affinityScore += 70
		}
		if affinityScore == 0 && !singleTarget {
			continue
		}
		if affinityScore == 0 {
			zeroScoreCandidates = append(zeroScoreCandidates, candidate)
			continue
		}

		score := affinityScore
		if strings.Contains(candidate, "_virtual_") {
			score += 30
		}
		if strings.HasPrefix(candidate, "sensor.gosungrow_") {
			score += 10
		}

		if bestCandidate == "" || score > bestScore || (score == bestScore && len(candidate) < len(bestCandidate)) {
			bestCandidate = candidate
			bestScore = score
		}
	}

	if bestCandidate != "" {
		return bestCandidate
	}
	if singleTarget && len(zeroScoreCandidates) == 1 {
		return zeroScoreCandidates[0]
	}
	return bestCandidate
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
