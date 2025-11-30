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

// UploadContainer uploads data to a container field
func (c *Client) UploadContainer(ctx context.Context, tableName string, id string, fieldName string, data io.Reader) error {
	// Read all data
	content, err := io.ReadAll(data)
	if err != nil {
		return err
	}

	// Encode to Base64
	encoded := base64.StdEncoding.EncodeToString(content)

	// Prepare payload
	payload := map[string]interface{}{
		fieldName: encoded,
	}

	jsonData, err := json.Marshal(payload)
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

// DownloadContainer downloads data from a container field
func (c *Client) DownloadContainer(ctx context.Context, tableName string, id string, fieldName string) ([]byte, error) {
	url := fmt.Sprintf("%s/%s('%s')/%s/$value", c.baseURL, tableName, id, fieldName)
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

	return io.ReadAll(resp.Body)
}
