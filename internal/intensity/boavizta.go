package intensity

import (
	"net/http"
	"time"
)

// BoaviztaClient fetches ADP and CED intensity factors from the Boavizta API.
// GET /v1/consumption_profile/country?country={code}
type BoaviztaClient struct {
	baseURL string
	http    *http.Client
}

func NewBoaviztaClient(baseURL string) *BoaviztaClient {
	return &BoaviztaClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}
