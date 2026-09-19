package types

import "fmt"

type AppSettings struct {
	Debug         bool   `json:"debug"`
	Port          string `json:"port"`
	UseFileSystem bool   `json:"useFileSystem"`
	Timeout       int    `json:"timeout"`
	UseMySQL      bool   `json:"useMysql"`
	UseSqlite     bool   `json:"useSqlite"`
	AuthToken     string `json:"authToken"`
	MySqlSettings struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Host     string `json:"host"`
		Port     string `json:"port"`
		Database string `json:"database"`
	} `json:"mysql"`
}

// Validate verifies required settings and returns an error when configuration is invalid.
func (s AppSettings) Validate() error {
	if s.Port == "" {
		return fmt.Errorf("PORT must be set")
	}
	if s.Timeout <= 0 {
		return fmt.Errorf("TIMEOUT must be > 0")
	}
	if s.UseMySQL {
		if s.MySqlSettings.Username == "" || s.MySqlSettings.Host == "" || s.MySqlSettings.Database == "" {
			return fmt.Errorf("MYSQL_USERNAME, MYSQL_HOST and MYSQL_DATABASE must be set when USE_MYSQL=true")
		}
	}
	return nil
}
