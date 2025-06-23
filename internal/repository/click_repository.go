package repository

import (
	"fmt"

	"github.com/axellelanca/urlshortener/internal/models"
	"gorm.io/gorm"
)

// ClickRepository est une interface qui définit les méthodes d'accès aux données
type ClickRepository interface {
	CreateClick(click *models.Click) error
	CountClicksByLinkID(linkID uint) (int, error)
}

// GormClickRepository est l'implémentation de l'interface ClickRepository utilisant GORM.
type GormClickRepository struct {
	db *gorm.DB
}

// NewClickRepository crée et retourne une nouvelle instance de GormClickRepository.
func NewClickRepository(db *gorm.DB) *GormClickRepository {
	return &GormClickRepository{db: db}
}

// CreateClick insère un nouvel enregistrement de clic dans la base de données.
func (r *GormClickRepository) CreateClick(click *models.Click) error {
	if err := r.db.Create(click).Error; err != nil {
		return fmt.Errorf("failed to create click: %w", err)
	}
	return nil
}

// CountClicksByLinkID compte le nombre total de clics pour un ID de lien donné.
func (r *GormClickRepository) CountClicksByLinkID(linkID uint) (int, error) {
	var count int64
	if err := r.db.Model(&models.Click{}).Where("link_id = ?", linkID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count clicks for link ID %d: %w", linkID, err)
	}
	return int(count), nil
}
