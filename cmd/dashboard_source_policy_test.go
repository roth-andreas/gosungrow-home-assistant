package cmd

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

type dashboardSnapshotReader struct {
	config map[string]any
	err    error
}

func (r dashboardSnapshotReader) GetConfig(context.Context, string) (map[string]any, error) {
	return r.config, r.err
}

func TestReconciliationRechecksConcurrentManualChoiceBeforeSaving(t *testing.T) {
	original := map[string]any{"overrides": map[string]any{"p13199": "sensor.old"}}
	changed := map[string]any{"overrides": map[string]any{"p13199": "sensor.new_manual"}}
	if err := verifyDashboardConfigSnapshot(context.Background(), dashboardSnapshotReader{config: original}, "gosungrow-flow", original); err != nil {
		t.Fatal(err)
	}
	err := verifyDashboardConfigSnapshot(context.Background(), dashboardSnapshotReader{config: changed}, "gosungrow-flow", original)
	var conflict *dashboardConfigChangedError
	if !errors.As(err, &conflict) {
		t.Fatalf("concurrent manual source was not protected: %v", err)
	}
	if changed["overrides"].(map[string]any)["p13199"] != "sensor.new_manual" {
		t.Fatal("conflict check changed the saved choice")
	}
	if err := verifyDashboardConfigSnapshot(context.Background(), dashboardSnapshotReader{err: errors.New("connection reset")}, "gosungrow-flow", original); err == nil {
		t.Fatal("failed read authorized an overwrite")
	}
}

func TestDashboardSourceMeaningIsIndependentOfLiveAvailability(t *testing.T) {
	target := dailyDashboardTarget()
	now := time.Now()
	for _, test := range []struct {
		name, identity, value, unit, verification, automatic, manual string
	}{
		{"unverified MWh magnitude", "GoSungrow-virtual-100_11_0_0-total_energy", "9.92", "MWh", "unverified", "withheld", "warned"},
		{"verified genuine zero", "GoSungrow-virtual-100_11_0_0-p13199", "0", "kWh", "verified", "normal", "normal"},
		{"known meaning missing current reading", "GoSungrow-virtual-100_11_0_0-p13199", "unavailable", "kWh", "verified", "normal", "normal"},
		{"known opposite direction", "GoSungrow-virtual-100_11_0_0-p13173", "4", "kWh", "incompatible", "withheld", "withheld"},
		{"foreign plant", "GoSungrow-virtual-999_11_0_0-p13199", "4", "kWh", "incompatible", "withheld", "withheld"},
		{"wrong physical quantity", "GoSungrow-virtual-100_11_0_0-p13199", "4", "W", "incompatible", "withheld", "withheld"},
		{"missing unit evidence", "GoSungrow-virtual-100_11_0_0-p13199", "unavailable", "", "unverified", "withheld", "warned"},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := dashboardTestCanonicalState("sensor.renamed_consumption_2", test.identity, test.value, test.unit)
			for _, manual := range []bool{false, true} {
				policy := assessDashboardSource(target, "p13199", state.EntityID, manual, []haState{state}, now)
				decision := test.automatic
				if manual {
					decision = test.manual
				}
				if policy.EntityID != state.EntityID || policy.Verification != test.verification || policy.DisplayDecision != decision {
					t.Fatalf("manual=%t: %#v", manual, policy)
				}
				if strings.Contains(policy.Reason, "9.92") || strings.Contains(strings.ToLower(policy.Reason), "lifetime") {
					t.Fatal("magnitude was treated as semantic evidence")
				}
			}
		})
	}
}

