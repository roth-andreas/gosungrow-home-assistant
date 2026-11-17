package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/MickMake/GoUnify/Unify"
	"github.com/spf13/cobra"
)

type dashboardTestDeadline struct {
	delay time.Duration
	fire  chan time.Time
}

func dashboardTestReceive[T any](t *testing.T, channel <-chan T) T {
	t.Helper()
	select {
	case value := <-channel:
		return value
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for dashboard worker")
		var zero T
		return zero
	}
}

func TestDashboardCoordinatorSerializesAndKeepsFirstDebounceDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &dashboardCoordinator{changes: make(chan struct{}, 1)}
	deadlines := make(chan dashboardTestDeadline, 16)
	after := func(delay time.Duration) dashboardRuntimeTimer {
		deadline := dashboardTestDeadline{delay: delay, fire: make(chan time.Time, 1)}
		deadlines <- deadline
		return dashboardRuntimeTimer{channel: deadline.fire}
	}
	started, finish := make(chan bool, 4), make(chan error, 4)
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.run(ctx, []time.Duration{30 * time.Second}, after, func(availability bool) error {
			started <- availability
			select {
			case err := <-finish:
				return err
			case <-ctx.Done():
				return ctx.Err()
			}
		}, func(error) {})
	}()
	startup := dashboardTestReceive(t, deadlines)
	if startup.delay != 30*time.Second {
		t.Fatal(startup.delay)
	}
	c.changes <- struct{}{}
	first := dashboardTestReceive(t, deadlines)
	if first.delay != 5*time.Second {
		t.Fatal(first.delay)
	}
	c.changes <- struct{}{}
	first.fire <- time.Now()
	if !dashboardTestReceive(t, started) {
		t.Fatal("availability trigger lost")
	}
	c.changes <- struct{}{}
	second := dashboardTestReceive(t, deadlines)
	second.fire <- time.Now()
	startup.fire <- time.Now() // Scheduled and event work still use one slot.
	finish <- nil
	_ = dashboardTestReceive(t, started)
	select {
	case <-started:
		t.Fatal("concurrent reconciliation")
	default:
	}
	cancel()
	dashboardTestReceive(t, done)
}

func TestDashboardCoordinatorBoundsRetriesAndAcceptsNewTransition(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &dashboardCoordinator{changes: make(chan struct{}, 1)}
	deadlines := make(chan dashboardTestDeadline, 16)
	after := func(delay time.Duration) dashboardRuntimeTimer {
		d := dashboardTestDeadline{delay, make(chan time.Time, 1)}
		deadlines <- d
		return dashboardRuntimeTimer{channel: d.fire}
	}
	failures := make(chan error, 8)
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.run(ctx, nil, after, func(bool) error { return errors.New("i/o timeout") }, func(err error) { failures <- err })
	}()
	c.changes <- struct{}{}
	for _, delay := range []time.Duration{5 * time.Second, 30 * time.Second, 90 * time.Second, 300 * time.Second} {
		deadline := dashboardTestReceive(t, deadlines)
		if deadline.delay != delay {
			t.Fatalf("deadline = %s, want %s", deadline.delay, delay)
		}
		deadline.fire <- time.Now()
		dashboardTestReceive(t, failures)
	}
	c.changes <- struct{}{}
	deadline := dashboardTestReceive(t, deadlines)
	if deadline.delay != 5*time.Second {
		t.Fatalf("new transition did not start a new bounded attempt: %s", deadline.delay)
	}
	cancel()
	dashboardTestReceive(t, done)
}

func TestPendingTransitionKeepsItsDeadlineAcrossScheduledFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &dashboardCoordinator{changes: make(chan struct{})}
	deadlines := make(chan dashboardTestDeadline, 8)
	cleared := make(chan time.Duration, 8)
	after := func(delay time.Duration) dashboardRuntimeTimer {
		deadline := dashboardTestDeadline{delay: delay, fire: make(chan time.Time, 1)}
		deadlines <- deadline
		return dashboardRuntimeTimer{channel: deadline.fire, stop: func() bool { cleared <- delay; return true }}
	}
	started, finish := make(chan bool, 2), make(chan error)
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.run(ctx, []time.Duration{30 * time.Second}, after, func(availability bool) error {
			started <- availability
			select {
			case err := <-finish:
				return err
			case <-ctx.Done():
				return ctx.Err()
			}
		}, func(error) {})
	}()
	startup := dashboardTestReceive(t, deadlines)
	startup.fire <- time.Now()
	if dashboardTestReceive(t, started) {
		t.Fatal("scheduled work was mislabeled")
	}
	dashboardTestReceive(t, cleared)
	c.changes <- struct{}{}
	window := dashboardTestReceive(t, deadlines)
	if window.delay != 5*time.Second {
		t.Fatal(window.delay)
	}
	window.fire <- time.Now()
	dashboardTestReceive(t, cleared) // The first window elapsed while work ran.
	c.changes <- struct{}{}          // A later event must not extend that ready window.
	finish <- errors.New("i/o timeout")
	if !dashboardTestReceive(t, started) {
		t.Fatal("pending availability work was discarded")
	}
	select {
	case deadline := <-deadlines:
		t.Fatalf("ready follow-up was delayed again: %s", deadline.delay)
	default:
	}
	cancel()
	dashboardTestReceive(t, done)
}

