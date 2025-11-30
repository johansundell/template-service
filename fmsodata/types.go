package fmsodata

import "time"

// ODataResponse represents a generic OData response
type ODataResponse struct {
	Context string                   `json:"@odata.context,omitempty"`
	Count   int                      `json:"@odata.count,omitempty"`
	Value   []map[string]interface{} `json:"value,omitempty"`
}

// ODataError represents an OData error response
type ODataError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// Record represents a generic FileMaker record
type Record map[string]interface{}

// TokenResponse represents the response when requesting a session token (if needed in future)
type TokenResponse struct {
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// ClientConfig holds configuration for the OData client
type ClientConfig struct {
	Host     string
	Database string
	Username string
	Password string
	Timeout  time.Duration
}

// ScriptResult represents the result of a script execution
type ScriptResult struct {
	Code            int         `json:"code"`
	ResultParameter interface{} `json:"resultParameter"`
}

// ScriptResponse represents the response from a script execution
type ScriptResponse struct {
	ScriptResult ScriptResult `json:"scriptResult"`
}

// FieldDefinition represents a field definition for creating a table
type FieldDefinition struct {
	Name    string      `json:"name"`
	Type    string      `json:"type"`
	Primary bool        `json:"primary,omitempty"`
	Unique  bool        `json:"unique,omitempty"`
	Global  bool        `json:"global,omitempty"`
	Default interface{} `json:"default,omitempty"`
}

// TableDefinition represents a table definition for creating a table
type TableDefinition struct {
	TableName string            `json:"tableName"`
	Fields    []FieldDefinition `json:"fields"`
}
