package cli

import (
	"fmt"
	"log"

	"github.com/axellelanca/urlshortener/cmd"
	"github.com/axellelanca/urlshortener/internal/config"
	"github.com/axellelanca/urlshortener/internal/models"
	"github.com/glebarez/sqlite"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

// MigrateCmd represents the 'migrate' command
// This command handles database schema creation and updates
var MigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Executes database migrations to create or update tables.",
	Long: `This command connects to the configured database (SQLite)
and executes GORM automatic migrations to create 'links' and 'clicks' tables
based on the Go models.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration to get database connection settings
		// This ensures we connect to the correct database file
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}

		// Initialize database connection using GORM with SQLite driver
		// Uses the database name specified in the configuration
		db, err := gorm.Open(sqlite.Open(cfg.Database.Name), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}

		// Get the underlying SQL database connection for proper resource management
		// This allows us to close the connection when migration is complete
		sqlDB, err := db.DB()
		if err != nil {
			log.Fatalf("FATAL: Failed to get underlying SQL database: %v", err)
		}
		defer sqlDB.Close() // Ensure connection is closed when function exits

		// Execute GORM automatic migrations
		// This creates tables based on the struct definitions in our models
		// It also handles adding new columns if the models have been updated
		if err := db.AutoMigrate(&models.Link{}, &models.Click{}); err != nil {
			log.Fatalf("Failed to migrate database: %v", err)
		}

		// Inform the user that migration completed successfully
		fmt.Println("Database migrations executed successfully.")
	},
}

func init() {
	// Register this command with the root command so it can be executed via CLI
	cmd.RootCmd.AddCommand(MigrateCmd)
}
