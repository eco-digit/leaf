package intensity

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type mockMix struct {
	gwp float64
	adp float64
	ced float64
	wue float64
	err error
}

func (m *mockMix) FetchGWP(_ string) (float64, error) { return m.gwp, m.err }
func (m *mockMix) FetchADP(_ string) (float64, error) { return m.adp, m.err }
func (m *mockMix) FetchCED(_ string) (float64, error) { return m.ced, m.err }
func (m *mockMix) FetchWUE(_ string) (float64, error) { return m.wue, m.err }

type callCountMix struct {
	inner *mockMix
	fail  *atomic.Bool
}

func (c *callCountMix) FetchGWP(code string) (float64, error) {
	if c.fail.Load() {
		return 0, fmt.Errorf("mix unavailable")
	}
	return c.inner.FetchGWP(code)
}

func (c *callCountMix) FetchADP(code string) (float64, error) {
	if c.fail.Load() {
		return 0, fmt.Errorf("mix unavailable")
	}
	return c.inner.FetchADP(code)
}

func (c *callCountMix) FetchCED(code string) (float64, error) {
	if c.fail.Load() {
		return 0, fmt.Errorf("mix unavailable")
	}
	return c.inner.FetchCED(code)
}

func (c *callCountMix) FetchWUE(code string) (float64, error) {
	if c.fail.Load() {
		return 0, fmt.Errorf("mix unavailable")
	}
	return c.inner.FetchWUE(code)
}

func liveCfg(zone, country string) Config {
	return Config{TTL: time.Hour, Zone: zone, Country: country}
}

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

func TestMixReader_Germany(t *testing.T) {
	r, err := LoadMixData("../../docs/electricity_mixes.csv")
	if err != nil {
		t.Fatalf("LoadMixData: %v", err)
	}

	f, err := r.Lookup("DEU")
	if err != nil {
		t.Fatalf("Lookup DEU: %v", err)
	}
	if f.ADP != 0.00000008787 {
		t.Errorf("ADP: got %g, want 0.00000008787", f.ADP)
	}
	if f.CED != 8.7477 {
		t.Errorf("CED: got %g, want 8.7477", f.CED)
	}
	if f.WUE != 1.947 {
		t.Errorf("WUE: got %g, want 1.947", f.WUE)
	}
}

func TestMixReader_CaseInsensitive(t *testing.T) {
	r, err := LoadMixData("../../docs/electricity_mixes.csv")
	if err != nil {
		t.Fatalf("LoadMixData: %v", err)
	}
	if _, err := r.Lookup("deu"); err != nil {
		t.Errorf("lowercase lookup failed: %v", err)
	}
}

func TestMixReader_UnknownCountry(t *testing.T) {
	r, err := LoadMixData("../../docs/electricity_mixes.csv")
	if err != nil {
		t.Fatalf("LoadMixData: %v", err)
	}
	if _, err := r.Lookup("ZZZ"); err == nil {
		t.Fatal("expected error for unknown country code")
	}
}
