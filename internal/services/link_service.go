// Package services contains the business logic layer for the URL shortener application
package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	"gorm.io/gorm"

	"github.com/axellelanca/urlshortener/internal/models"
	"github.com/axellelanca/urlshortener/internal/repository"
)

// charset defines the character set used for generating short codes.
// Uses alphanumeric characters (both cases) for a total of 62 possible characters.
// This gives us 62^6 = ~56 billion possible combinations for 6-character codes.
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// LinkService provides business logic methods for managing shortened links.
// It acts as an intermediary between the HTTP handlers and the data repository.
type LinkService struct {
	linkRepo repository.LinkRepository // Repository interface for database operations
}

// NewLinkService creates and returns a new instance of LinkService.
// This is a constructor function following Go conventions.
func NewLinkService(linkRepo repository.LinkRepository) *LinkService {
	return &LinkService{
		linkRepo: linkRepo,
	}
}

// GenerateShortCode generates a cryptographically secure random short code.
// Parameters:
//   - length: the desired length of the generated code
//
// Returns:
//   - string: the generated random code
//   - error: any error that occurred during generation
func (s *LinkService) GenerateShortCode(length int) (string, error) {
	// Create a byte slice to hold our random characters
	code := make([]byte, length)

	// Generate each character randomly from our charset
	for i := range code {
		// Use crypto/rand for cryptographically secure random numbers
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		// Map the random number to a character from our charset
		code[i] = charset[num.Int64()]
	}
	return string(code), nil
}

// CreateLink creates a new shortened link with collision detection and retry logic.
// This method ensures that each generated short code is unique in the database.
// Parameters:
//   - longURL: the original URL to be shortened
//
// Returns:
//   - *models.Link: the created link with its short code
//   - error: any error that occurred during creation
func (s *LinkService) CreateLink(longURL string) (*models.Link, error) {
	var shortCode string
	maxRetries := 5 // Maximum number of attempts to generate a unique code

	// Retry loop to handle short code collisions
	for i := 0; i < maxRetries; i++ {
		// Generate a new 6-character short code
		code, err := s.GenerateShortCode(6)
		if err != nil {
			return nil, fmt.Errorf("failed to generate short code: %w", err)
		}

		// Check if the generated code already exists in the database
		_, err = s.linkRepo.GetLinkByShortCode(code)
		if err != nil {
			// If the error is 'record not found', the code is unique and we can use it
			if errors.Is(err, gorm.ErrRecordNotFound) {
				shortCode = code
				break // Exit the retry loop - we found a unique code
			}
			// If it's any other database error, return it immediately
			return nil, fmt.Errorf("database error checking short code uniqueness: %w", err)
		}

		// If we reach here, the code already exists (collision detected)
		log.Printf("Short code '%s' already exists, retrying generation (%d/%d)...", code, i+1, maxRetries)
	}

	// If we exhausted all retries without finding a unique code
	if shortCode == "" {
		return nil, errors.New("failed to generate unique short code after maximum retries")
	}

	// Create a new Link instance with the generated unique short code
	link := &models.Link{
		ShortCode: shortCode,
		LongURL:   longURL,
		CreatedAt: time.Now(), // Set creation timestamp
	}

	// Persist the new link to the database via the repository layer
	if err := s.linkRepo.CreateLink(link); err != nil {
		return nil, fmt.Errorf("failed to create link: %w", err)
	}

	return link, nil
}

// GetLinkByShortCode retrieves a link from the database using its short code.
// This is the primary method used during URL redirection.
// Parameters:
//   - shortCode: the short code to look up
//
// Returns:
//   - *models.Link: the found link
//   - error: gorm.ErrRecordNotFound if not found, or other database errors
func (s *LinkService) GetLinkByShortCode(shortCode string) (*models.Link, error) {
	link, err := s.linkRepo.GetLinkByShortCode(shortCode)
	if err != nil {
		return nil, err
	}
	return link, nil
}

// GetLinkStats retrieves statistics for a given short code.
// This includes the link details and the total number of clicks recorded.
// Parameters:
//   - shortCode: the short code to get statistics for
//
// Returns:
//   - *models.Link: the link information
//   - int: total number of clicks
//   - error: any error that occurred during retrieval
func (s *LinkService) GetLinkStats(shortCode string) (*models.Link, int, error) {
	// First, retrieve the link by its shortCode
	link, err := s.linkRepo.GetLinkByShortCode(shortCode)
	if err != nil {
		return nil, 0, err
	}

	// Count the total number of clicks for this link's ID
	totalClicks, err := s.linkRepo.CountClicksByLinkID(link.ID)
	if err != nil {
		return nil, 0, err
	}

	return link, totalClicks, nil
}
