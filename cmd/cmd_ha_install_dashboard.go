package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud"
	gosungrowoutput "github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct/output"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	defaultHADashboardAssetDir = "/opt/gosungrow/assets"
	defaultHAWebsocketURL      = "ws://supervisor/core/websocket"
	defaultHAConfigDir         = "/homeassistant"
	defaultDashboardURLPath    = "gosungrow-flow"
	defaultDashboardTitle      = "GoSungrow Flow"
	defaultDashboardIcon       = "mdi:solar-power"
	dashboardTemplateFile      = "home-assistant-sungrow-flow.yaml"
	dashboardStateFileName     = "dashboard_state.json"
	dashboardCardSourceFile    = "gosungrow-energy-flow-card-v2.js"
	dashboardCardFilePrefix    = "gosungrow-dashboard-cards."
	dashboardCardResourceDir   = "gosungrow"
	dashboardCardResourceType  = "module"
)

type haDashboardInstallOptions struct {
	AssetDir           string
	HomeAssistantDir   string
	HomeAssistantWSURL string
	SupervisorToken    string
	Language           string
	DiagnosticContext  string
	DashboardURLPath   string
	DashboardTitle     string
	DashboardIcon      string
	ShowInSidebar      bool
	RequireAdmin       bool
	ForceUpdate        bool
}

type haDashboardTarget struct {
	PsID            string
	PsKey           string
	ViewTitle       string
	ViewPath        string
	PlantName       string
	DeviceName      string
	DeviceType      int64
	SelectionSource string
	PlantDevices    []dashboardPlantDevice
}

type haDashboardState struct {
	DashboardURLPath       string                       `json:"dashboard_url_path"`
	DashboardHash          string                       `json:"dashboard_hash"`
	DashboardStructureHash string                       `json:"dashboard_structure_hash,omitempty"`
	TargetPsKeys           []string                     `json:"target_ps_keys,omitempty"`
	SourceOverrides        map[string]map[string]string `json:"source_overrides,omitempty"`
	AssetMode              string                       `json:"asset_mode,omitempty"`
	AssetURL               string                       `json:"asset_url,omitempty"`
	AssetHash              string                       `json:"asset_hash,omitempty"`
	PreviousAssetURL       string                       `json:"previous_asset_url,omitempty"`
	PreviousAssetHash      string                       `json:"previous_asset_hash,omitempty"`
	UpdatedAt              string                       `json:"updated_at"`
}

type haDashboardMetadata struct {
	ID      string `json:"id"`
	URLPath string `json:"url_path"`
	Title   string `json:"title"`
}

type haResourceMetadata struct {
	ID           any    `json:"id"`
	URL          string `json:"url"`
	ResourceType string `json:"type,omitempty"`
	ResType      string `json:"res_type,omitempty"`
}

type haState struct {
	EntityID         string         `json:"entity_id"`
	State            string         `json:"state"`
	LastChanged      string         `json:"last_changed,omitempty"`
	LastUpdated      string         `json:"last_updated,omitempty"`
	Attributes       map[string]any `json:"attributes,omitempty"`
	RegistryUniqueID string         `json:"-"`
	RegistryDeviceID string         `json:"-"`
	RegistryPlatform string         `json:"-"`
}

type haEntityRegistryEntry struct {
	EntityID string `json:"entity_id"`
	UniqueID string `json:"unique_id"`
	DeviceID string `json:"device_id"`
	Platform string `json:"platform"`
}

type dashboardInstallDiagnostics struct {
	DiagnosticContext     string
	Debug                 bool
	HAStatesLoaded        int
	HAStatesLoadError     string
	GoSungrowStatesFound  int
	DashboardRefsFound    int
	RemappedRefs          int
	RemappedPreview       []dashboardEntityRemap
	UnresolvedRefs        []dashboardUnresolvedEntityRef
	MetricTraces          []dashboardMetricTrace
	AggregateHints        []dashboardAggregateHint
	BatteryDetectionKnown bool
	BatteryTargetsFound   int
	BatteryTargetsTotal   int
	TargetDiagnostics     []dashboardTargetDiagnostics
	DashboardSaved        bool
	DashboardSaveReason   string
	AssetPhase            string
	AssetHash             string
	AssetURL              string
	AssetHTTPRoute        string
	AssetMetadataStatus   int
	AssetMetadataOutcome  string
	AssetDiscoveredPort   int
	AssetDiscoveredTLS    string
	AssetHTTPStatus       int
	AssetMIMEType         string
	ResourceAction        string
	DashboardMode         string
	RollbackResult        string
	CleanupResult         string
}

type dashboardTargetDiagnostics struct {
	PlantName              string
	DeviceName             string
	PsID                   string
	PsKey                  string
	ViewPath               string
	DeviceType             int64
	SelectionSource        string
	PlantDevices           []dashboardPlantDevice
	GoSungrowStates        int
	VirtualStates          int
	ExampleGoSungrowStates []string
	ExampleVirtualStates   []string
}

type dashboardPlantDevice struct {
	PlantName       string
	DeviceName      string
	PsID            string
	PsKey           string
	DeviceType      int64
	Selected        bool
	SelectionSource string
}

type dashboardAggregateHint struct {
	Metric string
	Entity string
	State  string
	Unit   string
	Source string
}

type haWSCallError struct {
	Code    string
	Message string
}

func (e *haWSCallError) Error() string {
	return fmt.Sprintf("home assistant websocket error (%s): %s", e.Code, e.Message)
}

func (e *haWSCallError) IsCode(code string) bool {
	return e != nil && e.Code == code
}

