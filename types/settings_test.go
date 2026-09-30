package types

import "testing"

func TestAppSettingsValidate(t *testing.T) {
	tests := []struct {
		name    string
		s       AppSettings
		wantErr bool
	}{
		{"valid defaults", AppSettings{Port: ":8080", Timeout: 10, Storage: StorageSQLite}, false},
		{"missing storage", AppSettings{Port: ":8080", Timeout: 10}, true},
		{"unknown storage", AppSettings{Port: ":8080", Timeout: 10, Storage: "postgres"}, true},
		{"missing port", AppSettings{Port: "", Timeout: 10, Storage: StorageSQLite}, true},
		{"invalid timeout", AppSettings{Port: ":8080", Timeout: 0, Storage: StorageSQLite}, true},
		{"mysql missing fields", AppSettings{Port: ":8080", Timeout: 10, Storage: StorageMySQL, MySqlSettings: struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Host     string `json:"host"`
			Port     string `json:"port"`
			Database string `json:"database"`
		}{}}, true},
		{"mysql provided", AppSettings{Port: ":8080", Timeout: 10, Storage: StorageMySQL, MySqlSettings: struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Host     string `json:"host"`
			Port     string `json:"port"`
			Database string `json:"database"`
		}{Username: "u", Host: "localhost", Database: "db"}}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.s.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
