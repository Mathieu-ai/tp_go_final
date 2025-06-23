package cli

import (
	"fmt"
	"log"

	"github.com/axellelanca/urlshortener/cmd"
	"github.com/axellelanca/urlshortener/internal/config"
	"github.com/axellelanca/urlshortener/internal/models"
	"github.com/spf13/cobra"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MigrateCmd représente la commande 'migrate'
var MigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Exécute les migrations de la base de données pour créer ou mettre à jour les tables.",
	Long: `Cette commande se connecte à la base de données configurée (SQLite)
et exécute les migrations automatiques de GORM pour créer les tables 'links' et 'clicks'
basées sur les modèles Go.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Charger la configuration chargée globalement via cmd.cfg
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("Échec du chargement de la configuration : %v", err)
		}

		db, err := gorm.Open(sqlite.Open(cfg.Database.Name), &gorm.Config{})
		if err != nil {
			log.Fatalf("Échec de la connexion à la base de données : %v", err)
		}

		sqlDB, err := db.DB()
		if err != nil {
			log.Fatalf("FATAL: Échec de l'obtention de la base de données SQL sous-jacente: %v", err)
		}
		defer sqlDB.Close()

		// Exécuter les migrations automatiques de GORM.
		if err := db.AutoMigrate(&models.Link{}, &models.Click{}); err != nil {
			log.Fatalf("Échec de la migration de la base de données : %v", err)
		}

		fmt.Println("Migrations de la base de données exécutées avec succès.")
	},
}

func init() {
	cmd.RootCmd.AddCommand(MigrateCmd)
}
func init() {
	// TODO : Ajouter la commande à RootCmd
}
