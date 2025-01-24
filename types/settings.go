package types

type AppSettings struct {
	Debug         bool   `json:"debug"`
	Port          string `json:"port"`
	UseFileSystem bool   `json:"useFileSystem"`
	Timeout       int    `json:"timeout"`
	UseMysql      bool   `json:"useMysql"`
	UseSqlite     bool   `json:"useSqlite"`
	MySqlSettings struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Host     string `json:"host"`
		Port     string `json:"port"`
		Database string `json:"database"`
	} `json:"mysql"`
}
