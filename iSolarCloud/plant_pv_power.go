package iSolarCloud

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct/valueTypes"
)

type PlantTopologyDevice struct {
	PsID       string
	PsKey      string
	UUID       int64
	UpUUID     int64
	DeviceType int64
}

type PlantTopology struct {
	PsID     string
	Devices  map[string]PlantTopologyDevice
	Complete bool
	Reason   string
}

type PlantPVPowerResult struct {
	Added        bool
	Source       string
	Contributors int
	Reason       string
}

type plantPowerCandidate struct {
	entry *api.DataEntry
	value float64
	valid bool
	rank  int
}

type producerCandidates struct {
	device PlantTopologyDevice
	ac     *plantPowerCandidate
	dc     *plantPowerCandidate
}

var plantNativePVPointOrder = map[string]int{
	"p83076": 0, "p83076_map": 1, "p83033": 2, "p83002": 3,
	"plant_power": 4, "pv_power": 5, "solar_power": 6,
}

var plantACPointOrder = map[string]int{
	"p24": 0, "inverter_ac_power": 1, "total_active_power": 2, "active_power": 3,
}

var plantDCPointOrder = map[string]int{
	"total_dc_power": 0, "dc_power": 1,
}

func BuildPlantTopologies(trees PsTrees) map[string]PlantTopology {
	topologies := make(map[string]PlantTopology, len(trees))
	uuidOwners := make(map[int64]string)
	duplicateUUID := make(map[int64]bool)

	plantIDs := make([]string, 0, len(trees))
	for psID := range trees {
		plantIDs = append(plantIDs, strings.TrimSpace(psID))
	}
	sort.Strings(plantIDs)

	for _, psID := range plantIDs {
		topology := PlantTopology{PsID: psID, Devices: make(map[string]PlantTopologyDevice), Complete: true}
		if len(trees[psID].Devices) == 0 {
			topology.Complete = false
			topology.Reason = "topology contains no devices"
		}
		for _, raw := range trees[psID].Devices {
			device := PlantTopologyDevice{
				PsID: strings.TrimSpace(raw.PsId.String()), PsKey: strings.TrimSpace(raw.PsKey.String()),
				UUID: raw.UUID.Value(), UpUUID: raw.UpUUID.Value(), DeviceType: raw.DeviceType.Value(),
			}
			if device.PsID == "" {
				device.PsID = psID
			}
			if device.PsID != psID || device.PsKey == "" {
				topology.Complete = false
				topology.Reason = "topology contains a missing key or cross-plant device"
				continue
			}
			topology.Devices[device.PsKey] = device
			if device.UUID == 0 {
				continue
			}
			if owner, exists := uuidOwners[device.UUID]; exists {
				duplicateUUID[device.UUID] = true
				if owner == psID {
					topology.Complete = false
					topology.Reason = "topology contains duplicate UUIDs"
				}
			} else {
				uuidOwners[device.UUID] = psID
			}
		}
		topologies[psID] = topology
	}

	for _, psID := range plantIDs {
		topology := topologies[psID]
		known := make(map[int64]bool, len(topology.Devices))
		for _, device := range topology.Devices {
			if device.UUID != 0 {
				known[device.UUID] = true
			}
		}
		for _, device := range topology.Devices {
			if device.UpUUID == 0 {
				continue
			}
			owner, exists := uuidOwners[device.UpUUID]
			switch {
			case duplicateUUID[device.UpUUID]:
				topology.Complete = false
				topology.Reason = "topology parent UUID is ambiguous"
			case exists && owner != psID:
				topology.Complete = false
				topology.Reason = "topology contains a cross-plant parent"
			case !known[device.UpUUID]:
				topology.Complete = false
				topology.Reason = "topology contains a dangling parent"
			}
		}
		topologies[psID] = topology
	}

	return topologies
}

