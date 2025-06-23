package models

import "time"

// Link représente un lien raccourci dans la base de données.
type Link struct {
	ID        uint   `gorm:"primaryKey"`
	ShortCode string `gorm:"uniqueIndex;size:10;not null"`
	LongURL   string `gorm:"not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}
