// Package intensity fetches operational intensity factors (GWP, ADP, CED) from
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
	SourceBoavizta        Source = "boavizta"
	SourceStatic          Source = "static"
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
}

// Config holds the runtime settings for the Provider, populated from config.yaml.
// Static fallback fields are optional — zero means no static fallback for that factor.
type Config struct {
	TTL     time.Duration
	Zone    string
	Country string
	// Optional static fallback values used when the API is unavailable and no cached value exists.
	GWPStatic float64
	ADPStatic float64
	CEDStatic float64
}

// GWPFetcher, Electricity maps factor
type GWPFetcher interface {
	FetchGWP(zone string) (float64, error)
}

// BoaviztaFetcher, Boavizta API for now
type BoaviztaFetcher interface {
	FetchADP(countryCode string) (float64, error)
	FetchCED(countryCode string) (float64, error)
}

// Provider gets intensity factors and caches them for the configured TTL.
type Provider struct {
	cfg      Config
	gwp      GWPFetcher
	boavizta BoaviztaFetcher

	mu        sync.RWMutex
	cached    *IntensityFactors
	fetchedAt time.Time
}

// NewProvider wires up a Provider. Pass nil for gwp or boavizta
// - fall back to cached or static values.
func NewProvider(cfg Config, gwp GWPFetcher, boavizta BoaviztaFetcher) *Provider {
	return &Provider{cfg: cfg, gwp: gwp, boavizta: boavizta}
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
		log.Printf("intensity: GWP fetch failed (%v) — using fallback", err)
		fv, fbErr := p.fallback(func(f IntensityFactors) FactorValue { return f.GWP }, p.cfg.GWPStatic, "g_co2eq_per_kwh")
		if fbErr != nil {
			return fv, fmt.Errorf("gwp: fetch: %w; fallback: %v", err, fbErr)
		}
		return fv, fmt.Errorf("gwp: %w", err)
	}
	return p.fallback(func(f IntensityFactors) FactorValue { return f.GWP }, p.cfg.GWPStatic, "g_co2eq_per_kwh")
}

func (p *Provider) resolveADP() (FactorValue, error) {
	if p.boavizta != nil {
		v, err := p.boavizta.FetchADP(p.cfg.Country)
		if err == nil {
			return FactorValue{Value: v, Unit: "kg_sb_eq_per_kwh", Source: SourceBoavizta}, nil
		}
		log.Printf("intensity: ADP fetch failed (%v) — using fallback", err)
		fv, fbErr := p.fallback(func(f IntensityFactors) FactorValue { return f.ADP }, p.cfg.ADPStatic, "kg_sb_eq_per_kwh")
		if fbErr != nil {
			return fv, fmt.Errorf("adp: fetch: %w; fallback: %v", err, fbErr)
		}
		return fv, fmt.Errorf("adp: %w", err)
	}
	return p.fallback(func(f IntensityFactors) FactorValue { return f.ADP }, p.cfg.ADPStatic, "kg_sb_eq_per_kwh")
}

func (p *Provider) resolveCED() (FactorValue, error) {
	if p.boavizta != nil {
		v, err := p.boavizta.FetchCED(p.cfg.Country)
		if err == nil {
			return FactorValue{Value: v, Unit: "mj_per_kwh", Source: SourceBoavizta}, nil
		}
		log.Printf("intensity: CED fetch failed (%v) — using fallback", err)
		fv, fbErr := p.fallback(func(f IntensityFactors) FactorValue { return f.CED }, p.cfg.CEDStatic, "mj_per_kwh")
		if fbErr != nil {
			return fv, fmt.Errorf("ced: fetch: %w; fallback: %v", err, fbErr)
		}
		return fv, fmt.Errorf("ced: %w", err)
	}
	return p.fallback(func(f IntensityFactors) FactorValue { return f.CED }, p.cfg.CEDStatic, "mj_per_kwh")
}

// fallback tries the last cached value first, then the configured static value.
// Returns error if we have nothing to fall back to.
func (p *Provider) fallback(get func(IntensityFactors) FactorValue, staticVal float64, unit string) (FactorValue, error) {
	if p.cached != nil {
		fv := get(*p.cached)
		if fv.Value != 0 {
			return fv, nil
		}
	}
	if staticVal != 0 {
		return FactorValue{Value: staticVal, Unit: unit, Source: SourceStatic}, nil
	}
	return FactorValue{}, fmt.Errorf("no cached or static value available")
}
