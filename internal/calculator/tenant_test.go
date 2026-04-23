package calculator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eco-digit/leaf/internal/collector"
	"github.com/eco-digit/leaf/internal/infrastructure"
	"github.com/eco-digit/leaf/internal/intensity"
	"github.com/eco-digit/leaf/internal/model"
)

func makeComputeProfile(cores int, memGB float64) *infrastructure.Profile {
	return &infrastructure.Profile{
		DefaultLifespanYears: 4,
		Hardware: infrastructure.Hardware{
			CPU:      infrastructure.CPU{Cores: cores},
			MemoryGB: memGB,
		},
	}
}

type computeNode struct {
	id    string
	cores int
	memGB float64
}

func makeTenantInfra(nodes []computeNode) *infrastructure.Infrastructure {
	devices := make([]infrastructure.ResolvedDevice, len(nodes))
	for i, n := range nodes {
		devices[i] = infrastructure.ResolvedDevice{
			ID:        n.id,
			Component: "compute",
			Profile:   makeComputeProfile(n.cores, n.memGB),
		}
	}
	return &infrastructure.Infrastructure{
		Environment: infrastructure.Environment{ID: "test-dc", PUE: 1.2},
		Devices:     devices,
	}
}

// makeRaw builds a *RawMetrics. deviceMetrics maps deviceID to value.
func makeRaw(
	deviceMetrics map[string]map[string]float64,
	vmMetrics map[string]map[string]float64,
	vms []collector.VMInfo,
) *collector.RawMetrics {
	raw := &collector.RawMetrics{
		Devices: make(map[string]*collector.DeviceRaw),
		VMs:     vms,
	}
	for id, metrics := range deviceMetrics {
		raw.Devices[id] = &collector.DeviceRaw{
			Metrics:   metrics,
			VMMetrics: make(map[string]map[string]float64),
		}
	}
	for devID, vmMap := range vmMetrics {
		if raw.Devices[devID] == nil {
			raw.Devices[devID] = &collector.DeviceRaw{
				Metrics:   make(map[string]float64),
				VMMetrics: make(map[string]map[string]float64),
			}
		}
		raw.Devices[devID].VMMetrics["kepler_vm"] = vmMap
	}
	return raw
}

// makeContextRS builds the provider result set that TenantResults reads.
func makeContextRS(computeKWh, networkKWh, controlKWh, totalKWh, embGWP, embWater float64) model.ResultSet {
	energy := func(component string, val float64) model.ImpactResult {
		return model.ImpactResult{
			Subject: model.SubjectProvider, Provider: "test-dc", Datacenter: "test-dc",
			ImpactPhase: model.PhaseOperational, Category: model.CategoryEnergy,
			Component: component, Value: val, Unit: "kwh",
		}
	}
	emb := func(cat model.Category, unit string, val float64) model.ImpactResult {
		return model.ImpactResult{
			Subject: model.SubjectProvider, Provider: "test-dc", Datacenter: "test-dc",
			ImpactPhase: model.PhaseEmbodied, Category: cat,
			Component: "total", Value: val, Unit: unit,
		}
	}
	return model.ResultSet{
		energy("compute", computeKWh),
		energy("network", networkKWh),
		energy("control", controlKWh),
		energy("total", totalKWh),
		emb(model.CategoryGWP, "kg_co2eq", embGWP),
		emb(model.CategoryWater, "m3", embWater),
	}
}

func TestComputeCapacity(t *testing.T) {
	infra := makeTenantInfra([]computeNode{
		{"compute01", 10, 100},
		{"compute02", 10, 100},
	})
	vcpuTotal, memTotal := computeCapacity(infra)
	assert.Equal(t, 20.0, vcpuTotal)
	assert.Equal(t, 200.0, memTotal)
}

func TestComputeCapacity_SkipsNonCompute(t *testing.T) {
	infra := &infrastructure.Infrastructure{
		Environment: infrastructure.Environment{ID: "dc", PUE: 1.2},
		Devices: []infrastructure.ResolvedDevice{
			{ID: "sw01", Component: "network", Profile: makeComputeProfile(4, 32)},
			{ID: "cp01", Component: "compute", Profile: makeComputeProfile(8, 64)},
		},
	}
	vcpuTotal, memTotal := computeCapacity(infra)
	assert.Equal(t, 8.0, vcpuTotal)
	assert.Equal(t, 64.0, memTotal)
}

func TestTenantAllocations(t *testing.T) {
	vms := []collector.VMInfo{
		{VMID: "vm1", ProjectID: "proj-a", ProjectName: "Alpha", VCPUs: 4, MemoryGB: 8},
		{VMID: "vm2", ProjectID: "proj-a", ProjectName: "Alpha", VCPUs: 4, MemoryGB: 8},
		{VMID: "vm3", ProjectID: "proj-b", ProjectName: "Beta", VCPUs: 2, MemoryGB: 4},
		{VMID: "vm4", ProjectID: "proj-c", ProjectName: "Zero", VCPUs: 0, MemoryGB: 0},
	}
	allocs := tenantAllocations(vms, 20, 40)

	require.Len(t, allocs, 2, "zero-resource VM must be excluded")

	// Alpha: vcpu=8, mem=16 → ratio = (8/20 + 16/40) / 2 = 0.4
	assert.InDelta(t, 0.4, allocs["proj-a"].ratio, 1e-9)
	assert.Equal(t, "Alpha", allocs["proj-a"].projectName)

	// Beta: vcpu=2, mem=4 → ratio = (2/20 + 4/40) / 2 = 0.1
	assert.InDelta(t, 0.1, allocs["proj-b"].ratio, 1e-9)

	_, hasZero := allocs["proj-c"]
	assert.False(t, hasZero)
}

