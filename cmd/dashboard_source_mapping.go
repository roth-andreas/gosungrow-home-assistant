package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	dashboardSourceMappingCardType  = "custom:gosungrow-source-mapping-card-v1"
	dashboardSourceMappingSchema    = 1
	dashboardSourceValidationSchema = 1
	dashboardSourceRecommendedLimit = 5
	dashboardSourceAdditionalLimit  = 15
	dashboardSourceTargetLimit      = 340
)

type dashboardSourceMetricDefinition struct {
	Metric string
	Group  string
	Icon   string
	Label  string
}

var dashboardSourceMetricDefinitions = []dashboardSourceMetricDefinition{
	{Metric: "pv_power", Group: "live_power", Icon: "mdi:solar-power", Label: "Solar power"},
	{Metric: "load_power", Group: "live_power", Icon: "mdi:home-lightning-bolt-outline", Label: "Home consumption power"},
	{Metric: "grid_power", Group: "live_power", Icon: "mdi:transmission-tower", Label: "Grid power"},
	{Metric: "battery_power", Group: "battery", Icon: "mdi:battery-charging", Label: "Battery power"},
	{Metric: "p13141", Group: "battery", Icon: "mdi:battery-medium", Label: "Battery state of charge"},
	{Metric: "pv_to_load_power", Group: "live_power", Icon: "mdi:home-import-outline", Label: "Solar to home power"},
	{Metric: "pv_to_battery_power", Group: "live_power", Icon: "mdi:battery-arrow-up", Label: "Solar to battery power"},
	{Metric: "pv_to_grid_power", Group: "live_power", Icon: "mdi:transmission-tower-export", Label: "Solar to grid power"},
	{Metric: "grid_to_load_power", Group: "live_power", Icon: "mdi:transmission-tower-import", Label: "Grid to home power"},
	{Metric: "battery_to_load_power", Group: "live_power", Icon: "mdi:battery-arrow-down", Label: "Battery to home power"},
	{Metric: "p13112", Group: "today_energy", Icon: "mdi:white-balance-sunny", Label: "Solar production today"},
	{Metric: "p13116", Group: "today_energy", Icon: "mdi:home-lightning-bolt-outline", Label: "Direct solar consumption"},
	{Metric: "p13174", Group: "today_energy", Icon: "mdi:battery-charging-medium", Label: "Solar energy to battery"},
	{Metric: "p13173", Group: "today_energy", Icon: "mdi:upload-network-outline", Label: "Grid export today"},
	{Metric: "p13147", Group: "today_energy", Icon: "mdi:download-network-outline", Label: "Grid import today"},
	{Metric: "p13199", Group: "energy_summary", Icon: "mdi:home-lightning-bolt", Label: "Home consumption today"},
	{Metric: "p13029", Group: "energy_summary", Icon: "mdi:battery-arrow-down-outline", Label: "Battery discharge today"},
}

