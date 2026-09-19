package main

import (
	"testing"

	"github.com/johansundell/template-service/types"
)

func TestAppSettingsValidate_MySQLMissingRequiredFields(t *testing.T) {
	settings := types.AppSettings{
		Port:     ":8080",
		Timeout:  15,
		UseMySQL: true,
	}

	if err := settings.Validate(); err == nil {
		t.Fatal("expected validation to fail when MySQL required fields are missing")
	}
}
