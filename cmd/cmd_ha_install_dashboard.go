package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	defaultHADashboardAssetDir = "/opt/gosungrow/assets"
	defaultHAWebsocketURL      = "ws://supervisor/core/websocket"
	defaultDashboardURLPath    = "gosungrow-flow"
	defaultDashboardTitle      = "GoSungrow Flow"
	defaultDashboardIcon       = "mdi:solar-power"
	dashboardTemplateFile      = "home-assistant-sungrow-flow.yaml"
	dashboardStateFileName     = "dashboard_state.json"
	dashboardImageDirectory    = "gosungrow"
)

type haDashboardInstallOptions struct {
	AssetDir           string
	HomeAssistantWSURL string
	SupervisorToken    string
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
		HomeAssistantWSURL: defaultHAWebsocketURL,
		SupervisorToken:    os.Getenv("SUPERVISOR_TOKEN"),
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
	cmd.Flags().StringVar(&opts.HomeAssistantWSURL, "ha-ws-url", opts.HomeAssistantWSURL, "Home Assistant websocket endpoint.")
	cmd.Flags().StringVar(&opts.SupervisorToken, "supervisor-token", opts.SupervisorToken, "Supervisor token used to access the Home Assistant websocket.")
	cmd.Flags().StringVar(&opts.DashboardURLPath, "url-path", opts.DashboardURLPath, "Dashboard URL path.")
	cmd.Flags().StringVar(&opts.DashboardTitle, "title", opts.DashboardTitle, "Dashboard title.")
	cmd.Flags().StringVar(&opts.DashboardIcon, "icon", opts.DashboardIcon, "Dashboard sidebar icon.")
	cmd.Flags().BoolVar(&opts.ShowInSidebar, "show-in-sidebar", opts.ShowInSidebar, "Show the dashboard in the Home Assistant sidebar.")
	cmd.Flags().BoolVar(&opts.RequireAdmin, "require-admin", opts.RequireAdmin, "Restrict dashboard access to Home Assistant administrators.")
	cmd.Flags().BoolVar(&opts.ForceUpdate, "force-update", opts.ForceUpdate, "Replace an existing dashboard even if it was modified outside GoSungrow.")
	_ = cmd.Flags().MarkHidden("supervisor-token")

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

	if err := copyDashboardAssets(opts.AssetDir); err != nil {
		return err
	}

	templatePath := filepath.Join(opts.AssetDir, dashboardTemplateFile)
	config, err := renderDashboardConfig(templatePath, opts.DashboardTitle, targets)
	if err != nil {
		return err
	}

	desiredHash, err := hashCanonicalJSON(config)
	if err != nil {
		return err
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

	if currentHash != desiredHash || currentConfig == nil || opts.ForceUpdate {
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

	fmt.Printf("Managed GoSungrow dashboard ready at /%s with %d view(s).\n", opts.DashboardURLPath, len(targets))
	return nil
}

func (c *CmdHa) discoverDashboardTargets(args []string) ([]haDashboardTarget, error) {
	trees, err := cmds.Api.SunGrow.PsTreeMenu(args...)
	if err != nil {
		return nil, err
	}

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
		for _, device := range tree.Devices {
			if !device.DeviceType.Match(14) {
				continue
			}

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
			collected = append(collected, target)
			perPlantCounts[psID]++
		}
	}

	if len(collected) == 0 {
		return nil, fmt.Errorf("no Sungrow device type 14 devices were discovered")
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

func renderDashboardConfig(templatePath string, dashboardTitle string, targets []haDashboardTarget) (map[string]any, error) {
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

	prototype := rawViews[0]
	generatedViews := make([]any, 0, len(targets))
	for _, target := range targets {
		view, err := deepCopyJSONValue(prototype)
		if err != nil {
			return nil, err
		}
		view = replaceDashboardPlaceholder(view, "YOUR_ESS_PS_KEY", target.PsKey)
		viewMap, ok := view.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("dashboard view prototype has unexpected type %T", view)
		}
		viewMap["title"] = target.ViewTitle
		viewMap["path"] = target.ViewPath
		generatedViews = append(generatedViews, viewMap)
	}

	template["title"] = dashboardTitle
	template["views"] = generatedViews
	return template, nil
}

func copyDashboardAssets(assetDir string) error {
	sourcePattern := filepath.Join(assetDir, "SungrowEnergy2*.png")
	files, err := filepath.Glob(sourcePattern)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no dashboard images found in %q", assetDir)
	}

	destRoots := existingDashboardAssetRoots()
	if len(destRoots) == 0 {
		return fmt.Errorf("no Home Assistant config root available for dashboard assets")
	}

	for _, root := range destRoots {
		destDir := filepath.Join(root, "www", dashboardImageDirectory)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return err
		}

		for _, src := range files {
			data, err := os.ReadFile(src)
			if err != nil {
				return err
			}
			destPath := filepath.Join(destDir, filepath.Base(src))
			if err := os.WriteFile(destPath, data, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

func existingDashboardAssetRoots() []string {
	candidates := []string{
		strings.TrimSpace(os.Getenv("GOSUNGROW_HOMEASSISTANT_CONFIG_DIR")),
		strings.TrimSpace(os.Getenv("HOMEASSISTANT_CONFIG_DIR")),
		"/config",
		"/homeassistant",
	}

	seen := make(map[string]bool)
	roots := make([]string, 0, len(candidates))
	for _, root := range candidates {
		if root == "" || seen[root] {
			continue
		}
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue
		}
		seen[root] = true
		roots = append(roots, root)
	}

	return roots
}

func targetPSKeys(targets []haDashboardTarget) []string {
	keys := make([]string, 0, len(targets))
	for _, target := range targets {
		keys = append(keys, target.PsKey)
	}
	return keys
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
