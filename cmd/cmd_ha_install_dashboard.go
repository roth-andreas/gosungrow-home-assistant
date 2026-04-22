package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud"
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
	defaultSupervisorCoreURL   = "http://supervisor/core"
	dashboardTemplateFile      = "home-assistant-sungrow-flow.yaml"
	dashboardStateFileName     = "dashboard_state.json"
	dashboardCardFileName      = "gosungrow-energy-flow-card-v2.js"
	dashboardCardResourceDir   = "gosungrow"
	dashboardCardResourceType  = "module"
)

type haDashboardInstallOptions struct {
	AssetDir           string
	HomeAssistantDir   string
	HomeAssistantWSURL string
	HomeAssistantURL   string
	SupervisorToken    string
	Language           string
	DashboardURLPath   string
	DashboardTitle     string
	DashboardIcon      string
	ShowInSidebar      bool
	RequireAdmin       bool
	ForceUpdate        bool
}

type haDashboardTarget struct {
	PsID      string
	PsKey     string
	ViewTitle string
	ViewPath  string
}

type haDashboardState struct {
	DashboardURLPath string   `json:"dashboard_url_path"`
	DashboardHash    string   `json:"dashboard_hash"`
	TargetPsKeys     []string `json:"target_ps_keys,omitempty"`
	UpdatedAt        string   `json:"updated_at"`
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
	EntityID   string         `json:"entity_id"`
	State      string         `json:"state"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

type dashboardInstallDiagnostics struct {
	HAStatesLoaded        int
	HAStatesLoadError     string
	GoSungrowStatesFound  int
	DashboardRefsFound    int
	RemappedRefs          int
	UnresolvedRefs        []dashboardUnresolvedEntityRef
	BatteryDetectionKnown bool
	BatteryTargetsFound   int
	BatteryTargetsTotal   int
	DashboardSaved        bool
	DashboardSaveReason   string
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
		HomeAssistantURL:   defaultSupervisorCoreURL,
		SupervisorToken:    os.Getenv("SUPERVISOR_TOKEN"),
		Language:           "auto",
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
	cmd.Flags().StringVar(&opts.HomeAssistantURL, "ha-url", opts.HomeAssistantURL, "Home Assistant base URL used to verify custom card assets.")
	cmd.Flags().StringVar(&opts.SupervisorToken, "supervisor-token", opts.SupervisorToken, "Supervisor token used to access the Home Assistant websocket.")
	cmd.Flags().StringVar(&opts.Language, "language", opts.Language, "Dashboard language (auto, en, de, sv, or locale such as de-DE).")
	cmd.Flags().StringVar(&opts.DashboardURLPath, "url-path", opts.DashboardURLPath, "Dashboard URL path.")
	cmd.Flags().StringVar(&opts.DashboardTitle, "title", opts.DashboardTitle, "Dashboard title.")
	cmd.Flags().StringVar(&opts.DashboardIcon, "icon", opts.DashboardIcon, "Dashboard sidebar icon.")
	cmd.Flags().BoolVar(&opts.ShowInSidebar, "show-in-sidebar", opts.ShowInSidebar, "Show the dashboard in the Home Assistant sidebar.")
	cmd.Flags().BoolVar(&opts.RequireAdmin, "require-admin", opts.RequireAdmin, "Restrict dashboard access to Home Assistant administrators.")
	cmd.Flags().BoolVar(&opts.ForceUpdate, "force-update", opts.ForceUpdate, "Replace an existing dashboard even if it was modified outside GoSungrow.")
	_ = cmd.Flags().MarkHidden("supervisor-token")
	_ = cmd.Flags().MarkHidden("ha-config-dir")
	_ = cmd.Flags().MarkHidden("ha-url")

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

	templatePath := filepath.Join(opts.AssetDir, dashboardTemplateFile)

	localResourceURL, assetVersion, err := installDashboardCardAsset(opts.AssetDir, opts.HomeAssistantDir)
	if err != nil {
		return err
	}
	resourceURL := localResourceURL
	ok, verifyErr := verifyDashboardCardResource(opts.HomeAssistantURL, opts.SupervisorToken, localResourceURL)
	if verifyErr != nil || !ok {
		resourceURL, err = dashboardCardDataURI(filepath.Join(opts.AssetDir, dashboardCardFileName), assetVersion)
		if err != nil {
			return err
		}
		fmt.Printf("Managed GoSungrow custom card local asset unavailable; using embedded fallback resource.\n")
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

	config, err := renderDashboardConfig(templatePath, opts.DashboardTitle, targets, localeBundle)
	if err != nil {
		return err
	}

	diagnostics := dashboardInstallDiagnostics{
		BatteryTargetsTotal: len(targets),
		DashboardSaveReason: "unchanged",
	}
	states, listErr := client.ListStates(ctx)
	var remapReport dashboardRemapReport
	if listErr != nil {
		diagnostics.HAStatesLoadError = listErr.Error()
		_, remapReport = remapDashboardEntitiesWithReport(config, targets, nil)
	} else {
		diagnostics.HAStatesLoaded = len(states)
		diagnostics.GoSungrowStatesFound = countDashboardGoSungrowStates(states)
		diagnostics.BatteryDetectionKnown = true
		diagnostics.BatteryTargetsFound = countDashboardBatteryTargets(targets, states)
		config = pruneDashboardForMissingBattery(config, targets, states)
		config, remapReport = remapDashboardEntitiesWithReport(config, targets, states)
	}
	diagnostics.DashboardRefsFound = remapReport.TotalRefs
	diagnostics.RemappedRefs = len(remapReport.Remapped)
	diagnostics.UnresolvedRefs = remapReport.Unresolved

	desiredHash, err := hashCanonicalJSON(config)
	if err != nil {
		return err
	}

	if err := client.EnsureResource(ctx, resourceURL, dashboardCardResourceType); err != nil {
		return err
	}

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

	currentConfig, err := client.GetConfig(ctx, opts.DashboardURLPath)
	if err != nil {
		wsErr, ok := err.(*haWSCallError)
		if !(ok && wsErr.IsCode("config_not_found")) {
			return err
		}
		currentConfig = nil
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
	if exists && managedByState && state.DashboardHash != "" && currentHash != "" && currentHash != state.DashboardHash && !opts.ForceUpdate {
		return fmt.Errorf("dashboard %q was modified outside GoSungrow; set dashboard_force_update to true to replace it", opts.DashboardURLPath)
	}

	if exists {
		if err := client.UpdateDashboard(ctx, existing.ID, opts); err != nil {
			wsErr, ok := err.(*haWSCallError)
			if !(ok && wsErr.IsCode("not_found")) {
				return err
			}
			exists = false
		}
	}
	if !exists {
		if err := client.CreateDashboard(ctx, opts); err != nil {
			return err
		}
	}

	shouldSaveConfig := currentHash != desiredHash || currentConfig == nil || opts.ForceUpdate
	diagnostics.DashboardSaved = shouldSaveConfig
	diagnostics.DashboardSaveReason = dashboardSaveReason(shouldSaveConfig, currentConfig == nil, currentHash != desiredHash, opts.ForceUpdate)
	if shouldSaveConfig {
		if err := client.SaveConfig(ctx, opts.DashboardURLPath, config); err != nil {
			return err
		}
	}

	if err := saveDashboardState(statePath, &haDashboardState{
		DashboardURLPath: opts.DashboardURLPath,
		DashboardHash:    desiredHash,
		TargetPsKeys:     targetPSKeys(targets),
		UpdatedAt:        time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return err
	}

	viewCount := 0
	if views, ok := config["views"].([]any); ok {
		viewCount = len(views)
	}
	if viewCount == 0 {
		viewCount = len(targets)
	}
	printDashboardInstallDiagnostics(diagnostics)
	fmt.Printf("Managed GoSungrow dashboard ready at /%s with %d view(s).\n", opts.DashboardURLPath, viewCount)
	return nil
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
		psID       string
		psKey      string
		plantName  string
		deviceName string
	}

	collected := make([]deviceTarget, 0)
	perPlantCounts := make(map[string]int)
	for _, psID := range psIDs {
		tree := trees[psID]
		plantTargets := make([]deviceTarget, 0)
		fallbackTargets := make([]deviceTarget, 0)
		for _, device := range tree.Devices {
			target := deviceTarget{
				psID:       psID,
				psKey:      strings.TrimSpace(device.PsKey.String()),
				plantName:  cleanDashboardLabel(device.PsName.String()),
				deviceName: cleanDashboardLabel(device.DeviceName.String()),
			}
			if target.psKey == "" {
				continue
			}
			if target.plantName == "" {
				target.plantName = fmt.Sprintf("Plant %s", psID)
			}
			if device.DeviceType.Match(14) {
				plantTargets = append(plantTargets, target)
				continue
			}
			fallbackTargets = append(fallbackTargets, target)
		}

		if len(plantTargets) == 0 && len(fallbackTargets) > 0 {
			plantTargets = append(plantTargets, fallbackTargets[0])
		}
		if len(plantTargets) == 0 {
			continue
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
			PsID:      target.psID,
			PsKey:     target.psKey,
			ViewTitle: title,
			ViewPath:  dashboardSlug(target.psKey),
		})
	}

	return ret, nil
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
	sourcePath := filepath.Join(assetDir, dashboardCardFileName)
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", "", err
	}

	candidates := uniqueNonEmptyStrings([]string{homeAssistantDir, "/homeassistant", "/config"})
	var lastErr error
	var wrote bool
	for _, dir := range candidates {
		targetDir := filepath.Join(dir, "www", dashboardCardResourceDir)
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			lastErr = err
			continue
		}

		targetPath := filepath.Join(targetDir, dashboardCardFileName)
		if err := os.WriteFile(targetPath, data, 0644); err != nil {
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

	sum := sha256.Sum256(data)
	version := hex.EncodeToString(sum[:6])
	return fmt.Sprintf("/local/%s/%s?v=%s", dashboardCardResourceDir, dashboardCardFileName, version), version, nil
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

func verifyDashboardCardResource(homeAssistantURL string, supervisorToken string, resourceURL string) (bool, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(homeAssistantURL), "/")
	if baseURL == "" {
		return false, fmt.Errorf("home assistant url is empty")
	}

	parsed, err := url.Parse(resourceURL)
	if err != nil {
		return false, err
	}

	verifyURL := baseURL + parsed.Path
	req, err := http.NewRequest(http.MethodGet, verifyURL, nil)
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(supervisorToken) != "" {
		req.Header.Set("Authorization", "Bearer "+supervisorToken)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode == http.StatusOK, nil
}

func dashboardCardDataURI(sourcePath string, version string) (string, error) {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	version = strings.TrimSpace(version)
	if version == "" {
		return "data:text/javascript;base64," + encoded, nil
	}
	return fmt.Sprintf("data:text/javascript;base64,%s#v=%s", encoded, version), nil
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
	return os.WriteFile(path, data, 0600)
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
	if err := c.call(map[string]any{"type": "lovelace/resources"}, &resources); err != nil {
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
	return strings.Contains(base, "/"+dashboardCardResourceDir+"/"+dashboardCardFileName) || strings.Contains(base, dashboardCardFileName)
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
