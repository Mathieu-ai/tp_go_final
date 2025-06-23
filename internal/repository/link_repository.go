package repository

import (
	"fmt"

	"github.com/axellelanca/urlshortener/internal/models"
	"gorm.io/gorm"
)

// LinkRepository is an interface that defines data access methods for link operations.
// This interface abstracts the underlying database implementation, making the code
// more testable and maintainable by allowing easy mocking and database switching.
type LinkRepository interface {
	// CreateLink inserts a new shortened link into the database.
	// Used when users create new short URLs via API or CLI.
	CreateLink(link *models.Link) error

	// GetLinkByShortCode retrieves a link record using its unique short code.
	// This is the primary method used during URL redirection to find the target URL.
	GetLinkByShortCode(shortCode string) (*models.Link, error)

	// GetAllLinks retrieves all link records from the database.
	// Used by the URL monitor to check the health of all registered URLs.
	GetAllLinks() ([]models.Link, error)

	// CountClicksByLinkID returns the total number of clicks for a specific link.
	// Used for generating statistics and analytics reports.
	CountClicksByLinkID(linkID uint) (int, error)
}

// GormLinkRepository is the GORM-based implementation of LinkRepository interface.
// It uses GORM ORM to provide a high-level interface to the underlying SQLite database.
type GormLinkRepository struct {
	db *gorm.DB // GORM database connection instance for all database operations
}

// NewLinkRepository creates and returns a new instance of GormLinkRepository.
// This constructor function initializes the repository with the provided database connection.
// Parameters:
//   - db: GORM database connection to use for all repository operations
//
// Returns:
//   - *GormLinkRepository: configured repository instance ready for database operations
func NewLinkRepository(db *gorm.DB) *GormLinkRepository {
	return &GormLinkRepository{db: db}
}

// CreateLink inserts a new link record into the database.
// This method is called when users create new shortened URLs through the API or CLI.
// The link contains the short code, long URL, and creation timestamp.
// Parameters:
//   - link: pointer to the Link model containing short code, long URL, and metadata
//
// Returns:
//   - error: nil on success, or database error if insertion fails (e.g., duplicate short code)
func (r *GormLinkRepository) CreateLink(link *models.Link) error {
	if err := r.db.Create(link).Error; err != nil {
		return fmt.Errorf("failed to create link: %w", err)
	}
	return nil
}

// GetLinkByShortCode retrieves a link record from the database using its short code.
// This is the most frequently called method, used during URL redirection to find
// the original long URL associated with a short code (e.g., "abc123" -> "https://google.com").
// Parameters:
//   - shortCode: the unique short code identifier to search for
//
// Returns:
//   - *models.Link: pointer to the found link record with all its data
//   - error: gorm.ErrRecordNotFound if short code doesn't exist, or other database errors
func (r *GormLinkRepository) GetLinkByShortCode(shortCode string) (*models.Link, error) {
	var link models.Link
	// Use GORM's Where() to filter by short_code column and First() to get single result
	// First() returns ErrRecordNotFound if no matching record exists
	if err := r.db.Where("short_code = ?", shortCode).First(&link).Error; err != nil {
		return nil, err
	}
	return &link, nil
}

// GetAllLinks retrieves all link records from the database.
// This method is primarily used by the URL monitoring system to periodically
// check the health status of all registered URLs. It returns all links without pagination.
// Returns:
//   - []models.Link: slice containing all link records in the database
//   - error: nil on success, or database error if query fails
func (r *GormLinkRepository) GetAllLinks() ([]models.Link, error) {
	var links []models.Link
	// Use GORM's Find() to retrieve all records from the links table
	// Find() populates the slice with all matching records
	if err := r.db.Find(&links).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve all links: %w", err)
	}
	return links, nil
}

// CountClicksByLinkID counts the total number of clicks for a given link ID.
// This method provides analytics data by counting click records associated with a link.
// It performs a SQL COUNT query on the clicks table filtered by link_id.
// Parameters:
//   - linkID: the database ID of the link to count clicks for
//
// Returns:
//   - int: total number of clicks recorded for this link
//   - error: nil on success, or database error if query fails
func (r *GormLinkRepository) CountClicksByLinkID(linkID uint) (int, error) {
	var count int64
	// Use GORM's Model() to specify the clicks table and Count() for aggregation
	// The Where() clause filters to only count clicks for the specified link ID
	if err := r.db.Model(&models.Click{}).Where("link_id = ?", linkID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count clicks for link ID %d: %w", linkID, err)
	}
	// Convert int64 to int for consistency with interface return type
	return int(count), nil
}
