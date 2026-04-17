// Package intensity fetches operational intensity factors (GWP, ADP, CED, WUE) from
// external APIs and caches them.
package intensity

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type Source string

const (
	SourceElectricityMaps Source = "electricity_maps"
	SourceMixData         Source = "mix_data"
)

type FactorValue struct {
	Value  float64
	Unit   string
	Source Source
}

// IntensityFactors is what the calculator needs.
type IntensityFactors struct {
	GWP FactorValue
	ADP FactorValue
	CED FactorValue
	WUE FactorValue
}

// Config holds the runtime settings for the Provider, populated from config.yaml.
type Config struct {
	TTL     time.Duration
	Zone    string
	Country string
}

type GWPFetcher interface {
	FetchGWP(zone string) (float64, error)
}

type StaticFetcher interface {
	FetchGWP(countryCode string) (float64, error)
	FetchADP(countryCode string) (float64, error)
	FetchCED(countryCode string) (float64, error)
	FetchWUE(countryCode string) (float64, error)
}

// Provider gets intensity factors and caches them for the configured TTL.
// When a live fetch fails it falls back to the last known cached value.
type Provider struct {
	cfg       Config
	gwp       GWPFetcher
	mix       StaticFetcher
	mu        sync.RWMutex
	cached    *IntensityFactors
	fetchedAt time.Time
}

// NewProvider wires up a Provider.
func NewProvider(cfg Config, gwp GWPFetcher, mix StaticFetcher) *Provider {
	return &Provider{cfg: cfg, gwp: gwp, mix: mix}
}

// Fetch returns the current intensity factors.
func (p *Provider) Fetch() (IntensityFactors, error) {
	p.mu.RLock()
	if p.cached != nil && time.Since(p.fetchedAt) < p.cfg.TTL {
		f := *p.cached
		p.mu.RUnlock()
		return f, nil
	}
	p.mu.RUnlock()

	return p.refresh()
}

func (p *Provider) refresh() (IntensityFactors, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cached != nil && time.Since(p.fetchedAt) < p.cfg.TTL {
		return *p.cached, nil
	}

	var errs []error
	factors := IntensityFactors{}

	gwpVal, err := p.resolveGWP()
	if err != nil {
		errs = append(errs, err)
	}
	factors.GWP = gwpVal

	adpVal, err := p.resolveADP()
	if err != nil {
		errs = append(errs, err)
	}
	factors.ADP = adpVal

	cedVal, err := p.resolveCED()
	if err != nil {
		errs = append(errs, err)
	}
	factors.CED = cedVal

	wueVal, err := p.resolveWUE()
	if err != nil {
		errs = append(errs, err)
	}
	factors.WUE = wueVal

	// Cache even on partial failure so the TTL window still applies.
	p.cached = &factors
	p.fetchedAt = time.Now()

	if len(errs) > 0 {
		return factors, fmt.Errorf("intensity refresh: %v", errs)
	}
	return factors, nil
}

func (p *Provider) resolveGWP() (FactorValue, error) {
	if p.gwp != nil {
		v, err := p.gwp.FetchGWP(p.cfg.Zone)
		if err == nil {
			return FactorValue{Value: v, Unit: "g_co2eq_per_kwh", Source: SourceElectricityMaps}, nil
		}
		log.Printf("intensity: GWP live fetch failed (%v) — falling back to CSV", err)
	}
	// Fall back to the CSV value, which is a static but reasonable approximation.
	if p.mix != nil {
		v, err := p.mix.FetchGWP(p.cfg.Country)
		if err == nil {
			return FactorValue{Value: v * 1000, Unit: "g_co2eq_per_kwh", Source: SourceMixData}, nil
		}
	}
	return p.cachedOrError(func(f IntensityFactors) FactorValue { return f.GWP }, "gwp")
}

func (p *Provider) resolveADP() (FactorValue, error) {
	if p.mix != nil {
		v, err := p.mix.FetchADP(p.cfg.Country)
		if err == nil {
			return FactorValue{Value: v, Unit: "kg_sb_eq_per_kwh", Source: SourceMixData}, nil
		}
	}
	return p.cachedOrError(func(f IntensityFactors) FactorValue { return f.ADP }, "adp")
}

func (p *Provider) resolveCED() (FactorValue, error) {
	if p.mix != nil {
		v, err := p.mix.FetchCED(p.cfg.Country)
		if err == nil {
			return FactorValue{Value: v, Unit: "mj_per_kwh", Source: SourceMixData}, nil
		}
	}
	return p.cachedOrError(func(f IntensityFactors) FactorValue { return f.CED }, "ced")
}

func (p *Provider) resolveWUE() (FactorValue, error) {
	if p.mix != nil {
		v, err := p.mix.FetchWUE(p.cfg.Country)
		if err == nil {
			return FactorValue{Value: v, Unit: "liters_per_kwh", Source: SourceMixData}, nil
		}
	}
	return p.cachedOrError(func(f IntensityFactors) FactorValue { return f.WUE }, "wue")
}

// cachedOrError returns the last cached value if one exists, otherwise an error.
func (p *Provider) cachedOrError(get func(IntensityFactors) FactorValue, name string) (FactorValue, error) {
	if p.cached != nil {
		if fv := get(*p.cached); fv.Value != 0 {
			return fv, nil
		}
	}
	return FactorValue{}, fmt.Errorf("%s: no data available", name)
}
