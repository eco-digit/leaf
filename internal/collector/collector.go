// Package collector queries Prometheus for raw energy metrics
// and returns structured RawMetrics for the Model Calculator.
package collector

import (
	"time"

	prommodel "github.com/prometheus/common/model"
)

// Querier executes a PromQL query and returns results.
type Querier interface {
	QueryMetric(query string) (prommodel.Value, error)
}

type DeviceRaw struct {
	Metrics   map[string]float64
	VMMetrics map[string]map[string]float64
}

type RackRaw struct {
	Metrics map[string]float64 // source_name → value
}

// RawMetrics is the full set of raw  metric values from one collection run.
type RawMetrics struct {
	Timestamp time.Time
	Devices   map[string]*DeviceRaw
	Racks     map[string]*RackRaw
	Warnings  []string
}
