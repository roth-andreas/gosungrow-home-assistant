package iSolarCloud

import (
	"time"

	"github.com/MickMake/GoSungrow/iSolarCloud/AppService/getPsList"
	"github.com/MickMake/GoSungrow/iSolarCloud/WebIscmAppService/getPsTreeMenu"
	"github.com/MickMake/GoSungrow/iSolarCloud/api/GoStruct/valueTypes"
	"github.com/MickMake/GoUnify/Only"
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

// PsTreeMenu - WebIscmAppService.getPsTreeMenu
func (sg *SunGrow) PsTreeMenu(psIDs ...string) (PsTrees, error) {
	ret := make(PsTrees)

	for range Only.Once {
		pids := sg.SetPsIds(psIDs...)
		if sg.Error != nil {
			break
		}

		for _, psID := range pids {
			ep := sg.GetByStruct(
				getPsTreeMenu.EndPointName,
				getPsTreeMenu.RequestData{PsId: psID},
				time.Hour,
			)
			if sg.IsError() {
				break
			}

			data := getPsTreeMenu.Assert(ep)
			ret[psID.String()] = PsTree{Devices: data.Response.ResultData.List}
		}
	}

	return ret, sg.Error
}
