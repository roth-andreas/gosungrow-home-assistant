package cmd

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud"
)

type dashboardCoordinator struct {
	mu      sync.RWMutex
	targets []haDashboardTarget
	updates chan struct{}
	changes chan struct{}
}

type dashboardRuntimeTimer struct {
	channel <-chan time.Time
	stop    func() bool
}

func newDashboardRuntimeTimer(delay time.Duration) dashboardRuntimeTimer {
	timer := time.NewTimer(delay)
	return dashboardRuntimeTimer{channel: timer.C, stop: timer.Stop}
}

func (t *dashboardRuntimeTimer) clear() {
	if t.stop != nil {
		t.stop()
	}
	*t = dashboardRuntimeTimer{}
}

func copyDashboardTargets(targets []haDashboardTarget) []haDashboardTarget {
	ret := append([]haDashboardTarget(nil), targets...)
	for i := range ret {
		ret[i].PlantDevices = append([]dashboardPlantDevice(nil), ret[i].PlantDevices...)
	}
	return ret
}

func (c *dashboardCoordinator) updateTargets(targets []haDashboardTarget) {
	c.mu.Lock()
	c.targets = copyDashboardTargets(targets)
	c.mu.Unlock()
	select {
	case c.updates <- struct{}{}:
	default:
	}
}

func (c *dashboardCoordinator) snapshotTargets() []haDashboardTarget {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return copyDashboardTargets(c.targets)
}

func dashboardRuntimeOptions(force bool) haDashboardInstallOptions {
	assets := strings.TrimSpace(os.Getenv("GOSUNGROW_ASSET_DIR"))
	if assets == "" {
		assets = defaultHADashboardAssetDir
	}
	language := strings.TrimSpace(os.Getenv("GOSUNGROW_DASHBOARD_LANGUAGE"))
	if language == "" {
		language = "auto"
	}
	return haDashboardInstallOptions{
		AssetDir: assets, HomeAssistantDir: defaultHAConfigDir,
		HomeAssistantWSURL: defaultHAWebsocketURL, SupervisorToken: os.Getenv("SUPERVISOR_TOKEN"),
		Language: language, DiagnosticContext: "runtime", DashboardURLPath: defaultDashboardURLPath,
		DashboardTitle: defaultDashboardTitle, DashboardIcon: defaultDashboardIcon,
		ShowInSidebar: true, RequireAdmin: true, ForceUpdate: force,
	}
}

func dashboardStartupDelays(raw string) ([]time.Duration, error) {
	if strings.TrimSpace(raw) == "" {
		raw = "30 90 180 300"
	}
	var delays []time.Duration
	for _, value := range strings.Fields(raw) {
		seconds, err := strconv.ParseFloat(value, 64)
		if err != nil || math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds <= 0 || seconds > float64((1<<63-1)/int64(time.Second)) || time.Duration(seconds*float64(time.Second)) <= 0 {
			return nil, fmt.Errorf("GOSUNGROW_DASHBOARD_RECONCILE_DELAYS must contain positive seconds")
		}
		delays = append(delays, time.Duration(seconds*float64(time.Second)))
	}
	return delays, nil
}

