# fmsodata

`fmsodata` is a Go client library for the Claris FileMaker OData API. It provides a simple and idiomatic way to interact with FileMaker databases, supporting CRUD operations, script execution, and container data handling.

## Features

-   **CRUD Operations**: Create, Read, Update, and Delete records.
-   **Script Execution**: Run FileMaker scripts with parameters and receive results.
-   **Container Data**: Upload and download files to/from container fields.
-   **Context Support**: All methods support `context.Context` for cancellation and timeouts.
-   **Basic Authentication**: Supports standard Basic Auth.

## Installation

```bash
go get github.com/johansundell/template-service/fmsodata
```

## Usage

### Initialization

Create a new client with your FileMaker Server details:

```go
package main

import (
	"time"
	"github.com/johansundell/template-service/fmsodata"
)

func main() {
	config := fmsodata.ClientConfig{
		Host:     "https://your-filemaker-server.com",
		Database: "YourDatabase",
		Username: "username",
		Password: "password",
		Timeout:  30 * time.Second,
	}

	client := fmsodata.NewClient(config)
}
```

### CRUD Operations

#### Get Records

```go
import (
    "context"
    "net/url"
)

query := url.Values{}
query.Set("$filter", "Name eq 'John Doe'")
query.Set("$top", "10")

// Multiple search parameters
// query.Set("$filter", "Name eq 'John Doe' and Age eq 30")

records, err := client.GetRecords(context.Background(), "TableName", query)
if err != nil {
    // handle error
}
```

#### Get Single Record

```go
record, err := client.GetRecord(context.Background(), "TableName", "RecordID")
```

#### Create Record

```go
data := map[string]interface{}{
    "Name": "Jane Doe",
    "Age":  30,
}

id, err := client.CreateRecord(context.Background(), "TableName", data)
```

#### Update Record

```go
data := map[string]interface{}{
    "Age": 31,
}

err := client.UpdateRecord(context.Background(), "TableName", "RecordID", data)
```

#### Delete Record

```go
err := client.DeleteRecord(context.Background(), "TableName", "RecordID")
```

### Script Execution

Run a FileMaker script:

```go
result, err := client.RunScript(context.Background(), "ScriptName", "ScriptParameter")
if err != nil {
    // handle error
}

fmt.Printf("Script Result: %s (Code: %d)\n", result.ResultParameter, result.Code)
```

### Container Data

#### Upload to Container

```go
file, _ := os.Open("image.jpg")
defer file.Close()

// Uploads the file content to the specified container field
err := client.UploadContainer(context.Background(), "TableName", "RecordID", "ContainerFieldName", file)
```

#### Download from Container

```go
data, err := client.DownloadContainer(context.Background(), "TableName", "RecordID", "ContainerFieldName")
if err != nil {
    // handle error
}

// 'data' contains the raw bytes of the file
```

### Schema Modification

#### Create Table

```go
tableDef := fmsodata.TableDefinition{
    TableName: "NewTable",
    Fields: []fmsodata.FieldDefinition{
        {Name: "ID", Type: "NUMERIC", Primary: true, Unique: true},
        {Name: "Name", Type: "VARCHAR"},
        {Name: "Description", Type: "VARCHAR"},
    },
}

err := client.CreateTable(context.Background(), tableDef)
```

#### Delete Table

```go
err := client.DeleteTable(context.Background(), "NewTable")
```

#### Create Index

```go
err := client.CreateIndex(context.Background(), "TableName", "FieldName")
```

#### Delete Index

```go
err := client.DeleteIndex(context.Background(), "TableName", "IndexName")
```

### Example: Upload and Verify Container Data

```go
// 1. Read the file
fileData, _ := os.ReadFile("logo.png")

// 2. Upload to a specific record
err := client.UploadContainer(context.Background(), "sudde_odata2", "RecordID", "image", bytes.NewReader(fileData))
if err != nil {
    log.Fatal(err)
}

// 3. Download back
downloadedData, err := client.DownloadContainer(context.Background(), "sudde_odata2", "RecordID", "image")
if err != nil {
    log.Fatal(err)
}

// 4. Verify integrity
if bytes.Equal(fileData, downloadedData) {
    fmt.Println("Success: Downloaded data matches uploaded file.")
}
```

### Example: Migration Script

This example demonstrates how to fetch data from an external API and migrate it to a FileMaker table.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/johansundell/template-service/fmsodata"
)

type LogEntry struct {
	ID        int         `json:"id"`
	Status    int         `json:"status"`
	Method    string      `json:"method"`
	Error     string      `json:"error"`
	Endpoint  string      `json:"endpoint"`
	CreatedAt string      `json:"created_at"`
	Response  interface{} `json:"response"`
	Request   interface{} `json:"request"`
}

func main() {
	// 1. Initialize Client
	config := fmsodata.ClientConfig{
		Host:     "https://your-filemaker-server.com",
		Database: "YourDatabase",
		Username: "username",
		Password: "password",
		Timeout:  60 * time.Second,
	}
	client := fmsodata.NewClient(config)
	ctx := context.Background()

	tableName := "Logs"

	// 2. Create Table
	tableDef := fmsodata.TableDefinition{
		TableName: tableName,
		Fields: []fmsodata.FieldDefinition{
			{Name: "ID", Type: "NUMERIC", Primary: true, Unique: true},
			{Name: "Status", Type: "NUMERIC"},
			{Name: "Method", Type: "VARCHAR"},
			{Name: "Error", Type: "VARCHAR"},
			{Name: "Endpoint", Type: "VARCHAR"},
			{Name: "CreatedAt", Type: "TIMESTAMP"},
			{Name: "Response", Type: "VARCHAR"},
			{Name: "Request", Type: "VARCHAR"},
		},
	}

	err := client.CreateTable(ctx, tableDef)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	// 3. Fetch Logs (Example source)
	resp, err := http.Get("http://api.example.com/logs")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var logs []LogEntry
	if err := json.NewDecoder(resp.Body).Decode(&logs); err != nil {
		log.Fatal(err)
	}

	// 4. Upload Logs
	for _, entry := range logs {
		responseJSON, _ := json.Marshal(entry.Response)
		requestJSON, _ := json.Marshal(entry.Request)

		record := map[string]interface{}{
			"ID":        entry.ID,
			"Status":    entry.Status,
			"Method":    entry.Method,
			"Error":     entry.Error,
			"Endpoint":  entry.Endpoint,
			"CreatedAt": entry.CreatedAt,
			"Response":  string(responseJSON),
			"Request":   string(requestJSON),
		}

		_, err := client.CreateRecord(ctx, tableName, record)
		if err != nil {
			log.Printf("Failed to create record %d: %v", entry.ID, err)
		}
	}
}
```
