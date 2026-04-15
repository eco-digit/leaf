// Package collector queries Prometheus for raw energy metrics
// and returns structured RawMetrics for the Model Calculator.
package collector

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/OSBA-eco-digit/leaf/internal/infrastructure"
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

// Collect queries Prometheus for every metric source defined in infra.MetricSources.
func Collect(q Querier, infra *infrastructure.Infrastructure, window string, at time.Time) (*RawMetrics, error) {
	if infra == nil {
		return nil, fmt.Errorf("collector: infra must not be nil")
	}

	raw := &RawMetrics{
		Timestamp: at,
		Devices:   make(map[string]*DeviceRaw, len(infra.Devices)),
		Racks:     make(map[string]*RackRaw),
	}
	for _, dev := range infra.Devices {
		raw.Devices[dev.ID] = &DeviceRaw{
			Metrics:   make(map[string]float64),
			VMMetrics: make(map[string]map[string]float64),
		}
	}

	devIdx := buildDeviceIndex(infra.Devices)

	for name, src := range infra.MetricSources {
		if err := collectSource(q, name, src, raw, infra.Devices, devIdx, window); err != nil {
			warn := fmt.Sprintf("%s query failed: %v", name, err)
			log.Printf("collector warning: %s", warn)
			raw.Warnings = append(raw.Warnings, warn)
		}
	}

	return raw, nil
}

// collectSource
func collectSource(
	q Querier,
	name string,
	src infrastructure.MetricSourceDef,
	raw *RawMetrics,
	devices []infrastructure.ResolvedDevice,
	devIdx map[string]string,
	window string,
) error {
	query := strings.ReplaceAll(src.Query, "{{ .Window }}", window)

	val, err := q.QueryMetric(query)
	if err != nil {
		return err
	}

	vec, _ := val.(prommodel.Vector)
	for _, s := range vec {
		labelVal := string(s.Metric[prommodel.LabelName(src.MatchLabel)])
		if labelVal == "" {
			continue
		}

		devID := matchDevice(labelVal, src.MatchStrategy, devices, devIdx)

		value := float64(s.Value)

		if devID != "" {
			d := raw.Devices[devID]
			if src.VMLabel != "" {
				vmID := string(s.Metric[prommodel.LabelName(src.VMLabel)])
				if vmID == "" {
					continue
				}
				if d.VMMetrics[name] == nil {
					d.VMMetrics[name] = make(map[string]float64)
				}
				d.VMMetrics[name][vmID] = value
			} else {
				d.Metrics[name] = value
			}
		} else {
			rack := raw.Racks[labelVal]
			if rack == nil {
				rack = &RackRaw{Metrics: make(map[string]float64)}
				raw.Racks[labelVal] = rack
			}
			rack.Metrics[name] = value
		}
	}

	return nil
}

// buildDeviceIndex returns a map from device ID to device ID.
func buildDeviceIndex(devices []infrastructure.ResolvedDevice) map[string]string {
	idx := make(map[string]string, len(devices))
	for _, d := range devices {
		idx[d.ID] = d.ID
	}
	return idx
}

// matchDevice maps a Prometheus label value to a device ID using the configured strategy.
func matchDevice(
	labelVal string,
	strategy string,
	devices []infrastructure.ResolvedDevice,
	devIdx map[string]string,
) string {
	switch strategy {
	case "hostname_extract":
		return matchInstanceToDevice(labelVal, devices)
	default:
		return devIdx[strings.ToLower(labelVal)]
	}
}

// matchInstanceToDevice maps a BMC Prometheus instance label to a device ID.
//
//	 instance is a Redfish URL:
//		https://<device_id>.bmc.<domain>/redfish/v1/something something
func matchInstanceToDevice(instance string, devices []infrastructure.ResolvedDevice) string {
	u, err := url.Parse(instance)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	host := u.Hostname()
	if i := strings.Index(host, "."); i > 0 {
		host = host[:i]
	}
	host = strings.ToLower(host)
	for _, d := range devices {
		if strings.ToLower(d.ID) == host {
			return d.ID
		}
	}
	return ""
}
