package queryDeviceList

import (
	"testing"

	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct"
)

func TestVirtualPointBuildersSkipMissingSourcePoints(t *testing.T) {
	endpoint := EndPoint{}
	entries := api.NewDataMap()
	epp := GoStruct.NewEndPointPath("virtual", "100_14_1_1")

	assertDoesNotPanic(t, "SetBatteryPoints", func() {
		endpoint.SetBatteryPoints(epp, entries)
	})
	assertDoesNotPanic(t, "SetPvPoints", func() {
		endpoint.SetPvPoints(epp, entries)
	})
	assertDoesNotPanic(t, "SetGridPoints", func() {
		endpoint.SetGridPoints(epp, entries)
	})
	assertDoesNotPanic(t, "SetLoadPoints", func() {
		endpoint.SetLoadPoints(epp, entries)
	})
}

func assertDoesNotPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("%s panicked with missing source points: %v", name, recovered)
		}
	}()
	fn()
}
