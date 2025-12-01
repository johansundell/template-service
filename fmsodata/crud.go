package fmsodata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// GetRecords retrieves records from a table
func (c *Client) GetRecords(ctx context.Context, tableName string, query url.Values) ([]map[string]interface{}, error) {
	u, err := url.Parse(fmt.Sprintf("%s/%s", c.baseURL, tableName))
	if err != nil {
		return nil, err
	}
	u.RawQuery = strings.ReplaceAll(query.Encode(), "+", "%20")

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var odataResp ODataResponse
	if err := json.NewDecoder(resp.Body).Decode(&odataResp); err != nil {
		return nil, err
	}

	return odataResp.Value, nil
}

// GetRecord retrieves a single record by ID
func (c *Client) GetRecord(ctx context.Context, tableName string, id string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/%s('%s')", c.baseURL, tableName, id)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var record map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&record); err != nil {
		return nil, err
	}

	return record, nil
}

// CreateRecord creates a new record
func (c *Client) CreateRecord(ctx context.Context, tableName string, data map[string]interface{}) (map[string]interface{}, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/%s", c.baseURL, tableName)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var record map[string]interface{}
	// If the response has a body, try to decode it
	if resp.ContentLength != 0 {
		if err := json.NewDecoder(resp.Body).Decode(&record); err != nil {
			// If decoding fails but status is OK, we might just return nil or the error
			// For now, let's assume if we can't decode, it's an issue if we expect a record back
			// However, sometimes POST might return 204 No Content or empty body?
			// OData usually returns the created entity.
			return nil, err
		}
	}

	return record, nil
}

// UpdateRecord updates an existing record
func (c *Client) UpdateRecord(ctx context.Context, tableName string, id string, data map[string]interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s('%s')", c.baseURL, tableName, id)
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return c.handleError(resp)
	}

	return nil
}

// DeleteRecord deletes a record
func (c *Client) DeleteRecord(ctx context.Context, tableName string, id string) error {
	url := fmt.Sprintf("%s/%s('%s')", c.baseURL, tableName, id)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return c.handleError(resp)
	}

	return nil
}

func (c *Client) handleError(resp *http.Response) error {
	bodyBytes, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("OData request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
}
