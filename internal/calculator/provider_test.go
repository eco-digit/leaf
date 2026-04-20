package calculator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eco-digit/leaf/internal/collector"
	"github.com/eco-digit/leaf/internal/infrastructure"
	"github.com/eco-digit/leaf/internal/intensity"
	"github.com/eco-digit/leaf/internal/model"
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

func TestAggregateEnergyByComponent(t *testing.T) {
	// Two compute devices (0.4 + 0.3 = 0.7) and one storage (0.2).
	deviceResults := model.ResultSet{
		{Subject: model.SubjectDevice, Component: "compute", Category: model.CategoryEnergy, Value: 0.4, Unit: "kwh"},
		{Subject: model.SubjectDevice, Component: "compute", Category: model.CategoryEnergy, Value: 0.3, Unit: "kwh"},
		{Subject: model.SubjectDevice, Component: "storage", Category: model.CategoryEnergy, Value: 0.2, Unit: "kwh"},
	}

	rs := aggregateEnergyByComponent(deviceResults, "test-dc", "test-dc", testTS)

	byComponent := make(map[string]model.ImpactResult)
	for _, r := range rs {
		byComponent[r.Component] = r
	}

	require.Contains(t, byComponent, "compute")
	require.Contains(t, byComponent, "storage")
	require.Contains(t, byComponent, "total")

	assert.Equal(t, model.SubjectProvider, byComponent["compute"].Subject)
	assert.InDelta(t, 0.7, byComponent["compute"].Value, 1e-9)
	assert.InDelta(t, 0.2, byComponent["storage"].Value, 1e-9)
	assert.InDelta(t, 0.9, byComponent["total"].Value, 1e-9)
}

func TestOperationalImpactByComponent(t *testing.T) {
	energyRS := model.ResultSet{
		{Subject: model.SubjectProvider, Provider: "dc", Datacenter: "dc", Component: "compute", Category: model.CategoryEnergy, Value: 1.0},
		{Subject: model.SubjectProvider, Provider: "dc", Datacenter: "dc", Component: "storage", Category: model.CategoryEnergy, Value: 0.5},
		{Subject: model.SubjectProvider, Provider: "dc", Datacenter: "dc", Component: "total", Category: model.CategoryEnergy, Value: 1.5},
	}
	factors := intensity.IntensityFactors{
		GWP: intensity.FactorValue{Value: 400, Unit: "g_co2eq_per_kwh"},
		ADP: intensity.FactorValue{Value: 0.423, Unit: "kg_sb_eq_per_kwh"},
		CED: intensity.FactorValue{Value: 10.0, Unit: "mj_per_kwh"},
	}
	pue := 1.5

	rs := operationalImpactByComponent(energyRS, pue, factors, testTS)

	// 3 com x 3 cat = 9 records
	require.Len(t, rs, 9)

	type key struct {
		component string
		cat       model.Category
	}
	byKey := make(map[key]model.ImpactResult)
	for _, r := range rs {
		byKey[key{r.Component, r.Category}] = r
	}

	assert.Equal(t, model.PhaseOperational, byKey[key{"compute", model.CategoryGWP}].ImpactPhase)
	assert.Equal(t, "kg_co2eq", byKey[key{"compute", model.CategoryGWP}].Unit)
	assert.InDelta(t, 0.6, byKey[key{"compute", model.CategoryGWP}].Value, 1e-9)
	assert.InDelta(t, 0.6345, byKey[key{"compute", model.CategoryADP}].Value, 1e-9)
	assert.InDelta(t, 15.0, byKey[key{"compute", model.CategoryCED}].Value, 1e-9)

	// storage: 0.5 kWh x 1.5 x 400 / 1000 = 0.3 kg GWP
	assert.InDelta(t, 0.3, byKey[key{"storage", model.CategoryGWP}].Value, 1e-9)

	// total: 1.5 kWh x1.5 x 400 / 1000 = 0.9 kg GWP
	assert.InDelta(t, 0.9, byKey[key{"total", model.CategoryGWP}].Value, 1e-9)
}

