// Package infrastructure parses infrastructure.yaml and profile.yaml.
package infrastructure

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// MetricSourceDef defines a single Prometheus query source.
type MetricSourceDef struct {
	Query         string `yaml:"query"`
	MatchLabel    string `yaml:"match_label"`
	MatchStrategy string `yaml:"match_strategy"`
	VMLabel       string `yaml:"vm_label"`
	Unit          string `yaml:"unit"`
}

type infraFile struct {
	Version     int         `yaml:"version"`
	Environment Environment `yaml:"environment"`
	Devices     []device    `yaml:"devices"`
}

// Environment describes the logical data-center environment.
type Environment struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

// device is the raw YAML representation of a device entry.
type device struct {
	ID      string `yaml:"id"`
	Role    string `yaml:"role"`
	Profile string `yaml:"profile"`
	Rack    string `yaml:"rack"`
}

type profileFile struct {
	Version  int                `yaml:"version"`
	Profiles map[string]Profile `yaml:"profiles"`
}

// Profile is the hardware specification and embodied-impact data for a device model.
type Profile struct {
	Kind                 string         `yaml:"kind"`
	Vendor               string         `yaml:"vendor"`
	Model                string         `yaml:"model"`
	DefaultLifespanYears int            `yaml:"default_lifespan_years"`
	Hardware             Hardware       `yaml:"hardware"`
	EmbodiedImpact       EmbodiedImpact `yaml:"embodied_impact"`
}

// Hardware holds the hardware specification of a device profile.
type Hardware struct {
	CPU               CPU           `yaml:"cpu"`
	MemoryGB          float64       `yaml:"memory_gb"`
	StorageDisks      []StorageDisk `yaml:"storage"`
	SwitchingCapacity string        `yaml:"switching_capacity"`
}

// CPU describes the processor configuration.
type CPU struct {
	Cores int `yaml:"cores"`
}

// StorageDisk is a single storage device in the hardware profile.
type StorageDisk struct {
	Type   string  `yaml:"type"`
	Count  int     `yaml:"count"`
	SizeGB float64 `yaml:"size_gb"`
	SizeTB float64 `yaml:"size_tb"`
}

// ImpactValue holds a numeric impact value and its unit as defined in profile.yaml.
// The value is stored as a string to safely handle varied decimal notations from
// LCA source data. Use ParseDecimal to obtain a float64.
type ImpactValue struct {
	Value string `yaml:"value"`
	Unit  string `yaml:"unit"`
}

// EmbodiedImpact holds the total manufacturing impact over the device lifespan.
// All four categories share the same {value, unit} structure in profile.yaml.
type EmbodiedImpact struct {
	GWP   ImpactValue `yaml:"gwp"`
	ADP   ImpactValue `yaml:"adp"`
	CED   ImpactValue `yaml:"ced"`
	Water ImpactValue `yaml:"water"`
}

// ParseGWP returns the GWP value as float64 (kg CO₂eq).
func (e EmbodiedImpact) ParseGWP() (float64, error) {
	return ParseDecimal(e.GWP.Value)
}

// ParseADP returns the ADP value as float64 (kg Sb eq).
func (e EmbodiedImpact) ParseADP() (float64, error) {
	return ParseDecimal(e.ADP.Value)
}

// ParseCED returns the CED value as float64 (MJ).
func (e EmbodiedImpact) ParseCED() (float64, error) {
	return ParseDecimal(e.CED.Value)
}

// ParseWater returns the water value as float64 (m³).
func (e EmbodiedImpact) ParseWater() (float64, error) {
	return ParseDecimal(e.Water.Value)
}

// ParseDecimal parses the result as float64 "20803,838" and "20803.838" are accepted
func ParseDecimal(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	s = strings.ReplaceAll(s, ",", ".")
	return strconv.ParseFloat(s, 64)
}

// ResolvedDevice is a device entry with its profile attached and its role
// mapped to one of Leaf's component categories.
type ResolvedDevice struct {
	ID        string
	Role      string
	Component string // compute | storage | network | control
	Rack      string
	Profile   *Profile // nil when the referenced profile is not defined in profile.yaml
}

// Infrastructure is the fully resolved representation of infrastructure.yaml and profile.yaml.
type Infrastructure struct {
	Environment Environment
	Devices     []ResolvedDevice
}

// RoleToComponent maps a device role string to one of Leaf's four component ategories. Unknown roles return unknown.
func RoleToComponent(role string) string {
	switch strings.ToLower(role) {
	case "compute":
		return "compute"
	case "storage":
		return "storage"
	case "network", "management-network":
		return "network"
	case "controller", "control", "manager":
		return "control"
	default:
		return "unknown"
	}
}

// Load parses infraPath (infrastructure.yaml) and profilePath (profile.yaml),
// resolves every device to its profile, and returns the typed Infrastructure.
//
// Error gets returned when:
//   - either file cannot be read or parsed
//   - one or more device profile references have no matching entry in profile.yaml
func Load(infraPath, profilePath string) (*Infrastructure, error) {
	infra, err := loadInfraFile(infraPath)
	if err != nil {
		return nil, fmt.Errorf("infrastructure: %w", err)
	}

	profiles, err := loadProfileFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("profiles: %w", err)
	}

	resolved, err := resolveDevices(infra.Devices, profiles)
	if err != nil {
		return nil, err
	}

	return &Infrastructure{
		Environment: infra.Environment,
		Devices:     resolved,
	}, nil
}

// --- Internal helpers ---

func loadInfraFile(path string) (*infraFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var f infraFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &f, nil
}

func loadProfileFile(path string) (map[string]Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var f profileFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return f.Profiles, nil
}

func resolveDevices(devices []device, profiles map[string]Profile) ([]ResolvedDevice, error) {
	var missing []string
	resolved := make([]ResolvedDevice, 0, len(devices))

	for _, d := range devices {
		rd := ResolvedDevice{
			ID:        d.ID,
			Role:      d.Role,
			Component: RoleToComponent(d.Role),
			Rack:      d.Rack,
		}

		if p, ok := profiles[d.Profile]; ok {
			pCopy := p
			rd.Profile = &pCopy
		} else {
			missing = append(missing, fmt.Sprintf("%s (device %s)", d.Profile, d.ID))
		}

		resolved = append(resolved, rd)
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("unresolved profiles: [%s]", strings.Join(missing, ", "))
	}

	return resolved, nil
}
