package cli

import (
	"fmt"
	"log"
	"os"

	"github.com/axellelanca/urlshortener/cmd"
	"github.com/axellelanca/urlshortener/internal/config"
	"github.com/axellelanca/urlshortener/internal/repository"
	"github.com/axellelanca/urlshortener/internal/services"
	"github.com/glebarez/sqlite"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

var shortCodeFlag string

// StatsCmd représente la commande 'stats'
var StatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Get statistics for a short URL",
	Long:  `Get click statistics for the provided short code.`,
	Run:   runStats,
}

func init() {
	// Définir le flag --code pour la commande stats
	StatsCmd.Flags().StringVar(&shortCodeFlag, "code", "", "The short code to get statistics for")
	StatsCmd.MarkFlagRequired("code")

	cmd.RootCmd.AddCommand(StatsCmd)
}

// runStats exécute la logique pour la commande stats
func runStats(cmd *cobra.Command, args []string) {
	if shortCodeFlag == "" {
		fmt.Println("Error: --code flag is required")
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
	link, totalClicks, err := linkService.GetLinkStats(shortCodeFlag)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			fmt.Printf("Error: Short code '%s' not found\n", shortCodeFlag)
		} else {
			fmt.Printf("Error retrieving statistics: %v\n", err)
		}
		os.Exit(1)
	}

	// Afficher les résultats
	fmt.Printf("Statistiques pour le code court: %s\n", shortCodeFlag)
	fmt.Printf("URL longue: %s\n", link.LongURL)
	fmt.Printf("Total de clics: %d\n", totalClicks)
	fmt.Printf("Date de création: %s\n", link.CreatedAt.Format("2006-01-02 15:04:05"))
}
