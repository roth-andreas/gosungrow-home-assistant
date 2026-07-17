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
		"view_overview":              "Overview",
		"view_aggregates":            "Aggregates",
		"view_trends":                "Trends",
		"view_data_sources":          "Data Sources",
		"source_title":               "Data Sources",
		"source_subtitle":            "Review automatic matches or choose a dashboard override.",
		"source_automatic":           "Automatic",
		"source_manual":              "Manual",
		"source_needs_review":        "Needs review",
		"source_configure":           "Configure",
		"source_recommended":         "Recommended",
		"source_other":               "Other compatible entities",
		"source_search":              "Search entities",
		"source_use":                 "Use this source",
		"source_reset":               "Reset to automatic",
		"source_cancel":              "Cancel",
		"source_saved":               "Data source saved.",
		"source_readonly":            "Only Home Assistant administrators can change data sources.",
		"source_unavailable_warning": "The selected entity is unavailable or non-numeric.",
		"source_stale_warning":       "The selected entity has not updated recently.",
		"source_physical_warning":    "Selected value ({value}) exceeds solar production ({reference}). Review this source.",
		"source_confirm_warning":     "Use this source anyway?",
		"source_stale":               "Dashboard changed; reload and try again.",
		"source_incompatible":        "This entity is no longer an available compatible source.",
		"source_save_error":          "Could not save the data source. Check your administrator access and connection, then try again.",
		"source_confidence_high":     "High confidence",
		"source_confidence_medium":   "Medium confidence",
		"source_confidence_low":      "Low confidence",
		"source_confidence_manual":   "User selected",
		"source_confidence_unknown":  "Confidence unavailable",
		"source_compatible":          "Compatible source",
		"source_group_live":          "Live power",
		"source_group_today":         "Today's energy",
		"source_group_battery":       "Battery",
		"source_group_summary":       "Energy summary",
		"source_metric_pv_power":     "Solar power", "source_metric_load_power": "Home consumption power", "source_metric_grid_power": "Grid power", "source_metric_battery_power": "Battery power",
		"source_metric_p13141": "Battery state of charge", "source_metric_pv_to_load_power": "Solar to home power", "source_metric_pv_to_battery_power": "Solar to battery power",
		"source_metric_pv_to_grid_power": "Solar to grid power", "source_metric_grid_to_load_power": "Grid to home power", "source_metric_battery_to_load_power": "Battery to home power",
		"source_metric_p13112": "Solar production today", "source_metric_p13116": "Direct solar consumption", "source_metric_p13174": "Solar energy to battery",
		"source_metric_p13173": "Grid export today", "source_metric_p13147": "Grid import today", "source_metric_p13199": "Home consumption today", "source_metric_p13029": "Battery discharge today",
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
		"heading_energy_summary":   "Energy Summary",
		"name_pv_yield":            "PV Yield",
		"name_pv_to_load":          "PV To Load",
		"name_pv_to_battery":       "PV To Battery",
		"name_pv_to_grid":          "PV To Grid",
		"name_feed_in":             "Feed-in",
		"name_grid_import":         "Grid Import",
		"name_production":          "Production",
		"name_consumption":         "Consumption",
		"name_to_grid":             "To Grid",
		"name_from_grid":           "From Grid",
		"name_to_battery":          "To Battery",
		"name_from_battery":        "From Battery",
		"name_grid_to_load":        "Grid To Load",
		"name_grid_net":            "Grid Net",
		"name_battery_soc":         "Battery SOC",
		"name_battery_power":       "Battery Power",
		"name_charge":              "Charge",
		"name_discharge":           "Discharge",
		"period_day":               "Day",
		"period_month":             "Month",
		"period_year":              "Year",
		"unavailable":              "Unavailable",
		"statistics_unavailable":   "Statistics unavailable",
		"no_statistics":            "No statistics yet",
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
