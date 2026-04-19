package calculator

import (
	"testing"
	"time"

	"github.com/OSBA-eco-digit/leaf/internal/collector"
	"github.com/OSBA-eco-digit/leaf/internal/infrastructure"
	"github.com/OSBA-eco-digit/leaf/internal/model"
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

var testTS = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func TestDeviceEnergyKWh(t *testing.T) {
	infra := makeInfra([]infrastructure.ResolvedDevice{
		{ID: "compute01", Component: "compute"},
		{ID: "storage01", Component: "storage"},
	})
	raw := makeRawDeviceData(map[string]map[string]float64{
		"compute01": {"bmc": 400},
		"storage01": {"bmc": 200},
	})

	rs, warnings := deviceEnergyResults(raw, infra, testTS)

	assert.Empty(t, warnings)
	assert.Len(t, rs, 2)

	byDevice := make(map[string]model.ImpactResult)
	for _, r := range rs {
		byDevice[r.Device] = r
	}

	checkResult(t, byDevice["compute01"], 0.4, "compute")
	checkResult(t, byDevice["storage01"], 0.2, "storage")
}

func checkResult(t *testing.T, r model.ImpactResult, wantKWh float64, wantComponent string) {
	t.Helper()
	assert.Equal(t, model.SubjectDevice, r.Subject)
	assert.Equal(t, model.CategoryEnergy, r.Category)
	assert.Equal(t, wantComponent, r.Component)
	assert.InDelta(t, wantKWh, r.Value, 1e-9)
	assert.Equal(t, "kwh", r.Unit)
}
