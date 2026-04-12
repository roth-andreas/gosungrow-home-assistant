package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLocalizedDashboardBundleFallbackToEnglish(t *testing.T) {
	assetDir := filepath.Join("..", "addon", "gosungrow", "assets")

	bundle, locale, err := localizedDashboardBundle(assetDir, "fr-FR")
	if err != nil {
		t.Fatalf("localizedDashboardBundle: %v", err)
	}
	if locale != "en" {
		t.Fatalf("expected english fallback locale, got %q", locale)
	}
	if bundle.Dashboard["heading_live_flow"] != defaultDashboardLocaleBundle.Dashboard["heading_live_flow"] {
		t.Fatalf("unexpected fallback heading: %q", bundle.Dashboard["heading_live_flow"])
	}
}

func TestLocalizedDashboardBundleSwedish(t *testing.T) {
	assetDir := filepath.Join("..", "addon", "gosungrow", "assets")

	bundle, locale, err := localizedDashboardBundle(assetDir, "sv-SE")
	if err != nil {
		t.Fatalf("localizedDashboardBundle: %v", err)
	}
	if locale != "sv" {
		t.Fatalf("expected swedish locale, got %q", locale)
	}
	if bundle.Dashboard["heading_live_flow"] == defaultDashboardLocaleBundle.Dashboard["heading_live_flow"] {
		t.Fatal("expected localized swedish heading")
	}
}

func TestDashboardLocaleParity(t *testing.T) {
	assetDir := filepath.Join("..", "addon", "gosungrow", "assets", dashboardLocaleDir)
	entries, err := os.ReadDir(assetDir)
	if err != nil {
		t.Fatalf("read locale dir: %v", err)
	}

	base := defaultDashboardLocaleBundle
	baseDashboardKeys := sortedStringKeys(base.Dashboard)
	baseFlowCardKeys := sortedStringKeys(base.FlowCard)

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		content, err := os.ReadFile(filepath.Join(assetDir, entry.Name()))
		if err != nil {
			t.Fatalf("read locale file %q: %v", entry.Name(), err)
		}
		var locale dashboardLocaleBundle
		if err := yaml.Unmarshal(content, &locale); err != nil {
			t.Fatalf("parse locale file %q: %v", entry.Name(), err)
		}

		if got := sortedStringKeys(locale.Dashboard); len(got) != len(baseDashboardKeys) {
			t.Fatalf("locale %q dashboard key count mismatch: got %d want %d", entry.Name(), len(got), len(baseDashboardKeys))
		}
		if got := sortedStringKeys(locale.FlowCard); len(got) != len(baseFlowCardKeys) {
			t.Fatalf("locale %q flow_card key count mismatch: got %d want %d", entry.Name(), len(got), len(baseFlowCardKeys))
		}

		for _, key := range baseDashboardKeys {
			if _, ok := locale.Dashboard[key]; !ok {
				t.Fatalf("locale %q missing dashboard key %q", entry.Name(), key)
			}
		}
		for _, key := range baseFlowCardKeys {
			if _, ok := locale.FlowCard[key]; !ok {
				t.Fatalf("locale %q missing flow_card key %q", entry.Name(), key)
			}
		}
	}
}

func TestAddonOptionTranslationParity(t *testing.T) {
	type optionName struct {
		Name string `yaml:"name"`
	}
	type addonTranslation struct {
		Configuration map[string]optionName `yaml:"configuration"`
	}

	basePath := filepath.Join("..", "addon", "gosungrow", "translations", "en.yaml")
	baseContent, err := os.ReadFile(basePath)
	if err != nil {
		t.Fatalf("read base translation: %v", err)
	}
	var base addonTranslation
	if err := yaml.Unmarshal(baseContent, &base); err != nil {
		t.Fatalf("parse base translation: %v", err)
	}

	dir := filepath.Join("..", "addon", "gosungrow", "translations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read translations dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" || entry.Name() == "en.yaml" {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("read translation %q: %v", entry.Name(), err)
		}
		var candidate addonTranslation
		if err := yaml.Unmarshal(content, &candidate); err != nil {
			t.Fatalf("parse translation %q: %v", entry.Name(), err)
		}
		if len(candidate.Configuration) != len(base.Configuration) {
			t.Fatalf("translation %q key count mismatch: got %d want %d", entry.Name(), len(candidate.Configuration), len(base.Configuration))
		}
		for key := range base.Configuration {
			if _, ok := candidate.Configuration[key]; !ok {
				t.Fatalf("translation %q missing configuration key %q", entry.Name(), key)
			}
		}
	}
}
