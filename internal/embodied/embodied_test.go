package embodied

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/OSBA-eco-digit/leaf/internal/infrastructure"
	"github.com/OSBA-eco-digit/leaf/internal/model"
)

const infraYAML = `
version: 1
environment:
  id: test-dc
  name: Test Datacenter
devices:
  - id: compute01
    role: compute
    profile: compute
    rack: rack01
  - id: compute02
    role: compute
    profile: compute
    rack: rack01
  - id: storage01
    role: storage
    profile: storage
    rack: rack01
  - id: switch01
    role: network
    profile: switch
    rack: rack01
`

// compute: lifespan 4y (just an assumption being made) / divisor = 4 * 8760
//
//	GWP/h  = 20803.838 / 35040 = 0.59370
//	ADP/h  = 0.691     / 35040 = 0.0000197
//	CED/h  = 257957.649/ 35040 = 7.36202
//	Water/h= 8680.273  / 35040  0.24772
const profileYAML = `
version: 1
profiles:
  compute:
    kind: server
    vendor: MiTAC
    model: E8020-A
    default_lifespan_years: 4
    hardware:
      cpu:
        cores: 64
      memory_gb: 512
    embodied_impact:
      gwp:
        value: "20803.838"
        unit: kg_co2eq
      adp:
        value: "0.691"
        unit: kg_sb_eq
      ced:
        value: "257957.649"
        unit: mj
      water:
        value: "8680.273"
        unit: m3

  storage:
    kind: server
    vendor: MiTAC
    model: E8020-A
    default_lifespan_years: 4
    hardware:
      cpu:
        cores: 16
      memory_gb: 128
    embodied_impact:
      gwp:
        value: "15000.0"
        unit: kg_co2eq
      adp:
        value: "0.5"
        unit: kg_sb_eq
      ced:
        value: "180000.0"
        unit: mj
      water:
        value: "6000.0"
        unit: m3

  switch:
    kind: switch
    vendor: Edgecore
    model: 7726-32x
    hardware:
      switching_capacity: 100g
    embodied_impact:
      gwp:
        value: "840"
        unit: kg_co2eq
      adp:
        value: "0.035"
        unit: kg_sb_eq
      ced:
        value: "10.800"
        unit: mj
      water:
        value: "1.8"
        unit: m3
`

func loadTestInfra(t *testing.T) *infrastructure.Infrastructure {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "infrastructure.yaml", infraYAML)
	writeFile(t, dir, "profile.yaml", profileYAML)
	infra, err := infrastructure.Load(
		filepath.Join(dir, "infrastructure.yaml"),
		filepath.Join(dir, "profile.yaml"),
	)
	if err != nil {
		t.Fatalf("infrastructure.Load: %v", err)
	}
	return infra
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("writeFile %s: %v", name, err)
	}
}

func approxEqual(a, b, tol float64) bool {
	return math.Abs(a-b) <= tol
}

// TestCalculate_ProviderTotal
func TestCalculate_ProviderTotal(t *testing.T) {
	infra := loadTestInfra(t)
	ts := time.Now().Truncate(time.Hour)

	rs, err := Calculate(infra, ts)
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}

	// Total GWP = 2xcompute + 1xstorage + 1xswitch
	divisor := 4.0 * 8760
	wantTotal := (2*20803.838 + 15000.0 + 840.0) / divisor

	totalGWP := rs.
		FilterBySubject(model.SubjectProvider).
		FilterByComponent("total").
		FilterByPhase(model.PhaseEmbodied).
		FilterByCategory(model.CategoryGWP)

	if len(totalGWP) != 1 {
		t.Fatalf("total GWP records: got %d, want 1", len(totalGWP))
	}
	if !approxEqual(totalGWP[0].Value, wantTotal, 1e-9) {
		t.Errorf("total GWP = %.10f, want %.10f", totalGWP[0].Value, wantTotal)
	}
}

func TestCalculate_RecordCount(t *testing.T) {
	infra := loadTestInfra(t)
	ts := time.Now().Truncate(time.Hour)

	rs, err := Calculate(infra, ts)
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}

	// 4 devices x 4 categories = 16 device records
	deviceRecords := rs.FilterBySubject(model.SubjectDevice)
	if len(deviceRecords) != 16 {
		t.Errorf("device records: got %d, want 16", len(deviceRecords))
	}

	// 3 components (compute, storage, network) x  4 categories = 12 component records
	// + 4 total records = 16 provider records
	providerRecords := rs.FilterBySubject(model.SubjectProvider)
	if len(providerRecords) != 16 {
		t.Errorf("provider records: got %d, want 16", len(providerRecords))
	}

	totalRecords := rs.FilterBySubject(model.SubjectProvider).FilterByComponent("total")
	if len(totalRecords) != 4 {
		t.Errorf("total records: got %d, want 4", len(totalRecords))
	}
}
