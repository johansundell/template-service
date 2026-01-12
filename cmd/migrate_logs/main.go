package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"os"

	"github.com/johansundell/template-service/fmsodata"
	"github.com/joho/godotenv"
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
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default/environment values")
	}

	config := fmsodata.ClientConfig{
		Host:     os.Getenv("FMS_HOST"),
		Database: os.Getenv("FMS_DATABASE"),
		Username: os.Getenv("FMS_USERNAME"),
		Password: os.Getenv("FMS_PASSWORD"),
		Timeout:  60 * time.Second,
	}
	client := fmsodata.NewClient(config)
	ctx := context.Background()

	tableName := "Logs"

	// 2. Create Table
	fmt.Println("Creating table...")
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

	// Delete table if exists to start fresh (optional, but good for this task)
	_ = client.DeleteTable(ctx, tableName)

	err := client.CreateTable(ctx, tableDef)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
	fmt.Println("Table created.")

	// 3. Fetch Logs
	fmt.Println("Fetching logs...")
	req, err := http.NewRequest("GET", "http://localhost:8081/logs/2023-01-01/2026-06-01", nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", os.Getenv("AUTH_TOKEN"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Failed to fetch logs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Failed to fetch logs, status: %d, body: %s", resp.StatusCode, string(body))
	}

	var logs []LogEntry
	if err := json.NewDecoder(resp.Body).Decode(&logs); err != nil {
		log.Fatalf("Failed to decode logs: %v", err)
	}
	fmt.Printf("Fetched %d logs.\n", len(logs))

	// 4. Upload Logs
	fmt.Println("Uploading logs...")
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
			log.Printf("Failed to create record for log ID %d: %v", entry.ID, err)
		} else {
			fmt.Printf("Uploaded log ID %d\n", entry.ID)
		}
	}

	fmt.Println("Migration complete.")
}