func applyDashboardSourceMappings(config map[string]any, current map[string]any, persisted map[string]map[string]string, targets []haDashboardTarget, states []haState, traces []dashboardMetricTrace, dashboardURL string, locale dashboardLocaleBundle) (map[string]any, map[string]map[string]string) {
	requestedOverrides := extractDashboardSourceOverrides(current)
	acceptedOverrides := make(map[string]map[string]string)
	singleTarget := len(targets) == 1

	defaultsByTarget := make(map[string]map[string]string, len(targets))
	for _, trace := range traces {
		key := strings.ToLower(strings.TrimSpace(trace.TargetPsKey))
		metric := strings.ToLower(strings.TrimSpace(trace.Metric))
		if key == "" || metric == "" || trace.Resolved == "" {
			continue
		}
		if defaultsByTarget[key] == nil {
			defaultsByTarget[key] = make(map[string]string)
		}
		defaultsByTarget[key][metric] = trace.Resolved
	}

	for _, target := range targets {
		key := strings.ToLower(strings.TrimSpace(target.PsKey))
		mappingID := dashboardSourceMappingID(target)
		defaults := defaultsByTarget[key]
		card := findDashboardSourceMappingCard(config, target.PsKey)
		if card == nil {
			continue
		}
		card["schema_version"] = dashboardSourceMappingSchema
		card["mapping_id"] = mappingID
		card["dashboard_url_path"] = dashboardURL
		card["labels"] = dashboardSourceMappingLabels(locale)
		if len(defaults) == 0 {
			card["defaults"] = map[string]any{}
			card["overrides"] = map[string]any{}
			card["metrics"] = []any{}
			card["candidates"] = map[string]any{}
			card["bindings"] = map[string]any{}
			continue
		}
		viewIndexes := dashboardTargetViewIndexes(config, target, targets)
		bindings := dashboardSourceBindingPaths(config, viewIndexes, defaults)
		candidatesByMetric := make(map[string][]any, len(defaults))
		remainingCandidates := dashboardSourceTargetLimit
		for _, def := range dashboardSourceMetricDefinitions {
			defaultEntity := defaults[def.Metric]
			if defaultEntity == "" || remainingCandidates <= 0 {
				continue
			}
			candidates := dashboardSourceCandidates(target, def.Metric, defaultEntity, states, singleTarget)
			if len(candidates) > remainingCandidates {
				candidates = candidates[:remainingCandidates]
			}
			remainingCandidates -= len(candidates)
			candidatesByMetric[def.Metric] = candidates
		}

		validOverrides := validateDashboardSourceOverrides(mappingID, defaults, requestedOverrides[mappingID], persisted[mappingID], target, singleTarget, states)
		if len(validOverrides) > 0 {
			acceptedOverrides[mappingID] = validOverrides
		}
		for metric, entity := range validOverrides {
			candidatesByMetric[metric] = ensureDashboardSourceCandidate(candidatesByMetric[metric], metric, entity)
		}
		for metric, rawPaths := range bindings {
			paths, _ := rawPaths.([]any)
			next := defaults[metric]
			if override := validOverrides[metric]; override != "" {
				next = override
			}
			for _, rawPath := range paths {
				_ = setDashboardJSONPointer(config, stringValue(rawPath), next)
			}
		}
		card["defaults"] = stringMapToAny(defaults)
		card["overrides"] = stringMapToAny(validOverrides)
		card["metrics"] = dashboardSourceMetricConfigs(defaults, validOverrides, candidatesByMetric, locale)
		card["candidates"] = candidateMapToAny(candidatesByMetric)
		card["bindings"] = bindings
	}

	return config, acceptedOverrides
}

func extractDashboardSourceOverrides(config map[string]any) map[string]map[string]string {
	ret := make(map[string]map[string]string)
	var walk func(any)
	walk = func(value any) {
		switch typed := value.(type) {
		case map[string]any:
			if stringValue(typed["type"]) == dashboardSourceMappingCardType && intValue(typed["schema_version"]) == dashboardSourceMappingSchema {
				mappingID := strings.ToLower(strings.TrimSpace(stringValue(typed["mapping_id"])))
				if mappingID != "" {
					values := make(map[string]string)
					if raw, ok := typed["overrides"].(map[string]any); ok {
						for metric, entity := range raw {
							if text := strings.TrimSpace(stringValue(entity)); text != "" {
								values[strings.ToLower(strings.TrimSpace(metric))] = text
							}
						}
					}
					if len(values) > 0 {
						ret[mappingID] = values
					}
				}
			}
			for _, entry := range typed {
				walk(entry)
			}
		case []any:
			for _, entry := range typed {
				walk(entry)
			}
		}
	}
	if config != nil {
		walk(config)
	}
	return ret
}

func dashboardSourceMetricConfigs(defaults, overrides map[string]string, candidatesByMetric map[string][]any, locale dashboardLocaleBundle) []any {
	ret := make([]any, 0, len(defaults))
	for _, def := range dashboardSourceMetricDefinitions {
		defaultEntity := defaults[def.Metric]
		if defaultEntity == "" {
			continue
		}
		metric := map[string]any{
			"key": def.Metric, "group": def.Group, "icon": def.Icon, "label": localeText(locale, "source_metric_"+def.Metric, def.Label),
			"default": defaultEntity,
		}
		rules := []any{map[string]any{"type": "freshness", "max_age_seconds": int(dashboardSourceFreshness(def.Metric).Seconds())}}
		selected := defaultEntity
		if override := overrides[def.Metric]; override != "" {
			selected = override
		}
		for _, raw := range candidatesByMetric[def.Metric] {
			candidate, _ := raw.(map[string]any)
			if strings.EqualFold(stringValue(candidate["entity_id"]), selected) {
				metric["confidence"] = candidate["confidence"]
				metric["reason"] = candidate["reason"]
				break
			}
		}
		if def.Metric == "p13116" && defaults["p13112"] != "" {
			rules = append(rules, map[string]any{"type": "not_materially_greater_than", "metric": "p13112", "relative_tolerance": 0.05, "absolute_tolerance": 0.1})
		}
		metric["validation"] = map[string]any{"schema_version": dashboardSourceValidationSchema, "rules": rules}
		ret = append(ret, metric)
	}
	return ret
}

