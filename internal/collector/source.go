package collector

// DeviceEnergySource provides raw energy readings per device and per VM
type DeviceEnergySource interface {
	MetricValue(deviceID, sourceName string) (float64, bool)
	VMMetricValues(deviceID, sourceName string) map[string]float64
}

// VMMetadataSource provides VM-to-tenant mapping and resource sizing
type VMMetadataSource interface {
	VMInfos() []VMInfo
}

// RawSource is the combined raw collection result needed by the tenant calculator.
// *RawMetrics satisfies this interface without any changes.
type RawSource interface {
	DeviceEnergySource
	VMMetadataSource
}

// RackMetricSource provides raw metric values collected at rack level
type RackMetricSource interface {
	RackMetricValue(rackID, sourceName string) (float64, bool)
}
