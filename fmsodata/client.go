package fmsodata

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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

// sendRequest is a helper to send requests with JSON body
func (c *Client) sendRequest(ctx context.Context, method, url string, payload interface{}) (*http.Response, error) {
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, c.handleError(resp)
	}

	return resp, nil
}

// CreateTable creates a new table with the given definition
func (c *Client) CreateTable(ctx context.Context, table TableDefinition) error {
	url := fmt.Sprintf("%s/FileMaker_Tables", c.baseURL)
	resp, err := c.sendRequest(ctx, http.MethodPost, url, table)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// DeleteTable deletes a table by name
func (c *Client) DeleteTable(ctx context.Context, tableName string) error {
	url := fmt.Sprintf("%s/FileMaker_Tables/%s", c.baseURL, tableName)
	resp, err := c.sendRequest(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// CreateIndex creates an index for a field in a table
func (c *Client) CreateIndex(ctx context.Context, tableName string, fieldName string) error {
	url := fmt.Sprintf("%s/FileMaker_Indexes/%s", c.baseURL, tableName)
	payload := map[string]string{
		"indexName": fieldName,
	}
	resp, err := c.sendRequest(ctx, http.MethodPost, url, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// DeleteIndex deletes an index
func (c *Client) DeleteIndex(ctx context.Context, tableName string, indexName string) error {
	url := fmt.Sprintf("%s/FileMaker_Indexes/%s/%s", c.baseURL, tableName, indexName)
	resp, err := c.sendRequest(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// Ping checks the connection to the FileMaker OData API
func (c *Client) Ping(ctx context.Context) error {
	// Request the service document (base URL)
	req, err := http.NewRequest("GET", c.baseURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.handleError(resp)
	}

	return nil
}
