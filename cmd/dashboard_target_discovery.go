package cmd

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/roth-andreas/gosungrow-home-assistant/defaults"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/getDeviceList"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/getPsList"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/WebIscmAppService/getPsTreeMenu"
)

const (
	dashboardCoverageIncomplete   = "target_coverage_incomplete"
	dashboardContinuityUnverified = "target_continuity_unverified"
	dashboardIdentityConflict     = "target_identity_conflict"
)

// dashboardDiscovery keeps cloud observations separate from saved dashboard decisions.
type dashboardDiscovery struct {
	plants      []string
	trees       iSolarCloud.PsTrees
	inventories iSolarCloud.PsTrees
	denied      map[string]bool
	summary     sync.Once
}

func dashboardPlantNames(list getPsList.ResultData) map[string]string {
	names := make(map[string]string)
	for _, plant := range list.PageList {
		id := strings.TrimSpace(plant.PsId.String())
		if !plant.PsId.HasValue() || id == "" {
			continue
		}
		name := cleanDashboardLabel(plant.PsName.String())
		if previous, found := names[id]; found && previous != name {
			name = ""
		}
		names[id] = name
	}
	return names
}

func dashboardDiscoveryFromCloud(sg *iSolarCloud.SunGrow, args []string) (*dashboardDiscovery, error) {
	var names map[string]string
	plants := uniqueNonEmptyStrings(args)
	if len(args) == 0 {
		list, err := sg.GetPlantList()
		if err != nil {
			return nil, err
		}
		names = dashboardPlantNames(list)
		for id := range names {
			plants = append(plants, id)
		}
	}
	sort.Strings(plants)
	var attempt iSolarCloud.PlantTreeDiscovery
	trees, err := attempt.TreesForPlants(sg, plants)
	if err != nil && !attempt.HasDenials() {
		return nil, err
	}
	var inventory getDeviceList.Devices
	if err == nil {
		for _, id := range plants {
			if !attempt.Denied(id) {
				continue
			}
			var devices []getDeviceList.Device
			devices, err = sg.GetDeviceList(id)
			if err != nil {
				break
			}
			inventory = append(inventory, devices...)
		}
	}
	return collectDashboardDiscovery(plants, names, trees, attempt, inventory), err
}

func collectDashboardDiscovery(plants []string, names map[string]string, trees iSolarCloud.PsTrees, attempt iSolarCloud.PlantTreeDiscovery, rows getDeviceList.Devices) *dashboardDiscovery {
	discovery := &dashboardDiscovery{plants: plants, trees: trees, inventories: make(iSolarCloud.PsTrees), denied: make(map[string]bool)}
	for _, id := range plants {
		if attempt.Denied(id) {
			discovery.denied[id] = true
		}
	}
	if len(discovery.denied) == 0 {
		return discovery
	}
	byPlant := make(map[string]map[string]getPsTreeMenu.Ps)
	unresolved := make(map[string]bool)
	owners := make(map[string]string)
	for id, tree := range trees {
		for _, device := range tree.Devices {
			key := strings.TrimSpace(device.PsKey.String())
			if key != "" {
				owners[key] = id
			}
		}
	}
	for _, row := range rows {
		requestedID := row.RequestedPlantID
		if !discovery.denied[requestedID] {
			continue
		}
		key := strings.TrimSpace(row.PsKey.String())
		if key == "" {
			continue
		}
		if owner := owners[key]; owner != "" && owner != requestedID {
			unresolved[requestedID], unresolved[owner] = true, true
		}
		owners[key] = requestedID
		deviceType := row.DashboardDeviceType()
		if strings.TrimSpace(row.PsId.String()) != requestedID || deviceType == 0 {
			unresolved[requestedID] = true
			continue
		}
		if byPlant[requestedID] == nil {
			byPlant[requestedID] = make(map[string]getPsTreeMenu.Ps)
		}
		name := cleanDashboardLabel(row.DeviceName.String())
		if previous, found := byPlant[requestedID][key]; found {
			if previous.DeviceType.Value() != deviceType {
				unresolved[requestedID] = true
			}
			if name == "" || (previous.DeviceName.String() != "" && previous.DeviceName.String() <= name) {
				continue
			}
		}
		device := getPsTreeMenu.Ps{}
		device.PsId.SetString(requestedID)
		device.PsKey.SetValue(key)
		device.PsName.SetValue(names[requestedID])
		device.DeviceName.SetValue(name)
		device.DeviceType.SetValue(deviceType)
		byPlant[requestedID][key] = device
	}
	for id, devices := range byPlant {
		if unresolved[id] {
			continue
		}
		rows := make([]getPsTreeMenu.Ps, 0, len(devices))
		for _, device := range devices {
			rows = append(rows, device)
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].PsKey.String() < rows[j].PsKey.String() })
		discovery.inventories[id] = iSolarCloud.PsTree{Devices: rows}
	}
	return discovery
}