func TestDashboardEligibilityIgnoresNumericUpdatesAndTracksExpiry(t *testing.T) {
	target := haDashboardTarget{PsID: "100", PsKey: "100_55_1_1"}
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	state := haState{EntityID: "sensor.renamed_pv_2", RegistryUniqueID: dashboardPlantPVUniqueID(target), RegistryObserved: true, State: "2.4", LastUpdated: now.Add(-29 * time.Minute).Format(time.RFC3339), Attributes: map[string]any{"unit_of_measurement": "kW"}}
	before := dashboardEligibilityViewAt([]haDashboardTarget{target}, []haState{state}, now)
	state.State = "4.2"
	state.LastUpdated = now.Format(time.RFC3339)
	updated := dashboardEligibilityViewAt([]haDashboardTarget{target}, []haState{state}, now)
	if !reflect.DeepEqual(before, updated) {
		t.Fatal("ordinary numeric update changed eligibility")
	}
	expired := dashboardEligibilityViewAt([]haDashboardTarget{target}, []haState{state}, now.Add(31*time.Minute))
	if reflect.DeepEqual(updated, expired) {
		t.Fatal("freshness expiry did not change eligibility")
	}
	state.State = "unavailable"
	if reflect.DeepEqual(updated, dashboardEligibilityViewAt([]haDashboardTarget{target}, []haState{state}, now)) {
		t.Fatal("unavailable source did not change eligibility")
	}
}

func TestDashboardTargetSnapshotsDoNotShareMutableInventory(t *testing.T) {
	c := &dashboardCoordinator{updates: make(chan struct{}, 1)}
	original := []haDashboardTarget{{PsID: "opaque.plant", PlantDevices: []dashboardPlantDevice{{PsKey: "opaque_device"}}}}
	c.updateTargets(original)
	original[0].PlantDevices[0].PsKey = "changed"
	snapshot := c.snapshotTargets()
	snapshot[0].PlantDevices[0].PsKey = "also changed"
	if c.snapshotTargets()[0].PlantDevices[0].PsKey != "opaque_device" {
		t.Fatal("runtime inventory aliases the MQTT discovery owner")
	}
	command := &CmdMqtt{DashboardEnabled: false, DashboardForceUpdate: true}
	command.startDashboardRuntime(context.Background())()
	if command.dashboardRuntime != nil {
		t.Fatal("force permission enabled dashboard work")
	}
}

func TestDashboardBooleanPrecedenceThroughCLIFramework(t *testing.T) {
	for _, scenario := range []struct {
		name, persisted, environment, flag string
		want, invalid                      bool
	}{
		{name: "default"},
		{name: "persisted", persisted: "true", want: true},
		{name: "environment false", persisted: "true", environment: "false"},
		{name: "flag false", persisted: "true", environment: "true", flag: "false"},
		{name: "flag true", environment: "false", flag: "true", want: true},
		{name: "invalid environment", environment: "wrong", invalid: true},
		{name: "invalid persisted", persisted: "wrong", invalid: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			previous := cmds
			t.Cleanup(func() { cmds = previous })
			framework := Unify.New(Unify.Options{BinaryName: "GoSungrow", BinaryVersion: "test", EnvPrefix: "GOSUNGROW"}, Unify.Flags{})
			path := filepath.Join(t.TempDir(), "config.json")
			config := map[string]string{}
			if scenario.persisted != "" {
				config["dashboard-enabled"] = scenario.persisted
			}
			data, _ := json.Marshal(config)
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			framework.Commands.CmdConfig.SetPath(path)
			command := NewCmdMqtt("")
			command.AttachFlags(framework.GetCmd(), framework.GetViper())
			cmds.Unify, cmds.Mqtt = framework, command
			called := false
			framework.GetCmd().AddCommand(&cobra.Command{Use: "probe", RunE: func(*cobra.Command, []string) error {
				called = true
				if command.DashboardEnabled != scenario.want {
					t.Fatalf("dashboard-enabled = %t", command.DashboardEnabled)
				}
				return nil
			}})
			if scenario.environment != "" {
				t.Setenv("GOSUNGROW_DASHBOARD_ENABLED", scenario.environment)
			}
			args := []string{"probe", "--config=" + path}
			if scenario.flag != "" {
				args = append(args, "--dashboard-enabled="+scenario.flag)
			}
			framework.GetCmd().SetArgs(args)
			err := framework.Execute()
			if scenario.invalid {
				if err == nil || called {
					t.Fatalf("invalid Boolean reached runtime: %v", err)
				}
			} else if err != nil || !called {
				t.Fatalf("CLI failed: %v", err)
			}
		})
	}
}