func TestSavedManualSelectionSurvivesFallbackAndPromotionBeforeStateFileUpdate(t *testing.T) {
	target := dailyDashboardTarget()
	id := dashboardSourceMappingID(target)
	entity := "sensor.gosungrow_100_total_energy"
	current := map[string]any{"views": []any{map[string]any{"cards": []any{map[string]any{
		"type": dashboardSourceMappingCardType, "schema_version": 1, "matcher_version": 4, "mapping_id": id,
		"defaults": map[string]any{"p13199": entity}, "pinned_defaults": map[string]any{"p13199": entity}, "overrides": map[string]any{"p13199": entity},
	}}}}}
	states := []haState{dashboardTestCanonicalState(entity, "GoSungrow-virtual-100_11_0_0-total_energy", "9.92", "MWh")}
	config, accepted, report := composeDashboardSources(dailyDashboardTemplate(t, []haDashboardTarget{target}), current, nil, []haDashboardTarget{target}, states, "gosungrow-flow", defaultDashboardLocaleBundle)
	if report.SourcePolicyError != nil {
		t.Fatal(report.SourcePolicyError)
	}
	if accepted[id]["p13199"] != entity {
		t.Fatalf("saved manual selection was discarded: %#v", accepted)
	}
	trace := dailyTrace(t, report, target.PsKey, "p13199")
	if trace.Resolution.Policy.DisplayDecision != "warned" || trace.Source != "manual" {
		t.Fatalf("missing persistent manual warning: %#v", trace)
	}
	fallback, err := nativeDashboardFallback(config, defaultDashboardLocaleBundle)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(extractDashboardSourceOverrides(fallback), accepted) {
		t.Fatal("native fallback lost the saved override")
	}
	var notice map[string]any
	var walk func(any)
	walk = func(value any) {
		switch typed := value.(type) {
		case map[string]any:
			if metadata, ok := typed["gosungrow_source_mapping"].(map[string]any); ok {
				notice = metadata
				return
			}
			for _, child := range typed {
				walk(child)
			}
		case []any:
			for _, child := range typed {
				walk(child)
			}
		}
	}
	walk(fallback)
	if notice == nil || len(notice) != 8 {
		t.Fatalf("invalid retained notice: %#v", notice)
	}
	for _, forbidden := range []string{"bindings", "candidates", "source_policy", "native_template", "native_bindings"} {
		if _, exists := notice[forbidden]; exists {
			t.Fatal("notice retained stale or transient metadata: " + forbidden)
		}
	}
	promoted, _, next := composeDashboardSources(dailyDashboardTemplate(t, []haDashboardTarget{target}), fallback, nil, []haDashboardTarget{target}, states, "gosungrow-flow", defaultDashboardLocaleBundle)
	if next.SourcePolicyError != nil {
		t.Fatal(next.SourcePolicyError)
	}
	if extractDashboardSourceOverrides(promoted)[id]["p13199"] != entity {
		t.Fatal("promotion replaced the accepted identity")
	}
	// A reset in current configuration wins over the older diagnostic copy.
	findDashboardSourceMappingCard(promoted, id)["overrides"] = map[string]any{}
	_, reset, _ := composeDashboardSources(dailyDashboardTemplate(t, []haDashboardTarget{target}), promoted, accepted, []haDashboardTarget{target}, states, "gosungrow-flow", defaultDashboardLocaleBundle)
	if reset[id]["p13199"] != "" {
		t.Fatal("diagnostic state resurrected a reset override")
	}
}

func TestSourcePolicyCompositionHasDeterministicNativeBindings(t *testing.T) {
	target := dailyDashboardTarget()
	states := []haState{dashboardTestCanonicalState("sensor.renamed_yield", "GoSungrow-virtual-100_11_0_0-p13112", "0", "kWh")}
	var baseline string
	for i := 0; i < 15; i++ {
		config, _, report := composeDashboardSources(dailyDashboardTemplate(t, []haDashboardTarget{target}), nil, nil, []haDashboardTarget{target}, states, "gosungrow-flow", defaultDashboardLocaleBundle)
		if report.SourcePolicyError != nil {
			t.Fatal(report.SourcePolicyError)
		}
		hash, err := hashCanonicalJSON(config)
		if err != nil {
			t.Fatal(err)
		}
		if i > 0 && hash != baseline {
			t.Fatal("unchanged composition produced different bindings")
		}
		baseline = hash
	}
}
