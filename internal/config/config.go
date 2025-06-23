package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the main structure mapping the entire application configuration.
// This struct uses mapstructure tags to map YAML/JSON keys to Go struct fields.
type Config struct {
	// Server configuration section containing HTTP server settings
	Server struct {
		Port    int    `mapstructure:"port"`     // HTTP server port (default: 8080)
		BaseURL string `mapstructure:"base_url"` // Base URL for generating short links
	} `mapstructure:"server"`

	// Database configuration section for SQLite settings
	Database struct {
		Name string `mapstructure:"name"` // SQLite database file name
	} `mapstructure:"database"`

	// Analytics configuration for asynchronous click tracking
	Analytics struct {
		BufferSize  int `mapstructure:"buffer_size"`  // Size of the click event channel buffer
		WorkerCount int `mapstructure:"worker_count"` // Number of worker goroutines for processing clicks
	} `mapstructure:"analytics"`

	// Monitor configuration for URL health checking
	Monitor struct {
		IntervalMinutes int `mapstructure:"interval_minutes"` // Interval in minutes between URL health checks
	} `mapstructure:"monitor"`
}

// LoadConfig loads the application configuration using Viper.
// It supports environment variable overrides and YAML configuration files.
// Returns a populated Config struct or an error if configuration loading fails.
func LoadConfig() (*Config, error) {
	// Enable automatic environment variable binding
	// This allows config values to be overridden via environment variables
	viper.AutomaticEnv()

	// Replace dots with underscores in environment variable names
	// e.g., "server.port" becomes "SERVER_PORT"
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Specify the directory path where Viper should look for config files
	viper.AddConfigPath("./configs")

	// Specify the name of the config file (without the extension)
	viper.SetConfigName("config")

	// Specify the type/format of the config file (YAML in this case)
	viper.SetConfigType("yaml")

	// Set default values for all configuration options
	// These will be used if no config file is found or if specific keys are missing
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.base_url", "http://localhost:8080")
	viper.SetDefault("database.name", "url_shortener.db")
	viper.SetDefault("analytics.buffer_size", 1000)
	viper.SetDefault("analytics.worker_count", 5)
	viper.SetDefault("monitor.interval_minutes", 5)

	// Attempt to read the config file
	if err := viper.ReadInConfig(); err != nil {
		// Check if the error is specifically "config file not found"
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// This is not a fatal error - we'll use default values
			log.Println("Config file not found, using default values")
		} else {
			// Any other error (permissions, malformed YAML, etc.) is fatal
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal the loaded configuration into our Config structure
	// This converts the Viper configuration into our strongly-typed struct
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Log the loaded configuration for debugging and verification purposes
	log.Printf("Configuration loaded: Server Port=%d, DB Name=%s, Analytics Buffer=%d, Monitor Interval=%dmin",
		cfg.Server.Port, cfg.Database.Name, cfg.Analytics.BufferSize, cfg.Monitor.IntervalMinutes)

	// Return the successfully loaded and parsed configuration
	return &cfg, nil
}
