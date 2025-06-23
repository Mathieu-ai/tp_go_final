package api

import (
	"errors"
	"log"
	"net/http"
	"time"

	customerrors "github.com/axellelanca/urlshortener/internal/errors"
	"github.com/axellelanca/urlshortener/internal/models"
	"github.com/axellelanca/urlshortener/internal/services"
	"github.com/gin-gonic/gin"
)

// ClickEventsChannel is the global channel used to send click events
// This channel enables asynchronous processing of click analytics without blocking URL redirection
var ClickEventsChannel chan models.ClickEvent

// SetupRoutes configures all Gin API routes and injects necessary dependencies
// This function is the main routing configuration that sets up all HTTP endpoints
// Parameters:
//   - router: Gin engine instance to configure routes on
//   - linkService: business logic service for link operations
//   - bufferSize: size of the click events channel buffer for async processing
func SetupRoutes(router *gin.Engine, linkService *services.LinkService, bufferSize int) {
	// Initialize the global click events channel if it hasn't been created yet
	// This channel is used throughout the application for async click tracking
	if ClickEventsChannel == nil {
		ClickEventsChannel = make(chan models.ClickEvent, bufferSize)
	}

	// Health Check Route - used for monitoring service availability
	router.GET("/health", HealthCheckHandler)

	// API Routes Group - all business logic endpoints under /api/v1 prefix
	api := router.Group("/api/v1")
	{
		// POST endpoint for creating new shortened links (supports single and multiple URLs)
		api.POST("/links", CreateShortLinkHandler(linkService))
		// GET endpoint for retrieving click statistics for a specific short code
		api.GET("/links/:shortCode/stats", GetLinkStatsHandler(linkService))
	}

	// Redirection Route - handles the actual URL redirection at root level
	// This is where users access their short URLs (e.g., localhost:8080/abc123)
	router.GET("/:shortCode", RedirectHandler(linkService))
}

// HealthCheckHandler handles the /health route to verify service status
// This endpoint is typically used by load balancers and monitoring systems
func HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// CreateLinkRequest represents the JSON request body for creating one or multiple links
// This struct supports both backward compatibility (single URL) and new functionality (multiple URLs)
// Supports both single URL and multiple URLs formats:
// Single: {"long_url": "https://example.com"}
// Multiple: {"long_urls": ["https://example.com", "https://google.com"]}
type CreateLinkRequest struct {
	LongURL  string   `json:"long_url" binding:"omitempty,url"`       // Single URL (optional) - for backward compatibility
	LongURLs []string `json:"long_urls" binding:"omitempty,dive,url"` // Multiple URLs (optional) - new feature
}

// CreateLinkResponse represents the response for a single link creation
// This struct is used both for single URL responses and as elements in the results array for multiple URLs
type CreateLinkResponse struct {
	ShortCode    string `json:"short_code"`      // The generated short code (e.g., "abc123")
	LongURL      string `json:"long_url"`        // The original long URL that was shortened
	FullShortURL string `json:"full_short_url"`  // Complete shortened URL ready to use
	Success      bool   `json:"success"`         // Whether this particular URL was successfully shortened
	Error        string `json:"error,omitempty"` // Error message if shortening failed (omitted if successful)
}

// CreateLinksResponse represents the response for multiple link creation
// This provides both individual results and aggregate statistics
type CreateLinksResponse struct {
	Results []CreateLinkResponse `json:"results"` // Array of individual results for each URL processed
	Summary struct {
		Total      int `json:"total"`      // Total number of URLs that were attempted to be shortened
		Successful int `json:"successful"` // Number of URLs successfully shortened
		Failed     int `json:"failed"`     // Number of URLs that failed to be shortened
	} `json:"summary"` // Aggregate statistics for the batch operation
}

