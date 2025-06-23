package config

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

// Main structure mapping the entire application configuration.
type Config struct {
	Server struct {
		Port    int    `mapstructure:"port"`
		BaseURL string `mapstructure:"base_url"`
	} `mapstructure:"server"`

	Database struct {
		Name string `mapstructure:"name"`
	} `mapstructure:"database"`

	Analytics struct {
		BufferSize  int `mapstructure:"buffer_size"`
		WorkerCount int `mapstructure:"worker_count"`
	} `mapstructure:"analytics"`

	Monitor struct {
		IntervalMinutes int `mapstructure:"interval_minutes"`
	} `mapstructure:"monitor"`
}

// Loads the application configuration using Viper.
func LoadConfig() (*Config, error) {
	// Enable environment variable overrides
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	// Specify the path where Viper should look for config files.
	viper.AddConfigPath("./configs")

	// Specify the name of the config file (without the extension).
	viper.SetConfigName("config")

	// Specify the type of config file.
	viper.SetConfigType("yaml")

	// Set default values for all configuration options
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.base_url", "http://localhost:8080")
	viper.SetDefault("database.name", "url_shortener.db")
	viper.SetDefault("analytics.buffer_size", 1000)
	viper.SetDefault("analytics.worker_count", 5)
	viper.SetDefault("monitor.interval_minutes", 5)

	// Reads config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("Config file not found, using default values")
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal the loaded configuration into the Config structure
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Log  pour vérifier la config chargée
	log.Printf("Configuration loaded: Server Port=%d, DB Name=%s, Analytics Buffer=%d, Monitor Interval=%dmin",
		cfg.Server.Port, cfg.Database.Name, cfg.Analytics.BufferSize, cfg.Monitor.IntervalMinutes)

	// Return the config
	return &cfg, nil
}
