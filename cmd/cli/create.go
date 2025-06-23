package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/axellelanca/urlshortener/cmd"
	"github.com/axellelanca/urlshortener/internal/config"
	"github.com/axellelanca/urlshortener/internal/repository"
	"github.com/axellelanca/urlshortener/internal/services"
	"github.com/glebarez/sqlite"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

// longURLFlag stores the URLs provided by the user via the --url flag
var longURLFlag string

// CreateCmd represents the 'create' command for the CLI application
// This command allows users to create shortened URLs from one or more long URLs via command line
var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates short URLs from one or more long URLs.",
	Long: `This command shortens one or more provided long URLs and displays the generated short codes.

Examples:
  url-shortener create --url="https://www.google.com"
  url-shortener create --url="https://www.google.com" --url="https://www.github.com"
  url-shortener create --url='["https://www.google.com", "https://www.github.com", "https://www.stackoverflow.com"]'
  url-shortener create --url="['https://www.google.com','https://www.github.com']"`,

	Run: func(cmd *cobra.Command, args []string) {
		// Validate that the --url flag has been provided
		if longURLFlag == "" {
			fmt.Println("Error: The --url flag is required")
			os.Exit(1)
		}

		// Parse URLs from the single flag value
		allURLs, err := parseURLFlag(longURLFlag)
		if err != nil {
			fmt.Printf("Error: Failed to parse URL flag '%s': %v\n", longURLFlag, err)
			os.Exit(1)
		}

		// Validate all parsed URLs before processing any of them
		for i, urlStr := range allURLs {
			_, err := url.ParseRequestURI(urlStr)
			if err != nil {
				fmt.Printf("Error: Invalid URL format for URL #%d (%s): %v\n", i+1, urlStr, err)
				os.Exit(1)
			}
		}

		// Load application configuration from config file or environment variables
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}

		// Initialize database connection using GORM with SQLite driver
		db, err := gorm.Open(sqlite.Open(cfg.Database.Name), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}

		// Get the underlying SQL database connection for proper cleanup
		sqlDB, err := db.DB()
		if err != nil {
			log.Fatalf("FATAL: Failed to get underlying SQL database: %v", err)
		}
		defer sqlDB.Close() // Ensure database connection is closed when function exits

		// Initialize the repository and service layers
		linkRepo := repository.NewLinkRepository(db)
		linkService := services.NewLinkService(linkRepo)

		// Process each URL and collect results
		fmt.Printf("Creating short URLs for %d URL(s)...\n\n", len(allURLs))

		successCount := 0
		for i, longURL := range allURLs {
			fmt.Printf("[%d/%d] Processing: %s\n", i+1, len(allURLs), longURL)

			// Call the LinkService to create the shortened link
			link, err := linkService.CreateLink(longURL)
			if err != nil {
				fmt.Printf("  ❌ Failed to create short link: %v\n\n", err)
				continue
			}

			// Build the full shortened URL using the base URL from configuration
			fullShortURL := fmt.Sprintf("%s/%s", cfg.Server.BaseURL, link.ShortCode)

			// Display the results for this URL
			fmt.Printf("  ✅ Short URL created successfully:\n")
			fmt.Printf("     Code: %s\n", link.ShortCode)
			fmt.Printf("     Full URL: %s\n\n", fullShortURL)

			successCount++
		}

		// Summary
		if successCount == len(allURLs) {
			fmt.Printf("🎉 All %d URL(s) shortened successfully!\n", successCount)
		} else {
			fmt.Printf("⚠️  %d out of %d URL(s) shortened successfully.\n", successCount, len(allURLs))
		}
	},
}

