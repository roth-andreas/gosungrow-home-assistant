package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud"
)

const (
	dashboardAssetModeEnhanced        = "enhanced"
	dashboardAssetModeFallback        = "native-fallback"
	dashboardHTTPRouteSupervisorInfo  = "supervisor-core-info"
	dashboardHTTPRouteWebsocketOrigin = "websocket-origin"
	dashboardSupervisorCoreInfoURL    = "http://supervisor/core/info"
)

var managedDashboardBundlePattern = regexp.MustCompile(`^gosungrow-dashboard-cards\.([0-9a-f]{12})\.js$`)

var dashboardAssetHTTPClient = &http.Client{
	Timeout: 15 * time.Second,
	CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

type dashboardAssetVerification struct {
	StatusCode         int
	MIMEType           string
	Route              string
	MetadataStatus     int
	MetadataOutcome    string
	DiscoveredPort     int
	DiscoveredTLS      bool
	StaticRouteOutcome string
	OperatorAction     string
}

const dashboardCoreRestartAction = "restart Home Assistant Core once; restarting the GoSungrow app is insufficient; GoSungrow reconciliation will retry automatically"

type dashboardStaticRouteUnavailableError struct{}

func (e *dashboardStaticRouteUnavailableError) Error() string {
	return "home assistant static route unavailable: staged dashboard asset returned HTTP 404; " + dashboardCoreRestartAction
}

func (e *dashboardStaticRouteUnavailableError) FailureClass() iSolarCloud.FailureClass {
	return iSolarCloud.FailureClassOperatorActionRequired
}

type dashboardResourceChange struct {
	Action    string
	Canonical haResourceMetadata
	Before    []haResourceMetadata
}

func dashboardHTTPResourceURL(websocketURL, resourceURL string) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(websocketURL))
	if err != nil {
		return "", "", err
	}
	path := strings.TrimSuffix(parsed.Path, "/")
	if strings.EqualFold(parsed.Hostname(), "supervisor") && strings.HasSuffix(path, "/core/websocket") {
		return "", dashboardHTTPRouteSupervisorInfo, fmt.Errorf("Supervisor websocket endpoint requires Core endpoint discovery")
	}
	switch parsed.Scheme {
	case "ws":
		parsed.Scheme = "http"
	case "wss":
		parsed.Scheme = "https"
	default:
		return "", "", fmt.Errorf("unsupported Home Assistant websocket scheme %q", parsed.Scheme)
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	if strings.HasSuffix(path, "/api/websocket") {
		path = strings.TrimSuffix(path, "/api/websocket")
	} else if strings.HasSuffix(path, "/websocket") {
		path = strings.TrimSuffix(path, "/websocket")
	}
	parsed.Path = strings.TrimSuffix(path, "/") + "/" + strings.TrimPrefix(resourceURL, "/")
	return parsed.String(), dashboardHTTPRouteWebsocketOrigin, nil
}

func verifyDashboardCardAsset(ctx context.Context, websocketURL, supervisorToken, resourceURL, expectedHash string) (dashboardAssetVerification, error) {
	return verifyDashboardCardAssetWithClient(ctx, dashboardAssetHTTPClient, websocketURL, supervisorToken, resourceURL, expectedHash)
}

type dashboardHTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type dashboardRequestError struct {
	operation string
	err       error
}

func (e *dashboardRequestError) Error() string {
	switch {
	case errors.Is(e.err, context.DeadlineExceeded):
		return e.operation + ": request timed out"
	case errors.Is(e.err, context.Canceled):
		return e.operation + ": request canceled"
	case errors.Is(e.err, syscall.ECONNREFUSED):
		return e.operation + ": connection refused"
	case errors.Is(e.err, syscall.ECONNRESET):
		return e.operation + ": connection reset"
	}
	var dnsErr *net.DNSError
	if errors.As(e.err, &dnsErr) {
		return e.operation + ": DNS lookup failed"
	}
	var netErr net.Error
	if errors.As(e.err, &netErr) && netErr.Timeout() {
		return e.operation + ": request timed out"
	}
	return e.operation + ": request failed"
}

