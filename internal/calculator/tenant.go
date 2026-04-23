package calculator

import (
	"fmt"
	"time"

	"github.com/eco-digit/leaf/internal/collector"
	"github.com/eco-digit/leaf/internal/infrastructure"
	"github.com/eco-digit/leaf/internal/intensity"
	"github.com/eco-digit/leaf/internal/model"
)

const (
	alphaNetwork = 1.0
	alphaControl = 1.0
)

// TenantResults computes tenant-level ImpactResults using the Idle + Dynamic model.
//
// contextRS must contain both provider component energy records
// and provider embodied total records
func TenantResults(
	raw collector.RawSource,
	infra *infrastructure.Infrastructure,
	factors intensity.IntensityFactors,
	contextRS model.ResultSet,
	ts time.Time,
) (model.ResultSet, []string) {
	datacenter := infra.Environment.ID
	provider := infra.Environment.ID
	pue := infra.Environment.PUE

	// Step 1: physical capacity  from compute profiles
	vcpuTotal, memTotal := computeCapacity(infra)
	if vcpuTotal == 0 && memTotal == 0 {
		return nil, []string{"tenant: no compute capacity found in profiles, skipping"}
	}

	// Step 2: per-tenant resource allocation ratios
	allocs := tenantAllocations(raw.VMInfos(), vcpuTotal, memTotal)
	if len(allocs) == 0 {
		return nil, nil
	}

	// Step 3: infrastructure-wide idle energy and per-tenant VM dynamic energy
	vmProjectMap := buildVMProjectMap(raw.VMInfos())
	idleTotal, vmDynamic := idleAndDynamic(raw, infra, vmProjectMap)

	providerComputeKWh := componentEnergyKWh(contextRS, "compute")
	providerNetKWh := componentEnergyKWh(contextRS, "network")
	providerCtrlKWh := componentEnergyKWh(contextRS, "control")
	providerTotalKWh := componentEnergyKWh(contextRS, "total")
	embodiedTotals := providerEmbodiedTotals(contextRS)

	type opSpec struct {
		cat   model.Category
		fac   float64
		unit  string
		scale float64
	}
	opSpecs := []opSpec{
		{model.CategoryGWP, factors.GWP.Value, "kg_co2eq", 1.0 / 1000.0},
		{model.CategoryADP, factors.ADP.Value, "kg_sb_eq", 1.0},
		{model.CategoryCED, factors.CED.Value, "mj", 1.0},
	}

	var rs model.ResultSet
	var warnings []string
	ratioSum := 0.0
	energySum := 0.0

	for pid, a := range allocs {
		ratioSum += a.ratio

		// Steps 3–6: compute tenant energy
		idleShare := idleTotal * a.ratio
		dynamic := vmDynamic[pid]

		// Step 4: compute fraction — used to attribute network and control overhead
		var frac float64
		if providerComputeKWh > 0 {
			frac = (idleShare + dynamic) / providerComputeKWh
		}

		// Step 5: proportional network and control overhead
		netShare := providerNetKWh * alphaNetwork * frac
		ctrlShare := providerCtrlKWh * alphaControl * frac

		// Step 6: total tenant energy
		E := idleShare + dynamic + netShare + ctrlShare
		energySum += E

		base := model.ImpactResult{
			Subject:     model.SubjectTenant,
			Provider:    provider,
			Datacenter:  datacenter,
			Component:   "total",
			ProjectID:   pid,
			ProjectName: a.projectName,
			Timestamp:   ts,
			PeriodHours: windowHours,
		}

		if E > 0 {
			r := base
			r.ImpactPhase = model.PhaseOperational
			r.Category = model.CategoryEnergy
			r.Value = E
			r.Unit = reportingUnit
			rs = append(rs, r)
		}

		// Steps 7+8+9: GWP, ADP, CED — operational + embodied + total
		for _, s := range opSpecs {
			var opVal float64
			if s.fac > 0 {
				opVal = E * pue * s.fac * s.scale
			}
			embVal := a.ratio * embodiedTotals[s.cat]

			if opVal > 0 {
				r := base
				r.ImpactPhase = model.PhaseOperational
				r.Category = s.cat
				r.Value = opVal
				r.Unit = s.unit
				rs = append(rs, r)
			}
			if embVal > 0 {
				r := base
				r.ImpactPhase = model.PhaseEmbodied
				r.Category = s.cat
				r.Value = embVal
				r.Unit = s.unit
				rs = append(rs, r)
			}
			if total := opVal + embVal; total > 0 {
				r := base
				r.ImpactPhase = model.PhaseTotal
				r.Category = s.cat
				r.Value = total
				r.Unit = s.unit
				rs = append(rs, r)
			}
		}

		// Water: embodied only in V1
		if waterEmb := a.ratio * embodiedTotals[model.CategoryWater]; waterEmb > 0 {
			r := base
			r.ImpactPhase = model.PhaseEmbodied
			r.Category = model.CategoryWater
			r.Value = waterEmb
			r.Unit = "m3"
			rs = append(rs, r)

			r2 := base
			r2.ImpactPhase = model.PhaseTotal
			r2.Category = model.CategoryWater
			r2.Value = waterEmb
			r2.Unit = "m3"
			rs = append(rs, r2)
		}
	}

	if ratioSum > 1.0+1e-9 {
		warnings = append(warnings, fmt.Sprintf("tenant: sum(ratio) = %.6f exceeds 1.0", ratioSum))
	}
	if providerTotalKWh > 0 && energySum > providerTotalKWh+1e-9 {
		warnings = append(warnings, fmt.Sprintf("tenant: sum(E_t) = %.6f exceeds provider total %.6f", energySum, providerTotalKWh))
	}

	return rs, warnings
}

