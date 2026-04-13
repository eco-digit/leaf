package model

import (
	"testing"
	"time"
)

// buildTestSet returns a small ResultSet, ensuring differnet filter functions works as aspect.
func buildTestSet() ResultSet {
	ts := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)
	return ResultSet{
		// Provider embodied GWP
		{
			Subject: SubjectProvider, Datacenter: "dc1", Component: "compute",
			ImpactPhase: PhaseEmbodied, Category: CategoryGWP,
			Value: 1.5, Unit: "kg_co2eq", Timestamp: ts, PeriodHours: 1,
		},
		// Provider embodied GWP
		{
			Subject: SubjectProvider, Datacenter: "dc1", Component: "storage",
			ImpactPhase: PhaseEmbodied, Category: CategoryGWP,
			Value: 0.8, Unit: "kg_co2eq", Timestamp: ts, PeriodHours: 1,
		},
		// Provider operational ADP
		{
			Subject: SubjectProvider, Datacenter: "dc1", Component: "compute",
			ImpactPhase: PhaseOperational, Category: CategoryADP,
			Value: 0.002, Unit: "kg_sb_eq", Timestamp: ts, PeriodHours: 1,
		},
		// Device embodied water
		{
			Subject: SubjectDevice, Datacenter: "dc1", Component: "compute", Device: "compute01",
			ImpactPhase: PhaseEmbodied, Category: CategoryWater,
			Value: 0.25, Unit: "m3", Timestamp: ts, PeriodHours: 1,
		},
		// Device energy
		{
			Subject: SubjectDevice, Datacenter: "dc1", Component: "compute", Device: "compute01",
			ImpactPhase: PhaseOperational, Category: CategoryEnergy,
			Value: 1.2, Unit: "kwh", Timestamp: ts, PeriodHours: 1,
		},
		// Tenant total GWP
		{
			Subject: SubjectTenant, Datacenter: "dc1", Component: "total",
			ProjectID: "x1839ca", ProjectName: "alpha",
			ImpactPhase: PhaseTotal, Category: CategoryGWP,
			Value: 0.3, Unit: "kg_co2eq", Timestamp: ts, PeriodHours: 1,
		},
		// Tenant total CED
		{
			Subject: SubjectTenant, Datacenter: "dc1", Component: "total",
			ProjectID: "proj-b", ProjectName: "beta",
			ImpactPhase: PhaseTotal, Category: CategoryCED,
			Value: 12.5, Unit: "mj", Timestamp: ts, PeriodHours: 1,
		},
	}
}

func TestFilterBySubject(t *testing.T) {
	rs := buildTestSet()

	providerResults := rs.FilterBySubject(SubjectProvider)
	if len(providerResults) != 3 {
		t.Errorf("FilterBySubject(provider): got %d, want 3", len(providerResults))
	}
	for _, r := range providerResults {
		if r.Subject != SubjectProvider {
			t.Errorf("unexpected subject %q in provider filter", r.Subject)
		}
	}

	deviceResults := rs.FilterBySubject(SubjectDevice)
	if len(deviceResults) != 2 {
		t.Errorf("FilterBySubject(device): got %d, want 2", len(deviceResults))
	}

	tenantResults := rs.FilterBySubject(SubjectTenant)
	if len(tenantResults) != 2 {
		t.Errorf("FilterBySubject(tenant): got %d, want 2", len(tenantResults))
	}
}

func TestFilterByPhase(t *testing.T) {
	rs := buildTestSet()

	embodied := rs.FilterByPhase(PhaseEmbodied)
	if len(embodied) != 3 {
		t.Errorf("FilterByPhase(embodied): got %d, want 3", len(embodied))
	}

	operational := rs.FilterByPhase(PhaseOperational)
	if len(operational) != 2 {
		t.Errorf("FilterByPhase(operational): got %d, want 2", len(operational))
	}

	total := rs.FilterByPhase(PhaseTotal)
	if len(total) != 2 {
		t.Errorf("FilterByPhase(total): got %d, want 2", len(total))
	}
}

func TestFilterByCategory(t *testing.T) {
	rs := buildTestSet()

	gwp := rs.FilterByCategory(CategoryGWP)
	if len(gwp) != 3 {
		t.Errorf("FilterByCategory(gwp): got %d, want 3", len(gwp))
	}

	water := rs.FilterByCategory(CategoryWater)
	if len(water) != 1 {
		t.Errorf("FilterByCategory(water): got %d, want 1", len(water))
	}

	energy := rs.FilterByCategory(CategoryEnergy)
	if len(energy) != 1 {
		t.Errorf("FilterByCategory(energy): got %d, want 1", len(energy))
	}
}

func TestFilterByDevice(t *testing.T) {
	rs := buildTestSet()

	compute01 := rs.FilterByDevice("compute01")
	if len(compute01) != 2 {
		t.Errorf("FilterByDevice(compute01): got %d, want 2", len(compute01))
	}

	none := rs.FilterByDevice("nonexistent")
	if len(none) != 0 {
		t.Errorf("FilterByDevice(nonexistent): got %d, want 0", len(none))
	}
}

func TestFilterByProject(t *testing.T) {
	rs := buildTestSet()

	projA := rs.FilterByProject("x1839ca")
	if len(projA) != 1 {
		t.Errorf("FilterByProject(x1839ca): got %d, want 1", len(projA))
	}
	if projA[0].ProjectName != "alpha" {
		t.Errorf("unexpected project name %q", projA[0].ProjectName)
	}
}
