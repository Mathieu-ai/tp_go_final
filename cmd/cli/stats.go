package cli

import (
	"fmt"
	"log"
	"os"

	"github.com/axellelanca/urlshortener/cmd"
	"github.com/axellelanca/urlshortener/internal/config"
	"github.com/axellelanca/urlshortener/internal/repository"
	"github.com/axellelanca/urlshortener/internal/services"
	"github.com/glebarez/sqlite"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

// shortCodeFlag stores the short code provided by the user via the --code flag
var shortCodeFlag string

// StatsCmd represents the 'stats' command
// This command allows users to view click statistics for a specific short URL
var StatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Get statistics for a short URL",
	Long:  `Get click statistics for the provided short code.`,
	Run:   runStats, // Delegate to separate function for better organization
}

func init() {
	// Define the --code flag for the stats command
	// This flag accepts the short code that the user wants statistics for
	StatsCmd.Flags().StringVar(&shortCodeFlag, "code", "", "The short code to get statistics for")

	// Mark the flag as required - Cobra will enforce this validation
	StatsCmd.MarkFlagRequired("code")

	// Register this command with the root command
	cmd.RootCmd.AddCommand(StatsCmd)
}

// runStats executes the logic for the stats command
// Separated into its own function for better readability and testing
func runStats(cmd *cobra.Command, args []string) {
	// Double-check that the required flag was provided
	// This is a safety check even though Cobra enforces required flags
	if shortCodeFlag == "" {
		fmt.Println("Error: --code flag is required")
		os.Exit(1)
	}

	// Load application configuration to get database settings
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connection using GORM with SQLite
	db, err := gorm.Open(sqlite.Open(cfg.Database.Name), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Get underlying SQL connection for proper cleanup
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("FATAL: Failed to get underlying SQL database: %v", err)
	}
	defer sqlDB.Close() // Ensure database connection is closed

	// Initialize repository and service layers
	// Repository handles database operations, service handles business logic
	linkRepo := repository.NewLinkRepository(db)
	linkService := services.NewLinkService(linkRepo)

	// Call GetLinkStats to retrieve the link and its statistics
	// This includes the link details and total click count
	link, totalClicks, err := linkService.GetLinkStats(shortCodeFlag)
	if err != nil {
		// Handle the case where the short code doesn't exist
		if err == gorm.ErrRecordNotFound {
			fmt.Printf("Error: Short code '%s' not found\n", shortCodeFlag)
		} else {
			// Handle other database or service errors
			fmt.Printf("Error retrieving statistics: %v\n", err)
		}
		os.Exit(1)
	}

	// Display the results in a user-friendly format
	fmt.Printf("Statistics for short code: %s\n", shortCodeFlag)
	fmt.Printf("Long URL: %s\n", link.LongURL)
	fmt.Printf("Total clicks: %d\n", totalClicks)
	fmt.Printf("Creation date: %s\n", link.CreatedAt.Format("2006-01-02 15:04:05"))
}
