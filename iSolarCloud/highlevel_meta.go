package iSolarCloud

import (
	"errors"
	"github.com/MickMake/GoSungrow/iSolarCloud/AppService/queryDeviceRealTimeDataByPsKeys"
	"github.com/MickMake/GoSungrow/iSolarCloud/AppService/queryUnitList"
	"github.com/MickMake/GoSungrow/iSolarCloud/WebAppService/getMqttConfigInfoByAppkey"
	"github.com/MickMake/GoSungrow/iSolarCloud/api/GoStruct/valueTypes"
	"github.com/MickMake/GoUnify/Only"
)

func (sg *SunGrow) MetaUnitList() error {
	for range Only.Once {
		data := sg.NewSunGrowData()
		data.SetArgs()
		data.SetEndpoints(queryUnitList.EndPointName)

		sg.Error = data.GetData()
		if sg.Error != nil {
			break
		}

		sg.Error = data.OutputDataTables()
		if sg.Error != nil {
			break
		}
	}

	return sg.Error
}

func (sg *SunGrow) GetIsolarcloudMqtt(appKey string) error {
	for range Only.Once {
		if appKey == "" {
			appKey = sg.GetAppKey()
		}

		data := sg.NewSunGrowData()
		data.SetArgs("AppKey:" + appKey)
		data.SetEndpoints(getMqttConfigInfoByAppkey.EndPointName)

		sg.Error = data.GetData()
		if sg.Error != nil {
			break
		}

		sg.Error = data.Output()
		if sg.Error != nil {
			break
		}

		// ep := sg.GetByStruct(getMqttConfigInfoByAppkey.EndPointName,
		// 	getMqttConfigInfoByAppkey.RequestData{AppKey: valueTypes.SetStringValue(appKey)},
		// 	DefaultCacheTimeout,
		// )
		// if sg.IsError() {
		// 	break
		// }
		//
		// data := getMqttConfigInfoByAppkey.Assert(ep)
		// table := data.GetEndPointResultTable()
		// if table.Error != nil {
		// 	sg.Error = table.Error
		// 	break
		// }
		//
		// table.SetTitle("MQTT info")
		// table.SetFilePrefix(data.SetFilenamePrefix(""))
		// table.SetGraphFilter("")
		// table.SetSaveFile(sg.SaveAsFile)
		// table.OutputType = sg.OutputType
		// sg.Error = table.Output()
		// if sg.IsError() {
		// 	break
		// }
	}

	return sg.Error
}

func (sg *SunGrow) GetRealTimeData(psKey string) error {
	for range Only.Once {
		if psKey == "" {
			var psKeys valueTypes.PsKeys
			psKeys, sg.Error = sg.GetPsKeys()
			if sg.IsError() {
				break
			}
			if psKeys.Length() == 0 {
				sg.Error = errors.New("could not auto-detect ps_key")
				break
			}

			// Prefer inverter keys for real-time power values.
			for _, key := range psKeys.PsKeys {
				if key.DeviceType == "14" {
					psKey = key.String()
					break
				}
			}
			if psKey == "" {
				psKey = psKeys.PsKeys[0].String()
			}
		}

		data := sg.NewSunGrowData()
		data.SetArgs("PsKeyList:" + psKey)
		data.SetEndpoints(queryDeviceRealTimeDataByPsKeys.EndPointName)
		sg.Error = data.GetData()
		if sg.Error != nil {
			break
		}

		sg.Error = data.OutputDataTables()
		if sg.IsError() {
			break
		}
	}

	return sg.Error
}
