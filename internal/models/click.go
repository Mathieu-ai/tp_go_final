package models

import "time"

// Click représente un événement de clic sur un lien raccourci.
// GORM utilisera ces tags pour créer la table 'clicks'.
type Click struct {
	ID        uint      `gorm:"primaryKey"`        // Clé primaire
	LinkID    uint      `gorm:"index"`             // Clé étrangère vers la table 'links', indexée pour des requêtes efficaces
	Link      Link      `gorm:"foreignKey:LinkID"` // Relation GORM: indique que LinkID est une FK vers le champ ID de Link
	Timestamp time.Time // Horodatage précis du clic
	UserAgent string    `gorm:"size:255"` // User-Agent de l'utilisateur qui a cliqué (informations sur le navigateur/OS)
	IPAddress string    `gorm:"size:50"`  // Adresse IP de l'utilisateur
}

// TODO créer la struct pour ClickEvent
// ClickEvent représente un événement de clic brut, destiné à être passé via un channel
// Ce n'est pas un modèle GORM direct.
// Un Click event a un LinkID(uint), un Timestamp (Time.Time), un UserAgent (string) et un IP (stringà
