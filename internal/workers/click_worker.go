package workers

import (
	"log"

	"github.com/axellelanca/urlshortener/internal/models"
	"github.com/axellelanca/urlshortener/internal/repository"
)

// StartClickWorkers launches a pool of worker goroutines to process click events asynchronously.
// This implements the worker pool pattern to handle high-volume click tracking without blocking.
// Parameters:
//   - workerCount: number of concurrent workers to spawn
//   - clickEventsChan: channel that receives click events to be processed
//   - clickRepo: repository interface for persisting clicks to database
func StartClickWorkers(workerCount int, clickEventsChan <-chan models.ClickEvent, clickRepo repository.ClickRepository) {
	log.Printf("Starting %d click worker(s)...", workerCount)

	// Spawn the specified number of worker goroutines
	// Each worker will listen on the same channel and process events concurrently
	for i := 0; i < workerCount; i++ {
		go clickWorker(clickEventsChan, clickRepo)
	}
}

// clickWorker is the function executed by each worker goroutine.
// It continuously listens for click events on the channel and processes them.
// When the channel is closed, the worker will exit gracefully.
func clickWorker(clickEventsChan <-chan models.ClickEvent, clickRepo repository.ClickRepository) {
	// Range over the channel - this will block until events arrive
	// When the channel is closed, the loop will exit and the goroutine will terminate
	for event := range clickEventsChan {
		// Convert the ClickEvent (which might be a lightweight event struct)
		// into a full Click model that matches our database schema
		click := &models.Click{
			LinkID:    event.LinkID,    // Which shortened link was clicked
			Timestamp: event.Timestamp, // When the click occurred
			UserAgent: event.UserAgent, // Browser/client information for analytics
			IPAddress: event.IPAddress, // Client IP for geolocation/analytics
		}

		// Persist the click to the database via the repository
		// This is the actual database write operation
		if err := clickRepo.CreateClick(click); err != nil {
			// Log error but don't crash - we want to continue processing other clicks
			// In production, you might want to add retry logic or dead letter queues
			log.Printf("ERROR: Failed to save click for LinkID %d (UserAgent: %s, IP: %s): %v",
				event.LinkID, event.UserAgent, event.IPAddress, err)
		} else {
			// Success case - click was recorded successfully
			log.Printf("Click recorded successfully for LinkID %d", event.LinkID)
		}
	}
	// Worker exits when channel is closed - this happens during graceful shutdown
}
