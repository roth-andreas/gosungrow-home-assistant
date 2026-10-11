package cmd

import "strings"

// composeDashboardSources keeps placeholders until the final capability
// decision, then builds bindings at their post-pruning positions.
func composeDashboardSources(config, current map[string]any, persisted map[string]map[string]string, targets []haDashboardTarget, states []haState, dashboardURL string, locale dashboardLocaleBundle) (map[string]any, map[string]map[string]string, dashboardRemapReport) {
	_, report := remapDashboardEntitiesWithReport(config, targets, states)
	traces, accepted := prepareDashboardSourceTraces(report.Traces, targets, states, extractDashboardSourceDefaults(current), extractDashboardSourceOverrides(current), persisted)
	config = pruneDashboardForUnavailableMetricsWithTraces(config, targets, states, traces)
	config, accepted = applyDashboardSourceMappingsWithTraces(config, current, targets, states, traces, accepted, dashboardURL, locale)
	return config, accepted, dashboardReportFromTraces(traces)
}

func prepareDashboardSourceTraces(traces []dashboardMetricTrace, targets []haDashboardTarget, states []haState, pinned, requested, persisted map[string]map[string]string) ([]dashboardMetricTrace, map[string]map[string]string) {
	ret := append([]dashboardMetricTrace(nil), traces...)
	accepted := make(map[string]map[string]string)
	stateByID := dashboardStateByEntityID(states)
	singleTarget := len(targets) == 1
	registryAvailable := dashboardRegistryMetadataAvailable(states)
	for _, target := range targets {
		key := strings.ToLower(strings.TrimSpace(target.PsKey))
		mappingID := dashboardSourceMappingID(target)
		defaults := make(map[string]string)
		matches := make(map[string]dashboardSemanticMatch)
		for i := range ret {
			trace := &ret[i]
			if !strings.EqualFold(strings.TrimSpace(trace.TargetPsKey), key) {
				continue
			}
			metric := strings.ToLower(strings.TrimSpace(trace.Metric))
			if trace.Placeholder == "" {
				trace.Placeholder = trace.Resolved
			}
			if !dashboardIsCanonicalDailyMetric(metric) {
				if trace.Resolved != "" {
					defaults[metric] = trace.Resolved
					if existing := strings.TrimSpace(pinned[mappingID][metric]); existing != "" {
						defaults[metric] = existing
					}
				}
				continue
			}
			match := dashboardSemanticMatch{}
			if trace.Resolution != nil {
				match = trace.Resolution.Match
			} else {
				match = dashboardSemanticRecommendation(target, metric, states, singleTarget)
			}
			resolution := &dashboardDailyResolution{Match: match}
			if match.Confident {
				resolution.Default = match.Entity
			}
			if existing := strings.TrimSpace(pinned[mappingID][metric]); existing != "" {
				resolution.Default, resolution.Pinned = existing, true
			}
			trace.Resolution = resolution
			defaults[metric] = resolution.Default // The metric is known even without an automatic entity.
			matches[metric] = match
		}
		valid := validateDashboardSourceOverridesWithMatches(mappingID, defaults, requested[mappingID], persisted[mappingID], target, singleTarget, states, matches)
		if len(valid) > 0 {
			accepted[mappingID] = valid
		}
		for i := range ret {
			trace := &ret[i]
			if !strings.EqualFold(strings.TrimSpace(trace.TargetPsKey), key) || trace.Resolution == nil {
				continue
			}
			resolution := trace.Resolution
			resolution.Override = valid[trace.Metric]
			trace.Resolved, trace.Source = resolution.Default, "canonical"
			if resolution.Pinned {
				trace.Source = "pinned"
			}
			if resolution.Override != "" {
				trace.Resolved, trace.Source = resolution.Override, "manual"
			}
			verified := false
			for _, candidate := range resolution.Match.Candidates {
				if registryAvailable && candidate.Compatibility == "compatible" && strings.EqualFold(candidate.Entity, trace.Resolved) {
					verified = true
					break
				}
			}
			resolution.NeedsReview = !verified || (resolution.Match.Confident && !strings.EqualFold(resolution.Match.Entity, resolution.Default))
			if state, ok := stateByID[strings.ToLower(trace.Resolved)]; ok && trace.Metric == "p13116" && dashboardIsUnsupportedCalculatedDirectSolar(state) {
				resolution.UnsupportedCalculated, resolution.NeedsReview = true, true
			}
			if trace.Resolved == "" {
				trace.Source = "unavailable"
			}
			trace.Candidates = append([]dashboardMetricCandidate(nil), resolution.Match.Candidates...)
			if len(trace.Candidates) > dashboardSourceRecommendedLimit {
				trace.Candidates = trace.Candidates[:dashboardSourceRecommendedLimit]
			}
		}
	}
	return ret, accepted
}

func dashboardReportFromTraces(traces []dashboardMetricTrace) dashboardRemapReport {
	report := dashboardRemapReport{TotalRefs: len(traces), Traces: traces}
	for _, trace := range traces {
		if trace.Resolved == "" {
			reason := trace.Reason
			if reason == "" {
				reason = "No usable source"
			}
			if trace.Resolution != nil {
				reason = trace.Resolution.Match.Reason
			}
			report.Unresolved = append(report.Unresolved, dashboardUnresolvedEntityRef{Entity: trace.Placeholder, Metric: trace.Metric, Reason: reason})
		} else if trace.Placeholder != "" && !strings.EqualFold(trace.Placeholder, trace.Resolved) {
			report.Remapped = append(report.Remapped, dashboardEntityRemap{From: trace.Placeholder, To: trace.Resolved, Metric: trace.Metric, Source: trace.Source})
		}
	}
	return report
}