// CreateShortLinkHandler handles the creation of one or multiple shortened URLs
// This handler supports both single URL (backward compatibility) and multiple URLs (new feature)
// It automatically detects the request format and routes to appropriate processing logic
func CreateShortLinkHandler(linkService *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateLinkRequest

		// Attempt to bind the JSON request to the CreateLinkRequest struct
		// Gin will validate URL formats based on the binding tags
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
			return
		}

		// Determine if this is a single URL or multiple URLs request by collecting all provided URLs
		var urlsToProcess []string

		// Check if single URL is provided (backward compatibility path)
		if req.LongURL != "" {
			urlsToProcess = append(urlsToProcess, req.LongURL)
		}

		// Check if multiple URLs are provided (new feature path)
		if len(req.LongURLs) > 0 {
			urlsToProcess = append(urlsToProcess, req.LongURLs...)
		}

		// Validate that at least one URL was provided in the request
		if len(urlsToProcess) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Either 'long_url' or 'long_urls' must be provided"})
			return
		}

		// Route to appropriate processing logic based on the number of URLs
		if len(urlsToProcess) > 1 {
			// Process multiple URLs with detailed result tracking
			handleMultipleURLs(c, linkService, urlsToProcess)
		} else {
			// Process single URL with backward-compatible response format
			handleSingleURL(c, linkService, urlsToProcess[0])
		}
	}
}

// handleSingleURL processes a single URL request (maintains backward compatibility)
// This function preserves the original API response format for single URL requests
// ensuring existing clients continue to work without modification
func handleSingleURL(c *gin.Context, linkService *services.LinkService, longURL string) {
	// Call the LinkService to create the new shortened link
	// The service handles short code generation, collision detection, and database storage
	link, err := linkService.CreateLink(longURL)
	if err != nil {
		// Handle the specific case where we can't generate a unique short code
		// This can happen if the system is under heavy load or has many existing codes
		if errors.Is(err, customerrors.ErrShortCodeGenerationFailed) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Unable to generate unique short code. Please try again later."})
			return
		}
		// Handle any other unexpected errors during link creation
		log.Printf("Error creating link: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create short link"})
		return
	}

	// Return the short code and long URL in the original JSON response format
	// This maintains backward compatibility with existing API clients
	c.JSON(http.StatusCreated, gin.H{
		"short_code":     link.ShortCode,
		"long_url":       link.LongURL,
		"full_short_url": "http://localhost:8080/" + link.ShortCode, // TODO: Use cfg.Server.BaseURL for dynamic configuration
	})
}

// handleMultipleURLs processes multiple URLs request with comprehensive error handling
// This function provides detailed results for each URL and aggregate statistics
// It ensures partial success scenarios are handled gracefully
func handleMultipleURLs(c *gin.Context, linkService *services.LinkService, urls []string) {
	var results []CreateLinkResponse
	successful := 0
	failed := 0

	// Process each URL individually and track results
	// This allows some URLs to succeed even if others fail
	for _, longURL := range urls {
		// Initialize result structure for this specific URL
		result := CreateLinkResponse{
			LongURL: longURL, // Always include the original URL for traceability
		}

		// Attempt to create the short link for this URL
		link, err := linkService.CreateLink(longURL)
		if err != nil {
			// Handle error for this specific URL without affecting others
			result.Success = false
			if errors.Is(err, customerrors.ErrShortCodeGenerationFailed) {
				result.Error = "Unable to generate unique short code"
			} else {
				result.Error = "Failed to create short link"
				log.Printf("Error creating link for %s: %v", longURL, err)
			}
			failed++
		} else {
			// Success case - populate all success fields
			result.Success = true
			result.ShortCode = link.ShortCode
			result.FullShortURL = "http://localhost:8080/" + link.ShortCode // TODO: Use cfg.Server.BaseURL for dynamic configuration
			successful++
		}

		// Add this result to the collection regardless of success/failure
		results = append(results, result)
	}

	// Create comprehensive response with individual results and summary statistics
	response := CreateLinksResponse{
		Results: results, // Detailed results for each URL
		Summary: struct {
			Total      int `json:"total"`
			Successful int `json:"successful"`
			Failed     int `json:"failed"`
		}{
			Total:      len(urls),  // Total number of URLs processed
			Successful: successful, // Count of successfully shortened URLs
			Failed:     failed,     // Count of URLs that failed to be shortened
		},
	}

	// Determine appropriate HTTP status code based on operation results
	var statusCode int
	if failed == 0 {
		statusCode = http.StatusCreated // All URLs successful (201)
	} else if successful == 0 {
		statusCode = http.StatusBadRequest // All URLs failed (400)
	} else {
		statusCode = http.StatusMultiStatus // Mixed results - some success, some failure (207)
	}

	c.JSON(statusCode, response)
}

