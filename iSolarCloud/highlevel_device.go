package iSolarCloud

import (
	"time"

	"github.com/MickMake/GoUnify/Only"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/getDeviceList"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct/valueTypes"
)

// GetDeviceList - AppService.getDeviceList
func (sg *SunGrow) GetDeviceList(psIds ...string) ([]getDeviceList.Device, error) {
	var ret []getDeviceList.Device

	for range Only.Once {
		pids := sg.SetPsIds(psIds...)
		if sg.Error != nil {
			break
		}

		for _, psID := range pids {
			ep := sg.GetByStruct(
				getDeviceList.EndPointName,
				getDeviceList.RequestData{PsId: plantRequestID(psID)},
				time.Hour*24,
			)
			if sg.IsError() {
				break
			}

			data := getDeviceList.Assert(ep)
			for _, device := range data.Response.ResultData.PageList {
				device.RequestedPlantID = psID.String()
				ret = append(ret, device)
			}
		}
	}

	return ret, sg.Error
}

func uniquePsKeys(values ...string) valueTypes.PsKeys {
	var keys valueTypes.PsKeys
	keys.Set(values...)
	return keys
}
