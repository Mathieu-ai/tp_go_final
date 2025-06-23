package cli

import (
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/axellelanca/urlshortener/cmd"
	"github.com/axellelanca/urlshortener/internal/config"
	"github.com/axellelanca/urlshortener/internal/repository"
	"github.com/axellelanca/urlshortener/internal/services"
	"github.com/glebarez/sqlite"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

// longURLFlag stores the URL provided by the user via the --url flag
var longURLFlag string

// CreateCmd represents the 'create' command for the CLI application
// This command allows users to create shortened URLs from long URLs via command line
var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates a short URL from a long URL.",
	Long: `This command shortens a provided long URL and displays the generated short code.

Example:
  url-shortener create --url="https://www.google.com/search?q=go+lang"`,
	Run: func(cmd *cobra.Command, args []string) {
		// Validate that the --url flag has been provided
		// This is a safety check even though we mark the flag as required
		if longURLFlag == "" {
			fmt.Println("Error: --url flag is required")
			os.Exit(1)
		}

		// Basic validation of URL format using Go's standard url package
		// This ensures the provided string is a valid URI before processing
		_, err := url.ParseRequestURI(longURLFlag)
		if err != nil {
			fmt.Printf("Error: Invalid URL format: %v\n", err)
			os.Exit(1)
		}

		// Load application configuration from config file or environment variables
		// This gives us database settings, server settings, etc.
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}

		// Initialize database connection using GORM with SQLite driver
		// The database file name comes from the loaded configuration
		db, err := gorm.Open(sqlite.Open(cfg.Database.Name), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}

		// Get the underlying SQL database connection for proper cleanup
		// This is necessary to close the connection when we're done
		sqlDB, err := db.DB()
		if err != nil {
			log.Fatalf("FATAL: Failed to get underlying SQL database: %v", err)
		}
		defer sqlDB.Close() // Ensure database connection is closed when function exits

		// Initialize the repository layer for database operations
		// This abstracts the database operations behind an interface
		linkRepo := repository.NewLinkRepository(db)

		// Initialize the service layer with business logic
		// This handles the actual URL shortening logic and validation
		linkService := services.NewLinkService(linkRepo)

		// Call the LinkService to create the shortened link
		// This will generate a unique short code and store it in the database
		link, err := linkService.CreateLink(longURLFlag)
		if err != nil {
			log.Fatalf("Failed to create short link: %v", err)
		}

		// Build the full shortened URL using the base URL from configuration
		// This gives users the complete URL they can share
		fullShortURL := fmt.Sprintf("%s/%s", cfg.Server.BaseURL, link.ShortCode)

		// Display the results to the user in a friendly format
		fmt.Printf("Short URL created successfully:\n")
		fmt.Printf("Code: %s\n", link.ShortCode)
		fmt.Printf("Full URL: %s\n", fullShortURL)
	},
}

func init() {
	// Define the --url flag for the create command
	// This flag is required and stores the long URL to be shortened
	CreateCmd.Flags().StringVar(&longURLFlag, "url", "", "The long URL to shorten")

	// Mark the flag as required - Cobra will enforce this
	CreateCmd.MarkFlagRequired("url")

	// Add this command to the root command so it can be executed
	cmd.RootCmd.AddCommand(CreateCmd)
}
