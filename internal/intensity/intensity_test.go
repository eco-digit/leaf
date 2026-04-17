package intensity

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

// Electricity tests
func TestElectricityMapsClient_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("auth-token") != "mykey" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.URL.Query().Get("zone") != "DE" {
			http.Error(w, "bad zone", http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(electricityMapsResponse{Zone: "DE", CarbonIntensity: 350.5})
	}))
	defer srv.Close()

	v, err := NewElectricityMapsClient("mykey", srv.URL).FetchGWP("DE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 350.5 {
		t.Errorf("got %g, want 350.5", v)
	}
}