func AddCanonicalPlantPVPower(data *api.DataMap, topology PlantTopology) PlantPVPowerResult {
	psID := strings.TrimSpace(topology.PsID)
	if data == nil || psID == "" {
		return PlantPVPowerResult{Reason: "plant ID is unavailable"}
	}

	entries := sortedLatestEntries(data)
	if native := selectNativePlantPVPower(entries, topology); native != nil {
		addPlantPVPowerEntry(data, psID, native.entry, native.value)
		return PlantPVPowerResult{Added: true, Source: "native_plant", Contributors: 1}
	}
	if !topology.Complete {
		return PlantPVPowerResult{Reason: topology.Reason}
	}

	producers := collectProducerCandidates(entries, topology)
	if len(producers) == 0 {
		return PlantPVPowerResult{Reason: "no producer candidates"}
	}
	leaves := producerLeaves(producers)
	if len(leaves) == 0 {
		return PlantPVPowerResult{Reason: "no producer leaves"}
	}

	selected, source, ok := completeProducerTier(leaves, true)
	if !ok {
		selected, source, ok = completeProducerTier(leaves, false)
	}
	if !ok {
		return PlantPVPowerResult{Reason: "no complete AC or DC contributor tier"}
	}

	total := 0.0
	sourceEntry := selected[0].entry
	for _, candidate := range selected {
		total += candidate.value
		if candidate.entry.Date.Time.Before(sourceEntry.Date.Time) {
			sourceEntry = candidate.entry
		}
	}
	total = valueTypes.SetPrecision(total, 3)
	addPlantPVPowerEntry(data, psID, sourceEntry, total)
	return PlantPVPowerResult{Added: true, Source: source, Contributors: len(selected)}
}

func sortedLatestEntries(data *api.DataMap) []*api.DataEntry {
	keys := data.Sort()
	entries := make([]*api.DataEntry, 0, len(keys))
	for _, key := range keys {
		entry := data.Map[key].GetEntry(api.LastEntry)
		if entry != nil && entry.Point != nil {
			entries = append(entries, entry)
		}
	}
	return entries
}

func selectNativePlantPVPower(entries []*api.DataEntry, topology PlantTopology) *plantPowerCandidate {
	var best *plantPowerCandidate
	for _, entry := range entries {
		if strings.HasPrefix(strings.ToLower(entry.EndPoint), "virtual.") || !plantScopedEntry(entry, topology) {
			continue
		}
		rank, ok := plantNativePVPointOrder[strings.ToLower(strings.TrimSpace(entry.Point.Id))]
		if !ok {
			continue
		}
		value, valid := plantPowerKilowatts(entry)
		candidate := &plantPowerCandidate{entry: entry, value: value, valid: valid, rank: rank}
		if valid && (best == nil || rank < best.rank || rank == best.rank && entry.EndPoint < best.entry.EndPoint) {
			best = candidate
		}
	}
	return best
}

func plantScopedEntry(entry *api.DataEntry, topology PlantTopology) bool {
	key := strings.TrimSpace(entry.Parent.Key)
	if key == topology.PsID {
		return true
	}
	parent := entry.Parent
	parent.Split()
	if parent.PsId == topology.PsID && parent.Type == "11" {
		return true
	}
	device, ok := topology.Devices[key]
	return ok && device.DeviceType == 11
}

func collectProducerCandidates(entries []*api.DataEntry, topology PlantTopology) map[string]*producerCandidates {
	producers := make(map[string]*producerCandidates)
	for _, entry := range entries {
		device, ok := topology.Devices[strings.TrimSpace(entry.Parent.Key)]
		if !ok || device.DeviceType == 11 {
			continue
		}
		pointID := strings.ToLower(strings.TrimSpace(entry.Point.Id))
		acRank, ac := plantACPointOrder[pointID]
		dcRank, dc := plantDCPointOrder[pointID]
		if !ac && !dc {
			continue
		}
		if ac && !hasInverterContext(entry, device) {
			continue
		}
		producer := producers[device.PsKey]
		if producer == nil {
			producer = &producerCandidates{device: device}
			producers[device.PsKey] = producer
		}
		value, valid := plantPowerKilowatts(entry)
		if ac {
			producer.ac = preferredPlantCandidate(producer.ac, &plantPowerCandidate{entry: entry, value: value, valid: valid, rank: acRank})
		}
		if dc {
			producer.dc = preferredPlantCandidate(producer.dc, &plantPowerCandidate{entry: entry, value: value, valid: valid, rank: dcRank})
		}
	}
	return producers
}

