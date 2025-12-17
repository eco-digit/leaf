package promclient

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	promapi "github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type Client struct {
	api v1.API
	url string
}

// BasicAuthTransport implements http.RoundTripper with basic auth
type BasicAuthTransport struct {
	Username  string
	Password  string
	Transport http.RoundTripper
}

func (t *BasicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Username != "" || t.Password != "" {
		req.SetBasicAuth(t.Username, t.Password)
	}
	return t.Transport.RoundTrip(req)
}

func NewClient(url, username, password string) (*Client, error) {
	httpClient := &http.Client{
		Transport: &BasicAuthTransport{
			Username:  username,
			Password:  password,
			Transport: http.DefaultTransport,
		},
	}

	config := promapi.Config{
		Address:      url,
		RoundTripper: httpClient.Transport,
	}

	client, err := promapi.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus API client: %w", err)
	}

	c := &Client{
		api: v1.NewAPI(client),
		url: url,
	}

	// Test connection
	if err := c.TestConnection(); err != nil {
		return nil, fmt.Errorf("failed to connect to Prometheus: %w", err)
	}

	log.Printf("Connected to Prometheus at %s", url)
	return c, nil
}

// TestConnection verifies connectivity to Prometheus
func (c *Client) TestConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.api.Buildinfo(ctx)
	if err != nil {
		return fmt.Errorf("connectivity test failed: %w", err)
	}

	return nil
}

func (c *Client) QueryMetric(metric string) (model.Value, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	val, warnings, err := c.api.Query(ctx, metric, time.Now())
	if err != nil {
		return nil, err
	}

	if len(warnings) > 0 {
		log.Printf("Prometheus warnings: %v", warnings)
	}

	return val, nil
}
