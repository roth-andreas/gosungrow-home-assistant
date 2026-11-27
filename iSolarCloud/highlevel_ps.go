package iSolarCloud

import (
	"errors"
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
	var ret valueTypes.PsIds

	for range Only.Once {
		ep := sg.GetByStruct(getPsList.EndPointName, nil, DefaultCacheTimeout)
		if sg.IsError() {
			break
		}

		data := getPsList.AssertResultData(ep)
		ret = data.GetPsIds()
	}

	return ret, sg.Error
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
}

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

// PsTreeMenu - WebIscmAppService.getPsTreeMenu
func (sg *SunGrow) PsTreeMenu(psIDs ...string) (PsTrees, error) {
	return sg.psTreeMenu(nil, psIDs...)
}

func (sg *SunGrow) psTreeMenu(discovery *PlantTreeDiscovery, psIDs ...string) (PsTrees, error) {
	ret := make(PsTrees)
	pids := sg.SetPsIds(psIDs...)
	if sg.Error != nil {
		return ret, sg.Error
	}

	for _, psID := range pids {
		key := psID.String()
		if discovery != nil && discovery.Denied(key) {
			continue
		}
		ep := sg.GetByStruct(getPsTreeMenu.EndPointName, getPsTreeMenu.RequestData{PsId: psID}, time.Hour)
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
