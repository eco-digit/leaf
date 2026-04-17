package collector_test

import (
	"testing"
	"time"

	prommodel "github.com/prometheus/common/model"

	"github.com/OSBA-eco-digit/leaf/internal/collector"
	"github.com/OSBA-eco-digit/leaf/internal/infrastructure"
)

// mockQuerier is a test double for collector.Querier.
type mockQuerier struct {
	responses map[string]prommodel.Value
	errs      map[string]error
}

func (m *mockQuerier) QueryMetric(query string) (prommodel.Value, error) {
	if err, ok := m.errs[query]; ok {
		return nil, err
	}
	if v, ok := m.responses[query]; ok {
		return v, nil
	}
	return prommodel.Vector{}, nil
}

// smpl creates a single Prometheus sample.
func smpl(labels prommodel.LabelSet, value float64) *prommodel.Sample {
	return &prommodel.Sample{
		Metric:    prommodel.Metric(labels),
		Value:     prommodel.SampleValue(value),
		Timestamp: prommodel.Now(),
	}
}

// vec builds a prommodel.Vector from samples.
func vec(samples ...*prommodel.Sample) prommodel.Vector {
	out := make(prommodel.Vector, len(samples))
	copy(out, samples)
	return out
}

// infraWithSources builds a minimal Infrastructure with two compute devices and
// a provided metric_sources map.
func infraWithSources(sources map[string]infrastructure.MetricSourceDef) *infrastructure.Infrastructure {
	return &infrastructure.Infrastructure{
		Environment: infrastructure.Environment{ID: "dc1", Name: "Test DC"},
		Devices: []infrastructure.ResolvedDevice{
			{ID: "compute01", Role: "compute", Component: "compute"},
			{ID: "compute02", Role: "compute", Component: "compute"},
		},
		MetricSources: sources,
	}
}

const testWindow = "1h"

// keplerIdleSrc mirrors the kepler_node_idle entry in infrastructure.yaml.
var keplerIdleSrc = infrastructure.MetricSourceDef{
	Query:      `increase(kepler_node_cpu_idle_joules_total{zone="package"}[{{ .Window }}])`,
	MatchLabel: "instance",
	Unit:       "joules",
}

// keplerActiveSrc mirrors the kepler_node_active entry in infrastructure.yaml.
var keplerActiveSrc = infrastructure.MetricSourceDef{
	Query:      `increase(kepler_node_cpu_active_joules_total{zone="package"}[{{ .Window }}])`,
	MatchLabel: "instance",
	Unit:       "joules",
}

// keplerVMSrc mirrors the kepler_vm entry in infrastructure.yaml.
var keplerVMSrc = infrastructure.MetricSourceDef{
	Query:      `increase(kepler_vm_cpu_joules_total{zone="package"}[{{ .Window }}])`,
	MatchLabel: "instance",
	VMLabel:    "vm_id",
	Unit:       "joules",
}

// bmcSrc mirrors the bmc entry in infrastructure.yaml.
var bmcSrc = infrastructure.MetricSourceDef{
	Query:         `avg_over_time(reading_watts[{{ .Window }}])`,
	MatchLabel:    "instance",
	MatchStrategy: "hostname_extract",
	Unit:          "watts",
}

func TestCollect_KeplerIdle(t *testing.T) {
	rendered := `increase(kepler_node_cpu_idle_joules_total{zone="package"}[1h])`

	q := &mockQuerier{responses: map[string]prommodel.Value{
		rendered: vec(
			smpl(prommodel.LabelSet{"instance": "compute01"}, 3600.0),
			smpl(prommodel.LabelSet{"instance": "compute02"}, 7200.0),
		),
	}}

	raw, err := collector.Collect(q, infraWithSources(map[string]infrastructure.MetricSourceDef{
		"kepler_node_idle": keplerIdleSrc,
	}), testWindow, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raw.Devices["compute01"].Metrics["kepler_node_idle"] != 3600.0 {
		t.Errorf("compute01 idle = %.1f, want 3600.0", raw.Devices["compute01"].Metrics["kepler_node_idle"])
	}
	if raw.Devices["compute02"].Metrics["kepler_node_idle"] != 7200.0 {
		t.Errorf("compute02 idle = %.1f, want 7200.0", raw.Devices["compute02"].Metrics["kepler_node_idle"])
	}
}

func TestCollect_KeplerActive(t *testing.T) {
	rendered := `increase(kepler_node_cpu_active_joules_total{zone="package"}[1h])`

	q := &mockQuerier{responses: map[string]prommodel.Value{
		rendered: vec(smpl(prommodel.LabelSet{"instance": "compute01"}, 1800.0)),
	}}

	raw, _ := collector.Collect(q, infraWithSources(map[string]infrastructure.MetricSourceDef{
		"kepler_node_active": keplerActiveSrc,
	}), testWindow, time.Now())

	if raw.Devices["compute01"].Metrics["kepler_node_active"] != 1800.0 {
		t.Errorf("active = %.1f, want 1800.0", raw.Devices["compute01"].Metrics["kepler_node_active"])
	}
}

func TestCollect_BMC_RedfishURL(t *testing.T) {
	infra := &infrastructure.Infrastructure{
		Environment:   infrastructure.Environment{ID: "dc1"},
		Devices:       []infrastructure.ResolvedDevice{{ID: "compute01", Role: "compute", Component: "compute"}},
		MetricSources: map[string]infrastructure.MetricSourceDef{"bmc": bmcSrc},
	}
	rendered := `avg_over_time(reading_watts[1h])`

	q := &mockQuerier{responses: map[string]prommodel.Value{
		rendered: vec(smpl(prommodel.LabelSet{
			"instance": "https://compute01.bmc.scs1.ber3.int.yco.de/redfish/v1/Chassis/Self/Power",
		}, 250.5)),
	}}

	raw, err := collector.Collect(q, infra, testWindow, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raw.Devices["compute01"].Metrics["bmc"] != 250.5 {
		t.Errorf("BMC watts = %.2f, want 250.5", raw.Devices["compute01"].Metrics["bmc"])
	}
}

func TestCollect_KeplerVM(t *testing.T) {
	rendered := `increase(kepler_vm_cpu_joules_total{zone="package"}[1h])`

	q := &mockQuerier{responses: map[string]prommodel.Value{
		rendered: vec(
			smpl(prommodel.LabelSet{"instance": "compute01", "vm_id": "instance-000001"}, 900.0),
			smpl(prommodel.LabelSet{"instance": "compute01", "vm_id": "instance-000002"}, 450.0),
			smpl(prommodel.LabelSet{"instance": "compute02", "vm_id": "instance-000003"}, 600.0),
		),
	}}

	raw, err := collector.Collect(q, infraWithSources(map[string]infrastructure.MetricSourceDef{
		"kepler_vm": keplerVMSrc,
	}), testWindow, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raw.Devices["compute01"].VMMetrics["kepler_vm"]["instance-000001"] != 900.0 {
		t.Errorf("vm1 joules = %.1f, want 900.0", raw.Devices["compute01"].VMMetrics["kepler_vm"]["instance-000001"])
	}
	if raw.Devices["compute01"].VMMetrics["kepler_vm"]["instance-000002"] != 450.0 {
		t.Errorf("vm2 joules = %.1f, want 450.0", raw.Devices["compute01"].VMMetrics["kepler_vm"]["instance-000002"])
	}
	if raw.Devices["compute02"].VMMetrics["kepler_vm"]["instance-000003"] != 600.0 {
		t.Errorf("vm3 joules = %.1f, want 600.0", raw.Devices["compute02"].VMMetrics["kepler_vm"]["instance-000003"])
	}
}
