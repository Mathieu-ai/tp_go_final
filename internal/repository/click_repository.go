package repository

import (
	"fmt"

	"github.com/axellelanca/urlshortener/internal/models"
	"gorm.io/gorm"
)

// ClickRepository is an interface that defines data access methods for click operations.
// This interface abstracts the underlying database implementation, making it easier to test
// and potentially switch database implementations in the future.
type ClickRepository interface {
	// CreateClick inserts a new click record into the database.
	// This method is used by background workers to persistently store click events.
	CreateClick(click *models.Click) error

	// CountClicksByLinkID returns the total number of clicks for a specific link ID.
	// This is used for analytics and statistics generation.
	CountClicksByLinkID(linkID uint) (int, error)
}

// GormClickRepository is the GORM-based implementation of the ClickRepository interface.
// It uses GORM ORM to interact with the underlying database (SQLite in this case).
type GormClickRepository struct {
	db *gorm.DB // GORM database connection instance
}

// NewClickRepository creates and returns a new instance of GormClickRepository.
// This is a constructor function that initializes the repository with a database connection.
// Parameters:
//   - db: GORM database connection to use for all operations
//
// Returns:
//   - *GormClickRepository: configured repository instance ready for use
func NewClickRepository(db *gorm.DB) *GormClickRepository {
	return &GormClickRepository{db: db}
}

// CreateClick inserts a new click record into the database.
// This method is typically called by background worker goroutines to avoid blocking
// the main URL redirection process. The click data includes timestamp, user agent,
// IP address, and the associated link ID for analytics purposes.
// Parameters:
//   - click: pointer to the Click model containing all click event data
//
// Returns:
//   - error: nil on success, or database error if insertion fails
func (r *GormClickRepository) CreateClick(click *models.Click) error {
	if err := r.db.Create(click).Error; err != nil {
		return fmt.Errorf("failed to create click: %w", err)
	}
	return nil
}

// CountClicksByLinkID counts the total number of clicks for a given link ID.
// This method is used for generating statistics and analytics reports.
// It performs a SQL COUNT query filtered by the link_id column.
// Parameters:
//   - linkID: the database ID of the link to count clicks for
//
// Returns:
//   - int: total number of clicks recorded for this link
//   - error: nil on success, or database error if query fails
func (r *GormClickRepository) CountClicksByLinkID(linkID uint) (int, error) {
	var count int64
	// Use GORM's Model() to specify the table and Count() to perform aggregation
	// The Where() clause filters records to only count clicks for the specified link
	if err := r.db.Model(&models.Click{}).Where("link_id = ?", linkID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count clicks for link ID %d: %w", linkID, err)
	}
	// Convert int64 to int for consistency with interface return type
	return int(count), nil
}
