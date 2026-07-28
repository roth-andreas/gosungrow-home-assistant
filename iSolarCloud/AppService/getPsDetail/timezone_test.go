package getPsDetail

import (
	"testing"
	"time"

	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct/valueTypes"
)

func TestPlantTimezoneLocation(t *testing.T) {
	tests := []struct {
		name, zone string
		when       time.Time
		offset     int
	}{
		{name: "Brisbane", zone: "Australia/Brisbane", when: time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC), offset: 10 * 3600},
		{name: "Berlin summer", zone: "Europe/Berlin", when: time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC), offset: 2 * 3600},
		{name: "UTC", zone: "UTC", when: time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC), offset: 0},
		{name: "fractional", zone: "GMT+05:30", when: time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC), offset: 5*3600 + 30*60},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			location, err := plantTimezoneLocation(tt.zone)
			if err != nil {
				t.Fatal(err)
			}
			_, got := tt.when.In(location).Zone()
			if got != tt.offset {
				t.Fatalf("offset=%d want=%d", got, tt.offset)
			}
		})
	}
	for _, invalid := range []string{"", "GMT+25", "not/a-zone"} {
		if _, err := plantTimezoneLocation(invalid); err == nil {
			t.Fatalf("expected timezone %q to fail", invalid)
		}
	}
}

func TestNormalizePlantDataLastUpdateTimePreservesWallClockAndAddsOffset(t *testing.T) {
	var timezone valueTypes.String
	timezone.SetString("Australia/Brisbane")
	data := ResultData{
		DataLastUpdateTime: valueTypes.SetDateTimeString("2026-07-28 02:35:00"),
		Timezone:           timezone,
	}
	if err := normalizePlantDataLastUpdateTime(&data); err != nil {
		t.Fatal(err)
	}
	if got := data.DataLastUpdateTime.Format(time.RFC3339); got != "2026-07-28T02:35:00+10:00" {
		t.Fatalf("timestamp=%s", got)
	}
	if got := data.DataLastUpdateTime.UTC().Format(time.RFC3339); got != "2026-07-27T16:35:00Z" {
		t.Fatalf("instant=%s", got)
	}
	if got := valueTypes.AnyToValueString(data.DataLastUpdateTime, 0, valueTypes.DateTimeFullLayout); got != "2026-07-28T02:35:00+10:00" {
		t.Fatalf("published value=%s", got)
	}
}

func TestNormalizePlantDataLastUpdateTimeAcrossPlantZones(t *testing.T) {
	tests := []struct {
		name, zone, wallClock, want string
	}{
		{name: "Berlin DST", zone: "Europe/Berlin", wallClock: "2026-07-28 02:35:00", want: "2026-07-28T02:35:00+02:00"},
		{name: "UTC", zone: "UTC", wallClock: "2026-01-28 02:35:00", want: "2026-01-28T02:35:00Z"},
		{name: "fractional offset", zone: "UTC+05:45", wallClock: "2026-01-28 02:35:00", want: "2026-01-28T02:35:00+05:45"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var timezone valueTypes.String
			timezone.SetString(tt.zone)
			data := ResultData{DataLastUpdateTime: valueTypes.SetDateTimeString(tt.wallClock), Timezone: timezone}
			if err := normalizePlantDataLastUpdateTime(&data); err != nil {
				t.Fatal(err)
			}
			if got := data.DataLastUpdateTime.Format(time.RFC3339); got != tt.want {
				t.Fatalf("timestamp=%s want=%s", got, tt.want)
			}
		})
	}
}

func TestNormalizePlantDataLastUpdateTimePreservesValueWhenTimezoneInvalid(t *testing.T) {
	data := ResultData{DataLastUpdateTime: valueTypes.SetDateTimeString("2026-07-28 02:35:00")}
	want := data.DataLastUpdateTime.Time
	if err := normalizePlantDataLastUpdateTime(&data); err == nil {
		t.Fatal("expected missing timezone error")
	}
	if !data.DataLastUpdateTime.Time.Equal(want) || data.DataLastUpdateTime.Location() != want.Location() {
		t.Fatalf("timestamp changed: %s != %s", data.DataLastUpdateTime.Time, want)
	}
}