func TestIdleAndDynamic(t *testing.T) {
	infra := makeTenantInfra([]computeNode{{"compute01", 10, 100}})
	raw := makeRaw(
		map[string]map[string]float64{
			"compute01": {"kepler_node_idle": 3_600_000}, // 1 kWh
		},
		map[string]map[string]float64{
			"compute01": {"vm-a": 720_000, "vm-b": 360_000}, // 0.2 and 0.1 kWh
		},
		nil,
	)
	vmMap := map[string]string{"vm-a": "proj-a", "vm-b": "proj-b"}

	idleTotal, vmDynamic := idleAndDynamic(raw, infra, vmMap)

	assert.InDelta(t, 1.0, idleTotal, 1e-9)
	assert.InDelta(t, 0.2, vmDynamic["proj-a"], 1e-9)
	assert.InDelta(t, 0.1, vmDynamic["proj-b"], 1e-9)
}

func TestIdleAndDynamic_UnknownVMIgnored(t *testing.T) {
	infra := makeTenantInfra([]computeNode{{"compute01", 10, 100}})
	raw := makeRaw(
		nil,
		map[string]map[string]float64{
			"compute01": {"vm-orphan": 720_000}, // no entry in vmMap
		},
		nil,
	)
	_, vmDynamic := idleAndDynamic(raw, infra, map[string]string{})
	assert.Empty(t, vmDynamic)
}

// end-to-end TenantResults
func TestTenantResults_HappyPath(t *testing.T) {
	// 1 compute node: 10 cores, 10 GB
	// Tenant A: 4 vcpu, 4 GB > ratio = (4/10 + 4/10) / 2 = 0.4
	//
	// idle_total  = 3_600_000 J / 3600 / 1000 = 1.0 kWh
	// idle_share  = 1.0 x 0.4 = 0.4 kWh
	// vm_dynamic  = 720_000 J / 3600 / 1000 = 0.2 kWh
	// frac        = (0.4 + 0.2) / 1.2 = 0.5
	// net_share   = 0.3 x 0.5 = 0.15 kWh
	// ctrl_share  = 0.1 x 0.5 = 0.05 kWh
	// E           = 0.4 + 0.2 + 0.15 + 0.05 = 0.8 kWh
	//
	// gwp_op  = 0.8 x 1.2 x 400 / 1000 = 0.384 kg
	// gwp_emb = 0.4 x 1.0 = 0.4 kg
	// gwp_tot = 0.784 kg
	// water   = 0.4 x 100.0 = 40.0 m3

	infra := makeTenantInfra([]computeNode{{"compute01", 10, 10}})
	raw := makeRaw(
		map[string]map[string]float64{
			"compute01": {"kepler_node_idle": 3_600_000},
		},
		map[string]map[string]float64{
			"compute01": {"vm-a1": 720_000},
		},
		[]collector.VMInfo{
			{VMID: "vm-a1", ProjectID: "proj-a", ProjectName: "Alpha", VCPUs: 4, MemoryGB: 4},
		},
	)
	contextRS := makeContextRS(1.2, 0.3, 0.1, 1.6, 1.0, 100.0)
	factors := intensity.IntensityFactors{
		GWP: intensity.FactorValue{Value: 400, Unit: "g_co2eq_per_kwh"},
	}

	rs, warnings := TenantResults(raw, infra, factors, contextRS, testTS)

	assert.Empty(t, warnings)
	require.NotEmpty(t, rs)

	type key struct {
		phase model.ImpactPhase
		cat   model.Category
	}
	byKey := make(map[key]model.ImpactResult)
	for _, r := range rs {
		if r.Subject == model.SubjectTenant && r.ProjectID == "proj-a" {
			byKey[key{r.ImpactPhase, r.Category}] = r
		}
	}

	assert.InDelta(t, 0.8, byKey[key{model.PhaseOperational, model.CategoryEnergy}].Value, 1e-9)
	assert.InDelta(t, 0.384, byKey[key{model.PhaseOperational, model.CategoryGWP}].Value, 1e-9)
	assert.InDelta(t, 0.4, byKey[key{model.PhaseEmbodied, model.CategoryGWP}].Value, 1e-9)
	assert.InDelta(t, 0.784, byKey[key{model.PhaseTotal, model.CategoryGWP}].Value, 1e-9)
	assert.InDelta(t, 40.0, byKey[key{model.PhaseEmbodied, model.CategoryWater}].Value, 1e-9)
	assert.InDelta(t, 40.0, byKey[key{model.PhaseTotal, model.CategoryWater}].Value, 1e-9)

	// labels
	r := byKey[key{model.PhaseOperational, model.CategoryEnergy}]
	assert.Equal(t, "proj-a", r.ProjectID)
	assert.Equal(t, "Alpha", r.ProjectName)
	assert.Equal(t, "total", r.Component)
	assert.Equal(t, model.SubjectTenant, r.Subject)
}
