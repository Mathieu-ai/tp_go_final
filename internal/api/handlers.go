// internal/api/handlers.go
package api

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/axellelanca/urlshortener/internal/models"
	"github.com/axellelanca/urlshortener/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ClickEventsChannel est le channel global utilisé pour envoyer les événements de clic
var ClickEventsChannel chan models.ClickEvent

// SetupRoutes configure toutes les routes de l'API Gin et injecte les dépendances nécessaires
func SetupRoutes(router *gin.Engine, linkService *services.LinkService, bufferSize int) {
	// Initialiser le channel si ce n'est pas déjà fait
	if ClickEventsChannel == nil {
		ClickEventsChannel = make(chan models.ClickEvent, bufferSize)
	}

	// Route de Health Check
	router.GET("/health", HealthCheckHandler)

	// Routes de l'API
	api := router.Group("/api/v1")
	{
		api.POST("/links", CreateShortLinkHandler(linkService))
		api.GET("/links/:shortCode/stats", GetLinkStatsHandler(linkService))
	}

	// Route de Redirection (au niveau racine pour les short codes)
	router.GET("/:shortCode", RedirectHandler(linkService))
}

// HealthCheckHandler gère la route /health pour vérifier l'état du service.
func HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// CreateLinkRequest représente le corps de la requête JSON pour la création d'un lien.
type CreateLinkRequest struct {
	LongURL string `json:"long_url" binding:"required,url"`
}

// CreateShortLinkHandler gère la création d'une URL courte.
func CreateShortLinkHandler(linkService *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateLinkRequest

		// Tente de lier le JSON de la requête à la structure CreateLinkRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
			return
		}

		// Appeler le LinkService pour créer le nouveau lien
		link, err := linkService.CreateLink(req.LongURL)
		if err != nil {
			log.Printf("Error creating link: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create short link"})
			return
		}

		// Retourne le code court et l'URL longue dans la réponse JSON
		c.JSON(http.StatusCreated, gin.H{
			"short_code":     link.ShortCode,
			"long_url":       link.LongURL,
			"full_short_url": "http://localhost:8080/" + link.ShortCode, // TODO: Utiliser cfg.Server.BaseURL
		})
	}
}

// RedirectHandler gère la redirection d'une URL courte vers l'URL longue et l'enregistrement asynchrone des clics.
func RedirectHandler(linkService *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Récupère le shortCode de l'URL
		shortCode := c.Param("shortCode")

		// Récupérer l'URL longue associée au shortCode
		link, err := linkService.GetLinkByShortCode(shortCode)
		if err != nil {
			// Si le lien n'est pas trouvé, retourner HTTP 404 Not Found
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Short URL not found"})
				return
			}
			// Gérer d'autres erreurs potentielles
			log.Printf("Error retrieving link for %s: %v", shortCode, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		// Créer un ClickEvent avec les informations pertinentes
		clickEvent := models.ClickEvent{
			LinkID:    link.ID,
			Timestamp: time.Now(),
			UserAgent: c.GetHeader("User-Agent"),
			IPAddress: c.ClientIP(),
		}

		// Envoyer le ClickEvent dans le ClickEventsChannel avec multiplexage
		select {
		case ClickEventsChannel <- clickEvent:
			// Événement envoyé avec succès
		default:
			// Channel plein, on abandonne l'événement
			log.Printf("Warning: ClickEventsChannel is full, dropping click event for %s.", shortCode)
		}

		// Effectuer la redirection HTTP 302 vers l'URL longue
		c.Redirect(http.StatusFound, link.LongURL)
	}
}

// GetLinkStatsHandler gère la récupération des statistiques pour un lien spécifique.
func GetLinkStatsHandler(linkService *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Récupère le shortCode de l'URL
		shortCode := c.Param("shortCode")

		// Appeler le LinkService pour obtenir le lien et le nombre total de clics
		link, totalClicks, err := linkService.GetLinkStats(shortCode)
		if err != nil {
			// Gérer le cas où le lien n'est pas trouvé
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Short URL not found"})
				return
			}
			// Gérer d'autres erreurs
			log.Printf("Error retrieving stats for %s: %v", shortCode, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		// Retourne les statistiques dans la réponse JSON
		c.JSON(http.StatusOK, gin.H{
			"short_code":   link.ShortCode,
			"long_url":     link.LongURL,
			"total_clicks": totalClicks,
		})
	}
}
