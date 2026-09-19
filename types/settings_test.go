package types

import "testing"

func TestAppSettingsValidate(t *testing.T) {
    tests := []struct{
        name string
        s AppSettings
        wantErr bool
    }{
        {"valid defaults", AppSettings{Port: ":8080", Timeout: 10}, false},
        {"missing port", AppSettings{Port: "", Timeout: 10}, true},
        {"invalid timeout", AppSettings{Port: ":8080", Timeout: 0}, true},
        {"mysql missing fields", AppSettings{Port: ":8080", Timeout: 10, UseMySQL: true, MySqlSettings: struct{
            Username string `json:"username"`
            Password string `json:"password"`
            Host     string `json:"host"`
            Port     string `json:"port"`
            Database string `json:"database"`
        }{}}, true},
        {"mysql provided", AppSettings{Port: ":8080", Timeout: 10, UseMySQL: true, MySqlSettings: struct{
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
