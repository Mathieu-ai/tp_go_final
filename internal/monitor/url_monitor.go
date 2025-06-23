package monitor

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/axellelanca/urlshortener/internal/repository"
)

// UrlMonitor manages periodic monitoring of long URLs to check their accessibility.
// It maintains a state map to track URL status changes and notify when they occur.
type UrlMonitor struct {
	linkRepo    repository.LinkRepository // Repository to fetch all links from database
	interval    time.Duration             // How often to check URLs (e.g., every 30 seconds)
	knownStates map[uint]bool             // Cache of previous URL states (ID -> accessible/not accessible)
	mu          sync.Mutex                // Protects concurrent access to knownStates map
	httpClient  *http.Client              // HTTP client for making requests
}

// NewUrlMonitor creates and returns a new instance of UrlMonitor.
// interval parameter determines how frequently URLs will be checked.
func NewUrlMonitor(linkRepo repository.LinkRepository, interval time.Duration) *UrlMonitor {
	return &UrlMonitor{
		linkRepo:    linkRepo,
		interval:    interval,
		knownStates: make(map[uint]bool),                     // Initialize empty state map
		httpClient:  &http.Client{Timeout: 10 * time.Second}, // Initialize HTTP client with timeout
	}
}

// Start launches the periodic URL monitoring loop.
// This is a blocking function that runs indefinitely until the program stops.
func (m *UrlMonitor) Start() {
	log.Printf("[MONITOR] Starting URL monitor with interval of %v...", m.interval)
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	// Execute an immediate check on startup before waiting for the first tick
	m.checkUrls()

	// Main monitoring loop - runs every 'interval' duration
	for range ticker.C {
		m.checkUrls()
	}
}

// checkUrls performs a status check on all registered long URLs.
// It compares current state with previous state and logs any changes.
func (m *UrlMonitor) checkUrls() {
	log.Println("[MONITOR] Starting URL status verification...")

	// Fetch all active long URLs from the repository
	links, err := m.linkRepo.GetAllLinks()
	if err != nil {
		log.Printf("[MONITOR] ERROR retrieving links for monitoring: %v", err)
		return
	}

	// Iterate through each link and check its current accessibility
	for _, link := range links {
		// Test if the URL is currently accessible via HTTP request
		currentState := m.isUrlAccessible(link.LongURL)

		// Thread-safe access to the state map since multiple goroutines might access it
		m.mu.Lock()
		previousState, exists := m.knownStates[link.ID] // Check if we've seen this URL before
		m.knownStates[link.ID] = currentState           // Update the state cache
		m.mu.Unlock()

		// If this is the first time checking this link, just log the initial state
		if !exists {
			log.Printf("[MONITOR] Initial state for link %s (%s): %s",
				link.ShortCode, link.LongURL, formatState(currentState))
			continue
		}

		// Compare current state with previous state to detect changes
		// This is where we detect if a URL went from working to broken or vice versa
		if currentState != previousState {
			log.Printf("[NOTIFICATION] Link %s (%s) changed from %s to %s!",
				link.ShortCode, link.LongURL, formatState(previousState), formatState(currentState))
		}
	}
	log.Println("[MONITOR] URL status verification completed.")
}

// isUrlAccessible performs an HTTP HEAD request to check if a URL is accessible.
// Returns true if the URL responds with a successful HTTP status code (2xx or 3xx).
func (m *UrlMonitor) isUrlAccessible(url string) bool {
	// Set a timeout to prevent hanging on slow/unresponsive URLs
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create HTTP HEAD request (faster than GET since we don't need the response body)
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		log.Printf("[MONITOR] Error creating request for URL '%s': %v", url, err)
		return false
	}

	// Execute the HTTP request
	resp, err := m.httpClient.Do(req)
	if err != nil {
		log.Printf("[MONITOR] Error accessing URL '%s': %v", url, err)
		return false
	}
	defer resp.Body.Close()

	// Consider URLs accessible if they return 2xx (success) or 3xx (redirect) status codes
	// 4xx (client error) and 5xx (server error) are considered inaccessible
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

// formatState is a utility function to make the state more readable in logs.
// Converts boolean accessibility state to human-readable string.
func formatState(accessible bool) string {
	if accessible {
		return "ACCESSIBLE"
	}
	return "INACCESSIBLE"
}
