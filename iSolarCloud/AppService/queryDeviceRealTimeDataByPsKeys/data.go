package queryDeviceRealTimeDataByPsKeys

import (
	"github.com/MickMake/GoSungrow/iSolarCloud/AppService/queryDeviceList"
	"github.com/MickMake/GoSungrow/iSolarCloud/api"
	"github.com/MickMake/GoSungrow/iSolarCloud/api/GoStruct"
	"github.com/MickMake/GoSungrow/iSolarCloud/api/GoStruct/valueTypes"

	"fmt"
)

// The legacy realtime endpoint currently rejects parameters for many accounts.
// Route through queryDeviceList using ps_id derived from ps_key.
const Url = "/v1/devService/queryDeviceList"
const Disabled = false
const EndPointName = "AppService.queryDeviceRealTimeDataByPsKeys"

type RequestData struct {
	PsKeyList valueTypes.String `json:"ps_key_list" required:"true"`
}

func (rd *RequestData) IsValid() error {
	err := GoStruct.VerifyOptionsRequired(*rd)
	if err != nil {
		return err
	}

	psKey := valueTypes.SetPsKeyString(rd.PsKeyList.String())
	if psKey.Error != nil {
		return psKey.Error
	}
	if psKey.PsId == "" {
		return fmt.Errorf("invalid PsKeyList value")
	}
	return nil
}

func (rd RequestData) Help() string {
	ret := fmt.Sprintf("")
	return ret
}

type ResultData = queryDeviceList.ResultData

func (e *EndPoint) GetData() api.DataMap {
	// Response structure is equivalent to queryDeviceList; reuse its data mapping logic.
	list := queryDeviceList.EndPoint{}
	list.Response.ResultData = e.Response.ResultData

	// Keep only the requested device when a ps_key is supplied.
	psKey := e.Request.PsKeyList.String()
	if psKey != "" {
		filtered := make([]queryDeviceList.Device, 0, len(list.Response.ResultData.PageList))
		for _, device := range list.Response.ResultData.PageList {
			if device.PsKey.String() == psKey {
				filtered = append(filtered, device)
			}
		}
		if len(filtered) > 0 {
			list.Response.ResultData.PageList = filtered
			list.Response.ResultData.RowCount = valueTypes.SetIntegerValue(int64(len(filtered)))
		}
	}

	if len(list.Response.ResultData.PageList) > 0 {
		list.Request.PsId = list.Response.ResultData.PageList[0].PsId
	}
	ret := list.GetData()
	ret.EndPoint = *e
	return ret
}
