package models

import "time"

// Click represents a click event on a shortened URL stored in the database.
// This model tracks user interactions for analytics and statistics purposes.
type Click struct {
	// ID is the primary key with auto-increment functionality
	ID uint `gorm:"primaryKey"`

	// LinkID is the foreign key referencing the Link that was clicked
	// - index: creates a database index for efficient queries when counting clicks per link
	LinkID uint `gorm:"index"`

	// Link establishes the GORM relationship to the Link model
	// - foreignKey:LinkID: explicitly defines LinkID as the foreign key field
	// This allows us to access link.Link.ShortCode, link.Link.LongURL, etc.
	Link Link `gorm:"foreignKey:LinkID"`

	// Timestamp records the exact moment when the click occurred
	// Used for analytics, tracking click patterns, and temporal analysis
	Timestamp time.Time

	// UserAgent stores the browser/client information from the HTTP request
	// - size:255: limits the database column to 255 characters
	// Useful for analytics: browser type, mobile vs desktop, OS information
	UserAgent string `gorm:"size:255"`

	// IPAddress stores the IP address of the user who clicked the link
	// - size:50: sufficient for both IPv4 and IPv6 addresses
	// Used for geographical analytics and potential abuse detection
	IPAddress string `gorm:"size:50"`
}

// ClickEvent represents a raw click event intended to be passed through channels.
// This lightweight struct is used for asynchronous processing between goroutines.
// It contains only the essential data needed to create a Click record later.
type ClickEvent struct {
	LinkID    uint      // The ID of the link that was clicked
	Timestamp time.Time // When the click occurred
	UserAgent string    // Browser/client information
	IPAddress string    // User's IP address
}
