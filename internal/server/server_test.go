package server_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/OSBA-eco-digit/leaf/internal/cache"
	"github.com/OSBA-eco-digit/leaf/internal/model"
	"github.com/OSBA-eco-digit/leaf/internal/server"
)

var fixedTS = time.Date(2026, 4, 10, 8, 0, 0, 0, time.UTC)

// seedCache returns a cache.
func seedCache(rs model.ResultSet) *cache.Cache {
	c := cache.New()
	c.Update(rs)
	return c
}

// get performs GET against h at path and returns the response body.
func get(t *testing.T, h http.Handler, path string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	body, _ := io.ReadAll(rec.Body)
	return rec.Code, string(body)
}

func TestMetricsHandler_ContentType(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	server.New(cache.New(), "").Handler().ServeHTTP(rec, req)

	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain prefix", ct)
	}
}

func TestMetricsHandler_EmptyCache(t *testing.T) {
	code, body := get(t, server.New(cache.New(), "").Handler(), "/metrics")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	if strings.Contains(body, "leaf_") {
		t.Errorf("expected no metrics for empty cache:\n%s", body)
	}
}

func TestMetricsHandler_AllCategories(t *testing.T) {
	cases := []struct {
		cat      model.Category
		phase    model.ImpactPhase
		wantName string
	}{
		{model.CategoryGWP, model.PhaseEmbodied, "leaf_provider_embodied_gwp_kg"},
		{model.CategoryADP, model.PhaseEmbodied, "leaf_provider_embodied_adp_kg_sb_eq"},
		{model.CategoryCED, model.PhaseEmbodied, "leaf_provider_embodied_ced_mj"},
		{model.CategoryWater, model.PhaseEmbodied, "leaf_provider_embodied_water_m3"},
		{model.CategoryEnergy, model.PhaseOperational, "leaf_provider_energy_kwh"},
	}
	for _, tc := range cases {
		t.Run(string(tc.cat), func(t *testing.T) {
			rs := model.ResultSet{{
				Subject:     model.SubjectProvider,
				Datacenter:  "dc1",
				Component:   "compute",
				ImpactPhase: tc.phase,
				Category:    tc.cat,
				Value:       1.0,
				Timestamp:   fixedTS,
			}}
			_, body := get(t, server.New(seedCache(rs), "").Handler(), "/metrics")
			if !strings.Contains(body, tc.wantName) {
				t.Errorf("expected %s in:\n%s", tc.wantName, body)
			}
		})
	}
}

func TestMetricsHandler_TenantLabels(t *testing.T) {
	rs := model.ResultSet{{
		Subject:     model.SubjectTenant,
		ImpactPhase: model.PhaseOperational,
		Category:    model.CategoryGWP,
		ProjectID:   "proj-abc",
		ProjectName: "some-openstack-project",
		Value:       0.25,
		Timestamp:   fixedTS,
	}}
	_, body := get(t, server.New(seedCache(rs), "").Handler(), "/metrics")

	wants := []string{
		"leaf_tenant_operational_gwp_kg",
		`project_id="proj-abc"`,
		`project_name="some-openstack-project"`,
	}
	for _, w := range wants {
		if !strings.Contains(body, w) {
			t.Errorf("missing %q in:\n%s", w, body)
		}
	}
}

func TestHealthHandler_EmptyCache(t *testing.T) {
	code, body := get(t, server.New(cache.New(), "").Handler(), "/health")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("unmarshal health response: %v", err)
	}
	if resp["status"] != "no_data" {
		t.Errorf("status = %v, want no_data", resp["status"])
	}
	if resp["cache_empty"] != true {
		t.Errorf("cache_empty = %v, want true", resp["cache_empty"])
	}
}

func TestHealthHandler_PopulatedCache(t *testing.T) {
	rs := model.ResultSet{{
		Subject:     model.SubjectProvider,
		Datacenter:  "dc1",
		Component:   "total",
		ImpactPhase: model.PhaseEmbodied,
		Category:    model.CategoryGWP,
		Value:       1.0,
		Timestamp:   fixedTS,
	}}
	code, body := get(t, server.New(seedCache(rs), "").Handler(), "/health")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("unmarshal health response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("status = %v, want ok", resp["status"])
	}
	if resp["cache_empty"] != false {
		t.Errorf("cache_empty = %v, want false", resp["cache_empty"])
	}
}
