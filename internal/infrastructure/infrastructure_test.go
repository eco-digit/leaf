package infrastructure

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeFile %s: %v", name, err)
	}
	return path
}

const infraYAML = `
version: 1
environment:
  id: test-env
  name: Test Environment
devices:
  - id: compute01
    role: compute
    profile: compute-standard
    rack: rack01
  - id: compute02
    role: compute
    profile: compute-standard
    rack: rack01
  - id: storage01
    role: storage
    profile: storage-standard
    rack: rack01
  - id: switch01
    role: network
    profile: switch-standard
    rack: rack01
  - id: controller01
    role: controller
    profile: controller-standard
    rack: rack02
  - id: manager01
    role: manager
    profile: controller-standard
    rack: rack02
  - id: mgmt-switch01
    role: management-network
    profile: switch-standard
    rack: rack02
`

const profileYAML = `
version: 1
profiles:
  compute-standard:
    kind: server
    vendor: MiTAC
    model: E8020-A
    default_lifespan_years: 4
    hardware:
      cpu:
        cores: 64
      memory_gb: 512
      storage:
        - type: sata_ssd
          size_gb: 960
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

  storage-standard:
    kind: server
    vendor: MiTAC
    model: E8020-A
    default_lifespan_years: 4
    hardware:
      cpu:
        cores: 16
      memory_gb: 128
      storage:
        - type: nvme_u2
          count: 4
          size_tb: 4
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

  controller-standard:
    kind: server
    vendor: MiTAC
    model: E8020-A
    default_lifespan_years: 4
    hardware:
      cpu:
        cores: 20
      memory_gb: 128
    embodied_impact:
      gwp:
        value: "18000.0"
        unit: kg_co2eq
      adp:
        value: "0.600"
        unit: kg_sb_eq
      ced:
        value: "200000.0"
        unit: mj
      water:
        value: "7000.0"
        unit: m3

  switch-standard:
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

func TestLoad_Success(t *testing.T) {
	dir := t.TempDir()
	infraPath := writeFile(t, dir, "infrastructure.yaml", infraYAML)
	profilePath := writeFile(t, dir, "profile.yaml", profileYAML)

	infra, err := Load(infraPath, profilePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if infra.Environment.ID != "test-env" {
		t.Errorf("environment.id = %q, want %q", infra.Environment.ID, "test-env")
	}
	if len(infra.Devices) != 7 {
		t.Errorf("devices count = %d, want 7", len(infra.Devices))
	}
}

func TestLoad_ProfileResolution(t *testing.T) {
	dir := t.TempDir()
	infraPath := writeFile(t, dir, "infrastructure.yaml", infraYAML)
	profilePath := writeFile(t, dir, "profile.yaml", profileYAML)

	infra, err := Load(infraPath, profilePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	byID := make(map[string]*ResolvedDevice)
	for i := range infra.Devices {
		d := &infra.Devices[i]
		byID[d.ID] = d
	}

	if byID["compute01"].Profile == nil {
		t.Error("compute01 profile should be resolved")
	}
	if byID["compute01"].Profile.Model != "E8020-A" {
		t.Errorf("compute01 profile.model = %q, want E8020-A", byID["compute01"].Profile.Model)
	}
	if byID["switch01"].Profile == nil {
		t.Error("switch01 profile should be resolved")
	}
	if byID["switch01"].Profile.Kind != "switch" {
		t.Errorf("switch01 profile.kind = %q, want switch", byID["switch01"].Profile.Kind)
	}
}

func TestLoad_MissingProfile(t *testing.T) {
	dir := t.TempDir()
	infraPath := writeFile(t, dir, "infrastructure.yaml", `
version: 1
environment:
  id: test
  name: Test
  topology:
    racks: []
devices:
  - id: compute01
    role: compute
    profile: nonexistent-profile
    rack: rack01
`)
	profilePath := writeFile(t, dir, "profile.yaml", `
