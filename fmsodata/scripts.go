package fmsodata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// RunScript runs a script in the FileMaker database
func (c *Client) RunScript(ctx context.Context, scriptName string, parameter interface{}) (ScriptResult, error) {
	url := fmt.Sprintf("%s/Script.%s", c.baseURL, scriptName)

	var req *http.Request
	var err error

	if parameter != nil {
		payload := map[string]interface{}{
			"scriptParameterValue": parameter,
		}
		jsonData, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return ScriptResult{}, marshalErr
		}
		req, err = http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	} else {
		req, err = http.NewRequest("POST", url, nil)
	}

	if err != nil {
		return ScriptResult{}, err
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return ScriptResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ScriptResult{}, c.handleError(resp)
	}

	var scriptResp ScriptResponse
	if err := json.NewDecoder(resp.Body).Decode(&scriptResp); err != nil {
		return ScriptResult{}, err
	}

	return scriptResp.ScriptResult, nil
}
