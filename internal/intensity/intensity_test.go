package intensity

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func staticCfg() Config {
	return Config{
		TTL:       time.Hour,
		GWPStatic: 400.0,
		ADPStatic: 1.5,
		CEDStatic: 8.0,
	}
}

func liveCfg(zone, country string) Config {
	return Config{TTL: time.Hour, Zone: zone, Country: country}
}

func liveCfgWithFallback(zone, country string) Config {
	return Config{
		TTL:       time.Hour,
		Zone:      zone,
		Country:   country,
		GWPStatic: 350.0,
		ADPStatic: 1.2,
		CEDStatic: 7.5,
	}
}

// TestProvider_StaticSource test
func TestProvider_StaticSource(t *testing.T) {
	p := NewProvider(staticCfg(), nil, nil)

	factors, err := p.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if factors.GWP.Value != 400.0 {
		t.Errorf("GWP: got %g, want 400.0", factors.GWP.Value)
	}
	if factors.GWP.Source != SourceStatic {
		t.Errorf("GWP source: got %q, want %q", factors.GWP.Source, SourceStatic)
	}
	if factors.ADP.Value != 1.5 {
		t.Errorf("ADP: got %g, want 1.5", factors.ADP.Value)
	}
	if factors.CED.Value != 8.0 {
		t.Errorf("CED: got %g, want 8.0", factors.CED.Value)
	}
}

func TestProvider_TTLCaching(t *testing.T) {
	var callCount atomic.Int32

	emSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		json.NewEncoder(w).Encode(electricityMapsResponse{Zone: "DE", CarbonIntensity: 300.0})
	}))
	defer emSrv.Close()

	bSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(boaviztaCountryResponse{ADPFactor: 1.0, PEFactor: 8.5})
	}))
	defer bSrv.Close()

	p := NewProvider(liveCfg("DE", "DE"), NewElectricityMapsClient("testkey", emSrv.URL), NewBoaviztaClient(bSrv.URL))

	f1, err := p.Fetch()
	if err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	if f1.GWP.Value != 300.0 {
		t.Errorf("GWP: got %g, want 300.0", f1.GWP.Value)
	}

	// Second call within TTL must not hit the API again.
	if _, err = p.Fetch(); err != nil {
		t.Fatalf("second fetch: %v", err)
	}
	if callCount.Load() != 1 {
		t.Errorf("expected 1 API call, got %d", callCount.Load())
	}
}