func TestTotalImpactByComponent(t *testing.T) {
	operationalRS := model.ResultSet{
		{Subject: model.SubjectProvider, Provider: "dc", Datacenter: "dc", Component: "compute", ImpactPhase: model.PhaseOperational, Category: model.CategoryGWP, Value: 0.6, Unit: "kg_co2eq"},
		{Subject: model.SubjectProvider, Provider: "dc", Datacenter: "dc", Component: "total", ImpactPhase: model.PhaseOperational, Category: model.CategoryGWP, Value: 0.9, Unit: "kg_co2eq"},
	}
	embodiedRS := model.ResultSet{
		{Subject: model.SubjectProvider, Provider: "dc", Datacenter: "dc", Component: "compute", ImpactPhase: model.PhaseEmbodied, Category: model.CategoryGWP, Value: 0.4, Unit: "kg_co2eq"},
		{Subject: model.SubjectProvider, Provider: "dc", Datacenter: "dc", Component: "total", ImpactPhase: model.PhaseEmbodied, Category: model.CategoryGWP, Value: 0.6, Unit: "kg_co2eq"},
		{Subject: model.SubjectProvider, Provider: "dc", Datacenter: "dc", Component: "total", ImpactPhase: model.PhaseEmbodied, Category: model.CategoryWater, Value: 2.0, Unit: "m3"},
	}

	rs := totalImpactByComponent(operationalRS, embodiedRS, testTS)

	type key struct {
		component string
		cat       model.Category
	}
	byKey := make(map[key]model.ImpactResult)
	for _, r := range rs {
		byKey[key{r.Component, r.Category}] = r
	}

	require.Len(t, rs, 3)

	assert.Equal(t, model.PhaseTotal, byKey[key{"compute", model.CategoryGWP}].ImpactPhase)
	assert.InDelta(t, 1.0, byKey[key{"compute", model.CategoryGWP}].Value, 1e-9)
	assert.InDelta(t, 1.5, byKey[key{"total", model.CategoryGWP}].Value, 1e-9)

	assert.InDelta(t, 2.0, byKey[key{"total", model.CategoryWater}].Value, 1e-9)
	assert.Equal(t, "m3", byKey[key{"total", model.CategoryWater}].Unit)
}

// Test Energy total
func TestValidateEnergy_valid(t *testing.T) {
	rs := model.ResultSet{
		{Subject: model.SubjectDevice, Component: "compute", Category: model.CategoryEnergy, Value: 0.4},
		{Subject: model.SubjectDevice, Component: "compute", Category: model.CategoryEnergy, Value: 0.3},
		{Subject: model.SubjectDevice, Component: "storage", Category: model.CategoryEnergy, Value: 0.2},
		{Subject: model.SubjectProvider, Component: "compute", Category: model.CategoryEnergy, Value: 0.7},
		{Subject: model.SubjectProvider, Component: "storage", Category: model.CategoryEnergy, Value: 0.2},
		{Subject: model.SubjectProvider, Component: "total", Category: model.CategoryEnergy, Value: 0.9},
	}
	assert.Empty(t, validateEnergy(rs))
}

// Test the compute pipline
func TestProviderResults_EndToEnd(t *testing.T) {
	infra := makeInfra([]infrastructure.ResolvedDevice{
		{ID: "compute01", Component: "compute"},
		{ID: "storage01", Component: "storage"},
	})
	raw := makeRawDeviceData(map[string]map[string]float64{
		"compute01": {"bmc": 300},
		"storage01": {"bmc": 200},
	})
	factors := intensity.IntensityFactors{
		GWP: intensity.FactorValue{Value: 500, Unit: ""},
	}
	embodiedRS := model.ResultSet{
		{Subject: model.SubjectProvider, Provider: "test-dc", Datacenter: "test-dc", Component: "compute", ImpactPhase: model.PhaseEmbodied, Category: model.CategoryGWP, Value: 0.1, Unit: "kg_co2eq"},
		{Subject: model.SubjectProvider, Provider: "test-dc", Datacenter: "test-dc", Component: "total", ImpactPhase: model.PhaseEmbodied, Category: model.CategoryGWP, Value: 0.15, Unit: "kg_co2eq"},
	}

	rs, warnings := ProviderResults(raw, infra, factors, embodiedRS, testTS)

	assert.Empty(t, warnings)

	type key struct {
		subject   model.SubjectType
		component string
		phase     model.ImpactPhase
		cat       model.Category
	}
	byKey := make(map[key]model.ImpactResult)
	for _, r := range rs {
		byKey[key{r.Subject, r.Component, r.ImpactPhase, r.Category}] = r
	}

	// device energy
	assert.InDelta(t, 0.3, byKey[key{model.SubjectDevice, "compute", model.PhaseOperational, model.CategoryEnergy}].Value, 1e-9)

	// component energy
	assert.InDelta(t, 0.3, byKey[key{model.SubjectProvider, "compute", model.PhaseOperational, model.CategoryEnergy}].Value, 1e-9)
	assert.InDelta(t, 0.5, byKey[key{model.SubjectProvider, "total", model.PhaseOperational, model.CategoryEnergy}].Value, 1e-9)

	assert.InDelta(t, 0.18, byKey[key{model.SubjectProvider, "compute", model.PhaseOperational, model.CategoryGWP}].Value, 1e-9)

	assert.InDelta(t, 0.28, byKey[key{model.SubjectProvider, "compute", model.PhaseTotal, model.CategoryGWP}].Value, 1e-9)
}
