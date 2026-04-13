package server

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/OSBA-eco-digit/leaf/internal/model"
)

type sample struct {
	name     string
	labels   [][2]string
	value    float64
	tsMillis int64
}

// toSamples converts a ResultSet into prometheus sample.
func toSamples(rs model.ResultSet) []sample {
	out := make([]sample, 0, len(rs))
	for _, r := range rs {
		name := metricName(r)
		if name == "" {
			continue
		}
		out = append(out, sample{
			name:     name,
			labels:   metricLabels(r),
			value:    r.Value,
			tsMillis: r.Timestamp.UnixMilli(),
		})
	}
	return out
}

// metricName derives the prometheus metric name from an ImpactResult.
func metricName(r model.ImpactResult) string {
	subj := string(r.Subject)
	if r.Category == model.CategoryEnergy {
		return fmt.Sprintf("leaf_%s_energy_kwh", subj)
	}
	phase := string(r.ImpactPhase)
	suffix := categorySuffix(r.Category)
	if phase == "" || suffix == "" {
		return ""
	}
	return fmt.Sprintf("leaf_%s_%s_%s", subj, phase, suffix)
}

// categorySuffix maps a Category to the metric name suffix.
func categorySuffix(c model.Category) string {
	switch c {
	case model.CategoryGWP:
		return "gwp_kg"
	case model.CategoryADP:
		return "adp_kg_sb_eq"
	case model.CategoryCED:
		return "ced_mj"
	case model.CategoryWater:
		return "water_m3"
	default:
		return ""
	}
}

// metricLabels returns ordered label pairs for an ImpactResult.
func metricLabels(r model.ImpactResult) [][2]string {
	switch r.Subject {
	case model.SubjectDevice:
		return [][2]string{
			{"datacenter", r.Datacenter},
			{"component", r.Component},
			{"device", r.Device},
		}
	case model.SubjectProvider:
		return [][2]string{
			{"datacenter", r.Datacenter},
			{"component", r.Component},
		}
	case model.SubjectTenant:
		return [][2]string{
			{"project_id", r.ProjectID},
			{"project_name", r.ProjectName},
		}
	default:
		return nil
	}
}

// writeMetrics serialises a ResultSet to the prometheus text format.
func writeMetrics(w io.Writer, rs model.ResultSet) {
	samples := toSamples(rs)

	var order []string
	groups := make(map[string][]sample)
	for _, s := range samples {
		if _, seen := groups[s.name]; !seen {
			order = append(order, s.name)
		}
		groups[s.name] = append(groups[s.name], s)
	}
	sort.Strings(order)

	for _, name := range order {
		fmt.Fprintf(w, "# HELP %s %s\n", name, helpText(name))
		fmt.Fprintf(w, "# TYPE %s gauge\n", name)
		for _, s := range groups[name] {
			writeSample(w, s)
		}
	}
}

// writeSample writes one prometheus sample line.
func writeSample(w io.Writer, s sample) {
	if len(s.labels) == 0 {
		fmt.Fprintf(w, "%s %g %d\n", s.name, s.value, s.tsMillis)
		return
	}
	parts := make([]string, len(s.labels))
	for i, l := range s.labels {
		parts[i] = fmt.Sprintf(`%s=%q`, l[0], l[1])
	}
	fmt.Fprintf(w, "%s{%s} %g %d\n", s.name, strings.Join(parts, ","), s.value, s.tsMillis)
}

// helpText returns a human-readable description for a Leaf metric name.
func helpText(name string) string {
	switch {
	case strings.HasSuffix(name, "_energy_kwh"):
		return "Energy consumption (kWh per hour)"
	case strings.Contains(name, "_embodied_gwp_"):
		return "Embodied global warming potential, amortized (kg CO2eq per hour)"
	case strings.Contains(name, "_embodied_adp_"):
		return "Embodied abiotic depletion potential, amortized (kg Sb eq per hour)"
	case strings.Contains(name, "_embodied_ced_"):
		return "Embodied cumulative energy demand, amortized (MJ per hour)"
	case strings.Contains(name, "_embodied_water_"):
		return "Embodied water consumption, amortized (m3 per hour)"
	case strings.Contains(name, "_operational_gwp_"):
		return "Operational global warming potential (kg CO2eq per hour)"
	case strings.Contains(name, "_operational_adp_"):
		return "Operational abiotic depletion potential (kg Sb eq per hour)"
	case strings.Contains(name, "_operational_ced_"):
		return "Operational cumulative energy demand (MJ per hour)"
	case strings.Contains(name, "_total_gwp_"):
		return "Total global warming potential, operational + embodied (kg CO2eq per hour)"
	case strings.Contains(name, "_total_adp_"):
		return "Total abiotic depletion potential, operational + embodied (kg Sb eq per hour)"
	case strings.Contains(name, "_total_ced_"):
		return "Total cumulative energy demand, operational + embodied (MJ per hour)"
	case strings.Contains(name, "_total_water_"):
		return "Total water consumption (m3 per hour)"
	default:
		return "Leaf environmental impact metric"
	}
}