// parseURLFlag parses a URL flag that can be either a single URL string or a JSON array of URLs
// parseURLFlag parses a URL flag that can be either a single URL string or a JSON array of URLs
func parseURLFlag(urlFlag string) ([]string, error) {
	log.Printf("DEBUG: parseURLFlag called with input: '%s'", urlFlag)

	// Trim whitespace
	urlFlag = strings.TrimSpace(urlFlag)
	log.Printf("DEBUG: After trimming whitespace: '%s'", urlFlag)

	// Check if it looks like a JSON array (starts with [ and ends with ])
	if strings.HasPrefix(urlFlag, "[") && strings.HasSuffix(urlFlag, "]") {
		log.Printf("DEBUG: Input appears to be JSON array format")

		// First try to parse as proper JSON array (with double quotes)
		var urls []string
		log.Printf("DEBUG: Attempting to parse as standard JSON array...")
		err := json.Unmarshal([]byte(urlFlag), &urls)
		if err == nil {
			log.Printf("DEBUG: Successfully parsed as JSON array, found %d URLs: %v", len(urls), urls)
			if len(urls) == 0 {
				log.Printf("ERROR: JSON array is empty")
				return nil, fmt.Errorf("JSON array cannot be empty")
			}
			log.Printf("DEBUG: Returning successfully parsed JSON array")
			return urls, nil
		}
		log.Printf("DEBUG: Standard JSON parsing failed: %v", err)

		// If JSON parsing fails, try to convert single quotes to double quotes and parse again
		normalizedJSON := strings.ReplaceAll(urlFlag, "'", "\"")
		log.Printf("DEBUG: Attempting to parse with normalized quotes: '%s'", normalizedJSON)
		err = json.Unmarshal([]byte(normalizedJSON), &urls)
		if err == nil {
			log.Printf("DEBUG: Successfully parsed normalized JSON, found %d URLs: %v", len(urls), urls)
			if len(urls) == 0 {
				log.Printf("ERROR: Normalized JSON array is empty")
				return nil, fmt.Errorf("JSON array cannot be empty")
			}
			log.Printf("DEBUG: Returning successfully parsed normalized JSON array")
			return urls, nil
		}
		log.Printf("DEBUG: Normalized JSON parsing also failed: %v", err)

		// If both JSON attempts fail, manually parse comma-separated values
		log.Printf("DEBUG: Attempting manual parsing of array content...")
		// Remove the outer brackets first
		content := strings.TrimSpace(urlFlag[1 : len(urlFlag)-1])
		log.Printf("DEBUG: Content after removing brackets: '%s'", content)
		if content == "" {
			log.Printf("ERROR: Array content is empty after removing brackets")
			return nil, fmt.Errorf("JSON array cannot be empty")
		}

		// Split by comma and clean each URL
		parts := strings.Split(content, ",")
		log.Printf("DEBUG: Split by comma into %d parts: %v", len(parts), parts)
		var parsedURLs []string
		for i, part := range parts {
			log.Printf("DEBUG: Processing part %d: '%s'", i+1, part)

			// Trim whitespace
			cleanURL := strings.TrimSpace(part)
			log.Printf("DEBUG: After trimming whitespace: '%s'", cleanURL)

			// Remove surrounding quotes (both single and double)
			cleanURL = removeQuotes(cleanURL)
			log.Printf("DEBUG: After removing quotes: '%s'", cleanURL)

			// Trim again after quote removal
			cleanURL = strings.TrimSpace(cleanURL)
			log.Printf("DEBUG: After final trim: '%s'", cleanURL)

			if cleanURL != "" {
				parsedURLs = append(parsedURLs, cleanURL)
				log.Printf("DEBUG: Added URL to result: '%s'", cleanURL)
			} else {
				log.Printf("DEBUG: Skipping empty URL after cleaning")
			}
		}

		log.Printf("DEBUG: Manual parsing completed, found %d valid URLs: %v", len(parsedURLs), parsedURLs)
		if len(parsedURLs) == 0 {
			log.Printf("ERROR: No valid URLs found after manual parsing")
			return nil, fmt.Errorf("no valid URLs found in array")
		}

		log.Printf("DEBUG: Returning manually parsed URLs")
		return parsedURLs, nil
	}

	// Not a JSON array, treat as single URL
	log.Printf("DEBUG: Input is not JSON array format, treating as single URL")
	result := []string{urlFlag}
	log.Printf("DEBUG: Returning single URL result: %v", result)
	return result, nil
}

// removeQuotes removes surrounding quotes from a string
// Handles both single and double quotes, and nested quotes
func removeQuotes(s string) string {
	// Keep removing quotes from both ends until no more quotes are found
	for {
		original := s

		// Remove leading quotes
		if strings.HasPrefix(s, "'") || strings.HasPrefix(s, "\"") {
			s = s[1:]
		}

		// Remove trailing quotes
		if strings.HasSuffix(s, "'") || strings.HasSuffix(s, "\"") {
			s = s[:len(s)-1]
		}

		// If no changes were made, we're done
		if s == original {
			break
		}
	}

	return s
}

func init() {
	// Define the --url flag for the create command as a single string
	// This allows JSON arrays or single URLs to be specified
	CreateCmd.Flags().StringVar(&longURLFlag, "url", "", "The long URL(s) to shorten (single URL or JSON array)")

	// Mark the flag as required - Cobra will enforce this
	CreateCmd.MarkFlagRequired("url")

	// Add this command to the root command so it can be executed
	cmd.RootCmd.AddCommand(CreateCmd)
}
