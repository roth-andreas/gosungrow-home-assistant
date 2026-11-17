package cmd

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type dashboardNativeOwnershipError struct{ cause error }

func (e *dashboardNativeOwnershipError) Error() string { return e.cause.Error() }
func (e *dashboardNativeOwnershipError) Unwrap() error { return e.cause }

// Restore the owned structural shape for comparison, but only after checking
// the rendered projection. Otherwise stripping metadata could conceal edits.
func normalizeDashboardNativeProjections(value any) error {
	switch typed := value.(type) {
	case map[string]any:
		if source, ok := typed["source_policy"].(map[string]any); ok {
			_, templatePresent := source["native_template"]
			_, bindingsPresent := source["native_bindings"]
			if templatePresent || bindingsPresent {
				expected, _, err := renderDashboardNativeProjection(source)
				if err != nil {
					return err
				}
				actualJSON, err := json.Marshal(typed)
				if err != nil {
					return err
				}
				expectedJSON, err := json.Marshal(expected)
				if err != nil {
					return err
				}
				if string(actualJSON) != string(expectedJSON) {
					return fmt.Errorf("native source projection was modified outside GoSungrow")
				}
				copied, err := deepCopyJSONValue(source["native_template"])
				if err != nil {
					return err
				}
				template := copied.(map[string]any)
				for metric, raw := range source["native_bindings"].(map[string]any) {
					for _, rawPath := range raw.([]any) {
						setDashboardJSONPointer(template, rawPath.(string), "__gosungrow_source__"+stringValue(source["mapping_id"])+"__"+metric)
					}
				}
				for key := range typed {
					delete(typed, key)
				}
				for key, entry := range template {
					typed[key] = entry
				}
			}
		}
		for _, child := range typed {
			if err := normalizeDashboardNativeProjections(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range typed {
			if err := normalizeDashboardNativeProjections(child); err != nil {
				return err
			}
		}
	}
	return nil
}

// Native templates are configuration, never hidden rendered cards. Relative
// ownership survives removing a graph series; absolute bindings are rebuilt.
func dashboardNativeCardType(kind string) bool {
	return kind == "tile" || kind == "entities" || kind == "history-graph"
}

func dashboardNativeMetricLabel(template map[string]any, paths []any) string {
	if name := strings.TrimSpace(stringValue(template["name"])); name != "" {
		return name
	}
	for _, raw := range paths {
		parts := strings.Split(stringValue(raw), "/")
		if len(parts) >= 3 && parts[1] == "entities" {
			if row, ok := getDashboardJSONPointer(template, "/entities/"+parts[2]); ok {
				if fields, ok := row.(map[string]any); ok {
					if name := strings.TrimSpace(stringValue(fields["name"])); name != "" {
						return name
					}
				}
			}
		}
	}
	if title := strings.TrimSpace(stringValue(template["title"])); title != "" {
		return title
	}
	return ""
}

func dashboardNativeEntityPath(template map[string]any, path string) bool {
	if stringValue(template["type"]) == "tile" {
		return path == "/entity"
	}
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[0] != "" || parts[1] != "entities" {
		return false
	}
	index, err := strconv.Atoi(parts[2])
	if err != nil || index < 0 || strconv.Itoa(index) != parts[2] {
		return false
	}
	return len(parts) == 3 || (len(parts) == 4 && parts[3] == "entity")
}

func dashboardNativeTemplateSafe(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			switch strings.ToLower(key) {
			case "source_policy", "gosungrow_source_mapping", "native_template", "native_bindings", "state", "attributes", "access_token", "password", "credentials", "raw_payload":
				return false
			}
			if !dashboardNativeTemplateSafe(child) {
				return false
			}
		}
	case []any:
		for _, child := range typed {
			if !dashboardNativeTemplateSafe(child) {
				return false
			}
		}
	}
	return true
}

