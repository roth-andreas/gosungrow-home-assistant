package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
	"time"
)

type dashboardEligibility struct {
	Entity, Identity, Unit, StateClass, DeviceClass string
	DeviceID, Platform                              string
	RegistryObserved                                bool
	Numeric, RecentPower, RecentEnergy              bool
}

type dashboardEligibilityMatch struct {
	Target, Metric, Entity string
	Confident              bool
}

type dashboardEligibilityView struct {
	Entities []dashboardEligibility
	Matches  []dashboardEligibilityMatch
}

func dashboardObserverRelevant(state haState, targets []haDashboardTarget) bool {
	if state.ObservedSelection {
		return true
	}
	for _, target := range targets {
		identity, name := strings.ToLower(state.RegistryUniqueID), strings.ToLower(state.EntityID)
		if strings.Contains(identity, "gosungrow") && dashboardCanonicalTargetAffinity(identity, target, false) ||
			strings.Contains(name, "gosungrow") && dashboardCanonicalTargetAffinity(name, target, false) {
			return true
		}
	}
	return false
}

func dashboardNextEligibilityExpiry(states []haState, targets []haDashboardTarget, now time.Time) time.Duration {
	// No polling loop: wake only when a relevant source crosses a freshness
	// boundary (including becoming eligible after a future-dated timestamp).
	next := 36 * time.Hour
	for _, state := range states {
		if !dashboardObserverRelevant(state, targets) {
			continue
		}
		raw := state.LastUpdated
		if raw == "" {
			raw = state.LastChanged
		}
		stamp, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			continue
		}
		for _, deadline := range []time.Time{stamp.Add(-5 * time.Minute), stamp.Add(30 * time.Minute), stamp.Add(36 * time.Hour)} {
			delay := deadline.Sub(now) + time.Millisecond
			if delay > 0 && delay < next {
				next = delay
			}
		}
	}
	return next
}

// Numeric values and update timestamps are intentionally absent from this
// comparison. They matter only when validity, freshness, or ambiguity changes.
func dashboardEligibilityViewAt(targets []haDashboardTarget, states []haState, now time.Time) dashboardEligibilityView {
	view := dashboardEligibilityView{}
	for _, state := range states {
		if dashboardObserverRelevant(state, targets) {
			view.Entities = append(view.Entities, dashboardEligibility{
				Entity: state.EntityID, Identity: state.RegistryUniqueID, Unit: dashboardStateUnit(state),
				StateClass:  stringValue(state.Attributes["state_class"]),
				DeviceClass: stringValue(state.Attributes["device_class"]),
				DeviceID:    state.RegistryDeviceID, Platform: state.RegistryPlatform, RegistryObserved: state.RegistryObserved,
				Numeric:      dashboardStateHasUsableNumericValue(state),
				RecentPower:  dashboardSourceStateRecent(state, "pv_power", now),
				RecentEnergy: dashboardSourceStateRecent(state, "p13199", now),
			})
		}
	}
	sort.Slice(view.Entities, func(i, j int) bool { return view.Entities[i].Entity < view.Entities[j].Entity })
	for _, target := range targets {
		for _, metric := range dashboardSourceMetricDefinitions {
			match := dashboardSemanticRecommendationAt(target, metric.Metric, states, len(targets) == 1, now)
			view.Matches = append(view.Matches, dashboardEligibilityMatch{Target: target.PsKey, Metric: metric.Metric, Entity: match.Entity, Confident: match.Confident})
		}
	}
	return view
}

func loadDashboardObserverSnapshot(ctx context.Context, opts haDashboardInstallOptions, targets []haDashboardTarget) ([]haState, map[string]bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	client, err := newHAWSClient(ctx, opts.HomeAssistantWSURL, opts.SupervisorToken)
	if err != nil {
		return nil, nil, err
	}
	defer client.Close()
	states, err := client.ListStates(ctx)
	if err != nil {
		return nil, nil, err
	}
	registry, err := client.ListEntityRegistry(ctx)
	if err != nil {
		return nil, nil, err
	}
	config, err := client.GetConfig(ctx, opts.DashboardURLPath)
	if err != nil {
		var wsErr *haWSCallError
		if !errors.As(err, &wsErr) || !wsErr.IsCode("config_not_found") {
			return nil, nil, err
		}
	}
	selected := make(map[string]bool)
	for _, choices := range []map[string]map[string]string{extractDashboardSourceDefaults(config), extractDashboardSourceOverrides(config)} {
		for _, target := range targets {
			for _, id := range []string{dashboardSourceMappingID(target), strings.ToLower(target.PsKey)} {
				for _, entity := range choices[id] {
					selected[strings.ToLower(entity)] = true
				}
			}
		}
	}
	states = enrichDashboardStatesWithRegistry(states, registry)
	for i := range states {
		states[i].ObservedSelection = selected[strings.ToLower(states[i].EntityID)]
	}
	return states, selected, nil
}

func (c *dashboardCoordinator) observe(ctx context.Context, opts haDashboardInstallOptions, report func(string, error)) {
	var previous *dashboardEligibilityView
	runDashboardObserver(ctx, newDashboardRuntimeTimer, func(ready func()) error {
		return c.observeConnection(ctx, opts, &previous, ready)
	}, func(err error) { report("observation", err) })
}

