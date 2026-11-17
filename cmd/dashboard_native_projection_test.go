package cmd

import (
	"reflect"
	"testing"
)

func nativeProjectionFixture(kind string) map[string]any {
	template := map[string]any{"type": kind, "title": "Energy", "layout_options": map[string]any{"grid_columns": 6}}
	owned := map[string]any{"p13199": []any{"/entity"}}
	if kind == "tile" {
		template["entity"] = "sensor.shared"
	} else {
		template["hours_to_show"] = 24
		template["entities"] = []any{map[string]any{"entity": "sensor.shared", "name": "Consumption"}, map[string]any{"entity": "sensor.shared", "name": "Production"}}
		owned = map[string]any{"p13199": []any{"/entities/0/entity"}, "p13112": []any{"/entities/1/entity"}}
	}
	metrics := map[string]any{
		"p13199": dashboardSourcePolicy{EntityID: "sensor.shared", SelectionOrigin: "automatic", Verification: "unverified", DisplayDecision: "withheld", Reason: "Daily meaning unknown"}.fields(),
		"p13112": dashboardSourcePolicy{EntityID: "sensor.shared", SelectionOrigin: "automatic", Verification: "verified", DisplayDecision: "normal", Reason: "Verified"}.fields(),
	}
	return map[string]any{"mapping_id": "source-plant", "metrics": metrics, "native_template": template, "native_bindings": owned}
}

func TestNativeProjectionWithholdsRestoresAndRebinds(t *testing.T) {
	for _, kind := range []string{"tile", "history-graph", "entities"} {
		t.Run(kind, func(t *testing.T) {
			source := nativeProjectionFixture(kind)
			blocked, bindings, err := renderDashboardNativeProjection(source)
			if err != nil {
				t.Fatal(err)
			}
			if stringValue(blocked["type"]) != "vertical-stack" {
				t.Fatalf("missing review explanation: %#v", blocked)
			}
			if kind != "tile" {
				rows, ok := getDashboardJSONPointer(blocked, "/cards/1/entities")
				if !ok || len(rows.([]any)) != 1 || rows.([]any)[0].(map[string]any)["name"] != "Production" {
					t.Fatalf("lost unrelated series: %#v", blocked)
				}
				paths := bindings["p13112"].([]any)
				if paths[len(paths)-1] != "/cards/1/entities/0/entity" {
					t.Fatalf("did not regenerate shifted rendered path: %#v", paths)
				}
			}
			beforeHash, err := hashDashboardStructure(blocked)
			if err != nil {
				t.Fatal(err)
			}
			policy := source["metrics"].(map[string]any)["p13199"].(map[string]any)
			policy["entity_id"], policy["verification"], policy["display_decision"] = "sensor.verified_daily", "verified", "normal"
			for _, raw := range source["native_bindings"].(map[string]any)["p13199"].([]any) {
				setDashboardJSONPointer(source["native_template"], raw.(string), "sensor.verified_daily")
			}
			restored, paths, err := renderDashboardNativeProjection(source)
			if err != nil {
				t.Fatal(err)
			}
			if restored["type"] != kind || restored["title"] != "Energy" || restored["layout_options"] == nil {
				t.Fatalf("did not restore native structure: %#v", restored)
			}
			for _, raw := range paths["p13199"].([]any) {
				if entity, ok := getDashboardJSONPointer(restored, raw.(string)); !ok || entity != "sensor.verified_daily" {
					t.Fatalf("binding %v is not current", raw)
				}
			}
			afterHash, err := hashDashboardStructure(restored)
			if err != nil || beforeHash != afterHash {
				t.Fatalf("source-only projection changed ownership: %s != %s (%v)", beforeHash, afterHash, err)
			}
			policy["display_decision"], policy["verification"] = "withheld", "incompatible"
			again, _, err := renderDashboardNativeProjection(source)
			if err != nil || again["type"] != "vertical-stack" {
				t.Fatalf("cannot withhold restored source: %#v %v", again, err)
			}
		})
	}
}

func TestNativeProjectionRejectsInvalidOwnershipWithoutMutation(t *testing.T) {
	for _, change := range []func(map[string]any){
		func(s map[string]any) { delete(s, "native_template") },
		func(s map[string]any) { delete(s, "native_bindings") },
		func(s map[string]any) { s["native_bindings"].(map[string]any)["p13199"] = []any{"/views/1/entity"} },
		func(s map[string]any) { s["native_bindings"].(map[string]any)["p13199"] = []any{"/entities/1/entity"} },
		func(s map[string]any) { s["native_bindings"].(map[string]any)["p13199"] = []any{"/entities/9/entity"} },
		func(s map[string]any) {
			s["metrics"].(map[string]any)["p13199"].(map[string]any)["entity_id"] = "sensor.other_target"
		},
		func(s map[string]any) { s["native_template"].(map[string]any)["source_policy"] = map[string]any{} },
	} {
		source := nativeProjectionFixture("history-graph")
		change(source)
		before, _ := deepCopyJSONValue(source)
		if _, _, err := renderDashboardNativeProjection(source); err == nil {
			t.Fatalf("accepted malformed metadata: %#v", source)
		}
		after, _ := deepCopyJSONValue(source)
		if !reflect.DeepEqual(before, after) {
			t.Fatal("failed projection mutated input")
		}
	}
}

func TestNativeProjectionOwnershipDetectsLayoutEdits(t *testing.T) {
	source := nativeProjectionFixture("history-graph")
	card, _, err := renderDashboardNativeProjection(source)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := hashDashboardStructure(card)
	if err != nil {
		t.Fatal(err)
	}
	card["cards"].([]any)[1].(map[string]any)["title"] = "User title"
	if _, err := hashDashboardStructure(card); err == nil {
		t.Fatal("rendered layout edit hidden by template normalization")
	}
	card, _, _ = renderDashboardNativeProjection(source)
	source["native_template"].(map[string]any)["title"] = "Changed template"
	card, _, _ = renderDashboardNativeProjection(source)
	changed, err := hashDashboardStructure(card)
	if err != nil || changed == baseline {
		t.Fatalf("template layout edit was hidden: %v", err)
	}
}
