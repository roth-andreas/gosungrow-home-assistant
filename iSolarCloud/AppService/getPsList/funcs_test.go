package getPsList

import (
	"reflect"
	"testing"

	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/Common"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct/valueTypes"
)

func testDevice(psID string, name string, serial string) Common.Device {
	return Common.Device{
		PsId:        valueTypes.SetPsIdString(psID),
		PsName:      valueTypes.SetStringValue(name),
		PsShortName: valueTypes.SetStringValue(serial),
	}
}

func TestGetPsIdsKeepsCompositeIdentifiers(t *testing.T) {
	result := ResultData{
		PageList: []Common.Device{
			testDevice("5520557_5520558", "Hybrid Plant", "H-1"),
			testDevice("5520559", "Numeric Plant", "N-1"),
			testDevice("--", "Placeholder", "P-1"),
			testDevice("", "Empty", "E-1"),
		},
	}

	got := result.GetPsIds()
	if len(got) != 2 {
		t.Fatalf("expected 2 ps_ids, got %d", len(got))
	}
	if got[0].String() != "5520557_5520558" {
		t.Fatalf("expected composite ps_id first, got %q", got[0].String())
	}
	if got[1].String() != "5520559" {
		t.Fatalf("expected numeric ps_id second, got %q", got[1].String())
	}
}

func TestGetPsNameAndSerialUseUsablePsIds(t *testing.T) {
	result := ResultData{
		PageList: []Common.Device{
			testDevice("5520557_5520558", "Hybrid Plant", "H-1"),
			testDevice("5520559", "Numeric Plant", "N-1"),
			testDevice("--", "Placeholder", "P-1"),
		},
	}

	if diff := reflect.DeepEqual(result.GetPsName(), []string{"Hybrid Plant", "Numeric Plant"}); !diff {
		t.Fatalf("unexpected ps names: %#v", result.GetPsName())
	}
	if diff := reflect.DeepEqual(result.GetPsSerial(), []string{"H-1", "N-1"}); !diff {
		t.Fatalf("unexpected ps serials: %#v", result.GetPsSerial())
	}
}