// RedirectHandler handles the redirection from a short URL to the original long URL
// This is the core functionality that users experience when clicking short links
// It also triggers asynchronous click tracking for analytics without blocking the redirect
func RedirectHandler(linkService *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the short code from the URL path parameter
		// This comes from routes like "/:shortCode" where shortCode is the generated identifier
		shortCode := c.Param("shortCode")

		// Retrieve the original long URL associated with this short code
		// This is the database lookup that resolves the short code to its target
		link, err := linkService.GetLinkByShortCode(shortCode)
		if err != nil {
			// Handle the case where the short code doesn't exist in our database
			if errors.Is(err, customerrors.ErrShortCodeNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Short URL not found"})
				return
			}
			// Handle any other unexpected database or service errors
			log.Printf("Error retrieving link for %s: %v", shortCode, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		// Create a ClickEvent with all relevant information for analytics
		// This captures the context of the click for later analysis
		clickEvent := models.ClickEvent{
			LinkID:    link.ID,                   // Database ID of the link that was clicked
			Timestamp: time.Now(),                // Exact time when the click occurred
			UserAgent: c.GetHeader("User-Agent"), // Browser/client information for device analytics
			IPAddress: c.ClientIP(),              // Client IP address for geographic analytics
		}

		// Send the ClickEvent to the processing channel using non-blocking select
		// This ensures that click tracking never delays the user's redirect experience
		select {
		case ClickEventsChannel <- clickEvent:
			// Event successfully queued for asynchronous processing
			log.Printf("Click event queued for link %s (ID: %d)", shortCode, link.ID)
		default:
			// Channel buffer is full - we drop the event rather than blocking the user
			// This prioritizes user experience over perfect analytics in high-load scenarios
			log.Printf("WARNING: ClickEventsChannel is full, dropping click event for %s (ID: %d)", shortCode, link.ID)
		}

		// Perform the HTTP 302 redirect to the original long URL
		// This is the primary function - getting the user to their intended destination
		c.Redirect(http.StatusFound, link.LongURL)
	}
}

// GetLinkStatsHandler handles the retrieval of statistics for a specific link
// This endpoint provides analytics data including click counts and link metadata
// Used by both the API and CLI to display usage statistics
func GetLinkStatsHandler(linkService *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the short code from the URL path parameter
		shortCode := c.Param("shortCode")

		// Call the LinkService to get both link information and aggregated click statistics
		// This single call provides all the data needed for a comprehensive stats response
		link, totalClicks, err := linkService.GetLinkStats(shortCode)
		if err != nil {
			// Handle the case where the requested short code doesn't exist
			if errors.Is(err, customerrors.ErrShortCodeNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Short URL not found"})
				return
			}
			// Handle any other database or service errors during stats retrieval
			log.Printf("Error retrieving stats for %s: %v", shortCode, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		// Return comprehensive statistics in JSON format
		// Includes link metadata and usage analytics
		c.JSON(http.StatusOK, gin.H{
			"short_code":   link.ShortCode,                               // The short code identifier
			"long_url":     link.LongURL,                                 // The original long URL
			"total_clicks": totalClicks,                                  // Aggregate count of all clicks
			"created_at":   link.CreatedAt.Format("2006-01-02 15:04:05"), // Human-readable creation timestamp
		})
	}
}