func dashboardSourceCandidates(target haDashboardTarget, metric, defaultEntity string, states []haState, singleTarget bool) []any {
	profile := dashboardMetricProfileFor(metric)
	values := make([]dashboardMetricCandidate, 0)
	for _, state := range states {
		entity, score, reason, ok := dashboardScoreMetricCandidate(target, metric, profile, state, singleTarget)
		if !ok || entity == "" {
			continue
		}
		recent := dashboardSourceStateRecent(state, metric, time.Now())
		if !recent && !strings.EqualFold(entity, defaultEntity) {
			continue
		}
		if strings.EqualFold(entity, defaultEntity) {
			reason = "Current automatic match"
			if !recent {
				reason += " (state is stale)"
				score = 0
			}
		}
		values = append(values, dashboardMetricCandidate{Entity: entity, Metric: metric, Score: score, State: state.State, Unit: dashboardStateUnit(state), Source: dashboardMetricSourceCategory(target, metric, entity), Reason: reason})
	}
	sort.SliceStable(values, func(i, j int) bool {
		iDefault := strings.EqualFold(values[i].Entity, defaultEntity)
		jDefault := strings.EqualFold(values[j].Entity, defaultEntity)
		if iDefault != jDefault {
			return iDefault
		}
		if values[i].Score == values[j].Score {
			return values[i].Entity < values[j].Entity
		}
		return values[i].Score > values[j].Score
	})
	limit := dashboardSourceRecommendedLimit + dashboardSourceAdditionalLimit
	if len(values) > limit {
		values = values[:limit]
	}
	stateByID := dashboardStateByEntityID(states)
	ret := make([]any, 0, len(values)+1)
	foundDefault := false
	for _, value := range values {
		if strings.EqualFold(value.Entity, defaultEntity) {
			foundDefault = true
		}
		state := stateByID[strings.ToLower(value.Entity)]
		device := ""
		if state.Attributes != nil {
			device = strings.TrimSpace(stringValue(state.Attributes["device_name"]))
		}
		ret = append(ret, map[string]any{"entity_id": value.Entity, "device": device, "point_id": metric, "score": value.Score, "confidence": dashboardSourceConfidence(value.Score), "reason": value.Reason, "source": value.Source, "recommended": len(ret) < dashboardSourceRecommendedLimit})
	}
	if !foundDefault && defaultEntity != "" {
		ret = append([]any{map[string]any{"entity_id": defaultEntity, "point_id": metric, "score": 0, "confidence": "low", "reason": "Current automatic match", "recommended": true}}, ret...)
		if len(ret) > limit {
			ret = ret[:limit]
		}
	}
	return ret
}

func dashboardSourceConfidence(score int) string {
	if score >= 200 {
		return "high"
	}
	if score >= 100 {
		return "medium"
	}
	return "low"
}

func findDashboardSourceMappingCard(config map[string]any, mappingID string) map[string]any {
	var found map[string]any
	var walk func(any)
	walk = func(value any) {
		if found != nil {
			return
		}
		switch typed := value.(type) {
		case map[string]any:
			if stringValue(typed["type"]) == dashboardSourceMappingCardType && strings.EqualFold(stringValue(typed["mapping_id"]), mappingID) {
				found = typed
				return
			}
			for _, entry := range typed {
				walk(entry)
			}
		case []any:
			for _, entry := range typed {
				walk(entry)
			}
		}
	}
	walk(config)
	return found
}

func dashboardTargetViewIndexes(config map[string]any, target haDashboardTarget, targets []haDashboardTarget) []int {
	views, _ := config["views"].([]any)
	ret := make([]int, 0)
	for index, raw := range views {
		view, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if len(targets) == 1 {
			ret = append(ret, index)
			continue
		}
		matched, ok := dashboardTargetForView(view, targets)
		if ok && strings.EqualFold(matched.PsKey, target.PsKey) {
			ret = append(ret, index)
			continue
		}
		path := strings.ToLower(strings.TrimSpace(stringValue(view["path"])))
		prefix := strings.ToLower(strings.TrimSpace(target.ViewPath))
		if prefix != "" && (path == prefix || strings.HasPrefix(path, prefix+"-")) {
			ret = append(ret, index)
		}
	}
	return ret
}