type haWSResponse struct {
	ID      int64           `json:"id"`
	Type    string          `json:"type"`
	Success bool            `json:"success"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type haWSClient struct {
	conn   *websocket.Conn
	nextID int64
}

func (c *CmdHa) newInstallDashboardCommand() *cobra.Command {
	opts := haDashboardInstallOptions{
		AssetDir:           defaultHADashboardAssetDir,
		HomeAssistantDir:   defaultHAConfigDir,
		HomeAssistantWSURL: defaultHAWebsocketURL,
		SupervisorToken:    os.Getenv("SUPERVISOR_TOKEN"),
		Language:           "auto",
		DiagnosticContext:  "manual",
		DashboardURLPath:   defaultDashboardURLPath,
		DashboardTitle:     defaultDashboardTitle,
		DashboardIcon:      defaultDashboardIcon,
		ShowInSidebar:      true,
		RequireAdmin:       false,
		ForceUpdate:        false,
	}

	cmd := &cobra.Command{
		Use:                   "install-dashboard [ps_id ...]",
		Aliases:               []string{"dashboard-install"},
		Annotations:           map[string]string{"group": "Ha"},
		Short:                 "Install or update the managed GoSungrow dashboard.",
		Long:                  "Install or update the managed GoSungrow Home Assistant dashboard without restarting Home Assistant.",
		DisableFlagParsing:    false,
		DisableFlagsInUseLine: false,
		PreRunE:               cmds.SunGrowArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			selected := opts
			return c.installManagedDashboard(args, selected)
		},
		Args: cobra.ArbitraryArgs,
	}

	cmd.Flags().StringVar(&opts.AssetDir, "asset-dir", opts.AssetDir, "Directory containing the dashboard template and image assets.")
	cmd.Flags().StringVar(&opts.HomeAssistantDir, "ha-config-dir", opts.HomeAssistantDir, "Home Assistant config directory containing the www folder.")
	cmd.Flags().StringVar(&opts.HomeAssistantWSURL, "ha-ws-url", opts.HomeAssistantWSURL, "Home Assistant websocket endpoint.")
	cmd.Flags().StringVar(&opts.SupervisorToken, "supervisor-token", opts.SupervisorToken, "Supervisor token used to access the Home Assistant websocket.")
	cmd.Flags().StringVar(&opts.Language, "language", opts.Language, "Dashboard language (auto, en, de, sv, or locale such as de-DE).")
	cmd.Flags().StringVar(&opts.DiagnosticContext, "diagnostic-context", opts.DiagnosticContext, "Diagnostic context label included in dashboard logs.")
	cmd.Flags().StringVar(&opts.DashboardURLPath, "url-path", opts.DashboardURLPath, "Dashboard URL path.")
	cmd.Flags().StringVar(&opts.DashboardTitle, "title", opts.DashboardTitle, "Dashboard title.")
	cmd.Flags().StringVar(&opts.DashboardIcon, "icon", opts.DashboardIcon, "Dashboard sidebar icon.")
	cmd.Flags().BoolVar(&opts.ShowInSidebar, "show-in-sidebar", opts.ShowInSidebar, "Show the dashboard in the Home Assistant sidebar.")
	cmd.Flags().BoolVar(&opts.RequireAdmin, "require-admin", opts.RequireAdmin, "Restrict dashboard access to Home Assistant administrators.")
	cmd.Flags().BoolVar(&opts.ForceUpdate, "force-update", opts.ForceUpdate, "Replace an existing dashboard even if it was modified outside GoSungrow.")
	_ = cmd.Flags().MarkHidden("supervisor-token")
	_ = cmd.Flags().MarkHidden("ha-config-dir")
	_ = cmd.Flags().MarkHidden("diagnostic-context")

	return cmd
}

func (c *CmdHa) installManagedDashboard(args []string, opts haDashboardInstallOptions) error {
	if strings.TrimSpace(opts.SupervisorToken) == "" {
		return fmt.Errorf("SUPERVISOR_TOKEN is not set")
	}

	if err := cmds.Api.ApiLogin(true); err != nil {
		return err
	}

	targets, err := c.discoverDashboardTargets(args)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return fmt.Errorf("no Sungrow ESS devices were discovered")
	}

	statePath := dashboardStatePath()
	state, err := loadDashboardState(statePath)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	client, err := newHAWSClient(ctx, opts.HomeAssistantWSURL, opts.SupervisorToken)
	if err != nil {
		return err
	}
	defer client.Close()

	preferredLanguage := strings.TrimSpace(opts.Language)
	if preferredLanguage == "" || strings.EqualFold(preferredLanguage, "auto") {
		preferredLanguage = client.GetPreferredLanguage(ctx)
	}
	localeBundle, _, err := localizedDashboardBundle(opts.AssetDir, preferredLanguage)
	if err != nil {
		return err
	}

	currentConfig, err := client.GetConfig(ctx, opts.DashboardURLPath)
	if err != nil {
		wsErr, ok := err.(*haWSCallError)
		if !(ok && wsErr.IsCode("config_not_found")) {
			return err
		}
		currentConfig = nil
	}

	templatePath := filepath.Join(opts.AssetDir, dashboardTemplateFile)
	config, err := renderDashboardConfig(templatePath, opts.DashboardTitle, targets, localeBundle)
	if err != nil {
		return err
	}

	diagnostics := dashboardInstallDiagnostics{
		DiagnosticContext:   dashboardDiagnosticContext(opts.DiagnosticContext),
		Debug:               dashboardDebugEnabled(),
		BatteryTargetsTotal: len(targets),
		DashboardSaveReason: "unchanged",
		TargetDiagnostics:   buildDashboardTargetDiagnostics(targets, nil),
	}
	states, listErr := client.ListStates(ctx)
	var remapReport dashboardRemapReport
	if listErr != nil {
		diagnostics.HAStatesLoadError = listErr.Error()
		_, remapReport = remapDashboardEntitiesWithReport(config, targets, nil)
	} else {
		if registry, registryErr := client.ListEntityRegistry(ctx); registryErr == nil {
			states = enrichDashboardStatesWithRegistry(states, registry)
		} else {
			fmt.Printf("Managed dashboard could not load the Home Assistant entity registry; canonical matching will use conservative fallbacks: %s\n", registryErr)
		}
		diagnostics.HAStatesLoaded = len(states)
		diagnostics.GoSungrowStatesFound = countDashboardGoSungrowStates(states)
		diagnostics.BatteryDetectionKnown = true
		diagnostics.BatteryTargetsFound = countDashboardBatteryTargets(targets, states)
		diagnostics.TargetDiagnostics = buildDashboardTargetDiagnostics(targets, states)
		diagnostics.AggregateHints = buildDashboardAggregateHints(targets, states)
		// Keep the metric-specific placeholders in the generated config until
		// source bindings are built. Two metrics may resolve to the same entity;
		// replacing strings here would make their binding paths indistinguishable.
		_, remapReport = remapDashboardEntitiesWithReport(config, targets, states)
		config = pruneDashboardForUnavailableMetricsWithPinned(config, targets, states, extractDashboardSourceDefaults(currentConfig))
	}
	diagnostics.DashboardRefsFound = remapReport.TotalRefs
	diagnostics.RemappedRefs = len(remapReport.Remapped)
	diagnostics.RemappedPreview = dashboardRemapPreview(remapReport.Remapped, 5)
	diagnostics.UnresolvedRefs = remapReport.Unresolved
	diagnostics.MetricTraces = remapReport.Traces
	persistedOverrides := map[string]map[string]string(nil)
	if state != nil {
		persistedOverrides = state.SourceOverrides
	}
	config, sourceOverrides := applyDashboardSourceMappings(config, currentConfig, persistedOverrides, targets, states, remapReport.Traces, opts.DashboardURLPath, localeBundle)

	metadata, err := client.ListDashboards(ctx)
	if err != nil {
		return err
	}

	var existing *haDashboardMetadata
	for _, entry := range metadata {
		if entry.URLPath == opts.DashboardURLPath {
			entryCopy := entry
			existing = &entryCopy
			break
		}
	}
	exists := existing != nil
	if exists && strings.TrimSpace(existing.ID) == "" {
		return fmt.Errorf("dashboard URL path %q is already in use by a non-storage dashboard; choose a different dashboard_url_path or remove the existing dashboard", opts.DashboardURLPath)
	}

	currentHash := ""
	if currentConfig != nil {
		currentHash, err = hashCanonicalJSON(currentConfig)
		if err != nil {
			return err
		}
	}

	managedByState := state != nil && state.DashboardURLPath == opts.DashboardURLPath
	if exists && !managedByState && !opts.ForceUpdate {
		return fmt.Errorf("dashboard %q already exists and is not managed by GoSungrow; set dashboard_force_update to true to replace it", opts.DashboardURLPath)
	}
	modified, err := dashboardModifiedOutsideGoSungrow(currentConfig, config, state)
	if err != nil {
		return err
	}
	if exists && managedByState && modified && !opts.ForceUpdate {
		return fmt.Errorf("dashboard %q was modified outside GoSungrow; set dashboard_force_update to true to replace it", opts.DashboardURLPath)
	}

	// Ownership and the complete desired dashboard are established before any
	// filesystem, resource-registry, or dashboard mutation.
	resourceURL, assetHash, err := installDashboardCardAsset(opts.AssetDir, opts.HomeAssistantDir)
	if err != nil {
		fmt.Printf("Dashboard asset lifecycle: phase=stage-failed hash=none url=none http_route=unknown http_status=0 mime=unknown resource_action=none dashboard_mode=unknown rollback=not-required cleanup=not-run error=%q\n", err.Error())
		return err
	}
	assetCommitted := false
	defer func() {
		if assetCommitted {
			return
		}
		if state != nil && strings.EqualFold(state.AssetHash, assetHash) {
			diagnostics.CleanupResult = "retained active version"
		} else if cleanupErr := removeDashboardAssetVersion(opts.HomeAssistantDir, assetHash); cleanupErr != nil {
			diagnostics.CleanupResult = "failed: " + cleanupErr.Error()
		} else {
			diagnostics.CleanupResult = "failed activation cleaned"
		}
		printDashboardInstallDiagnostics(diagnostics)
	}()
	diagnostics.AssetPhase = "staged"
	diagnostics.AssetURL = resourceURL
	diagnostics.AssetHash = assetHash

	assetVerification, activationErr := verifyDashboardCardAsset(ctx, opts.HomeAssistantWSURL, opts.SupervisorToken, resourceURL, assetHash)
	diagnostics.AssetHTTPRoute = assetVerification.Route
	diagnostics.AssetMetadataStatus = assetVerification.MetadataStatus
	diagnostics.AssetMetadataOutcome = assetVerification.MetadataOutcome
	diagnostics.AssetDiscoveredPort = assetVerification.DiscoveredPort
	if assetVerification.MetadataOutcome == "ok" {
		diagnostics.AssetDiscoveredTLS = strconv.FormatBool(assetVerification.DiscoveredTLS)
	}
	diagnostics.AssetHTTPStatus = assetVerification.StatusCode
	diagnostics.AssetMIMEType = assetVerification.MIMEType
	if activationErr == nil {
		diagnostics.AssetPhase = "http-verified"
	}

	var resourceChange *dashboardResourceChange
	if activationErr == nil {
		resourceChange, activationErr = client.ActivateManagedResource(ctx, resourceURL)
		if resourceChange != nil {
			diagnostics.ResourceAction = resourceChange.Action
		}
		if activationErr == nil && resourceChange.Action != "unchanged" {
			var reloadRequested bool
			reloadRequested, activationErr = client.ReloadResourcesIfSupported(ctx)
			if activationErr == nil && reloadRequested {
				diagnostics.ResourceAction += "+reload-requested"
			}
		}
		if activationErr == nil {
			diagnostics.AssetPhase = "resource-verified"
		}
	}

	existingWorking := exists && managedByState && currentConfig != nil
	if activationErr != nil {
		if resourceChange != nil {
			if rollbackErr := client.RestoreManagedResources(ctx, resourceChange.Before); rollbackErr != nil {
				diagnostics.RollbackResult = "failed: " + rollbackErr.Error()
				return fmt.Errorf("activate dashboard asset: %w; resource rollback failed: %v", activationErr, rollbackErr)
			}
			diagnostics.RollbackResult = "resource restored"
		}
		if existingWorking {
			return fmt.Errorf("activate dashboard asset while preserving the existing dashboard: %w", activationErr)
		}

		fallbackConfig, fallbackErr := nativeDashboardFallback(config, localeBundle)
		if fallbackErr != nil {
			return fmt.Errorf("activate dashboard asset: %w; build native fallback: %v", activationErr, fallbackErr)
		}
		if dashboardConfigContainsCustomGoSungrow(fallbackConfig) {
			return fmt.Errorf("native dashboard fallback still contains GoSungrow custom-card references")
		}
		config = fallbackConfig
		diagnostics.DashboardMode = dashboardAssetModeFallback
		diagnostics.AssetPhase = "native-fallback"
		fmt.Printf("Managed dashboard asset activation failed; installing the native fallback: %v\n", activationErr)
	} else {
		diagnostics.DashboardMode = dashboardAssetModeEnhanced
	}

	dashboardCreated := false
	if exists {
		if err := client.UpdateDashboard(ctx, existing.ID, opts); err != nil {
			wsErr, ok := err.(*haWSCallError)
			if !(ok && wsErr.IsCode("not_found")) {
				return rollbackDashboardInstall(ctx, client, opts.DashboardURLPath, false, currentConfig, resourceChange, fmt.Errorf("update dashboard metadata: %w", err), &diagnostics)
			}
			exists = false
		}
	}
	if !exists {
		if err := client.CreateDashboard(ctx, opts); err != nil {
			return rollbackDashboardInstall(ctx, client, opts.DashboardURLPath, false, currentConfig, resourceChange, fmt.Errorf("create managed dashboard: %w", err), &diagnostics)
		}
		dashboardCreated = true
	}

	desiredHash, err := hashCanonicalJSON(config)
	if err != nil {
		return rollbackDashboardInstall(ctx, client, opts.DashboardURLPath, dashboardCreated, currentConfig, resourceChange, err, &diagnostics)
	}
	desiredStructureHash, err := hashDashboardStructure(config)
	if err != nil {
		return rollbackDashboardInstall(ctx, client, opts.DashboardURLPath, dashboardCreated, currentConfig, resourceChange, err, &diagnostics)
	}

	shouldSaveConfig := currentHash != desiredHash || currentConfig == nil || opts.ForceUpdate
	diagnostics.DashboardSaved = shouldSaveConfig
	diagnostics.DashboardSaveReason = dashboardSaveReason(shouldSaveConfig, currentConfig == nil, currentHash != desiredHash, opts.ForceUpdate)
	if shouldSaveConfig {
		if err := client.SaveConfig(ctx, opts.DashboardURLPath, config); err != nil {
			return rollbackDashboardInstall(ctx, client, opts.DashboardURLPath, dashboardCreated, currentConfig, resourceChange, fmt.Errorf("save managed dashboard: %w", err), &diagnostics)
		}
	}
	verifiedConfig, err := client.GetConfig(ctx, opts.DashboardURLPath)
	if err != nil {
		return rollbackDashboardInstall(ctx, client, opts.DashboardURLPath, dashboardCreated, currentConfig, resourceChange, fmt.Errorf("re-read managed dashboard: %w", err), &diagnostics)
	}
	verifiedHash, err := hashCanonicalJSON(verifiedConfig)
	if err != nil || verifiedHash != desiredHash {
		if err == nil {
			err = fmt.Errorf("saved dashboard hash mismatch: got %s want %s", verifiedHash, desiredHash)
		}
		return rollbackDashboardInstall(ctx, client, opts.DashboardURLPath, dashboardCreated, currentConfig, resourceChange, err, &diagnostics)
	}

	if resourceChange != nil && diagnostics.DashboardMode == dashboardAssetModeEnhanced {
		if err := client.RemoveDuplicateManagedResources(ctx, resourceChange.Canonical.ID); err != nil {
			return rollbackDashboardInstall(ctx, client, opts.DashboardURLPath, dashboardCreated, currentConfig, resourceChange, fmt.Errorf("clean duplicate managed resources: %w", err), &diagnostics)
		}
	}

	previousAssetURL, previousAssetHash := previousDashboardAsset(state, resourceURL, assetHash, diagnostics.DashboardMode)
	activeURL, activeHash := resourceURL, assetHash
	if diagnostics.DashboardMode == dashboardAssetModeFallback {
		activeURL, activeHash = "", ""
	}

	if err := saveDashboardState(statePath, &haDashboardState{
		DashboardURLPath:       opts.DashboardURLPath,
		DashboardHash:          desiredHash,
		DashboardStructureHash: desiredStructureHash,
		TargetPsKeys:           targetPSKeys(targets),
		SourceOverrides:        sourceOverrides,
		AssetMode:              diagnostics.DashboardMode,
		AssetURL:               activeURL,
		AssetHash:              activeHash,
		PreviousAssetURL:       previousAssetURL,
		PreviousAssetHash:      previousAssetHash,
		UpdatedAt:              time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return rollbackDashboardInstall(ctx, client, opts.DashboardURLPath, dashboardCreated, currentConfig, resourceChange, fmt.Errorf("persist managed dashboard state: %w", err), &diagnostics)
	}
	diagnostics.AssetPhase = "committed"
	diagnostics.RollbackResult = dashboardDiagnosticDefault(diagnostics.RollbackResult, "not required")
	if err := cleanupDashboardAssetFiles(opts.HomeAssistantDir, activeHash, previousAssetHash); err != nil {
		diagnostics.CleanupResult = "failed: " + err.Error()
		return err
	}
	diagnostics.CleanupResult = "complete"
	assetCommitted = true

	viewCount := 0
	if views, ok := config["views"].([]any); ok {
		viewCount = len(views)
	}
	if viewCount == 0 {
		viewCount = len(targets)
	}
	printDashboardInstallDiagnostics(diagnostics)
	fmt.Printf("Managed GoSungrow dashboard ready at /%s with %d view(s) in %s mode.\n", opts.DashboardURLPath, viewCount, diagnostics.DashboardMode)
	return nil
}

func previousDashboardAsset(state *haDashboardState, nextURL, nextHash, mode string) (string, string) {
	if state == nil {
		return "", ""
	}
	if mode != dashboardAssetModeEnhanced || state.AssetURL == "" || state.AssetHash == "" || state.AssetURL == nextURL || state.AssetHash == nextHash {
		return state.PreviousAssetURL, state.PreviousAssetHash
	}
	return state.AssetURL, state.AssetHash
}

func rollbackDashboardInstall(ctx context.Context, client *haWSClient, urlPath string, dashboardCreated bool, currentConfig map[string]any, resourceChange *dashboardResourceChange, cause error, diagnostics *dashboardInstallDiagnostics) error {
	var rollbackErrors []string
	if dashboardCreated {
		if err := client.DeleteDashboard(ctx, urlPath); err != nil {
			rollbackErrors = append(rollbackErrors, "dashboard delete: "+err.Error())
		}
	} else if currentConfig != nil {
		if err := client.SaveConfig(ctx, urlPath, currentConfig); err != nil {
			rollbackErrors = append(rollbackErrors, "dashboard restore: "+err.Error())
		}
	}
	if resourceChange != nil {
		if err := client.RestoreManagedResources(ctx, resourceChange.Before); err != nil {
			rollbackErrors = append(rollbackErrors, "resource restore: "+err.Error())
		}
	}
	if len(rollbackErrors) == 0 {
		diagnostics.RollbackResult = "complete"
		return cause
	}
	diagnostics.RollbackResult = "failed: " + strings.Join(rollbackErrors, "; ")
	return fmt.Errorf("%w; rollback failed: %s", cause, strings.Join(rollbackErrors, "; "))
}

func dashboardDiagnosticDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func (c *CmdHa) discoverDashboardTargets(args []string) ([]haDashboardTarget, error) {
	trees, err := cmds.Api.SunGrow.PsTreeMenu(args...)
	if err != nil {
		return nil, err
	}
	return discoverDashboardTargetsFromTrees(trees)
}

func discoverDashboardTargetsFromTrees(trees map[string]iSolarCloud.PsTree) ([]haDashboardTarget, error) {
	psIDs := make([]string, 0, len(trees))
	for psID := range trees {
		psIDs = append(psIDs, psID)
	}
	sort.Strings(psIDs)

	type deviceTarget struct {
		psID            string
		psKey           string
		plantName       string
		deviceName      string
		deviceType      int64
		selectionSource string
		plantDevices    []dashboardPlantDevice
	}

	collected := make([]deviceTarget, 0)
	perPlantCounts := make(map[string]int)
	for _, psID := range psIDs {
		tree := trees[psID]
		plantTargets := make([]deviceTarget, 0)
		plantDevices := make([]dashboardPlantDevice, 0)
		bestFallbackIndex := -1
		bestFallbackRank := 1 << 30
		fallbackTargets := make([]deviceTarget, 0)
		for _, device := range tree.Devices {
			target := deviceTarget{
				psID:       psID,
				psKey:      strings.TrimSpace(device.PsKey.String()),
				plantName:  cleanDashboardLabel(device.PsName.String()),
				deviceName: cleanDashboardLabel(device.DeviceName.String()),
				deviceType: device.DeviceType.Value(),
			}
			if target.psKey == "" {
				continue
			}
			if target.plantName == "" {
				target.plantName = fmt.Sprintf("Plant %s", psID)
			}
			target.selectionSource = dashboardSelectionSourceForDeviceType(target.deviceType)
			plantDevices = append(plantDevices, dashboardPlantDevice{
				PlantName:       target.plantName,
				DeviceName:      target.deviceName,
				PsID:            target.psID,
				PsKey:           target.psKey,
				DeviceType:      target.deviceType,
				SelectionSource: target.selectionSource,
			})
			if device.DeviceType.Match(14) {
				plantTargets = append(plantTargets, target)
				continue
			}
			fallbackTargets = append(fallbackTargets, target)
			rank := preferredSungrowDeviceTypeRank(target.deviceType)
			if bestFallbackIndex == -1 || rank < bestFallbackRank {
				bestFallbackIndex = len(fallbackTargets) - 1
				bestFallbackRank = rank
			}
		}

		if len(plantTargets) == 0 && bestFallbackIndex >= 0 {
			plantTargets = append(plantTargets, fallbackTargets[bestFallbackIndex])
		}
		if len(plantTargets) == 0 {
			continue
		}

		for index := range plantTargets {
			plantTargets[index].plantDevices = markSelectedDashboardPlantDevices(plantDevices, plantTargets[index].psKey)
		}

		collected = append(collected, plantTargets...)
		perPlantCounts[psID] += len(plantTargets)
	}

	if len(collected) == 0 {
		return nil, fmt.Errorf("no Sungrow devices with a valid ps_key were discovered")
	}

	titleCounts := make(map[string]int)
	titles := make([]string, len(collected))
	for i, target := range collected {
		title := target.plantName
		if perPlantCounts[target.psID] > 1 {
			if target.deviceName != "" && target.deviceName != target.plantName {
				title = fmt.Sprintf("%s (%s)", target.plantName, target.deviceName)
			} else {
				title = fmt.Sprintf("%s (%s)", target.plantName, target.psKey)
			}
		}
		titles[i] = title
		titleCounts[title]++
	}

	ret := make([]haDashboardTarget, 0, len(collected))
	for i, target := range collected {
		title := titles[i]
		if titleCounts[title] > 1 {
			title = fmt.Sprintf("%s (%s)", title, target.psID)
		}
		ret = append(ret, haDashboardTarget{
			PsID:            target.psID,
			PsKey:           target.psKey,
			ViewTitle:       title,
			ViewPath:        dashboardSlug(target.psKey),
			PlantName:       target.plantName,
			DeviceName:      target.deviceName,
			DeviceType:      target.deviceType,
			SelectionSource: target.selectionSource,
			PlantDevices:    target.plantDevices,
		})
	}

	return ret, nil
}

func markSelectedDashboardPlantDevices(devices []dashboardPlantDevice, selectedPsKey string) []dashboardPlantDevice {
	ret := make([]dashboardPlantDevice, 0, len(devices))
	selectedPsKey = strings.TrimSpace(selectedPsKey)
	for _, device := range devices {
		entry := device
		entry.Selected = selectedPsKey != "" && strings.EqualFold(strings.TrimSpace(device.PsKey), selectedPsKey)
		ret = append(ret, entry)
	}
	return ret
}

func renderDashboardConfig(templatePath string, dashboardTitle string, targets []haDashboardTarget, localeBundle dashboardLocaleBundle) (map[string]any, error) {
	localeBundle = mergeDashboardLocale(defaultDashboardLocaleBundle, localeBundle)

	content, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, err
	}

	var template map[string]any
	if err := yaml.Unmarshal(content, &template); err != nil {
		return nil, err
	}

	rawViews, ok := template["views"].([]any)
	if !ok || len(rawViews) == 0 {
		return nil, fmt.Errorf("dashboard template %q does not contain any views", templatePath)
	}

	generatedViews := make([]any, 0, len(targets)*len(rawViews))
	for _, target := range targets {
		for _, rawView := range rawViews {
			view, err := deepCopyJSONValue(rawView)
			if err != nil {
				return nil, err
			}
			view = replaceDashboardPlaceholder(view, "YOUR_ESS_PS_KEY", target.PsKey)
			viewMap, ok := view.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("dashboard view prototype has unexpected type %T", view)
			}

			if len(rawViews) == 1 {
				viewMap["title"] = target.ViewTitle
				viewMap["path"] = target.ViewPath
			} else if len(targets) > 1 {
				prototypeTitle := cleanDashboardLabel(fmt.Sprint(viewMap["title"]))
				if prototypeTitle == "" {
					prototypeTitle = "View"
				}
				prototypePath := dashboardSlug(fmt.Sprint(viewMap["path"]))
				if prototypePath == "" {
					prototypePath = dashboardSlug(prototypeTitle)
				}
				viewMap["title"] = fmt.Sprintf("%s %s", target.ViewTitle, prototypeTitle)
				viewMap["path"] = dashboardSlug(fmt.Sprintf("%s-%s", target.ViewPath, prototypePath))
			}

			generatedViews = append(generatedViews, viewMap)
		}
	}

	template["title"] = dashboardTitle
	template["views"] = generatedViews

	localized := localizeDashboardValue(template, dashboardReplacementMap(localeBundle))
	localized = injectFlowCardLabels(localized, localeBundle.FlowCard)
	localizedTemplate, ok := localized.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("localized dashboard template has unexpected type %T", localized)
	}
	return localizedTemplate, nil
}

func installDashboardCardAsset(assetDir string, homeAssistantDir string) (string, string, error) {
	sourcePath := filepath.Join(assetDir, dashboardCardSourceFile)
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", "", err
	}

	candidates := uniqueNonEmptyStrings([]string{homeAssistantDir, "/homeassistant", "/config"})
	sum := sha256.Sum256(data)
	fullHash := hex.EncodeToString(sum[:])
	fileName := dashboardCardFilePrefix + fullHash[:12] + ".js"
	var lastErr error
	var wrote bool
	for _, dir := range candidates {
		targetDir := filepath.Join(dir, "www", dashboardCardResourceDir)
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			lastErr = err
			continue
		}

		targetPath := filepath.Join(targetDir, fileName)
		if err := gosungrowoutput.PlainFileWrite(targetPath, data, 0644); err != nil {
			lastErr = err
			continue
		}
		written, err := os.ReadFile(targetPath)
		if err != nil || sha256.Sum256(written) != sum {
			if err == nil {
				err = fmt.Errorf("staged dashboard asset hash mismatch")
			}
			_ = os.Remove(targetPath)
			lastErr = err
			continue
		}

		wrote = true
	}
	if !wrote {
		if lastErr == nil {
			lastErr = fmt.Errorf("home assistant config directory is empty")
		}
		return "", "", lastErr
	}

	return fmt.Sprintf("/local/%s/%s", dashboardCardResourceDir, fileName), fullHash, nil
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	ret := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		ret = append(ret, value)
	}
	return ret
}

func targetPSKeys(targets []haDashboardTarget) []string {
	keys := make([]string, 0, len(targets))
	for _, target := range targets {
		keys = append(keys, target.PsKey)
	}
	return keys
}

func countDashboardGoSungrowStates(states []haState) int {
	count := 0
	for _, state := range states {
		entityID := strings.ToLower(strings.TrimSpace(state.EntityID))
		if strings.HasPrefix(entityID, "sensor.") && strings.Contains(entityID, "gosungrow") {
			count++
		}
	}
	return count
}

func dashboardDiagnosticContext(context string) string {
	context = strings.TrimSpace(context)
	if context == "" {
		return "manual"
	}
	return context
}

func dashboardDebugEnabled() bool {
	if cmds.Debug {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("GOSUNGROW_DEBUG"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func countDashboardBatteryTargets(targets []haDashboardTarget, states []haState) int {
	count := 0
	singleTarget := len(targets) == 1
	for _, target := range targets {
		if dashboardTargetHasBattery(target, states, singleTarget) {
			count++
		}
	}
	return count
}

func buildDashboardTargetDiagnostics(targets []haDashboardTarget, states []haState) []dashboardTargetDiagnostics {
	ret := make([]dashboardTargetDiagnostics, 0, len(targets))
	singleTarget := len(targets) == 1
	for _, target := range targets {
		diagnostic := dashboardTargetDiagnostics{
			PlantName:       target.PlantName,
			DeviceName:      target.DeviceName,
			PsID:            target.PsID,
			PsKey:           target.PsKey,
			ViewPath:        target.ViewPath,
			DeviceType:      target.DeviceType,
			SelectionSource: target.SelectionSource,
			PlantDevices:    append([]dashboardPlantDevice(nil), target.PlantDevices...),
		}
		for _, state := range states {
			candidate := strings.ToLower(strings.TrimSpace(state.EntityID))
			if !strings.HasPrefix(candidate, "sensor.") || !strings.Contains(candidate, "gosungrow") {
				continue
			}
			if _, ok := dashboardEntityPlantAffinity(candidate, target, singleTarget); !ok {
				continue
			}
			diagnostic.GoSungrowStates++
			diagnostic.ExampleGoSungrowStates = appendDashboardDiagnosticExample(diagnostic.ExampleGoSungrowStates, candidate, 3)
			if dashboardStateMatchesTargetVirtualPrefix(candidate, target) {
				diagnostic.VirtualStates++
				diagnostic.ExampleVirtualStates = appendDashboardDiagnosticExample(diagnostic.ExampleVirtualStates, candidate, 3)
			}
		}
		ret = append(ret, diagnostic)
	}
	return ret
}

func buildDashboardAggregateHints(targets []haDashboardTarget, states []haState) []dashboardAggregateHint {
	if len(targets) == 0 || len(states) == 0 {
		return nil
	}

	metrics := []string{"pv_power", "p13112", "grid_power"}
	singleTarget := len(targets) == 1
	seen := make(map[string]struct{})
	candidates := make([]dashboardMetricCandidate, 0)
	for _, target := range targets {
		for _, metric := range metrics {
			profile := dashboardMetricProfileFor(metric)
			for _, state := range states {
				entity, score, reason, ok := dashboardScoreMetricCandidate(target, metric, profile, state, singleTarget)
				if !ok || entity == "" {
					continue
				}
				key := metric + "\x00" + entity
				if _, exists := seen[key]; exists {
					continue
				}
				seen[key] = struct{}{}
				candidates = append(candidates, dashboardMetricCandidate{
					Entity: entity,
					Metric: metric,
					Score:  score,
					State:  state.State,
					Unit:   dashboardStateUnit(state),
					Source: dashboardMetricSourceCategory(target, metric, entity),
					Reason: reason,
				})
			}
		}
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].Entity < candidates[j].Entity
		}
		return candidates[i].Score > candidates[j].Score
	})

	ret := make([]dashboardAggregateHint, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Metric == "" {
			continue
		}
		ret = append(ret, dashboardAggregateHint{
			Metric: candidate.Metric,
			Entity: candidate.Entity,
			State:  candidate.State,
			Unit:   candidate.Unit,
			Source: candidate.Source,
		})
		if len(ret) >= 12 {
			break
		}
	}
	return ret
}

func appendDashboardDiagnosticExample(values []string, value string, limit int) []string {
	if len(values) >= limit {
		return values
	}
	for _, entry := range values {
		if entry == value {
			return values
		}
	}
	return append(values, value)
}

func dashboardStateMatchesTargetVirtualPrefix(candidate string, target haDashboardTarget) bool {
	psKey := strings.ToLower(strings.TrimSpace(target.PsKey))
	psID := strings.ToLower(strings.TrimSpace(target.PsID))
	switch {
	case psKey != "" && strings.Contains(candidate, "gosungrow_virtual_"+psKey+"_"):
		return true
	case psID != "" && strings.Contains(candidate, "gosungrow_virtual_"+psID+"_"):
		return true
	default:
		return false
	}
}

func dashboardTargetProfileWarnings(target dashboardTargetDiagnostics) []string {
	warnings := make([]string, 0, 3)
	if target.DeviceType != 14 && strings.HasPrefix(target.SelectionSource, "fallback") {
		warnings = append(warnings, fmt.Sprintf("selected fallback non-ESS device_type=%d; full ESS virtual metrics may be unavailable", target.DeviceType))
	} else if target.DeviceType != 14 {
		warnings = append(warnings, fmt.Sprintf("selected non-ESS device_type=%d; some dashboard metrics may be unavailable", target.DeviceType))
	}
	inverterLike := 0
	unselectedInverterLike := 0
	for _, device := range target.PlantDevices {
		if !dashboardPlantDeviceLooksInverterLike(device) {
			continue
		}
		inverterLike++
		if !device.Selected {
			unselectedInverterLike++
		}
	}
	if inverterLike > 1 {
		warnings = append(warnings, fmt.Sprintf("multiple inverter-like devices found for this plant (%d); aggregation may be needed", inverterLike))
	}
	if strings.HasPrefix(target.SelectionSource, "fallback") && unselectedInverterLike > 0 {
		warnings = append(warnings, "selected fallback target while other inverter-like devices exist")
	}
	if target.GoSungrowStates == 0 {
		warnings = append(warnings, "no GoSungrow states matched this target in Home Assistant")
	}
	if target.VirtualStates == 0 {
		warnings = append(warnings, "no target-specific gosungrow_virtual states were found")
	}
	return warnings
}

func dashboardPlantDeviceLooksInverterLike(device dashboardPlantDevice) bool {
	name := strings.ToLower(strings.TrimSpace(device.DeviceName + " " + device.SelectionSource))
	if strings.Contains(name, "inverter") {
		return true
	}
	switch device.DeviceType {
	case 1, 11, 14:
		return true
	default:
		return false
	}
}

func dashboardRemapPreview(remapped []dashboardEntityRemap, limit int) []dashboardEntityRemap {
	if len(remapped) <= limit {
		return append([]dashboardEntityRemap(nil), remapped...)
	}
	return append([]dashboardEntityRemap(nil), remapped[:limit]...)
}

func dashboardSaveReason(saved bool, newConfig bool, changed bool, forceUpdate bool) string {
	switch {
	case forceUpdate:
		return "force update"
	case newConfig:
		return "new dashboard config"
	case changed:
		return "configuration changed"
	case !saved:
		return "unchanged"
	default:
		return "saved"
	}
}

func printDashboardInstallDiagnostics(diagnostics dashboardInstallDiagnostics) {
	writeDashboardInstallDiagnostics(os.Stdout, diagnostics)
}

func writeDashboardInstallDiagnostics(w io.Writer, diagnostics dashboardInstallDiagnostics) {
	fmt.Fprintln(w, "Dashboard diagnostics:")
	fmt.Fprintf(w, "- context: %s\n", dashboardDiagnosticContext(diagnostics.DiagnosticContext))
	fmt.Fprintf(w, "- asset: phase=%s hash=%s url=%s http_route=%s metadata_status=%d metadata_outcome=%s core_port=%d core_tls=%s http_status=%d mime=%s resource_action=%s dashboard_mode=%s rollback=%s cleanup=%s\n",
		dashboardDiagnosticDefault(diagnostics.AssetPhase, "not-started"),
		dashboardDiagnosticDefault(diagnostics.AssetHash, "none"),
		dashboardDiagnosticDefault(diagnostics.AssetURL, "none"),
		dashboardDiagnosticDefault(diagnostics.AssetHTTPRoute, "unknown"),
		diagnostics.AssetMetadataStatus,
		dashboardDiagnosticDefault(diagnostics.AssetMetadataOutcome, "not-used"),
		diagnostics.AssetDiscoveredPort,
		dashboardDiagnosticDefault(diagnostics.AssetDiscoveredTLS, "unknown"),
		diagnostics.AssetHTTPStatus,
		dashboardDiagnosticDefault(diagnostics.AssetMIMEType, "unknown"),
		dashboardDiagnosticDefault(diagnostics.ResourceAction, "none"),
		dashboardDiagnosticDefault(diagnostics.DashboardMode, "unknown"),
		dashboardDiagnosticDefault(diagnostics.RollbackResult, "not-required"),
		dashboardDiagnosticDefault(diagnostics.CleanupResult, "not-run"),
	)
	if diagnostics.HAStatesLoadError != "" {
		fmt.Fprintf(w, "- HA states loaded: failed (%s)\n", diagnostics.HAStatesLoadError)
	} else {
		fmt.Fprintf(w, "- HA states loaded: %d\n", diagnostics.HAStatesLoaded)
	}
	fmt.Fprintf(w, "- GoSungrow states found: %d\n", diagnostics.GoSungrowStatesFound)
	fmt.Fprintf(w, "- dashboard entity refs found: %d\n", diagnostics.DashboardRefsFound)
	fmt.Fprintf(w, "- remapped refs: %d\n", diagnostics.RemappedRefs)
	fmt.Fprintf(w, "- unresolved refs: %d\n", len(diagnostics.UnresolvedRefs))
	if diagnostics.BatteryDetectionKnown {
		if diagnostics.BatteryTargetsTotal == 1 {
			fmt.Fprintf(w, "- battery detected: %t\n", diagnostics.BatteryTargetsFound == 1)
		} else {
			fmt.Fprintf(w, "- battery detected: %d/%d targets\n", diagnostics.BatteryTargetsFound, diagnostics.BatteryTargetsTotal)
		}
	} else {
		fmt.Fprintln(w, "- battery detected: unknown")
	}
	fmt.Fprintf(w, "- dashboard saved: %s (%s)\n", dashboardYesNo(diagnostics.DashboardSaved), diagnostics.DashboardSaveReason)
	if len(diagnostics.TargetDiagnostics) > 0 {
		fmt.Fprintln(w, "Dashboard targets:")
		for index, target := range diagnostics.TargetDiagnostics {
			fmt.Fprintf(w, "- target[%d]: plant=%q device=%q ps_id=%s ps_key=%s device_type=%d selection=%s view=%s gosungrow_states=%d virtual_states=%d\n",
				index+1,
				target.PlantName,
				target.DeviceName,
				target.PsID,
				target.PsKey,
				target.DeviceType,
				target.SelectionSource,
				target.ViewPath,
				target.GoSungrowStates,
				target.VirtualStates,
			)
			for _, warning := range dashboardTargetProfileWarnings(target) {
				fmt.Fprintf(w, "  warning: %s\n", warning)
			}
			if len(target.ExampleVirtualStates) > 0 {
				fmt.Fprintf(w, "  example virtual states: %s\n", strings.Join(target.ExampleVirtualStates, ", "))
			}
			if len(target.ExampleGoSungrowStates) > 0 && len(target.ExampleVirtualStates) == 0 {
				fmt.Fprintf(w, "  example gosungrow states: %s\n", strings.Join(target.ExampleGoSungrowStates, ", "))
			}
			if len(target.PlantDevices) > 0 {
				fmt.Fprintln(w, "  plant devices:")
				for _, device := range target.PlantDevices {
					status := "available"
					if device.Selected {
						status = "selected"
					}
					fmt.Fprintf(w, "    - %s ps_id=%s ps_key=%s device=%q device_type=%d selection=%s\n",
						status,
						device.PsID,
						device.PsKey,
						device.DeviceName,
						device.DeviceType,
						device.SelectionSource,
					)
				}
			}
		}
	}

	if len(diagnostics.RemappedPreview) > 0 {
		fmt.Fprintln(w, "Remapped dashboard refs:")
		for _, remapped := range diagnostics.RemappedPreview {
			fmt.Fprintf(w, "- %s [%s]: %s -> %s\n", remapped.Metric, remapped.Source, remapped.From, remapped.To)
		}
	}

	if len(diagnostics.AggregateHints) > 0 {
		fmt.Fprintln(w, "Potential aggregate sources:")
		for _, hint := range diagnostics.AggregateHints {
			unitSuffix := ""
			if hint.Unit != "" {
				unitSuffix = " " + hint.Unit
			}
			fmt.Fprintf(w, "- %s [%s] state=%s%s entity=%s\n", hint.Metric, hint.Source, hint.State, unitSuffix, hint.Entity)
		}
	}

	if diagnostics.Debug && len(diagnostics.MetricTraces) > 0 {
		fmt.Fprintln(w, "Dashboard metric candidates:")
		for _, trace := range diagnostics.MetricTraces {
			fmt.Fprintf(w, "- metric=%s target=%s placeholder=%s resolved=%s source=%s\n",
				trace.Metric,
				trace.TargetPsKey,
				trace.Placeholder,
				trace.Resolved,
				trace.Source,
			)
			for _, candidate := range trace.Candidates {
				unitSuffix := ""
				if candidate.Unit != "" {
					unitSuffix = " " + candidate.Unit
				}
				fmt.Fprintf(w, "  candidate score=%d source=%s state=%s%s entity=%s reason=%s\n",
					candidate.Score,
					candidate.Source,
					candidate.State,
					unitSuffix,
					candidate.Entity,
					candidate.Reason,
				)
			}
		}
	}

	if len(diagnostics.UnresolvedRefs) == 0 {
		return
	}
	fmt.Fprintln(w, "Unresolved dashboard refs:")
	for _, unresolved := range diagnostics.UnresolvedRefs {
		fmt.Fprintf(w, "- %s: %s\n", unresolved.Entity, unresolved.Reason)
	}
}

func dashboardYesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func dashboardStatePath() string {
	configPath := strings.TrimSpace(os.Getenv("GOSUNGROW_CONFIG"))
	if configPath == "" {
		return filepath.Join(os.TempDir(), dashboardStateFileName)
	}
	return filepath.Join(filepath.Dir(configPath), dashboardStateFileName)
}

func loadDashboardState(path string) (*haDashboardState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var state haDashboardState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func saveDashboardState(path string, state *haDashboardState) error {
	if state == nil {
		return nil
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return gosungrowoutput.PlainFileWrite(path, data, 0600)
}

func deepCopyJSONValue(value any) (any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	var copied any
	if err := json.Unmarshal(data, &copied); err != nil {
		return nil, err
	}
	return copied, nil
}

func replaceDashboardPlaceholder(value any, placeholder string, replacement string) any {
	switch typed := value.(type) {
	case map[string]any:
		ret := make(map[string]any, len(typed))
		for key, entry := range typed {
			ret[key] = replaceDashboardPlaceholder(entry, placeholder, replacement)
		}
		return ret
	case []any:
		ret := make([]any, 0, len(typed))
		for _, entry := range typed {
			ret = append(ret, replaceDashboardPlaceholder(entry, placeholder, replacement))
		}
		return ret
	case string:
		return strings.ReplaceAll(typed, placeholder, replacement)
	default:
		return value
	}
}

func hashCanonicalJSON(value any) (string, error) {
	data, err := marshalCanonicalJSON(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func marshalCanonicalJSON(value any) ([]byte, error) {
	var buf bytes.Buffer
	if err := writeCanonicalJSON(&buf, value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeCanonicalJSON(buf *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if typed {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case string:
		data, err := json.Marshal(typed)
		if err != nil {
			return err
		}
		buf.Write(data)
	case float64, float32, int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8, json.Number:
		data, err := json.Marshal(typed)
		if err != nil {
			return err
		}
		buf.Write(data)
	case []any:
		buf.WriteByte('[')
		for i, entry := range typed {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeCanonicalJSON(buf, entry); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		buf.WriteByte('{')
		for i, key := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			keyData, err := json.Marshal(key)
			if err != nil {
				return err
			}
			buf.Write(keyData)
			buf.WriteByte(':')
			if err := writeCanonicalJSON(buf, typed[key]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return err
		}
		buf.Write(data)
	}

	return nil
}

func cleanDashboardLabel(value string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(value), " "))
}

var dashboardSlugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func dashboardSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = dashboardSlugPattern.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "gosungrow-flow"
	}
	return value
}

func newHAWSClient(ctx context.Context, endpoint string, token string) (*haWSClient, error) {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+token)

	dialer := websocket.Dialer{HandshakeTimeout: 15 * time.Second}
	conn, _, err := dialer.DialContext(ctx, endpoint, headers)
	if err != nil {
		return nil, err
	}

	client := &haWSClient{conn: conn}
	if err := client.authenticate(token); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

func (c *haWSClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *haWSClient) authenticate(token string) error {
	var authRequired struct {
		Type string `json:"type"`
	}
	if err := c.readJSON(&authRequired); err != nil {
		return err
	}
	if authRequired.Type != "auth_required" {
		return fmt.Errorf("unexpected websocket handshake response %q", authRequired.Type)
	}

	if err := c.conn.WriteJSON(map[string]any{
		"type":         "auth",
		"access_token": token,
	}); err != nil {
		return err
	}

	var authResponse struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	}
	if err := c.readJSON(&authResponse); err != nil {
		return err
	}
	if authResponse.Type != "auth_ok" {
		return fmt.Errorf("home assistant websocket authentication failed: %s", authResponse.Message)
	}
	return nil
}

func (c *haWSClient) readJSON(target any) error {
	if err := c.conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return err
	}
	return c.conn.ReadJSON(target)
}

func (c *haWSClient) call(request map[string]any, out any) error {
	c.nextID++
	request["id"] = c.nextID
	requestID := c.nextID

	if err := c.conn.SetWriteDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return err
	}
	if err := c.conn.WriteJSON(request); err != nil {
		return err
	}

	for {
		var response haWSResponse
		if err := c.readJSON(&response); err != nil {
			return err
		}
		if response.Type != "result" || response.ID != requestID {
			continue
		}
		if !response.Success {
			if response.Error == nil {
				return fmt.Errorf("home assistant websocket request failed")
			}
			return &haWSCallError{Code: response.Error.Code, Message: response.Error.Message}
		}
		if out == nil || len(response.Result) == 0 {
			return nil
		}
		return json.Unmarshal(response.Result, out)
	}
}

func (c *haWSClient) ListDashboards(_ context.Context) ([]haDashboardMetadata, error) {
	var dashboards []haDashboardMetadata
	if err := c.call(map[string]any{"type": "lovelace/dashboards/list"}, &dashboards); err != nil {
		return nil, err
	}
	return dashboards, nil
}

func (c *haWSClient) ListResources(_ context.Context) ([]haResourceMetadata, error) {
	var resources []haResourceMetadata
	if err := c.call(map[string]any{"type": "lovelace/resources/list"}, &resources); err != nil {
		return nil, err
	}
	return resources, nil
}

func (c *haWSClient) ListStates(_ context.Context) ([]haState, error) {
	var states []haState
	if err := c.call(map[string]any{"type": "get_states"}, &states); err != nil {
		return nil, err
	}

	deduped := make([]haState, 0, len(states))
	seen := make(map[string]struct{}, len(states))
	for _, state := range states {
		entityID := strings.TrimSpace(state.EntityID)
		if entityID == "" {
			continue
		}
		normalized := strings.ToLower(entityID)
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		state.EntityID = entityID
		deduped = append(deduped, state)
	}
	return deduped, nil
}

func (c *haWSClient) ListEntityRegistry(_ context.Context) ([]haEntityRegistryEntry, error) {
	var entries []haEntityRegistryEntry
	if err := c.call(map[string]any{"type": "config/entity_registry/list"}, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func enrichDashboardStatesWithRegistry(states []haState, registry []haEntityRegistryEntry) []haState {
	byEntity := make(map[string]haEntityRegistryEntry, len(registry))
	for _, entry := range registry {
		byEntity[strings.ToLower(strings.TrimSpace(entry.EntityID))] = entry
	}
	ret := append([]haState(nil), states...)
	for index := range ret {
		entry, ok := byEntity[strings.ToLower(strings.TrimSpace(ret[index].EntityID))]
		if !ok {
			continue
		}
		ret[index].RegistryUniqueID = strings.TrimSpace(entry.UniqueID)
		ret[index].RegistryDeviceID = strings.TrimSpace(entry.DeviceID)
		ret[index].RegistryPlatform = strings.TrimSpace(entry.Platform)
	}
	return ret
}

func (c *haWSClient) ListStateEntityIDs(ctx context.Context) ([]string, error) {
	states, err := c.ListStates(ctx)
	if err != nil {
		return nil, err
	}

	entityIDs := make([]string, 0, len(states))
	for _, state := range states {
		entityIDs = append(entityIDs, state.EntityID)
	}
	return entityIDs, nil
}

func (c *haWSClient) CreateResource(_ context.Context, url string, resourceType string) error {
	return c.call(map[string]any{
		"type":     "lovelace/resources/create",
		"url":      url,
		"res_type": resourceType,
	}, nil)
}

func (c *haWSClient) UpdateResource(_ context.Context, resourceID any, url string, resourceType string) error {
	return c.call(map[string]any{
		"type":        "lovelace/resources/update",
		"resource_id": resourceID,
		"url":         url,
		"res_type":    resourceType,
	}, nil)
}

func (c *haWSClient) EnsureResource(ctx context.Context, url string, resourceType string) error {
	resources, err := c.ListResources(ctx)
	if err != nil {
		return err
	}

	targetBase := resourceURLBase(url)
	isManagedDashboardCard := matchesManagedDashboardCardResource(url)
	for _, resource := range resources {
		resourceBase := resourceURLBase(resource.URL)
		if resourceBase != targetBase && !(isManagedDashboardCard && matchesManagedDashboardCardResource(resource.URL)) {
			continue
		}
		existingType := strings.TrimSpace(resource.ResourceType)
		if existingType == "" {
			existingType = strings.TrimSpace(resource.ResType)
		}
		if resource.URL == url && existingType == resourceType {
			return nil
		}
		return c.UpdateResource(ctx, resource.ID, url, resourceType)
	}

	return c.CreateResource(ctx, url, resourceType)
}

func resourceURLBase(url string) string {
	if strings.HasPrefix(url, "data:text/javascript;base64,") {
		return "data:text/javascript;base64,"
	}
	if idx := strings.IndexAny(url, "?#"); idx >= 0 {
		return url[:idx]
	}
	return url
}

func matchesManagedDashboardCardResource(url string) bool {
	base := resourceURLBase(strings.TrimSpace(url))
	if base == "data:text/javascript;base64," {
		return true
	}
	lower := strings.ToLower(base)
	if index := strings.LastIndex(lower, "/"); index >= 0 {
		lower = lower[index+1:]
	}
	return lower == dashboardCardSourceFile || managedDashboardBundlePattern.MatchString(lower)
}

func (c *haWSClient) CreateDashboard(_ context.Context, opts haDashboardInstallOptions) error {
	return c.call(map[string]any{
		"type":            "lovelace/dashboards/create",
		"url_path":        opts.DashboardURLPath,
		"title":           opts.DashboardTitle,
		"icon":            opts.DashboardIcon,
		"show_in_sidebar": opts.ShowInSidebar,
		"require_admin":   opts.RequireAdmin,
	}, nil)
}

func (c *haWSClient) UpdateDashboard(_ context.Context, dashboardID string, opts haDashboardInstallOptions) error {
	return c.call(map[string]any{
		"type":            "lovelace/dashboards/update",
		"dashboard_id":    dashboardID,
		"title":           opts.DashboardTitle,
		"icon":            opts.DashboardIcon,
		"show_in_sidebar": opts.ShowInSidebar,
		"require_admin":   opts.RequireAdmin,
	}, nil)
}

func (c *haWSClient) DeleteDashboard(ctx context.Context, urlPath string) error {
	dashboards, err := c.ListDashboards(ctx)
	if err != nil {
		return err
	}
	for _, dashboard := range dashboards {
		if dashboard.URLPath != urlPath || strings.TrimSpace(dashboard.ID) == "" {
			continue
		}
		return c.call(map[string]any{
			"type":         "lovelace/dashboards/delete",
			"dashboard_id": dashboard.ID,
		}, nil)
	}
	return nil
}

func (c *haWSClient) GetConfig(_ context.Context, urlPath string) (map[string]any, error) {
	var config map[string]any
	if err := c.call(map[string]any{
		"type":     "lovelace/config",
		"url_path": urlPath,
		"force":    true,
	}, &config); err != nil {
		return nil, err
	}
	return config, nil
}

func (c *haWSClient) SaveConfig(_ context.Context, urlPath string, config map[string]any) error {
	return c.call(map[string]any{
		"type":     "lovelace/config/save",
		"url_path": urlPath,
		"config":   config,
	}, nil)
}
