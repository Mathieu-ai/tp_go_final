package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/axellelanca/urlshortener/internal/config"
	"github.com/spf13/cobra"
)

// cfg est la variable globale qui contiendra la configuration chargée.
// Elle sera accessible à toutes les commandes Cobra.
var Cfg *config.Config

var RootCmd = &cobra.Command{
	Use:   "urlshortener",
	Short: "A URL shortener application",
	Long: `A URL shortener application that allows you to create shortened URLs,
track click statistics, and monitor URL health.`,
}

// Execute est le point d'entrée principal pour l'application Cobra.
// Il est appelé depuis 'main.go'.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur lors de l'exécution de la commande: %v\n", err)
		os.Exit(1)
	}
}

// init() est une fonction spéciale de Go qui s'exécute automatiquement
// avant la fonction main(). Elle est utilisée ici pour initialiser Cobra
// et ajouter toutes les sous-commandes.
func init() {
	cobra.OnInitialize(initConfig)

	// IMPORTANT : Ici, nous n'appelons PAS RootCmd.AddCommand() directement
	// pour les commandes 'server', 'create', 'stats', 'migrate'.
	// Ces commandes s'enregistreront elles-mêmes via leur propre fonction init().
}

// initConfig charge la configuration de l'application.
// Cette fonction est appelée au début de l'exécution de chaque commande Cobra
// grâce à `cobra.OnInitialize(initConfig)`.
func initConfig() {
	var err error
	Cfg, err = config.LoadConfig()
	if err != nil {
		// Loggue l'erreur mais ne fait pas un os.Exit(1) ici si LoadConfig()
		// gère déjà l'absence de fichier avec des valeurs par défaut.
		// Si LoadConfig() termine le programme en cas d'erreur fatale,
		// cette vérification est surtout pour les avertissements.
		log.Printf("Attention: Problème lors du chargement de la configuration: %v. Utilisation des valeurs par défaut.", err)
	}
	// La configuration est maintenant disponible via la variable globale 'cmd.cfg'.
}