func TestDashboardConfigWritePersistsExplicitFalse(t *testing.T) {
	previous := cmds
	t.Cleanup(func() { cmds = previous })
	framework := Unify.New(Unify.Options{BinaryName: "GoSungrow", BinaryVersion: "test", EnvPrefix: "GOSUNGROW"}, Unify.Flags{})
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"dashboard-enabled":true,"dashboard-force-update":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	framework.Commands.CmdConfig.SetPath(path)
	command := NewCmdMqtt("")
	command.AttachFlags(framework.GetCmd(), framework.GetViper())
	cmds.Unify, cmds.Mqtt = framework, command
	t.Setenv("GOSUNGROW_DASHBOARD_ENABLED", "true")
	t.Setenv("GOSUNGROW_DASHBOARD_FORCE_UPDATE", "true")
	framework.GetCmd().SetArgs([]string{"config", "write", "--config=" + path, "--dashboard-enabled=false", "--dashboard-force-update=false"})
	if err := framework.Execute(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]any
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"dashboard-enabled", "dashboard-force-update"} {
		if saved[key] != "false" && saved[key] != false {
			t.Fatalf("explicit false lost in config write: %s=%v", key, saved[key])
		}
	}
}

func TestDashboardStartupDelayValidation(t *testing.T) {
	want := []time.Duration{30 * time.Second, 90 * time.Second, 180 * time.Second, 300 * time.Second}
	got, err := dashboardStartupDelays("")
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("startup schedule changed: %v %v", got, err)
	}
	for _, invalid := range []string{"NaN", "+Inf", "-1", "0", "no", "1e100", "1e-100"} {
		if _, err := dashboardStartupDelays(invalid); err == nil {
			t.Fatalf("invalid retry delay accepted: %s", invalid)
		}
	}
}

func TestObserverReconnectBackoffAndCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deadlines := make(chan dashboardTestDeadline, 8)
	stopped := make(chan struct{}, 8)
	after := func(delay time.Duration) dashboardRuntimeTimer {
		deadline := dashboardTestDeadline{delay: delay, fire: make(chan time.Time, 1)}
		deadlines <- deadline
		return dashboardRuntimeTimer{channel: deadline.fire, stop: func() bool { stopped <- struct{}{}; return true }}
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		runDashboardObserver(ctx, after, func(func()) error { return errors.New("connection reset") }, func(error) {})
	}()
	for _, want := range []time.Duration{30 * time.Second, 90 * time.Second, 300 * time.Second, 300 * time.Second} {
		deadline := dashboardTestReceive(t, deadlines)
		if deadline.delay != want {
			t.Fatalf("observer deadline = %s, want %s", deadline.delay, want)
		}
		deadline.fire <- time.Now()
		dashboardTestReceive(t, stopped)
	}
	_ = dashboardTestReceive(t, deadlines)
	cancel()
	dashboardTestReceive(t, done)
	dashboardTestReceive(t, stopped)
}

func TestMissingDashboardPrerequisitesDoNotBecomeMQTTFailure(t *testing.T) {
	t.Setenv("SUPERVISOR_TOKEN", "")
	c := NewCmdMqtt("")
	c.DashboardEnabled = true
	c.startDashboardRuntime(context.Background())()
	if c.Error != nil || c.dashboardRuntime != nil {
		t.Fatal("missing dashboard access changed MQTT's failure state")
	}
}
