package models

import "time"

// Link represents a shortened URL link stored in the database.
// This struct uses GORM tags to define database schema and constraints.
type Link struct {
	// ID is the primary key with auto-increment functionality
	ID uint `gorm:"primaryKey"`

	// ShortCode is the unique identifier for the shortened URL (e.g., "abc123")
	// - uniqueIndex: ensures no duplicate short codes can exist
	// - size:10: limits the column size to 10 characters in the database
	// - not null: prevents empty short codes
	ShortCode string `gorm:"uniqueIndex;size:10;not null"`

	// LongURL stores the original URL that the short code redirects to
	// - not null: ensures every link has a destination URL
	LongURL string `gorm:"not null"`

	// CreatedAt automatically stores the timestamp when the record is created
	// - autoCreateTime: GORM automatically sets this field when inserting
	CreatedAt time.Time `gorm:"autoCreateTime"`
}
