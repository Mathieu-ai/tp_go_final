package cli

import (
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/axellelanca/urlshortener/cmd"
	"github.com/axellelanca/urlshortener/internal/config"
	"github.com/axellelanca/urlshortener/internal/repository"
	"github.com/axellelanca/urlshortener/internal/services"
	"github.com/spf13/cobra"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var longURLFlag string

// CreateCmd représente la commande 'create'
var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Crée une URL courte à partir d'une URL longue.",
	Long: `Cette commande raccourcit une URL longue fournie et affiche le code court généré.

Exemple:
  url-shortener create --url="https://www.google.com/search?q=go+lang"`,
	Run: func(cmd *cobra.Command, args []string) {
		// Valider que le flag --url a été fourni.
		if longURLFlag == "" {
			fmt.Println("Error: --url flag is required")
			os.Exit(1)
		}

		// Validation basique du format de l'URL
		_, err := url.ParseRequestURI(longURLFlag)
		if err != nil {
			fmt.Printf("Error: Invalid URL format: %v\n", err)
			os.Exit(1)
		}

		// Charger la configuration
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}

		db, err := gorm.Open(sqlite.Open(cfg.Database.Name), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}

		sqlDB, err := db.DB()
		if err != nil {
			log.Fatalf("FATAL: Échec de l'obtention de la base de données SQL sous-jacente: %v", err)
		}
		defer sqlDB.Close()

		// Initialiser les repositories et services nécessaires
		linkRepo := repository.NewLinkRepository(db)
		linkService := services.NewLinkService(linkRepo)

		// Appeler le LinkService pour créer le lien court.
		link, err := linkService.CreateLink(longURLFlag)
		if err != nil {
			log.Fatalf("Failed to create short link: %v", err)
		}

		fullShortURL := fmt.Sprintf("%s/%s", cfg.Server.BaseURL, link.ShortCode)
		fmt.Printf("URL courte créée avec succès:\n")
		fmt.Printf("Code: %s\n", link.ShortCode)
		fmt.Printf("URL complète: %s\n", fullShortURL)
	},
}

func init() {
	// Définir le flag --url pour la commande create.
	CreateCmd.Flags().StringVar(&longURLFlag, "url", "", "The long URL to shorten")

	// Marquer le flag comme requis
	CreateCmd.MarkFlagRequired("url")

	// Ajouter la commande à RootCmd
	cmd.RootCmd.AddCommand(CreateCmd)
}