// computeCapacity returns total physical CPU cores and memory across all
// compute devices that have profiles attached.
func computeCapacity(infra *infrastructure.Infrastructure) (vcpuTotal, memTotal float64) {
	for _, dev := range infra.Devices {
		if dev.Component != "compute" || dev.Profile == nil {
			continue
		}
		vcpuTotal += float64(dev.Profile.Hardware.CPU.Cores)
		memTotal += dev.Profile.Hardware.MemoryGB
	}
	return
}

type tenantAlloc struct {
	projectName string
	vcpu        int
	memGB       int
	ratio       float64
}

// tenantAllocations groups VMInfos by project and computes each tenant's
// resource-based allocation ratio: (vcpu_t/vcpu_total + mem_t/mem_total) / 2.
// VMs where both VCPUs and MemoryGB are zero are excluded.
func tenantAllocations(vms []collector.VMInfo, vcpuTotal, memTotal float64) map[string]tenantAlloc {
	type accum struct {
		name string
		vcpu int
		mem  int
	}
	byProject := make(map[string]*accum)
	for _, vm := range vms {
		if vm.VCPUs == 0 && vm.MemoryGB == 0 {
			continue
		}
		a := byProject[vm.ProjectID]
		if a == nil {
			a = &accum{name: vm.ProjectName}
			byProject[vm.ProjectID] = a
		}
		a.vcpu += vm.VCPUs
		a.mem += vm.MemoryGB
	}

	result := make(map[string]tenantAlloc, len(byProject))
	for pid, a := range byProject {
		var ratio float64
		if vcpuTotal > 0 {
			ratio += float64(a.vcpu) / vcpuTotal
		}
		if memTotal > 0 {
			ratio += float64(a.mem) / memTotal
		}
		ratio /= 2
		result[pid] = tenantAlloc{
			projectName: a.name,
			vcpu:        a.vcpu,
			memGB:       a.mem,
			ratio:       ratio,
		}
	}
	return result
}

// buildVMProjectMap creates a vmID to projectID lookup from VM metadata.
func buildVMProjectMap(vms []collector.VMInfo) map[string]string {
	m := make(map[string]string, len(vms))
	for _, vm := range vms {
		m[vm.VMID] = vm.ProjectID
	}
	return m
}

// idleAndDynamic computes infrastructure-wide idle energy (kWh) and
// per-project VM dynamic energy (kWh) from compute devices.
// Joules are converted tokWh.
func idleAndDynamic(
	raw collector.RawSource,
	infra *infrastructure.Infrastructure,
	vmProjectMap map[string]string,
) (idleTotal float64, vmDynamic map[string]float64) {
	vmDynamic = make(map[string]float64)
	for _, dev := range infra.Devices {
		if dev.Component != "compute" {
			continue
		}
		if idleJ, ok := raw.MetricValue(dev.ID, "kepler_node_idle"); ok && idleJ > 0 {
			idleTotal += idleJ / 3600 / 1000
		}
		for vmID, joules := range raw.VMMetricValues(dev.ID, "kepler_vm") {
			pid, ok := vmProjectMap[vmID]
			if !ok {
				continue
			}
			vmDynamic[pid] += joules / 3600 / 1000
		}
	}
	return
}

// componentEnergyKWh returns the provider operational energy for the given component.
func componentEnergyKWh(rs model.ResultSet, component string) float64 {
	for _, r := range rs {
		if r.Subject == model.SubjectProvider &&
			r.ImpactPhase == model.PhaseOperational &&
			r.Category == model.CategoryEnergy &&
			r.Component == component {
			return r.Value
		}
	}
	return 0
}

// providerEmbodiedTotals extracts provider-level embodied totals by category from contextRS.
func providerEmbodiedTotals(rs model.ResultSet) map[model.Category]float64 {
	totals := make(map[model.Category]float64)
	for _, r := range rs {
		if r.Subject == model.SubjectProvider &&
			r.ImpactPhase == model.PhaseEmbodied &&
			r.Component == "total" {
			totals[r.Category] = r.Value
		}
	}
	return totals
}
