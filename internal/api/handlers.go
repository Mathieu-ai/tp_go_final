// internal/api/handlers.go
package api

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/axellelanca/urlshortener/internal/models"
	"github.com/axellelanca/urlshortener/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ClickEventsChannel is the global channel used to send click events to background workers
// This channel decouples URL redirection from click recording for better performance
var ClickEventsChannel chan models.ClickEvent

// SetupRoutes configures all API routes for the Gin router and injects necessary dependencies
// This is where we define all the HTTP endpoints and their corresponding handlers
func SetupRoutes(router *gin.Engine, linkService *services.LinkService, bufferSize int) {
	// Initialize the click events channel if not already done
	// This channel is used for asynchronous click event processing
	if ClickEventsChannel == nil {
		ClickEventsChannel = make(chan models.ClickEvent, bufferSize)
	}

	// Health check route - used by load balancers and monitoring systems
	router.GET("/health", HealthCheckHandler)

	// API routes group with version prefix for better API versioning
	api := router.Group("/api/v1")
	{
		// POST endpoint to create new shortened URLs
		api.POST("/links", CreateShortLinkHandler(linkService))

		// GET endpoint to retrieve statistics for a specific short code
		api.GET("/links/:shortCode/stats", GetLinkStatsHandler(linkService))
	}

	// Redirection route at root level for clean short URLs (e.g., domain.com/abc123)
	// This must be at root level to avoid /api/v1 prefix in shortened URLs
	router.GET("/:shortCode", RedirectHandler(linkService))
}

// HealthCheckHandler handles the /health route for service health verification
// Used by monitoring systems, load balancers, and orchestration platforms
func HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// CreateLinkRequest represents the JSON request body for creating a new shortened link
// The binding tags enable automatic validation and JSON unmarshaling
type CreateLinkRequest struct {
	LongURL string `json:"long_url" binding:"required,url"` // URL validation ensures valid format
}

// CreateShortLinkHandler handles the creation of new shortened URLs
// This is a factory function that returns a Gin handler with injected dependencies
func CreateShortLinkHandler(linkService *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateLinkRequest

		// Attempt to bind JSON request body to our struct with automatic validation
		// Gin will validate the URL format and required fields automatically
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
			return
		}

		// Call the LinkService to create the new shortened link
		// This handles business logic like generating unique codes and database storage
		link, err := linkService.CreateLink(req.LongURL)
		if err != nil {
			log.Printf("Error creating link: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create short link"})
			return
		}

		// Return the created link details in JSON format
		// TODO: Use cfg.Server.BaseURL instead of hardcoded localhost
		c.JSON(http.StatusCreated, gin.H{
			"short_code":     link.ShortCode,
			"long_url":       link.LongURL,
			"full_short_url": "http://localhost:8080/" + link.ShortCode,
		})
	}
}

// RedirectHandler handles URL redirection and asynchronous click tracking
// This is the core functionality that makes shortened URLs work
func RedirectHandler(linkService *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the short code from the URL path parameter
		shortCode := c.Param("shortCode")

		// Look up the original long URL associated with this short code
		link, err := linkService.GetLinkByShortCode(shortCode)
		if err != nil {
			// Handle case where short code doesn't exist in database
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Short URL not found"})
				return
			}
			// Handle other potential database errors
			log.Printf("Error retrieving link for %s: %v", shortCode, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		// Create a click event with relevant information for analytics
		// This captures user interaction data for statistics and monitoring
		clickEvent := models.ClickEvent{
			LinkID:    link.ID,                   // Which link was clicked
			Timestamp: time.Now(),                // When the click occurred
			UserAgent: c.GetHeader("User-Agent"), // Browser/client information
			IPAddress: c.ClientIP(),              // User's IP address for analytics
		}

		// Send click event to background workers using non-blocking channel operation
		// If channel is full, we drop the event to avoid blocking the redirect
		select {
		case ClickEventsChannel <- clickEvent:
			// Click event successfully queued for processing
		default:
			// Channel is full, log warning but continue with redirect
			log.Printf("Warning: ClickEventsChannel is full, dropping click event for %s.", shortCode)
		}

		// Perform HTTP 302 redirect to the original long URL
		// This is the main functionality that users expect from a URL shortener
		c.Redirect(http.StatusFound, link.LongURL)
	}
}

// GetLinkStatsHandler handles retrieving statistics for a specific shortened link
// This provides analytics data about how many times a link has been clicked
func GetLinkStatsHandler(linkService *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the short code from the URL path parameter
		shortCode := c.Param("shortCode")

		// Call LinkService to get link details and click statistics
		// This returns both the link information and aggregated click count
		link, totalClicks, err := linkService.GetLinkStats(shortCode)
		if err != nil {
			// Handle case where the short code doesn't exist
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Short URL not found"})
				return
			}
			// Handle other database or service errors
			log.Printf("Error retrieving stats for %s: %v", shortCode, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		// Return statistics in JSON format for client consumption
		c.JSON(http.StatusOK, gin.H{
			"short_code":   link.ShortCode,
			"long_url":     link.LongURL,
			"total_clicks": totalClicks,
		})
	}
}
