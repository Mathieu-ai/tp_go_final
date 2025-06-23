package cli

import (
	"fmt"
	"log"
	"os"

	"github.com/axellelanca/urlshortener/cmd"
	"github.com/axellelanca/urlshortener/internal/config"
	"github.com/axellelanca/urlshortener/internal/repository"
	"github.com/axellelanca/urlshortener/internal/services"
	"github.com/spf13/cobra"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// StatsCmd représente la commande 'stats'
var StatsCmd = &cobra.Command{
	Use:   "stats [short-code]",
	Short: "Get statistics for a short URL",
	Long:  `Get click statistics for the provided short code.`,
	Args:  cobra.ExactArgs(1),
	Run:   runStats,
}

func init() {
	cmd.RootCmd.AddCommand(StatsCmd)
}

// runStats exécute la logique pour la commande stats
func runStats(cmd *cobra.Command, args []string) {
	shortCode := args[0]

	if shortCode == "" {
		fmt.Println("Error: short code is required")
		os.Exit(1)
	}

	// Charger la configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Échec du chargement de la configuration : %v", err)
	}

	// Initialiser la base de données
	db, err := gorm.Open(sqlite.Open(cfg.Database.Name), &gorm.Config{})
	if err != nil {
		log.Fatalf("Échec de la connexion à la base de données : %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("FATAL: Échec de l'obtention de la base de données SQL sous-jacente: %v", err)
	}
	defer sqlDB.Close()

	// Initialiser les repositories et services nécessaires
	linkRepo := repository.NewLinkRepository(db)
	linkService := services.NewLinkService(linkRepo)

	// Appeler GetLinkStats pour récupérer le lien et ses statistiques.
	link, totalClicks, err := linkService.GetLinkStats(shortCode)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			fmt.Printf("Error: Short code '%s' not found\n", shortCode)
		} else {
			fmt.Printf("Error retrieving statistics: %v\n", err)
		}
		os.Exit(1)
	}

	// Afficher les résultats
	fmt.Printf("Statistiques pour le code court: %s\n", shortCode)
	fmt.Printf("URL longue: %s\n", link.LongURL)
	fmt.Printf("Total de clics: %d\n", totalClicks)
	fmt.Printf("Date de création: %s\n", link.CreatedAt.Format("2006-01-02 15:04:05"))
}
