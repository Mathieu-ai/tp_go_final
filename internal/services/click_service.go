package services

import (
	"fmt"

	"github.com/axellelanca/urlshortener/internal/models"
	"github.com/axellelanca/urlshortener/internal/repository"
)

// ClickService provides business logic methods for managing click events.
// This service handles the recording and querying of user interactions with short links.
type ClickService struct {
	clickRepo repository.ClickRepository // Repository interface for click data operations
}

// NewClickService creates and returns a new instance of ClickService.
// This is a constructor function following Go conventions.
func NewClickService(clickRepo repository.ClickRepository) *ClickService {
	return &ClickService{
		clickRepo: clickRepo,
	}
}

// RecordClick saves a new click event to the database.
// This method is typically called asynchronously by worker goroutines
// to avoid blocking the URL redirection process.
// Parameters:
//   - click: the click event data to record
//
// Returns:
//   - error: any error that occurred during recording
func (s *ClickService) RecordClick(click *models.Click) error {
	if err := s.clickRepo.CreateClick(click); err != nil {
		return fmt.Errorf("failed to record click: %w", err)
	}
	return nil
}

// GetClicksCountByLinkID retrieves the total number of clicks for a specific link.
// This is used for generating statistics and analytics.
// Parameters:
//   - linkID: the database ID of the link to count clicks for
//
// Returns:
//   - int: the total number of clicks
//   - error: any error that occurred during counting
func (s *ClickService) GetClicksCountByLinkID(linkID uint) (int, error) {
	count, err := s.clickRepo.CountClicksByLinkID(linkID)
	if err != nil {
		return 0, err
	}
	return count, nil
}