func hasInverterContext(entry *api.DataEntry, device PlantTopologyDevice) bool {
	if device.DeviceType == 1 {
		return true
	}
	identity := strings.ToLower(strings.Join([]string{device.PsKey, entry.EndPoint, entry.Point.Description, entry.Point.GroupName}, " "))
	return strings.Contains(identity, "inverter")
}

func preferredPlantCandidate(current, candidate *plantPowerCandidate) *plantPowerCandidate {
	if current == nil || candidate.rank < current.rank || candidate.rank == current.rank && candidate.valid && !current.valid ||
		candidate.rank == current.rank && candidate.valid == current.valid && candidate.entry.EndPoint < current.entry.EndPoint {
		return candidate
	}
	return current
}

func producerLeaves(producers map[string]*producerCandidates) []*producerCandidates {
	parents := make(map[int64]bool)
	for _, producer := range producers {
		if producer.device.UpUUID != 0 {
			parents[producer.device.UpUUID] = true
		}
	}
	keys := make([]string, 0, len(producers))
	for key := range producers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	leaves := make([]*producerCandidates, 0, len(keys))
	for _, key := range keys {
		producer := producers[key]
		if producer.device.UUID != 0 && parents[producer.device.UUID] {
			continue
		}
		leaves = append(leaves, producer)
	}
	return leaves
}

func completeProducerTier(leaves []*producerCandidates, ac bool) ([]*plantPowerCandidate, string, bool) {
	selected := make([]*plantPowerCandidate, 0, len(leaves))
	for _, producer := range leaves {
		candidate := producer.dc
		if ac {
			candidate = producer.ac
		}
		if candidate == nil || !candidate.valid {
			return nil, "", false
		}
		selected = append(selected, candidate)
	}
	if ac {
		return selected, "summed_device_ac", true
	}
	return selected, "summed_device_dc", true
}

func plantPowerKilowatts(entry *api.DataEntry) (float64, bool) {
	if entry == nil || entry.Point == nil || !entry.Valid || !entry.Point.Valid || !entry.Value.Valid || !entry.Value.IsNumber() {
		return 0, false
	}
	value := entry.Value.ValueFloat()
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}
	switch strings.TrimSpace(entry.Value.Unit()) {
	case "W":
		value /= 1000
	case "kW":
	case "MW":
		value *= 1000
	default:
		return 0, false
	}
	return value, true
}

func addPlantPVPowerEntry(data *api.DataMap, psID string, source *api.DataEntry, value float64) {
	if source == nil || source.Current == nil {
		return
	}
	current := *source.Current
	current.DataStructure.Endpoint = GoStruct.NewEndPointPath("virtual", psID, "pv_power")
	current.DataStructure.PointId = "pv_power"
	current.DataStructure.PointName = "Plant PV Power"
	current.DataStructure.PointGroupName = "Plant"
	current.DataStructure.PointDevice = psID
	current.DataStructure.PointUnit = "kW"
	current.DataStructure.PointUpdateFreq = GoStruct.UpdateFreq5Mins
	current.DataStructure.PointValueType = "Power"
	current.DataStructure.ValueType = "Power"
	current.IsOk = true

	unitValue := valueTypes.SetUnitValueFloat("kW", "Power", valueTypes.SetPrecision(value, 3))
	unitValue.SetDeviceId(psID)
	current.SetUnitValue(unitValue)
	point := api.CreatePoint(&current, psID)
	point.Id = "pv_power"
	point.Description = "Plant PV Power"
	point.GroupName = "Plant"
	point.Unit = "kW"
	point.UpdateFreq = GoStruct.UpdateFreq5Mins
	point.ValueType = "Power"
	point.Valid = true

	endpoint := fmt.Sprintf("virtual.%s.pv_power", psID)
	data.Add(api.DataEntry{
		Current: &current, EndPoint: endpoint, Point: &point, Parent: api.NewParentDevice(psID),
		Date: source.Date, Value: unitValue, Valid: true,
	})
}
