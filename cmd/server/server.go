package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/axellelanca/urlshortener/cmd"
	"github.com/axellelanca/urlshortener/internal/api"
	"github.com/axellelanca/urlshortener/internal/config"
	"github.com/axellelanca/urlshortener/internal/models"
	"github.com/axellelanca/urlshortener/internal/monitor"
	"github.com/axellelanca/urlshortener/internal/repository"
	"github.com/axellelanca/urlshortener/internal/services"
	"github.com/axellelanca/urlshortener/internal/workers"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

// RunServerCmd represents the 'run-server' Cobra command
// This is the entry point for launching the application server
var RunServerCmd = &cobra.Command{
	Use:   "run-server",
	Short: "Launches the URL shortening API server and background processes.",
	Long: `This command initializes the database, configures the APIs,
starts asynchronous workers for click tracking and URL monitoring,
then launches the HTTP server.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load application configuration from files or environment variables
		// This contains all settings for database, server, analytics, and monitoring
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}

		// Initialize database connection using GORM with SQLite
		// GORM provides an ORM layer over the raw database operations
		db, err := gorm.Open(sqlite.Open(cfg.Database.Name), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}

		// Automatic migration of database models to create/update tables
		// This ensures the database schema matches our Go structs
		if err := db.AutoMigrate(&models.Link{}, &models.Click{}); err != nil {
			log.Fatalf("Failed to migrate database: %v", err)
		}

		// Initialize repository layer for data access
		// Repositories abstract database operations behind interfaces
		linkRepo := repository.NewLinkRepository(db)
		clickRepo := repository.NewClickRepository(db)

		// Log successful repository initialization for debugging
		log.Println("Repositories initialized.")

		// Initialize business logic services
		// Services contain the core business logic of the application
		linkService := services.NewLinkService(linkRepo)

		// Log successful service initialization for debugging
		log.Println("Business services initialized.")

		// Initialize click events channel for asynchronous click processing
		// This channel decouples URL redirection from click recording for better performance
		clickEventsChan := make(chan models.ClickEvent, cfg.Analytics.BufferSize)
		api.ClickEventsChannel = clickEventsChan // Set the global channel used by handlers

		// Start worker goroutines to process click events asynchronously
		// Workers run in background and save click data to database
		workers.StartClickWorkers(cfg.Analytics.WorkerCount, clickEventsChan, clickRepo)

		// Log the initialization of click processing system
		log.Printf("Click events channel initialized with buffer size %d. %d click worker(s) started.",
			cfg.Analytics.BufferSize, cfg.Analytics.WorkerCount)

		// Initialize and start the URL health monitoring system
		// This periodically checks if shortened URLs are still accessible
		monitorInterval := time.Duration(cfg.Monitor.IntervalMinutes) * time.Minute
		urlMonitor := monitor.NewUrlMonitor(linkRepo, monitorInterval)
		go urlMonitor.Start() // Run monitor in background goroutine
		log.Printf("URL monitor started with interval of %v.", monitorInterval)

		// Configure Gin router and API handlers
		// Gin is the HTTP framework used for routing and middleware
		router := gin.Default()
		api.SetupRoutes(router, linkService, cfg.Analytics.BufferSize)

		// Log successful API route configuration
		log.Println("API routes configured.")

		// Create HTTP server instance with Gin router
		// This prepares the server but doesn't start it yet
		serverAddr := fmt.Sprintf(":%d", cfg.Server.Port)
		srv := &http.Server{
			Addr:    serverAddr,
			Handler: router,
		}

		// Start the HTTP server in a separate goroutine to avoid blocking
		// This allows the main goroutine to handle shutdown signals
		go func() {
			log.Printf("Starting server on %s", serverAddr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Failed to start server: %v", err)
			}
		}()

		// Handle graceful shutdown of the server
		// Create a channel to receive OS signals (Ctrl+C, SIGTERM)
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) // Listen for interrupt signals

		// Block until a shutdown signal is received
		// This keeps the main goroutine alive while the server runs
		<-quit
		log.Println("Shutdown signal received. Stopping server...")

		// Graceful shutdown with timeout for workers to finish
		// Give background workers time to complete their current tasks
		log.Println("Shutting down... Giving workers time to finish.")
		time.Sleep(5 * time.Second)

		log.Println("Server stopped gracefully.")
	},
}

func init() {
	// Register this command with the root command so it can be executed
	cmd.RootCmd.AddCommand(RunServerCmd)
}
