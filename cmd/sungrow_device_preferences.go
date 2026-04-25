package cmd

func preferredSungrowDeviceTypeRank(deviceType int64) int {
	switch deviceType {
	case 14:
		return 0
	case 11:
		return 1
	case 7:
		return 2
	case 1:
		return 3
	default:
		return 100
	}
}

func dashboardSelectionSourceForDeviceType(deviceType int64) string {
	switch deviceType {
	case 14:
		return "preferred-device-type-14"
	case 11:
		return "preferred-device-type-11"
	case 7:
		return "preferred-device-type-7"
	case 1:
		return "preferred-device-type-1"
	default:
		return "fallback-first-valid-ps-key"
	}
}

func realtimeSelectionSourceForDeviceType(deviceType int64) string {
	switch deviceType {
	case 14:
		return "device-type-14"
	case 11:
		return "device-type-11"
	case 7:
		return "device-type-7"
	case 1:
		return "device-type-1"
	default:
		return "first-valid-ps-key-fallback"
	}
}
