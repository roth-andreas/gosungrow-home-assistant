package iSolarCloud

import (
	"time"

	"github.com/MickMake/GoUnify/Only"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/WebAppService/getDevicePointAttrs"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct/valueTypes"
)

// DevicePointAttrs - Return all points associated with psIds and device_type filter.
func (sg *SunGrow) DevicePointAttrs(deviceType string, psIDs ...string) (getDevicePointAttrs.Points, error) {
	var points getDevicePointAttrs.Points

	for range Only.Once {
		pids := sg.SetPsIds(psIDs...)
		if sg.Error != nil {
			break
		}

		for _, pid := range pids {
			p, err := sg.GetDevicePointAttrs(pid)
			if err != nil {
				sg.Error = err
				break
			}

			for _, point := range p {
				if deviceType == "" || point.DeviceType.MatchString(deviceType) {
					points = append(points, point)
				}
			}
		}
	}

	return points, sg.Error
}

// DevicePointAttrsMap - Return all points associated with psIds and device_type filter.
func (sg *SunGrow) DevicePointAttrsMap(deviceType string, psIDs ...string) (getDevicePointAttrs.PointsMap, error) {
	points := make(getDevicePointAttrs.PointsMap)

	for range Only.Once {
		pa, err := sg.DevicePointAttrs(deviceType, psIDs...)
		if err != nil {
			sg.Error = err
			break
		}

		for i := range pa {
			points[pa[i].Id.String()] = &pa[i]
		}
	}

	return points, sg.Error
}

// GetDevicePointAttrs - WebAppService.getDevicePointAttrs Uuid: PsId: DeviceType
func (sg *SunGrow) GetDevicePointAttrs(psID valueTypes.PsId) (getDevicePointAttrs.Points, error) {
	var ret getDevicePointAttrs.Points

	for range Only.Once {
		trees, err := sg.PsTreeMenu(psID.String())
		if err != nil {
			sg.Error = err
			break
		}

		for _, tree := range trees {
			for _, device := range tree.Devices {
				ep := sg.GetByStruct(
					getDevicePointAttrs.EndPointName,
					getDevicePointAttrs.RequestData{
						Uuid:        device.UUID,
						PsId2:       device.PsId,
						DeviceType2: device.DeviceType,
					},
					time.Hour*24,
				)
				if sg.IsError() {
					break
				}

				data := getDevicePointAttrs.Assert(ep)
				ret = append(ret, data.Points()...)
			}
		}
	}

	return ret, sg.Error
}
