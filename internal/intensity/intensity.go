// Package intensity fetches operational intensity factors (GWP, ADP, CED) from
// external APIs and caches them so we don't query the APIs on every calculation cycle.
package intensity

import (
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
	GWP FactorValue // g CO2eq/kWh
	ADP FactorValue // kg Sb eq/kWh
	CED FactorValue // MJ/kWh
}

// Config holds the runtime settings for the Provider, populated from config.yaml.
// Static fallback fields are optional — zero means no static fallback for that factor.
type Config struct {
	TTL     time.Duration
	Zone    string // electricity grid zone, e.g. "DE"
	Country string // ISO country code for Boavizta, e.g. "DE"
	// Optional static fallback values used when the API is unavailable and no cached value exists.
	GWPStatic float64 // g CO2eq/kWh
	ADPStatic float64 // kg Sb eq/kWh
	CEDStatic float64 // MJ/kWh
}

// GWPFetcher is satisfied by ElectricityMapsClient.
type GWPFetcher interface {
	FetchGWP(zone string) (float64, error)
}

// BoaviztaFetcher is satisfied by BoaviztaClient.
type BoaviztaFetcher interface {
	FetchADP(countryCode string) (float64, error)
	FetchCED(countryCode string) (float64, error)
}

// Provider gets intensity factors and caches them for the configured TTL.
// When a live fetch fails it falls back to the last known cached value,
// then to the static fallback if one is configured.
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
