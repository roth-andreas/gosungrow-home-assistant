package iSolarCloud

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/MickMake/GoUnify/Only"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/getPsList"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/WebIscmAppService/getPsTreeMenu"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct/valueTypes"
)

func (sg *SunGrow) SetPsIds(args ...string) valueTypes.PsIds {
	var pids valueTypes.PsIds
	for range Only.Once {
		if len(args) > 0 {
			pids = valueTypes.SetPsIdStrings(args)
			if len(pids) > 0 {
				break
			}
		}

		pids, sg.Error = sg.GetPsIds()
		if sg.Error != nil {
			break
		}
	}
	return pids
}

func (sg *SunGrow) GetPsIds() (valueTypes.PsIds, error) {
	data, err := sg.GetPlantList()
	if err != nil {
		return nil, err
	}
	return data.GetPsIds(), nil
}

// GetPlantList returns the cached or current plant list for the active session.
func (sg *SunGrow) GetPlantList() (getPsList.ResultData, error) {
	ep := sg.GetByStruct(getPsList.EndPointName, nil, DefaultCacheTimeout)
	if sg.IsError() {
		return getPsList.ResultData{}, sg.Error
	}
	return getPsList.AssertResultData(ep), nil
}

func (sg *SunGrow) GetPsKeys() (valueTypes.PsKeys, error) {
	var ret valueTypes.PsKeys

	for range Only.Once {
		trees, err := sg.PsTreeMenu()
		if err != nil {
			sg.Error = err
			break
		}

		keys := make([]string, 0)
		for _, tree := range trees {
			for _, device := range tree.Devices {
				if device.PsKey.String() == "" {
					continue
				}
				keys = append(keys, device.PsKey.String())
			}
		}
		if len(keys) > 0 {
			ret.Set(keys...)
			break
		}

		devices, err := sg.GetDeviceList()
		if err != nil {
			sg.Error = err
			break
		}
		for _, device := range devices {
			if device.PsKey.String() == "" {
				continue
			}
			keys = append(keys, device.PsKey.String())
		}
		ret.Set(keys...)
	}

	return ret, sg.Error
}

type PsTree struct {
	Devices []getPsTreeMenu.Ps
}

type PsTrees map[string]PsTree

// PlantTreeDiscovery shares plant-specific denials across one MQTT discovery attempt.
// Its zero value starts a new attempt.
type PlantTreeDiscovery struct {
	denied map[string]bool
	plants getPsList.ResultData
}

// Plants returns the plant list used to scope the tree requests.
func (d PlantTreeDiscovery) Plants() getPsList.ResultData { return d.plants }

// Denied reports whether the tree request for psID returned HTTP 403 in this attempt.
func (d PlantTreeDiscovery) Denied(psID string) bool {
	return d.denied[psID]
}

// HasDenials reports whether this attempt has any unavailable plant trees.
func (d PlantTreeDiscovery) HasDenials() bool {
	return len(d.denied) > 0
}

// Trees loads plant trees, skipping plants whose optional discovery was denied.
func (d *PlantTreeDiscovery) Trees(sg *SunGrow) (PsTrees, error) {
	return sg.psTreeMenu(d)
}

// TreesForPlants shares denials while querying only the supplied explicit plants.
func (d *PlantTreeDiscovery) TreesForPlants(sg *SunGrow, psIDs []string) (PsTrees, error) {
	return sg.loadPlantTrees(d, valueTypes.SetPsIdStrings(psIDs))
}

// PsTreeMenu - WebIscmAppService.getPsTreeMenu
func (sg *SunGrow) PsTreeMenu(psIDs ...string) (PsTrees, error) {
	return sg.psTreeMenu(nil, psIDs...)
}

func (sg *SunGrow) psTreeMenu(discovery *PlantTreeDiscovery, psIDs ...string) (PsTrees, error) {
	var pids valueTypes.PsIds
	if discovery != nil && len(psIDs) == 0 {
		plants, err := sg.GetPlantList()
		if err != nil {
			return PsTrees{}, err
		}
		discovery.plants = plants
		pids = plants.GetPsIds()
	} else {
		pids = sg.SetPsIds(psIDs...)
	}
	return sg.loadPlantTrees(discovery, pids)
}

func (sg *SunGrow) loadPlantTrees(discovery *PlantTreeDiscovery, pids valueTypes.PsIds) (PsTrees, error) {
	ret := make(PsTrees)
	if sg.Error != nil {
		return ret, sg.Error
	}
	for _, psID := range pids {
		key := strings.TrimSpace(psID.String())
		if discovery != nil && discovery.Denied(key) {
			continue
		}
		ep := sg.GetByStruct(getPsTreeMenu.EndPointName, getPsTreeMenu.RequestData{PsId: plantRequestID(psID)}, time.Hour)
		if sg.IsError() {
			if discovery != nil && errors.Is(sg.Error, api.ErrHTTPForbidden) {
				if discovery.denied == nil {
					discovery.denied = make(map[string]bool)
				}
				discovery.denied[key] = true
				sg.Error = nil
				continue
			}
			return ret, sg.Error
		}
		ret[key] = PsTree{Devices: getPsTreeMenu.Assert(ep).Response.ResultData.List}
	}
	return ret, nil
}

func plantRequestID(id valueTypes.PsId) valueTypes.PsId {
	// Keep the established wire representation for canonical numeric IDs.
	// A noncanonical spelling must stay a string to preserve the identity.
	if id.Numeric && id.String() != strconv.FormatInt(id.Value(), 10) {
		id.Numeric = false
	}
	return id
}