version: 1
profiles:
  some-other: {}
`)

	_, err := Load(infraPath, profilePath)
	if err == nil {
		t.Fatal("expected error for missing profile, got nil")
	}
	if !contains(err.Error(), "nonexistent-profile") {
		t.Errorf("error should mention missing profile name, got: %v", err)
	}
}

func TestLoad_MissingInfraFile(t *testing.T) {
	dir := t.TempDir()
	profilePath := writeFile(t, dir, "profile.yaml", profileYAML)

	_, err := Load(filepath.Join(dir, "nonexistent.yaml"), profilePath)
	if err == nil {
		t.Fatal("expected error for missing infra file")
	}
}

func TestLoad_MalformedInfraYAML(t *testing.T) {
	dir := t.TempDir()
	infraPath := writeFile(t, dir, "infrastructure.yaml", ":\tinvalid yaml :\t")
	profilePath := writeFile(t, dir, "profile.yaml", profileYAML)

	_, err := Load(infraPath, profilePath)
	if err == nil {
		t.Fatal("expected parse error for malformed YAML")
	}
}

func TestRoleToComponent(t *testing.T) {
	cases := []struct {
		role string
		want string
	}{
		{"compute", "compute"},
		{"COMPUTE", "compute"},
		{"storage", "storage"},
		{"network", "network"},
		{"management-network", "network"},
		{"controller", "control"},
		{"control", "control"},
		{"manager", "control"},
		{"unknown-role", "unknown"},
		{"", "unknown"},
	}

	for _, tc := range cases {
		got := RoleToComponent(tc.role)
		if got != tc.want {
			t.Errorf("RoleToComponent(%q) = %q, want %q", tc.role, got, tc.want)
		}
	}
}

func TestComponentAssignment(t *testing.T) {
	dir := t.TempDir()
	infraPath := writeFile(t, dir, "infrastructure.yaml", infraYAML)
	profilePath := writeFile(t, dir, "profile.yaml", profileYAML)

	infra, err := Load(infraPath, profilePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := map[string]string{
		"compute01":     "compute",
		"compute02":     "compute",
		"storage01":     "storage",
		"switch01":      "network",
		"controller01":  "control",
		"manager01":     "control",
		"mgmt-switch01": "network",
	}

	for _, d := range infra.Devices {
		if w, ok := want[d.ID]; ok && d.Component != w {
			t.Errorf("device %s component = %q, want %q", d.ID, d.Component, w)
		}
	}
}

func TestLoad_PUEDefault(t *testing.T) {
	dir := t.TempDir()
	infraPath := writeFile(t, dir, "infrastructure.yaml", infraYAML) // no pue field exists
	profilePath := writeFile(t, dir, "profile.yaml", profileYAML)

	infra, err := Load(infraPath, profilePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if infra.Environment.PUE != 1.0 {
		t.Errorf("PUE = %v, want 1.0 (default)", infra.Environment.PUE)
	}
}

func TestLoad_PUEExplicit(t *testing.T) {
	dir := t.TempDir()
	infraPath := writeFile(t, dir, "infrastructure.yaml", `
version: 1
environment:
  id: test-env
  name: Test
  pue: 1.4
devices:
  - id: compute01
    role: compute
    profile: compute-standard
    rack: rack01
