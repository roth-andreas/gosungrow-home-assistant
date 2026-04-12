package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	dashboardLocaleDir = "dashboard-locales"
)

type dashboardLocaleBundle struct {
	Dashboard map[string]string `yaml:"dashboard"`
	FlowCard  map[string]string `yaml:"flow_card"`
}

var defaultDashboardLocaleBundle = dashboardLocaleBundle{
	Dashboard: map[string]string{
		"view_overview":            "Overview",
		"view_trends":              "Trends",
		"heading_live_flow":        "Live Flow",
		"heading_today":            "Today",
		"heading_power_balance":    "Power Balance",
		"heading_battery":          "Battery",
		"heading_daily_energy":     "Daily Energy",
		"heading_solar_allocation": "Solar Allocation",
		"heading_load_sources":     "Load Sources",
		"heading_grid_exchange":    "Grid Exchange",
		"heading_battery_flow":     "Battery Flow",
		"heading_battery_soc":      "Battery SOC",
		"name_pv_yield":            "PV Yield",
		"name_pv_to_load":          "PV To Load",
		"name_pv_to_battery":       "PV To Battery",
		"name_pv_to_grid":          "PV To Grid",
		"name_feed_in":             "Feed-in",
		"name_grid_import":         "Grid Import",
		"name_grid_to_load":        "Grid To Load",
		"name_grid_net":            "Grid Net",
		"name_battery_soc":         "Battery SOC",
		"name_battery_power":       "Battery Power",
		"name_charge":              "Charge",
		"name_discharge":           "Discharge",
		"name_pv":                  "PV",
		"name_load":                "Load",
		"name_grid":                "Grid",
		"name_battery":             "Battery",
	},
	FlowCard: map[string]string{
		"node_pv":      "PV",
		"node_grid":    "Grid",
		"node_home":    "Home",
		"node_battery": "Battery",
	},
}

func dashboardLocaleCandidates(language string) []string {
	normalized := strings.ToLower(strings.TrimSpace(language))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	if normalized == "" {
		return []string{"en"}
	}

	seen := map[string]struct{}{}
	candidates := make([]string, 0, 4)
	appendCandidate := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		candidates = append(candidates, value)
	}

	appendCandidate(normalized)
	if idx := strings.Index(normalized, "-"); idx > 0 {
		appendCandidate(normalized[:idx])
	}
	appendCandidate("en")
	return candidates
}

func localizedDashboardBundle(assetDir string, language string) (dashboardLocaleBundle, string, error) {
	candidates := dashboardLocaleCandidates(language)
	for _, candidate := range candidates {
		path := filepath.Join(assetDir, dashboardLocaleDir, candidate+".yaml")
		content, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return dashboardLocaleBundle{}, "", err
		}

		var bundle dashboardLocaleBundle
		if err := yaml.Unmarshal(content, &bundle); err != nil {
			return dashboardLocaleBundle{}, "", fmt.Errorf("parse dashboard locale %q: %w", candidate, err)
		}

		merged := mergeDashboardLocale(defaultDashboardLocaleBundle, bundle)
		return merged, candidate, nil
	}

	return defaultDashboardLocaleBundle, "en", nil
}

func mergeDashboardLocale(base dashboardLocaleBundle, override dashboardLocaleBundle) dashboardLocaleBundle {
	ret := dashboardLocaleBundle{
		Dashboard: make(map[string]string, len(base.Dashboard)),
		FlowCard:  make(map[string]string, len(base.FlowCard)),
	}
	for key, value := range base.Dashboard {
		ret.Dashboard[key] = value
	}
	for key, value := range base.FlowCard {
		ret.FlowCard[key] = value
	}
	for key, value := range override.Dashboard {
		value = strings.TrimSpace(value)
		if value != "" {
			ret.Dashboard[key] = value
		}
	}
	for key, value := range override.FlowCard {
		value = strings.TrimSpace(value)
		if value != "" {
			ret.FlowCard[key] = value
		}
	}
	return ret
}

func dashboardReplacementMap(bundle dashboardLocaleBundle) map[string]string {
	ret := make(map[string]string, len(defaultDashboardLocaleBundle.Dashboard))
	for key, english := range defaultDashboardLocaleBundle.Dashboard {
		localized := strings.TrimSpace(bundle.Dashboard[key])
		if localized == "" {
			localized = english
		}
		ret[english] = localized
	}
	return ret
}

func localizeDashboardValue(value any, replacements map[string]string) any {
	switch typed := value.(type) {
	case map[string]any:
		ret := make(map[string]any, len(typed))
		for key, entry := range typed {
			ret[key] = localizeDashboardValue(entry, replacements)
		}
		return ret
	case []any:
		ret := make([]any, 0, len(typed))
		for _, entry := range typed {
			ret = append(ret, localizeDashboardValue(entry, replacements))
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

func injectFlowCardLabels(value any, labels map[string]string) any {
	switch typed := value.(type) {
	case map[string]any:
		ret := make(map[string]any, len(typed))
		for key, entry := range typed {
			ret[key] = injectFlowCardLabels(entry, labels)
		}
		if cardType, ok := ret["type"].(string); ok && cardType == "custom:gosungrow-energy-flow-card-v2" {
			ret["labels"] = copyStringMap(labels)
		}
		return ret
	case []any:
		ret := make([]any, 0, len(typed))
		for _, entry := range typed {
			ret = append(ret, injectFlowCardLabels(entry, labels))
		}
		return ret
	default:
		return value
	}
}

func copyStringMap(values map[string]string) map[string]string {
	ret := make(map[string]string, len(values))
	for key, value := range values {
		ret[key] = value
	}
	return ret
}

func sortedStringKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