// startDashboardRuntime is owned by the MQTT command, and returns a join
// function. Its worker never reads or mutates the cloud client or MQTT error.
func (c *CmdMqtt) startDashboardRuntime(ctx context.Context) func() {
	if !c.DashboardEnabled {
		return func() {}
	}
	opts := dashboardRuntimeOptions(c.DashboardForceUpdate)
	delays, err := dashboardStartupDelays(os.Getenv("GOSUNGROW_DASHBOARD_RECONCILE_DELAYS"))
	if err == nil && strings.TrimSpace(opts.SupervisorToken) == "" {
		err = fmt.Errorf("SUPERVISOR_TOKEN is not set; configure Home Assistant access or disable managed dashboards")
	}
	if err != nil {
		c.log.Info("Managed dashboard configuration: %s. MQTT will continue.\n", err)
		return func() {}
	}
	runtime := &dashboardCoordinator{targets: copyDashboardTargets(c.dashboardTargets), updates: make(chan struct{}, 1), changes: make(chan struct{}, 1)}
	c.dashboardRuntime = runtime
	workerCtx, cancel := context.WithCancel(ctx)
	var workers sync.WaitGroup
	workers.Add(2)
	logFailure := func(stage string, err error) {
		// Do not log websocket payloads, tokens, or projection templates.
		class := iSolarCloud.ClassifyFailure(err)
		action := "automatic dashboard recovery remains independent of MQTT and cloud login"
		if class == iSolarCloud.FailureClassNonRecoverable {
			action = "check Home Assistant permissions, dashboard ownership, and local asset configuration"
		} else if class == iSolarCloud.FailureClassOperatorActionRequired {
			action = dashboardCoreRestartAction
		}
		c.log.Info("Managed dashboard %s failed: class=%s action=%s; MQTT will continue.\n", stage, class, action)
	}
	go func() {
		defer workers.Done()
		runtime.observe(workerCtx, opts, logFailure)
	}()
	go func() {
		defer workers.Done()
		runtime.run(workerCtx, delays, newDashboardRuntimeTimer, func(available bool) error {
			attemptOptions := opts
			attemptOptions.AvailabilityDriven = available
			if available {
				attemptOptions.DiagnosticContext = "availability"
			}
			return reconcileManagedDashboard(workerCtx, runtime.snapshotTargets(), attemptOptions)
		}, func(err error) { logFailure("reconciliation", err) })
	}()
	return func() {
		cancel()
		workers.Wait()
		c.dashboardRuntime = nil
	}
}

// One worker serializes every trigger. Events during a run or retry consume
// one pending slot. The first event owns the fixed debounce deadline.
func (c *dashboardCoordinator) run(ctx context.Context, delays []time.Duration, after func(time.Duration) dashboardRuntimeTimer, reconcile func(bool) error, report func(error)) {
	var scheduled, debounce, retry dashboardRuntimeTimer
	defer scheduled.clear()
	defer debounce.clear()
	defer retry.clear()
	if len(delays) > 0 {
		scheduled = after(delays[0])
	}
	nextSchedule := 1
	results := make(chan error, 1)
	running, pending := false, false
	availability := false
	attempt := 0
	retryDelays := []time.Duration{30 * time.Second, 90 * time.Second, 300 * time.Second}
	start := func(fromAvailability bool) {
		if ctx.Err() != nil {
			pending = false
			return
		}
		running, pending, availability = true, false, fromAvailability
		go func() { results <- reconcile(fromAvailability) }()
	}
	defer func() {
		if running {
			<-results
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.changes:
			if retry.channel != nil {
				pending = true
			} else if !pending && debounce.channel == nil {
				pending = true
				debounce = after(5 * time.Second)
			}
		case <-debounce.channel:
			debounce.clear()
			if running || retry.channel != nil {
				pending = true
			} else {
				attempt = 0
				start(true)
			}
		case <-scheduled.channel:
			scheduled.clear()
			if nextSchedule < len(delays) {
				scheduled = after(delays[nextSchedule])
				nextSchedule++
			}
			if running || retry.channel != nil {
				pending = true
			} else {
				debounce.clear()
				attempt = 0
				start(false)
			}
		case <-retry.channel:
			retry.clear()
			debounce.clear()
			start(true)
		case err := <-results:
			running = false
			if err != nil {
				report(err)
				if availability && iSolarCloud.ShouldRecoverGatewayError(err) && attempt < len(retryDelays) {
					debounce.clear()
					retry = after(retryDelays[attempt])
					attempt++
					continue
				}
				if availability && iSolarCloud.ShouldRecoverGatewayError(err) && attempt == len(retryDelays) {
					// Exhaustion waits for a transition after the final attempt.
					pending = false
					debounce.clear()
				}
				attempt = 0
			}
			if pending {
				attempt = 0
				if debounce.channel == nil {
					start(true)
				}
			}
		}
	}
}