`)
	profilePath := writeFile(t, dir, "profile.yaml", profileYAML)

	infra, err := Load(infraPath, profilePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if infra.Environment.PUE != 1.4 {
		t.Errorf("PUE = %v, want 1.4", infra.Environment.PUE)
	}
}

func TestParseDecimal(t *testing.T) {
	cases := []struct {
		input   string
		want    float64
		wantErr bool
	}{
		{"20803,838", 20803.838, false},
		{"20803.838", 20803.838, false},
		{"6,91E-01", 0.691, false},
		{"6.91E-01", 0.691, false},
		{"257957,649", 257957.649, false},
		{"0", 0, false},
		{"", 0, false},
		{"  42,5  ", 42.5, false},
		{"not-a-number", 0, true},
	}

	for _, tc := range cases {
		got, err := ParseDecimal(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseDecimal(%q): expected error, got nil", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseDecimal(%q): unexpected error: %v", tc.input, err)
			continue
		}
		// Allow small floating point tolerance
		if diff := got - tc.want; diff > 1e-6 || diff < -1e-6 {
			t.Errorf("ParseDecimal(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestEmbodiedImpactParsers(t *testing.T) {
	e := EmbodiedImpact{
		GWP:   ImpactValue{Value: "20803.838", Unit: "kg_co2eq"},
		ADP:   ImpactValue{Value: "0.691", Unit: "kg_sb_eq"},
		CED:   ImpactValue{Value: "257957.649", Unit: "mj"},
		Water: ImpactValue{Value: "8680.273", Unit: "m3"},
	}

	if v, err := e.ParseGWP(); err != nil || abs(v-20803.838) > 1e-3 {
		t.Errorf("ParseGWP() = %v, %v", v, err)
	}
	if v, err := e.ParseADP(); err != nil || abs(v-0.691) > 1e-3 {
		t.Errorf("ParseADP() = %v, %v", v, err)
	}
	if v, err := e.ParseCED(); err != nil || abs(v-257957.649) > 1e-3 {
		t.Errorf("ParseCED() = %v, %v", v, err)
	}
	if v, err := e.ParseWater(); err != nil || abs(v-8680.273) > 1e-3 {
		t.Errorf("ParseWater() = %v, %v", v, err)
	}
}

func TestEmbodiedImpact_EmptyValues(t *testing.T) {
	e := EmbodiedImpact{}
	for name, fn := range map[string]func() (float64, error){
		"GWP":   e.ParseGWP,
		"ADP":   e.ParseADP,
		"CED":   e.ParseCED,
		"Water": e.ParseWater,
	} {
		if v, err := fn(); err != nil || v != 0 {
			t.Errorf("%s on empty: got (%v, %v), want (0, nil)", name, v, err)
		}
	}
}

func TestProfileStructPopulation(t *testing.T) {
	dir := t.TempDir()
	infraPath := writeFile(t, dir, "infrastructure.yaml", infraYAML)
	profilePath := writeFile(t, dir, "profile.yaml", profileYAML)

	infra, err := Load(infraPath, profilePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	var compute *ResolvedDevice
	for i := range infra.Devices {
		if infra.Devices[i].ID == "compute01" {
			compute = &infra.Devices[i]
			break
		}
	}
	if compute == nil {
		t.Fatal("compute01 not found")
	}

	p := compute.Profile
	if p.DefaultLifespanYears != 4 {
		t.Errorf("lifespan = %d, want 4", p.DefaultLifespanYears)
	}
	if p.Hardware.CPU.Cores != 64 {
		t.Errorf("cpu.cores = %d, want 64", p.Hardware.CPU.Cores)
	}
	if p.Hardware.MemoryGB != 512 {
		t.Errorf("memory_gb = %v, want 512", p.Hardware.MemoryGB)
	}
	if len(p.Hardware.StorageDisks) != 1 {
		t.Errorf("storage disks = %d, want 1", len(p.Hardware.StorageDisks))
	}
	if p.Hardware.StorageDisks[0].Type != "sata_ssd" {
		t.Errorf("storage[0].type = %q, want sata_ssd", p.Hardware.StorageDisks[0].Type)
	}
	if p.EmbodiedImpact.GWP.Unit != "kg_co2eq" {
		t.Errorf("gwp unit = %q, want kg_co2eq", p.EmbodiedImpact.GWP.Unit)
	}
	if v, err := p.EmbodiedImpact.ParseGWP(); err != nil || abs(v-20803.838) > 1e-3 {
		t.Errorf("ParseGWP() = %v, %v", v, err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 &&
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
