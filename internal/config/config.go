package config

type Config struct {
	Env  string `mapstructure:"env"`
	Http struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"http"`
	Security struct {
		Secret string `mapstructure:"secret"`
	} `mapstructure:"security"`
	Data struct {
		DB struct {
			User struct {
				Driver string `mapstructure:"driver"`
				DSN    string `mapstructure:"dsn"`
			} `mapstructure:"user"`
		} `mapstructure:"db"`
		Redis struct {
			Addr     string `mapstructure:"addr"`
			Password string `mapstructure:"password"`
			DB       int    `mapstructure:"db"`
		} `mapstructure:"redis"`
	} `mapstructure:"data"`
	Log struct {
		Level           string `mapstructure:"log_level"`
		Mode            string `mapstructure:"mode"`
		FileEncoding    string `mapstructure:"file_encoding"`
		ConsoleEncoding string `mapstructure:"console_encoding"`
		Encoding        string `mapstructure:"encoding"`
		LogPath         string `mapstructure:"log_path"`
		FileName        string `mapstructure:"log_file_name"`
		ErrorFileName   string `mapstructure:"error_file_name"`
		MaxBackups      int    `mapstructure:"max_backups"`
		MaxAge          int    `mapstructure:"max_age"`
		MaxSize         int    `mapstructure:"max_size"`
		Compress        bool   `mapstructure:"compress"`
	} `mapstructure:"log"`
}
