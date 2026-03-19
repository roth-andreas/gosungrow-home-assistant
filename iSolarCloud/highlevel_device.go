package iSolarCloud

import (
	"time"

	"github.com/MickMake/GoSungrow/iSolarCloud/AppService/getDeviceList"
	"github.com/MickMake/GoSungrow/iSolarCloud/api/GoStruct/valueTypes"
	"github.com/MickMake/GoUnify/Only"
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
				getDeviceList.RequestData{PsId: psID},
				time.Hour*24,
			)
			if sg.IsError() {
				break
			}

			data := getDeviceList.Assert(ep)
			ret = append(ret, data.Response.ResultData.PageList...)
		}
	}

	return ret, sg.Error
}

func uniquePsKeys(values ...string) valueTypes.PsKeys {
	var keys valueTypes.PsKeys
	keys.Set(values...)
	return keys
}