func (e *dashboardRequestError) Unwrap() error {
	return e.err
}

func verifyDashboardCardAssetWithClient(ctx context.Context, client dashboardHTTPDoer, websocketURL, supervisorToken, resourceURL, expectedHash string) (dashboardAssetVerification, error) {
	httpURL, verification, err := resolveDashboardHTTPResourceURL(ctx, client, websocketURL, supervisorToken, resourceURL)
	if err != nil {
		return verification, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, httpURL, nil)
	if err != nil {
		return verification, err
	}
	response, err := client.Do(req)
	if err != nil {
		return verification, &dashboardRequestError{operation: "fetch staged dashboard asset", err: err}
	}
	defer response.Body.Close()

	verification.StatusCode = response.StatusCode
	mediaType, _, parseErr := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if parseErr == nil {
		verification.MIMEType = strings.ToLower(mediaType)
	}
	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusNotFound {
			verification.StaticRouteOutcome = "home-assistant-static-route-unavailable"
			verification.OperatorAction = dashboardCoreRestartAction
			return verification, &dashboardStaticRouteUnavailableError{}
		}
		verification.StaticRouteOutcome = "http-error"
		return verification, fmt.Errorf("staged dashboard asset returned HTTP %d", response.StatusCode)
	}
	verification.StaticRouteOutcome = "available"
	if !dashboardJavaScriptMIMEType(verification.MIMEType) {
		return verification, fmt.Errorf("staged dashboard asset returned non-JavaScript MIME type %q", verification.MIMEType)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 16<<20))
	if err != nil {
		return verification, fmt.Errorf("read staged dashboard asset: %w", err)
	}
	sum := sha256.Sum256(body)
	if got := hex.EncodeToString(sum[:]); !strings.EqualFold(got, expectedHash) {
		return verification, fmt.Errorf("staged dashboard asset hash mismatch: got %s", got)
	}
	return verification, nil
}

