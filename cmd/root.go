package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/axellelanca/urlshortener/internal/config"
	"github.com/spf13/cobra"
)

// Cfg is the global variable that will contain the loaded configuration
// It will be accessible to all Cobra commands throughout the application
var Cfg *config.Config

// RootCmd is the base command for the CLI application
// All other commands (create, run-server, stats, migrate) are added as subcommands
var RootCmd = &cobra.Command{
	Use:   "urlshortener",
	Short: "A URL shortener application",
	Long: `A URL shortener application that allows you to create shortened URLs,
track click statistics, and monitor URL health.`,
}

// Execute is the main entry point for the Cobra application
// It is called from 'main.go' and handles command execution and error handling
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}

// init() is a special Go function that executes automatically before main()
// It's used here to initialize Cobra and set up command initialization hooks
func init() {
	// Set up configuration initialization to run before any command executes
	// This ensures configuration is loaded before any command needs it
	cobra.OnInitialize(initConfig)

	// IMPORTANT: We don't call RootCmd.AddCommand() directly here
	// for commands like 'server', 'create', 'stats', 'migrate'.
	// These commands register themselves via their own init() functions.
	// This design allows for better modularity and prevents import cycles.
}

// initConfig loads the application configuration
// This function is called at the beginning of every Cobra command execution
// thanks to `cobra.OnInitialize(initConfig)` set up above
func initConfig() {
	var err error

	// Load configuration from file, environment variables, and defaults
	// The config package handles the precedence and fallback logic
	Cfg, err = config.LoadConfig()
	if err != nil {
		// Log warning but don't exit if LoadConfig() handles missing files gracefully
		// If LoadConfig() terminates the program on fatal errors, this check is mainly for warnings
		log.Printf("Warning: Problem loading configuration: %v. Using default values.", err)
	}

	// Configuration is now available via the global variable 'cmd.Cfg'
	// All commands can access this configuration throughout their execution
}