func dashboardSourceBindingPaths(config map[string]any, viewIndexes []int, defaults map[string]string) map[string]any {
	ret := make(map[string]any)
	views, _ := config["views"].([]any)
	for metric, defaultEntity := range defaults {
		paths := make([]any, 0)
		for _, index := range viewIndexes {
			if index < 0 || index >= len(views) {
				continue
			}
			collectDashboardEntityPaths(views[index], defaultEntity, "/views/"+strconv.Itoa(index), &paths)
		}
		ret[metric] = paths
	}
	return ret
}

func validateDashboardSourceOverrides(mappingID string, defaults, requested, persisted map[string]string, target haDashboardTarget, singleTarget bool, states []haState) map[string]string {
	ret := make(map[string]string)
	stateByID := dashboardStateByEntityID(states)
	for metric, rawEntity := range requested {
		entity := strings.TrimSpace(rawEntity)
		if defaults[metric] == "" || entity == "" {
			fmtDashboardSourceWarning(mappingID, metric, entity, "unknown metric or empty entity")
			continue
		}
		entityKey := strings.ToLower(entity)
		state, currentlyPresent := stateByID[entityKey]
		currentlyCompatible := false
		structurallyCompatible := false
		if currentlyPresent {
			resolved, _, _, ok := dashboardScoreMetricCandidate(target, metric, dashboardMetricProfileFor(metric), state, singleTarget)
			structurallyCompatible = ok && strings.EqualFold(resolved, entity)
			currentlyCompatible = structurallyCompatible && dashboardSourceStateRecent(state, metric, time.Now())
		}
		previouslyAccepted := strings.EqualFold(strings.TrimSpace(persisted[metric]), entity)
		if currentlyCompatible || (previouslyAccepted && (!currentlyPresent || structurallyCompatible)) {
			ret[metric] = entity
			continue
		}
		fmtDashboardSourceWarning(mappingID, metric, entity, "entity is not a compatible candidate for this target")
	}
	return ret
}

func dashboardSourceFreshness(metric string) time.Duration {
	if dashboardMetricProfileFor(metric).Kind == dashboardMetricKindEnergy {
		return 36 * time.Hour
	}
	return 30 * time.Minute
}

func dashboardSourceStateRecent(state haState, metric string, now time.Time) bool {
	raw := strings.TrimSpace(state.LastUpdated)
	if raw == "" {
		raw = strings.TrimSpace(state.LastChanged)
	}
	if raw == "" {
		return true
	}
	updated, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return false
	}
	return !updated.After(now.Add(5*time.Minute)) && now.Sub(updated) <= dashboardSourceFreshness(metric)
}

func ensureDashboardSourceCandidate(candidates []any, metric, entity string) []any {
	for _, raw := range candidates {
		candidate, _ := raw.(map[string]any)
		if strings.EqualFold(stringValue(candidate["entity_id"]), entity) {
			return candidates
		}
	}
	manual := map[string]any{"entity_id": entity, "point_id": metric, "score": 10000, "confidence": "manual", "reason": "Current manual selection", "source": "manual", "recommended": true}
	ret := make([]any, 0, len(candidates)+1)
	if len(candidates) > 0 {
		ret = append(ret, candidates[0], manual)
		ret = append(ret, candidates[1:]...)
	} else {
		ret = append(ret, manual)
	}
	limit := dashboardSourceRecommendedLimit + dashboardSourceAdditionalLimit
	if len(ret) > limit {
		ret = ret[:limit]
	}
	for index, raw := range ret {
		if candidate, ok := raw.(map[string]any); ok {
			candidate["recommended"] = index < dashboardSourceRecommendedLimit
		}
	}
	return ret
}

func fmtDashboardSourceWarning(mappingID, metric, entity, reason string) {
	// Entity IDs are safe diagnostics; credentials and state values are intentionally excluded.
	fmt.Printf("Warning: ignoring dashboard source override mapping=%s metric=%s entity=%s: %s.\n", mappingID, metric, entity, reason)
}