func resolveDashboardHTTPResourceURL(ctx context.Context, client dashboardHTTPDoer, websocketURL, supervisorToken, resourceURL string) (string, dashboardAssetVerification, error) {
	parsed, err := url.Parse(strings.TrimSpace(websocketURL))
	if err != nil {
		return "", dashboardAssetVerification{}, err
	}
	path := strings.TrimSuffix(parsed.Path, "/")
	if !strings.EqualFold(parsed.Hostname(), "supervisor") || !strings.HasSuffix(path, "/core/websocket") {
		httpURL, route, err := dashboardHTTPResourceURL(websocketURL, resourceURL)
		return httpURL, dashboardAssetVerification{Route: route}, err
	}

	verification := dashboardAssetVerification{
		Route:           dashboardHTTPRouteSupervisorInfo,
		MetadataOutcome: "request-failed",
	}
	discoveryCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(discoveryCtx, http.MethodGet, dashboardSupervisorCoreInfoURL, nil)
	if err != nil {
		return "", verification, fmt.Errorf("discover Home Assistant Core endpoint: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+supervisorToken)
	response, err := client.Do(req)
	if err != nil {
		return "", verification, &dashboardRequestError{operation: "discover Home Assistant Core endpoint", err: err}
	}
	defer response.Body.Close()
	verification.MetadataStatus = response.StatusCode
	if response.StatusCode != http.StatusOK {
		verification.MetadataOutcome = "http-error"
		return "", verification, fmt.Errorf("discover Home Assistant Core endpoint: Supervisor returned HTTP %d", response.StatusCode)
	}

	var envelope struct {
		Result string `json:"result"`
		Data   struct {
			Port json.RawMessage `json:"port"`
			SSL  json.RawMessage `json:"ssl"`
		} `json:"data"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(&envelope); err != nil {
		verification.MetadataOutcome = "invalid-json"
		return "", verification, fmt.Errorf("discover Home Assistant Core endpoint: decode Supervisor response: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		verification.MetadataOutcome = "invalid-json"
		return "", verification, fmt.Errorf("discover Home Assistant Core endpoint: Supervisor response contains trailing JSON")
	}
	if envelope.Result != "ok" {
		verification.MetadataOutcome = "unsuccessful"
		return "", verification, fmt.Errorf("discover Home Assistant Core endpoint: Supervisor result was not ok")
	}
	var port int
	if len(envelope.Data.Port) == 0 || json.Unmarshal(envelope.Data.Port, &port) != nil || port < 1 || port > 65535 {
		verification.MetadataOutcome = "invalid-port"
		return "", verification, fmt.Errorf("discover Home Assistant Core endpoint: Supervisor port is missing or invalid")
	}
	var ssl bool
	if len(envelope.Data.SSL) == 0 || json.Unmarshal(envelope.Data.SSL, &ssl) != nil {
		verification.MetadataOutcome = "invalid-ssl"
		return "", verification, fmt.Errorf("discover Home Assistant Core endpoint: Supervisor ssl is missing or invalid")
	}

	verification.MetadataOutcome = "ok"
	verification.DiscoveredPort = port
	verification.DiscoveredTLS = ssl
	scheme := "http"
	if ssl {
		scheme = "https"
	}
	resource := &url.URL{
		Scheme: scheme,
		Host:   "homeassistant:" + strconv.Itoa(port),
		Path:   "/" + strings.TrimPrefix(resourceURL, "/"),
	}
	return resource.String(), verification, nil
}

func dashboardJavaScriptMIMEType(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "text/javascript", "application/javascript", "text/ecmascript", "application/ecmascript", "application/x-javascript":
		return true
	default:
		return false
	}
}

func resourceType(resource haResourceMetadata) string {
	if value := strings.TrimSpace(resource.ResourceType); value != "" {
		return value
	}
	return strings.TrimSpace(resource.ResType)
}

func (c *haWSClient) ActivateManagedResource(ctx context.Context, resourceURL string) (*dashboardResourceChange, error) {
	resources, err := c.ListResources(ctx)
	if err != nil {
		return nil, err
	}
	change := &dashboardResourceChange{Before: managedDashboardResources(resources)}
	if len(change.Before) == 0 {
		if err := c.CreateResource(ctx, resourceURL, dashboardCardResourceType); err != nil {
			return nil, err
		}
		change.Action = "created"
	} else {
		primary := change.Before[0]
		if primary.URL == resourceURL && resourceType(primary) == dashboardCardResourceType {
			change.Action = "unchanged"
		} else {
			if err := c.UpdateResource(ctx, primary.ID, resourceURL, dashboardCardResourceType); err != nil {
				return nil, err
			}
			change.Action = "updated"
		}
	}

	resources, err = c.ListResources(ctx)
	if err != nil {
		return change, err
	}
	for _, resource := range resources {
		if resource.URL == resourceURL && resourceType(resource) == dashboardCardResourceType {
			change.Canonical = resource
			return change, nil
		}
	}
	return change, fmt.Errorf("managed dashboard resource did not re-read as %q type %q", resourceURL, dashboardCardResourceType)
}

func managedDashboardResources(resources []haResourceMetadata) []haResourceMetadata {
	managed := make([]haResourceMetadata, 0)
	for _, resource := range resources {
		if matchesManagedDashboardCardResource(resource.URL) {
			managed = append(managed, resource)
		}
	}
	return managed
}

func resourceIDKey(value any) string {
	return fmt.Sprint(value)
}

func (c *haWSClient) RemoveDuplicateManagedResources(ctx context.Context, canonicalID any) error {
	resources, err := c.ListResources(ctx)
	if err != nil {
		return err
	}
	canonicalKey := resourceIDKey(canonicalID)
	for _, resource := range managedDashboardResources(resources) {
		if resourceIDKey(resource.ID) == canonicalKey {
			continue
		}
		if err := c.DeleteResource(ctx, resource.ID); err != nil {
			return err
		}
	}
	resources, err = c.ListResources(ctx)
	if err != nil {
		return err
	}
	if got := len(managedDashboardResources(resources)); got != 1 {
		return fmt.Errorf("expected one managed dashboard resource after cleanup, got %d", got)
	}
	return nil
}

func (c *haWSClient) RestoreManagedResources(ctx context.Context, before []haResourceMetadata) error {
	current, err := c.ListResources(ctx)
	if err != nil {
		return err
	}
	currentManaged := managedDashboardResources(current)
	beforeByID := make(map[string]haResourceMetadata, len(before))
	currentByID := make(map[string]haResourceMetadata, len(currentManaged))
	for _, resource := range before {
		beforeByID[resourceIDKey(resource.ID)] = resource
	}
	for _, resource := range currentManaged {
		currentByID[resourceIDKey(resource.ID)] = resource
	}
	for key, resource := range currentByID {
		if _, ok := beforeByID[key]; ok {
			continue
		}
		if err := c.DeleteResource(ctx, resource.ID); err != nil {
			return err
		}
	}
	for key, resource := range beforeByID {
		if currentResource, ok := currentByID[key]; ok {
			if currentResource.URL != resource.URL || resourceType(currentResource) != resourceType(resource) {
				if err := c.UpdateResource(ctx, resource.ID, resource.URL, resourceType(resource)); err != nil {
					return err
				}
			}
			continue
		}
		if err := c.CreateResource(ctx, resource.URL, resourceType(resource)); err != nil {
			return err
		}
	}
	return nil
}

func (c *haWSClient) DeleteResource(_ context.Context, resourceID any) error {
	return c.call(map[string]any{"type": "lovelace/resources/delete", "resource_id": resourceID}, nil)
}

func (c *haWSClient) ReloadResourcesIfSupported(_ context.Context) (bool, error) {
	var services map[string]map[string]json.RawMessage
	if err := c.call(map[string]any{"type": "get_services"}, &services); err != nil {
		return false, err
	}
	if !dashboardReloadServiceSupported(services) {
		return false, nil
	}
	if err := c.call(map[string]any{
		"type":    "call_service",
		"domain":  "lovelace",
		"service": "reload_resources",
	}, nil); err != nil {
		return false, err
	}
	return true, nil
}

func dashboardReloadServiceSupported(services map[string]map[string]json.RawMessage) bool {
	_, supported := services["lovelace"]["reload_resources"]
	return supported
}

func cleanupDashboardAssetFiles(homeAssistantDir string, keepHashes ...string) error {
	keep := make(map[string]struct{}, len(keepHashes))
	for _, hash := range keepHashes {
		if len(hash) >= 12 {
			keep[strings.ToLower(hash[:12])] = struct{}{}
		}
	}
	var failures []string
	for _, root := range uniqueNonEmptyStrings([]string{homeAssistantDir, "/homeassistant", "/config"}) {
		dir := filepath.Join(root, "www", dashboardCardResourceDir)
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			match := managedDashboardBundlePattern.FindStringSubmatch(entry.Name())
			legacy := entry.Name() == dashboardCardSourceFile
			if match == nil && !legacy {
				continue
			}
			if match != nil {
				if _, retained := keep[match[1]]; retained {
					continue
				}
			}
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil && !os.IsNotExist(err) {
				failures = append(failures, err.Error())
			}
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("dashboard asset cleanup failed: %s", strings.Join(failures, "; "))
	}
	return nil
}

func removeDashboardAssetVersion(homeAssistantDir, hash string) error {
	if len(hash) < 12 {
		return nil
	}
	name := dashboardCardFilePrefix + strings.ToLower(hash[:12]) + ".js"
	var failures []string
	for _, root := range uniqueNonEmptyStrings([]string{homeAssistantDir, "/homeassistant", "/config"}) {
		path := filepath.Join(root, "www", dashboardCardResourceDir, name)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			failures = append(failures, err.Error())
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("remove failed dashboard asset version: %s", strings.Join(failures, "; "))
	}
	return nil
}

func nativeDashboardFallback(config map[string]any, locale dashboardLocaleBundle) (map[string]any, error) {
	copied, err := deepCopyJSONValue(config)
	if err != nil {
		return nil, err
	}
	converted := convertDashboardCardsToNative(copied, locale)
	ret, ok := converted.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("native dashboard fallback has unexpected type %T", converted)
	}
	return ret, nil
}

func convertDashboardCardsToNative(value any, locale dashboardLocaleBundle) any {
	switch typed := value.(type) {
	case []any:
		ret := make([]any, 0, len(typed))
		for _, entry := range typed {
			ret = append(ret, convertDashboardCardsToNative(entry, locale))
		}
		return ret
	case map[string]any:
		cardType := stringValue(typed["type"])
		switch cardType {
		case dashboardEnergyFlowCardType:
			return nativeDashboardEntitiesCard(typed, dashboardNativeFlowEntityOrder, localeText(locale, "heading_live_flow", "Live Flow"), locale)
		case "custom:gosungrow-energy-summary-card-v1":
			return nativeDashboardEntitiesCard(typed, dashboardNativeSummaryEntityOrder, localeText(locale, "heading_energy_summary", "Energy Summary"), locale)
		case dashboardSourceMappingCardType:
			return map[string]any{
				"type":    "markdown",
				"content": localeText(locale, "native_fallback_source_notice", "Enhanced dashboard cards are temporarily unavailable. Live data remains available, but source editing may be unavailable until automatic recovery completes."),
			}
		default:
			ret := make(map[string]any, len(typed))
			for key, entry := range typed {
				ret[key] = convertDashboardCardsToNative(entry, locale)
			}
			return ret
		}
	default:
		return value
	}
}

var dashboardNativeFlowEntityOrder = []string{
	"solar_power", "load_power", "grid_power", "battery_power", "pv_to_load_power",
	"pv_to_battery_power", "pv_to_grid_power", "grid_to_load_power", "battery_to_load_power", "battery_soc",
}

var dashboardNativeSummaryEntityOrder = []string{
	"production", "consumption", "to_grid", "from_grid", "to_battery", "from_battery",
}

var dashboardNativeEntityLabels = map[string]string{
	"solar_power": "source_metric_pv_power", "load_power": "source_metric_load_power", "grid_power": "source_metric_grid_power",
	"battery_power": "source_metric_battery_power", "pv_to_load_power": "source_metric_pv_to_load_power",
	"pv_to_battery_power": "source_metric_pv_to_battery_power", "pv_to_grid_power": "source_metric_pv_to_grid_power",
	"grid_to_load_power": "source_metric_grid_to_load_power", "battery_to_load_power": "source_metric_battery_to_load_power",
	"battery_soc": "source_metric_p13141", "production": "name_production", "consumption": "name_consumption",
	"to_grid": "name_to_grid", "from_grid": "name_from_grid", "to_battery": "name_to_battery", "from_battery": "name_from_battery",
}

func nativeDashboardEntitiesCard(card map[string]any, order []string, title string, locale dashboardLocaleBundle) map[string]any {
	ret := map[string]any{"type": "entities", "title": title}
	if layout, ok := card["layout_options"]; ok {
		ret["layout_options"] = layout
	}
	entities, _ := card["entities"].(map[string]any)
	rows := make([]any, 0, len(order))
	for _, key := range order {
		entity := strings.TrimSpace(stringValue(entities[key]))
		if entity == "" {
			continue
		}
		fallback := strings.ReplaceAll(key, "_", " ")
		rows = append(rows, map[string]any{"entity": entity, "name": localeText(locale, dashboardNativeEntityLabels[key], fallback)})
	}
	ret["entities"] = rows
	return ret
}

func dashboardConfigContainsCustomGoSungrow(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		if strings.HasPrefix(stringValue(typed["type"]), "custom:gosungrow-") {
			return true
		}
		for _, entry := range typed {
			if dashboardConfigContainsCustomGoSungrow(entry) {
				return true
			}
		}
	case []any:
		for _, entry := range typed {
			if dashboardConfigContainsCustomGoSungrow(entry) {
				return true
			}
		}
	}
	return false
}
