package fmsodata

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetRecords(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Basic dXNlcjpwYXNz", r.Header.Get("Authorization"))
		assert.Equal(t, "/fmi/odata/v4/testdb/Table1", r.URL.Path)
		assert.Equal(t, "filter=field%20eq%20%27value%27", r.URL.RawQuery)

		response := ODataResponse{
			Value: []map[string]interface{}{
				{"ID": "1", "Name": "Test"},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		Host:     server.URL, // Use server URL as host, but NewClient appends /fmi/odata...
		Database: "testdb",
		Username: "user",
		Password: "pass",
		Timeout:  10 * time.Second,
	})
	// Hack to fix the base URL for the mock server since NewClient appends the path
	// In a real scenario, Host would be just the domain
	// Here server.URL includes http://ip:port
	// We need to adjust the baseURL in the client to match the mock server's expectation if we want to test exact paths
	// But NewClient does: fmt.Sprintf("%s/fmi/odata/v4/%s", config.Host, config.Database)
	// So if Host is http://ip:port, baseURL is http://ip:port/fmi/odata/v4/testdb
	// The mock server will receive requests at /fmi/odata/v4/testdb/...
	// So this is correct.

	query := url.Values{}
	query.Set("filter", "field eq 'value'")

	records, err := client.GetRecords(context.Background(), "Table1", query)
	assert.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "Test", records[0]["Name"])
}

func TestGetRecord(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/fmi/odata/v4/testdb/Table1('1')", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]interface{}{"ID": "1", "Name": "Test"})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		Host:     server.URL,
		Database: "testdb",
		Username: "user",
		Password: "pass",
	})

	record, err := client.GetRecord(context.Background(), "Table1", "1")
	assert.NoError(t, err)
	assert.Equal(t, "Test", record["Name"])
}

func TestCreateRecord(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/fmi/odata/v4/testdb/Table1", r.URL.Path)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		Host:     server.URL,
		Database: "testdb",
		Username: "user",
		Password: "pass",
	})

	_, err := client.CreateRecord(context.Background(), "Table1", map[string]interface{}{"Name": "New"})
	assert.NoError(t, err)
}

func TestRunScript(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/fmi/odata/v4/testdb/Script.TestScript", r.URL.Path)

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "param", body["scriptParameterValue"])

		response := ScriptResponse{
			ScriptResult: ScriptResult{
				Code:            0,
				ResultParameter: "Success",
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		Host:     server.URL,
		Database: "testdb",
		Username: "user",
		Password: "pass",
	})

	result, err := client.RunScript(context.Background(), "TestScript", "param")
	assert.NoError(t, err)
	assert.Equal(t, 0, result.Code)
	assert.Equal(t, "Success", result.ResultParameter)
}

func TestUploadContainer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		assert.Equal(t, "/fmi/odata/v4/testdb/Table1('1')", r.URL.Path)

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "dGVzdA==", body["ContainerField"]) // Base64 for "test"

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		Host:     server.URL,
		Database: "testdb",
		Username: "user",
		Password: "pass",
	})

	err := client.UploadContainer(context.Background(), "Table1", "1", "ContainerField", bytes.NewBufferString("test"))
	assert.NoError(t, err)
}

func TestDownloadContainer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/fmi/odata/v4/testdb/Table1('1')/ContainerField/$value", r.URL.Path)

		w.Write([]byte("test data"))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		Host:     server.URL,
		Database: "testdb",
		Username: "user",
		Password: "pass",
	})

	data, err := client.DownloadContainer(context.Background(), "Table1", "1", "ContainerField")
	assert.NoError(t, err)
	assert.Equal(t, []byte("test data"), data)
}

func TestPing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/fmi/odata/v4/testdb", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		Host:     server.URL,
		Database: "testdb",
		Username: "user",
		Password: "pass",
	})

	err := client.Ping(context.Background())
	assert.NoError(t, err)
}