func setDashboardJSONPointer(root any, path string, next any) bool {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if path == "" || len(parts) == 0 {
		return false
	}
	var current any = root
	for index, escaped := range parts {
		part := strings.ReplaceAll(strings.ReplaceAll(escaped, "~1", "/"), "~0", "~")
		last := index == len(parts)-1
		switch typed := current.(type) {
		case map[string]any:
			if last {
				if _, ok := typed[part]; !ok {
					return false
				}
				typed[part] = next
				return true
			}
			current = typed[part]
		case []any:
			position, err := strconv.Atoi(part)
			if err != nil || position < 0 || position >= len(typed) {
				return false
			}
			if last {
				typed[position] = next
				return true
			}
			current = typed[position]
		default:
			return false
		}
	}
	return false
}

func collectDashboardEntityPaths(value any, entity, path string, paths *[]any) {
	switch typed := value.(type) {
	case map[string]any:
		if typed == nil || (stringValue(typed["type"]) == dashboardSourceMappingCardType) {
			return
		}
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			collectDashboardEntityPaths(typed[key], entity, path+"/"+escapeJSONPointer(key), paths)
		}
	case []any:
		for index, entry := range typed {
			collectDashboardEntityPaths(entry, entity, path+"/"+strconv.Itoa(index), paths)
		}
	case string:
		if strings.EqualFold(typed, entity) {
			*paths = append(*paths, path)
		}
	}
}

func normalizeDashboardStructure(config map[string]any) (map[string]any, error) {
	if config == nil {
		return nil, nil
	}
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	var normalized map[string]any
	if err := json.Unmarshal(data, &normalized); err != nil {
		return nil, err
	}
	var walk func(any)
	walk = func(value any) {
		switch typed := value.(type) {
		case map[string]any:
			if stringValue(typed["type"]) == dashboardSourceMappingCardType {
				defaults := anyMapToStringMap(typed["defaults"])
				overrides := anyMapToStringMap(typed["overrides"])
				bindings, _ := typed["bindings"].(map[string]any)
				for metric, defaultEntity := range defaults {
					effective := defaultEntity
					if override := overrides[metric]; override != "" {
						effective = override
					}
					paths, _ := bindings[metric].([]any)
					for _, rawPath := range paths {
						path := stringValue(rawPath)
						if current, ok := getDashboardJSONPointer(normalized, path); ok && strings.EqualFold(stringValue(current), effective) {
							setDashboardJSONPointer(normalized, path, defaultEntity)
						}
					}
				}
				delete(typed, "overrides")
				delete(typed, "bindings")
				delete(typed, "candidates")
				if metrics, ok := typed["metrics"].([]any); ok {
					for _, rawMetric := range metrics {
						if metric, ok := rawMetric.(map[string]any); ok {
							for _, key := range []string{"candidates", "selected", "status", "warning", "value", "unit", "live_status", "confidence", "reason"} {
								delete(metric, key)
							}
						}
					}
				}
			}
			for _, entry := range typed {
				walk(entry)
			}
		case []any:
			for _, entry := range typed {
				walk(entry)
			}
		}
	}
	walk(normalized)
	return normalized, nil
}

func getDashboardJSONPointer(root any, path string) (any, bool) {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if path == "" || len(parts) == 0 {
		return nil, false
	}
	current := root
	for _, escaped := range parts {
		part := strings.ReplaceAll(strings.ReplaceAll(escaped, "~1", "/"), "~0", "~")
		switch typed := current.(type) {
		case map[string]any:
			var ok bool
			current, ok = typed[part]
			if !ok {
				return nil, false
			}
		case []any:
			position, err := strconv.Atoi(part)
			if err != nil || position < 0 || position >= len(typed) {
				return nil, false
			}
			current = typed[position]
		default:
			return nil, false
		}
	}
	return current, true
}

func hashDashboardStructure(config map[string]any) (string, error) {
	normalized, err := normalizeDashboardStructure(config)
	if err != nil {
		return "", err
	}
	return hashCanonicalJSON(normalized)
}

func dashboardModifiedOutsideGoSungrow(current, desired map[string]any, state *haDashboardState) (bool, error) {
	if current == nil || state == nil {
		return false, nil
	}
	currentHash, err := hashCanonicalJSON(current)
	if err != nil {
		return false, err
	}
	if state.DashboardHash != "" && currentHash == state.DashboardHash {
		return false, nil
	}
	currentStructureHash, err := hashDashboardStructure(current)
	if err != nil {
		return false, err
	}
	if state.DashboardStructureHash != "" {
		return currentStructureHash != state.DashboardStructureHash, nil
	}
	// State files written before 3.2.0 have no structure hash. Only accept a
	// source-only difference when the normalized current and desired structures match.
	desiredStructureHash, err := hashDashboardStructure(desired)
	if err != nil {
		return false, err
	}
	return currentStructureHash != desiredStructureHash, nil
}

