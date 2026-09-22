package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	dashboardAssetModeEnhanced = "enhanced"
	dashboardAssetModeFallback = "native-fallback"
)

var managedDashboardBundlePattern = regexp.MustCompile(`^gosungrow-dashboard-cards\.([0-9a-f]{12})\.js$`)

type dashboardAssetVerification struct {
	StatusCode int
	MIMEType   string
}

type dashboardResourceChange struct {
	Action    string
	Canonical haResourceMetadata
	Before    []haResourceMetadata
}

func dashboardHTTPResourceURL(websocketURL, resourceURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(websocketURL))
	if err != nil {
		return "", err
	}
	switch parsed.Scheme {
	case "ws":
		parsed.Scheme = "http"
	case "wss":
		parsed.Scheme = "https"
	default:
		return "", fmt.Errorf("unsupported Home Assistant websocket scheme %q", parsed.Scheme)
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	path := strings.TrimSuffix(parsed.Path, "/")
	if strings.HasSuffix(path, "/api/websocket") {
		path = strings.TrimSuffix(path, "/api/websocket")
	} else {
		path = strings.TrimSuffix(path, "/websocket")
	}
	parsed.Path = strings.TrimSuffix(path, "/") + "/" + strings.TrimPrefix(resourceURL, "/")
	return parsed.String(), nil
}

func verifyDashboardCardAsset(ctx context.Context, websocketURL, token, resourceURL, expectedHash string) (dashboardAssetVerification, error) {
	httpURL, err := dashboardHTTPResourceURL(websocketURL, resourceURL)
	if err != nil {
		return dashboardAssetVerification{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, httpURL, nil)
	if err != nil {
		return dashboardAssetVerification{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	response, err := client.Do(req)
	if err != nil {
		return dashboardAssetVerification{}, fmt.Errorf("fetch staged dashboard asset: %w", err)
	}
	defer response.Body.Close()

	verification := dashboardAssetVerification{StatusCode: response.StatusCode}
	mediaType, _, parseErr := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if parseErr == nil {
		verification.MIMEType = strings.ToLower(mediaType)
	}
	if response.StatusCode != http.StatusOK {
		return verification, fmt.Errorf("staged dashboard asset returned HTTP %d", response.StatusCode)
	}
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