type dashboardDiscoverySelection struct {
	targets         []haDashboardTarget
	verifiedPlants  int
	inventoryPlants []string
	treePlants      int
	protected       bool
	keepViews       bool
	reason          string
}

func (d *dashboardDiscovery) selectTargets(state *haDashboardState) dashboardDiscoverySelection {
	var selection dashboardDiscoverySelection
	marked := make(map[string]bool)
	savedKeys := make(map[string]bool)
	if state != nil {
		for _, id := range state.InventoryTargetPlants {
			marked[id] = true
		}
		for _, key := range state.TargetPsKeys {
			savedKeys[key] = true
		}
	}
	selection.protected = len(d.denied) > 0
	selectedTrees := make(iSolarCloud.PsTrees)
	for _, id := range d.plants {
		inventory := d.denied[id]
		tree := d.trees[id]
		if inventory {
			tree = d.inventories[id]
		}
		selection.protected = selection.protected || marked[id]
		if !dashboardTreeHasKey(tree) {
			continue
		}
		if inventory || marked[id] {
			tree = preferSavedDashboardTarget(tree, savedKeys)
		}
		selectedTrees[id] = tree
		selection.verifiedPlants++
		if !inventory {
			selection.treePlants++
		}
		if inventory || marked[id] {
			selection.inventoryPlants = append(selection.inventoryPlants, id)
		}
	}
	selection.targets, _ = selectDashboardTargetsFromTrees(selectedTrees, d.denied)
	if selection.protected && selection.verifiedPlants != len(d.plants) {
		selection.reason = dashboardCoverageIncomplete
		return selection
	}
	if selection.protected && state != nil {
		sameTargets := len(selection.targets) == len(savedKeys)
		for _, target := range selection.targets {
			sameTargets = sameTargets && savedKeys[target.PsKey]
		}
		selection.keepViews = sameTargets
		if len(d.denied) > 0 && !sameTargets {
			selection.reason = dashboardContinuityUnverified
		}
	}
	return selection
}

func dashboardTreeHasKey(tree iSolarCloud.PsTree) bool {
	for _, device := range tree.Devices {
		if strings.TrimSpace(device.PsKey.String()) != "" {
			return true
		}
	}
	return false
}

func preferSavedDashboardTarget(tree iSolarCloud.PsTree, savedKeys map[string]bool) iSolarCloud.PsTree {
	best, saved := -1, -1
	for i, device := range tree.Devices {
		key := strings.TrimSpace(device.PsKey.String())
		if key == "" {
			continue
		}
		rank := preferredSungrowDeviceTypeRank(device.DeviceType.Value())
		if rank == 0 {
			return tree // All type-14 devices are selected in discovery order.
		}
		if best == -1 || rank < preferredSungrowDeviceTypeRank(tree.Devices[best].DeviceType.Value()) {
			best, saved = i, -1
		}
		if rank == preferredSungrowDeviceTypeRank(tree.Devices[best].DeviceType.Value()) && savedKeys[key] && saved == -1 {
			saved = i
		}
	}
	if saved <= best || best < 0 {
		return tree
	}
	devices := append([]getPsTreeMenu.Ps(nil), tree.Devices...)
	copy(devices[best+1:saved+1], devices[best:saved])
	devices[best] = tree.Devices[saved]
	return iSolarCloud.PsTree{Devices: devices}
}

