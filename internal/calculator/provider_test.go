package calculator

import (
	"testing"
	"time"

	"github.com/OSBA-eco-digit/leaf/internal/collector"
	"github.com/OSBA-eco-digit/leaf/internal/infrastructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeInfra(devices []infrastructure.ResolvedDevice) *infrastructure.Infrastructure {
	return &infrastructure.Infrastructure{
		Environment: infrastructure.Environment{
			ID:  "test-dc",
			PUE: 1.2,
		},
		Devices: devices,
	}
}

func makeRawDeviceData(deviceMetrics map[string]map[string]float64) *collector.RawMetrics {
	raw := &collector.RawMetrics{
		Timestamp: time.Time{},
		Devices:   make(map[string]*collector.DeviceRaw, len(deviceMetrics)),
	}
	for id, metrics := range deviceMetrics {
		raw.Devices[id] = &collector.DeviceRaw{Metrics: metrics, VMMetrics: make(map[string]map[string]float64)}
	}
	return raw
}

func TestDeviceEnergy(t *testing.T) {
	d := &collector.DeviceRaw{Metrics: map[string]float64{"bmc": 300}}
	kwh, fallback := deviceEnergyKWh(d)

	require.False(t, fallback, "expected BMC path, got Kepler fallback")
	assert.InDelta(t, 0.3, kwh, 1e-9)
}

func TestDeviceEnergyKWh(t *testing.T) {
}
