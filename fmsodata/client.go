package fmsodata

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
)

// Client is the OData client for FileMaker
type Client struct {
	client  *http.Client
	config  ClientConfig
	baseURL string
}

// NewClient creates a new OData client
func NewClient(config ClientConfig) *Client {
	baseURL := fmt.Sprintf("%s/fmi/odata/v4/%s", config.Host, config.Database)
	return &Client{
		client: &http.Client{
			Timeout: config.Timeout,
		},
		config:  config,
		baseURL: baseURL,
	}
}

// getBasicAuthHeader returns the Basic Auth header value
func (c *Client) getBasicAuthHeader() string {
	auth := c.config.Username + ":" + c.config.Password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", c.getBasicAuthHeader())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return c.client.Do(req)
}