func runDashboardObserver(ctx context.Context, after func(time.Duration) dashboardRuntimeTimer, connect func(func()) error, report func(error)) {
	delays := []time.Duration{30 * time.Second, 90 * time.Second, 300 * time.Second}
	attempt := 0
	for ctx.Err() == nil {
		err := connect(func() { attempt = 0 })
		if ctx.Err() != nil {
			return
		}
		report(err)
		index := attempt
		if index >= len(delays) {
			index = len(delays) - 1
		}
		timer := after(delays[index])
		select {
		case <-ctx.Done():
			timer.clear()
			return
		case <-timer.channel:
		}
		timer.clear()
		attempt++
	}
}

type dashboardObserverEvent struct {
	Type  string `json:"type"`
	Event struct {
		EventType string `json:"event_type"`
		Data      struct {
			EntityID string   `json:"entity_id"`
			NewState *haState `json:"new_state"`
		} `json:"data"`
	} `json:"event"`
}

func (c *dashboardCoordinator) observeConnection(ctx context.Context, opts haDashboardInstallOptions, previous **dashboardEligibilityView, ready func()) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	client, err := newHAWSClient(ctx, opts.HomeAssistantWSURL, opts.SupervisorToken)
	if err != nil {
		return err
	}
	defer client.Close()
	// One event subscription and one reader own this dedicated connection.
	// Snapshot requests always use a separate request/response connection.
	if err := client.call(map[string]any{"type": "subscribe_events"}, nil); err != nil {
		return err
	}
	events := make(chan dashboardObserverEvent, 128)
	readErrors := make(chan error, 1)
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		_ = client.conn.SetReadDeadline(time.Time{})
		for {
			var raw json.RawMessage
			if err := client.conn.ReadJSON(&raw); err != nil {
				readErrors <- err
				return
			}
			var event dashboardObserverEvent
			var envelope struct {
				Type  string `json:"type"`
				Event struct {
					EventType string `json:"event_type"`
				} `json:"event"`
			}
			if err := json.Unmarshal(raw, &envelope); err != nil {
				readErrors <- err
				return
			}
			if envelope.Type != "event" || (envelope.Event.EventType != "state_changed" && envelope.Event.EventType != "entity_registry_updated" && envelope.Event.EventType != "lovelace_updated") {
				continue
			}
			if err := json.Unmarshal(raw, &event); err != nil {
				readErrors <- err
				return
			}
			select {
			case events <- event:
			case <-ctx.Done():
				return
			}
		}
	}()
	defer func() {
		cancel()
		_ = client.Close()
		<-readerDone
	}()
	states, selected, err := loadDashboardObserverSnapshot(ctx, opts, c.snapshotTargets())
	if err != nil {
		return err
	}
	ready()
	check := func() {
		view := dashboardEligibilityViewAt(c.snapshotTargets(), states, time.Now())
		if *previous != nil && !reflect.DeepEqual(**previous, view) {
			select {
			case c.changes <- struct{}{}:
			default:
			}
		}
		*previous = &view
	}
	check()
	expiry := time.NewTimer(dashboardNextEligibilityExpiry(states, c.snapshotTargets(), time.Now()))
	defer expiry.Stop()
	resetExpiry := func() {
		if !expiry.Stop() {
			select {
			case <-expiry.C:
			default:
			}
		}
		expiry.Reset(dashboardNextEligibilityExpiry(states, c.snapshotTargets(), time.Now()))
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-readErrors:
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		case <-expiry.C:
			check()
			resetExpiry()
		case <-c.updates:
			states, selected, err = loadDashboardObserverSnapshot(ctx, opts, c.snapshotTargets())
			if err != nil {
				return err
			}
			check()
			resetExpiry()
		case event := <-events:
			if event.Event.EventType == "entity_registry_updated" || event.Event.EventType == "lovelace_updated" {
				states, selected, err = loadDashboardObserverSnapshot(ctx, opts, c.snapshotTargets())
				if err != nil {
					return err
				}
			} else {
				data := event.Event.Data
				index := -1
				for i := range states {
					if states[i].EntityID == data.EntityID {
						index = i
						break
					}
				}
				if data.NewState == nil {
					if index >= 0 {
						states = append(states[:index], states[index+1:]...)
					}
				} else if index >= 0 {
					next := *data.NewState
					old := states[index]
					// Keep registry identity on a state-only event.
					next.RegistryUniqueID, next.RegistryDeviceID, next.RegistryPlatform, next.RegistryObserved = old.RegistryUniqueID, old.RegistryDeviceID, old.RegistryPlatform, old.RegistryObserved
					next.ObservedSelection = old.ObservedSelection
					nextTime, nextErr := time.Parse(time.RFC3339Nano, next.LastUpdated)
					oldTime, oldErr := time.Parse(time.RFC3339Nano, old.LastUpdated)
					if nextErr == nil && oldErr == nil && nextTime.Before(oldTime) {
						continue
					}
					states[index] = next
					if !dashboardObserverRelevant(next, c.snapshotTargets()) {
						continue
					}
				} else {
					if !selected[strings.ToLower(data.EntityID)] && !dashboardObserverRelevant(*data.NewState, c.snapshotTargets()) {
						continue
					}
					// Creation may precede its registry event; take one fresh
					// joined snapshot rather than inferring identity from a name.
					states, selected, err = loadDashboardObserverSnapshot(ctx, opts, c.snapshotTargets())
					if err != nil {
						return err
					}
				}
			}
			check()
			resetExpiry()
		}
	}
}