func (d *dashboardDiscovery) logSummary(selection dashboardDiscoverySelection) {
	if len(d.denied) == 0 && selection.reason != dashboardIdentityConflict {
		return
	}
	d.summary.Do(func() {
		source := "none"
		switch {
		case selection.treePlants > 0 && selection.verifiedPlants > selection.treePlants:
			source = "mixed"
		case selection.treePlants > 0:
			source = "tree"
		case selection.verifiedPlants > 0:
			source = "device_list"
		}
		decision, reason := "complete", "none"
		if selection.reason != "" {
			decision, reason = "deferred", selection.reason
		}
		fmt.Printf("Dashboard target discovery: app_version=%s source=%s requested_plants=%d verified_plants=%d verified_targets=%d decision=%s reason=%s\n",
			defaults.BinaryVersion, source, len(d.plants), selection.verifiedPlants, len(selection.targets), decision, reason)
	})
}

func dashboardTargetIdentityConflict(targets []haDashboardTarget, mappingID func(haDashboardTarget) string) bool {
	type identity struct{ plant, key string }
	mappings := make(map[string]identity, len(targets))
	for i, target := range targets {
		key := strings.TrimSpace(target.PsKey)
		current := identity{strings.TrimSpace(target.PsID), key}
		id := mappingID(target)
		if previous, found := mappings[id]; found && previous != current {
			return true
		}
		mappings[id] = current
		for _, other := range targets[:i] {
			otherKey := strings.TrimSpace(other.PsKey)
			if otherKey != key && strings.EqualFold(key, otherKey) {
				return true
			}
		}
	}
	return false
}

func dashboardViewPathConflict(config map[string]any) bool {
	views, _ := config["views"].([]any)
	paths := make(map[string]bool, len(views))
	for _, raw := range views {
		view, _ := raw.(map[string]any)
		path := stringValue(view["path"])
		if paths[path] {
			return true
		}
		paths[path] = true
	}
	return false
}

func preserveDashboardViews(desired, current map[string]any, state *haDashboardState, targets []haDashboardTarget) bool {
	newViews, _ := desired["views"].([]any)
	oldViews, _ := current["views"].([]any)
	if len(targets) == 0 || len(newViews) == 0 || len(newViews) != len(oldViews) || len(newViews)%len(targets) != 0 {
		return false
	}
	// The saved structure verifies the order of targets and prototype views.
	// Source-only choices can legitimately differ from the last full hash.
	oldHash, _ := hashCanonicalJSON(current)
	verified := state.DashboardHash != "" && state.DashboardHash == oldHash
	if !verified && state.DashboardStructureHash != "" {
		structure, err := hashDashboardStructure(current)
		verified = err == nil && state.DashboardStructureHash == structure
	}
	if !verified {
		return false
	}
	oldIndex := make(map[string]int, len(state.TargetPsKeys))
	for i, key := range state.TargetPsKeys {
		if _, exists := oldIndex[key]; exists {
			return false
		}
		oldIndex[key] = i
	}
	if len(oldIndex) != len(targets) {
		return false
	}
	perTarget := len(newViews) / len(targets)
	savedViews := make([]map[string]any, len(newViews))
	for i, target := range targets {
		index, found := oldIndex[target.PsKey]
		if !found {
			return false
		}
		for j := 0; j < perTarget; j++ {
			view, ok := oldViews[index*perTarget+j].(map[string]any)
			if !ok || stringValue(view["path"]) == "" || stringValue(view["title"]) == "" {
				return false
			}
			savedViews[i*perTarget+j] = view
		}
	}
	for i, raw := range newViews {
		view := raw.(map[string]any) // Rendering already validated the view shape.
		view["path"], view["title"] = savedViews[i]["path"], savedViews[i]["title"]
	}
	return true
}
