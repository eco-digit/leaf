package intensity

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ElectricityMapsClient fetches GWP carbon intensity from the Electricity Maps API.
// GET /v3/carbon-intensity/latest?zone={zone}
// https://api.electricitymaps.com/v3/carbon-intensity/past?zone=DE&datetime=2026-04-15+19%3A42
type ElectricityMapsClient struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

func NewElectricityMapsClient(apiKey, baseURL string) *ElectricityMapsClient {
	return &ElectricityMapsClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

type electricityMapsResponse struct {
	Zone            string  `json:"zone"`
	CarbonIntensity float64 `json:"carbonIntensity"` // g CO₂eq/kWh
}

// FetchGWP returns the current carbon intensity.
func (c *ElectricityMapsClient) FetchGWP(zone string) (float64, error) {
	if zone == "" {
		return 0, fmt.Errorf("electricity_maps: zone must not be empty")
	}

	url := fmt.Sprintf("%s/v3/carbon-intensity/latest?zone=%s", c.baseURL, zone)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("electricity_maps: build request: %w", err)
	}
	req.Header.Set("auth-token", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("electricity_maps: request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("electricity_maps: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("electricity_maps: HTTP %d: %s", resp.StatusCode, body)
	}

	var result electricityMapsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("electricity_maps: decode response: %w", err)
	}

	if result.CarbonIntensity < 0 {
		return 0, fmt.Errorf("electricity_maps: negative carbon intensity %g for zone %s", result.CarbonIntensity, zone)
	}

	return result.CarbonIntensity, nil
}