func anyMapToStringMap(value any) map[string]string {
	raw, _ := value.(map[string]any)
	ret := make(map[string]string, len(raw))
	for key, entry := range raw {
		if text := strings.TrimSpace(stringValue(entry)); text != "" {
			ret[strings.ToLower(strings.TrimSpace(key))] = text
		}
	}
	return ret
}

func dashboardSourceMappingLabels(locale dashboardLocaleBundle) map[string]any {
	return map[string]any{
		"title": localeText(locale, "source_title", "Data Sources"), "subtitle": localeText(locale, "source_subtitle", "Review automatic matches or choose a dashboard override."),
		"automatic": localeText(locale, "source_automatic", "Automatic"), "manual": localeText(locale, "source_manual", "Manual"), "needs_review": localeText(locale, "source_needs_review", "Needs review"), "unavailable": localeText(locale, "unavailable", "Unavailable"),
		"configure": localeText(locale, "source_configure", "Configure"), "recommended": localeText(locale, "source_recommended", "Recommended"), "other": localeText(locale, "source_other", "Other compatible entities"),
		"search": localeText(locale, "source_search", "Search entities"), "use_source": localeText(locale, "source_use", "Use this source"), "reset": localeText(locale, "source_reset", "Reset to automatic"), "cancel": localeText(locale, "source_cancel", "Cancel"),
		"saved": localeText(locale, "source_saved", "Data source saved."), "readonly": localeText(locale, "source_readonly", "Only Home Assistant administrators can change data sources."),
		"source_unavailable_warning": localeText(locale, "source_unavailable_warning", "The selected entity is unavailable or non-numeric."),
		"source_stale_warning":       localeText(locale, "source_stale_warning", "The selected entity has not updated recently."),
		"source_physical_warning":    localeText(locale, "source_physical_warning", "Selected value ({value}) exceeds solar production ({reference}). Review this source."),
		"source_confirm_warning":     localeText(locale, "source_confirm_warning", "Use this source anyway?"),
		"source_stale":               localeText(locale, "source_stale", "Dashboard changed; reload and try again."),
		"source_incompatible":        localeText(locale, "source_incompatible", "This entity is no longer an available compatible source."),
		"source_save_error":          localeText(locale, "source_save_error", "Could not save the data source. Check your administrator access and connection, then try again."),
		"confidence_high":            localeText(locale, "source_confidence_high", "High confidence"),
		"confidence_medium":          localeText(locale, "source_confidence_medium", "Medium confidence"),
		"confidence_low":             localeText(locale, "source_confidence_low", "Low confidence"),
		"confidence_manual":          localeText(locale, "source_confidence_manual", "User selected"),
		"confidence_unknown":         localeText(locale, "source_confidence_unknown", "Confidence unavailable"),
		"source_compatible":          localeText(locale, "source_compatible", "Compatible source"),
		"groups":                     map[string]any{"live_power": localeText(locale, "source_group_live", "Live power"), "today_energy": localeText(locale, "source_group_today", "Today's energy"), "battery": localeText(locale, "source_group_battery", "Battery"), "energy_summary": localeText(locale, "source_group_summary", "Energy summary")},
	}
}

func localeText(locale dashboardLocaleBundle, key, fallback string) string {
	if text := strings.TrimSpace(locale.Dashboard[key]); text != "" {
		return text
	}
	return fallback
}
func dashboardSourceMappingID(target haDashboardTarget) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(target.PsID) + "\x00" + strings.TrimSpace(target.PsKey)))
	return "source-" + hex.EncodeToString(sum[:6])
}
func stringMapToAny(values map[string]string) map[string]any {
	ret := make(map[string]any, len(values))
	for key, value := range values {
		ret[key] = value
	}
	return ret
}
func candidateMapToAny(values map[string][]any) map[string]any {
	ret := make(map[string]any, len(values))
	for key, value := range values {
		ret[key] = value
	}
	return ret
}
func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return 0
	}
}
func escapeJSONPointer(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "~", "~0"), "/", "~1")
}
