// Package orchestrator schedules and executes calculation cycles.
package orchestrator

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/eco-digit/leaf/internal/cache"
	"github.com/eco-digit/leaf/internal/calculator"
	"github.com/eco-digit/leaf/internal/collector"
	"github.com/eco-digit/leaf/internal/infrastructure"
	"github.com/eco-digit/leaf/internal/intensity"
	"github.com/eco-digit/leaf/internal/model"
	"github.com/eco-digit/leaf/internal/promclient"
)

// Orchestrator runs the Collect -> Calculate -> Cache cycle per each ticker.
type Orchestrator struct {
	infra      *infrastructure.Infrastructure
	embodiedRS model.ResultSet
	cache      *cache.Cache
	intensity  *intensity.Provider
	querier    collector.Querier
	window     string
	interval   time.Duration
}

// CycleResult holds the output of a single calculation cycle.
type CycleResult struct {
	RS       model.ResultSet
	Factors  intensity.IntensityFactors
	Warnings []string
}

// New constructs an Orchestrator from config. Builds the Prometheus client and
// intensity provider internally so callers only need the domain objects.
func New(
	promURL, promUser, promPass string,
	intensityCfg intensity.Config,
	emapAPIKey, emapBaseURL string,
	mixDataPath string,
	interval time.Duration,
	infra *infrastructure.Infrastructure,
	embodiedRS model.ResultSet,
	c *cache.Cache,
) (*Orchestrator, error) {
	q, err := promclient.NewClient(promURL, promUser, promPass)
	if err != nil {
		return nil, fmt.Errorf("orchestrator: prometheus client: %w", err)
	}

	var gwpClient intensity.GWPFetcher
	if emapAPIKey != "" {
		gwpClient = intensity.NewElectricityMapsClient(emapAPIKey, emapBaseURL)
	}

	var mixClient intensity.StaticFetcher
	if mix, err := intensity.LoadMixData(mixDataPath); err != nil {
		log.Printf("orchestrator: intensity mix data unavailable (%v) — ADP/CED will be zero", err)
	} else {
		mixClient = mix
	}

	ip := intensity.NewProvider(intensityCfg, gwpClient, mixClient)

	window := interval.String()

	return &Orchestrator{
		infra:      infra,
		embodiedRS: embodiedRS,
		cache:      c,
		intensity:  ip,
		querier:    q,
		window:     window,
		interval:   interval,
	}, nil
}

// RunCycle executes one full collection and calculation cycle and updates the cache.
// Returns the full ResultSet written to cache, the intensity factors used, and any warnings.
func (o *Orchestrator) RunCycle() (CycleResult, error) {
	ts := time.Now().Truncate(o.interval)

	raw, err := collector.Collect(o.querier, o.infra, o.window, ts)
	if err != nil {
		return CycleResult{}, fmt.Errorf("collect: %w", err)
	}

	factors, err := o.intensity.Fetch()
	if err != nil {
		log.Printf("orchestrator: intensity partial failure: %v", err)
	}

	providerRS, warnings := calculator.ProviderResults(raw, o.infra, factors, o.embodiedRS, ts)
	warnings = append(raw.Warnings, warnings...)

	full := make(model.ResultSet, 0, len(o.embodiedRS)+len(providerRS))
	full = append(full, o.embodiedRS...)
	full = append(full, providerRS...)

	o.cache.Update(full)

	return CycleResult{RS: full, Factors: factors, Warnings: warnings}, nil
}

// Start runs RunCycle on the configured interval until ctx is cancelled.
// The first cycle is executed immediately
// subsequent ones fire on ticker.
func (o *Orchestrator) Start(ctx context.Context) {
	log.Printf("orchestrator: starting, interval=%s", o.interval)

	if _, err := o.RunCycle(); err != nil {
		log.Printf("orchestrator: initial cycle failed: %v", err)
	}

	ticker := time.NewTicker(o.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if _, err := o.RunCycle(); err != nil {
				log.Printf("orchestrator: cycle failed, serving last good result: %v", err)
			}
		case <-ctx.Done():
			log.Printf("orchestrator: stopping ...")
			return
		}
	}
}