// renderDashboardNativeProjection validates before returning a replacement. It
// does not mutate the supplied template or policy, including on failure.
func renderDashboardNativeProjection(source map[string]any) (map[string]any, map[string]any, error) {
	template, templateOK := source["native_template"].(map[string]any)
	owned, bindingsOK := source["native_bindings"].(map[string]any)
	metrics, metricsOK := source["metrics"].(map[string]any)
	if !templateOK || !bindingsOK || !metricsOK || stringValue(source["mapping_id"]) == "" ||
		!dashboardNativeCardType(stringValue(template["type"])) || !dashboardNativeTemplateSafe(template) {
		return nil, nil, fmt.Errorf("invalid native source projection metadata; refresh the dashboard")
	}
	copied, err := deepCopyJSONValue(template)
	if err != nil {
		return nil, nil, err
	}
	rendered := copied.(map[string]any)
	owners := make(map[string]string)
	blockedRows := make(map[int]bool)
	blockedTile := false
	warnings := make([]string, 0)
	keys := make([]string, 0, len(owned))
	for metric := range owned {
		keys = append(keys, metric)
	}
	sort.Strings(keys)
	for _, metric := range keys {
		known := false
		for _, definition := range dashboardSourceMetricDefinitions {
			if definition.Metric == metric {
				known = true
				break
			}
		}
		policy, ok := metrics[metric].(map[string]any)
		paths, pathsOK := owned[metric].([]any)
		if !known || !ok || !pathsOK || len(paths) == 0 || strings.TrimSpace(stringValue(policy["reason"])) == "" {
			return nil, nil, fmt.Errorf("invalid native metric ownership; refresh the dashboard")
		}
		decision := stringValue(policy["display_decision"])
		verification := stringValue(policy["verification"])
		origin := stringValue(policy["selection_origin"])
		if (origin != "automatic" && origin != "manual") || (verification != "verified" && verification != "unverified" && verification != "incompatible") {
			return nil, nil, fmt.Errorf("invalid native source policy; refresh the dashboard")
		}
		if !((decision == "normal" && verification == "verified") ||
			(decision == "warned" && verification == "unverified" && origin == "manual") || decision == "withheld") {
			return nil, nil, fmt.Errorf("invalid native source policy; refresh the dashboard")
		}
		for _, raw := range paths {
			path, ok := raw.(string)
			entity, resolves := getDashboardJSONPointer(template, path)
			if !ok || !resolves || !dashboardNativeEntityPath(template, path) ||
				!reflect.DeepEqual(entity, policy["entity_id"]) || owners[path] != "" {
				return nil, nil, fmt.Errorf("invalid native entity binding; refresh the dashboard")
			}
			owners[path] = metric
			if decision == "withheld" {
				if path == "/entity" {
					blockedTile = true
				} else {
					index, _ := strconv.Atoi(strings.Split(path, "/")[2])
					blockedRows[index] = true
				}
			}
		}
		if decision != "normal" {
			label := dashboardNativeMetricLabel(template, paths)
			if label != "" {
				label += ": "
			}
			warnings = append(warnings, label+stringValue(policy["reason"]))
		}
	}
	indexMap := make(map[int]int)
	if rows, ok := rendered["entities"].([]any); ok {
		kept := make([]any, 0, len(rows))
		for i, row := range rows {
			if !blockedRows[i] {
				indexMap[i] = len(kept)
				kept = append(kept, row)
			}
		}
		rendered["entities"] = kept
	}
	visible := !blockedTile
	if rows, ok := rendered["entities"].([]any); ok && len(rows) == 0 {
		visible = false
	}
	result := rendered
	renderPrefix := ""
	if len(warnings) > 0 {
		cards := []any{map[string]any{"type": "markdown", "content": strings.Join(warnings, "\n\n")}}
		if visible {
			cards = append(cards, rendered)
			renderPrefix = "/cards/1"
		}
		result = map[string]any{"type": "vertical-stack", "cards": cards}
		if layout, ok := rendered["layout_options"]; ok {
			result["layout_options"] = layout
			delete(rendered, "layout_options")
		}
	}
	result["source_policy"] = source
	bindings := make(map[string]any)
	for _, metric := range keys {
		paths := []any{"/source_policy/metrics/" + escapeJSONPointer(metric) + "/entity_id"}
		for _, raw := range owned[metric].([]any) {
			path := raw.(string)
			paths = append(paths, "/source_policy/native_template"+path)
			if !visible {
				continue
			}
			if path != "/entity" {
				parts := strings.Split(path, "/")
				i, _ := strconv.Atoi(parts[2])
				next, retained := indexMap[i]
				if !retained {
					continue
				}
				parts[2] = strconv.Itoa(next)
				path = strings.Join(parts, "/")
			}
			paths = append(paths, renderPrefix+path)
		}
		bindings[metric] = paths
	}
	return result, bindings, nil
}
