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
