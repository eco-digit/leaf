// Package calculator computes provider ImpactResults.
package calculator

import (
	"time"

	"github.com/OSBA-eco-digit/leaf/internal/collector"
	"github.com/OSBA-eco-digit/leaf/internal/infrastructure"
	"github.com/OSBA-eco-digit/leaf/internal/model"
	"github.com/OSBA-eco-digit/leaf/research/assets"
)

// TODO orchestrator.reporting_interval: "1h"

// deviceEnergyKWh returns the energie in kWh for a single device
// over the reporting.interval.
func deviceEnergyKWh(d *collector.DeviceRaw) (kwg float64, keplerFallback bool) {
	if bmc, ok := d.Metrics["bmc"]; ok && bmc > 0 {
		return bmc * assets.windowHours / 1000.0, false
	}

	// TODO kepler fallback
	return kwg, false
}

// deviceEnergyResults creates a energy ImpatResult per device.
func deviceEnergyResults(
	raw *collector.RawMetrics,
	infra *infrastructure.Infrastructure,
	ts time.Time,
) (model.ResultSet, []string) {
	var rs model.ResultSet
	var warnings []string

	return rs, warnings
}
