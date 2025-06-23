package monitor

import (
	"log"
	"net/http"
	"sync" // Pour protéger l'accès concurrentiel à knownStates
	"time"

	_ "github.com/axellelanca/urlshortener/internal/models"   // Importe les modèles de liens
	"github.com/axellelanca/urlshortener/internal/repository" // Importe le repository de liens
)

// UrlMonitor gère la surveillance périodique des URLs longues.
type UrlMonitor struct {
	linkRepo    repository.LinkRepository // Pour récupérer les URLs à surveiller
	interval    time.Duration             // Intervalle entre chaque vérification (ex: 5 minutes)
	knownStates map[uint]bool             // État connu de chaque URL: map[LinkID]estAccessible (true/false)
	mu          sync.Mutex                // Mutex pour protéger l'accès concurrentiel à knownStates
}

// TODO finir cette fonction
// NewUrlMonitor crée et retourne une nouvelle instance de UrlMonitor.
// Attention: retourne un pointeur
func NewUrlMonitor(linkRepo repository.LinkRepository, interval time.Duration) *UrlMonitor {
	return
}

// Start lance la boucle de surveillance périodique des URLs.
// Cette fonction est conçue pour être lancée dans une goroutine séparée.
func (m *UrlMonitor) Start() {
	log.Printf("[MONITOR] Démarrage du moniteur d'URLs avec un intervalle de %v...", m.interval)
	ticker := time.NewTicker(m.interval) // Crée un ticker qui envoie un signal à chaque intervalle
	defer ticker.Stop()                  // S'assure que le ticker est arrêté quand Start se termine

	// Exécute une première vérification immédiatement au démarrage
	m.checkUrls()

	// Boucle principale du moniteur, déclenchée par le ticker
	for range ticker.C {
		m.checkUrls()
	}
}

// checkUrls effectue une vérification de l'état de toutes les URLs longues enregistrées.
func (m *UrlMonitor) checkUrls() {
	log.Println("[MONITOR] Lancement de la vérification de l'état des URLs...")

	// TODO : Récupérer toutes les URLs longues actives depuis le linkRepo (GetAllLinks).
	// Gérer l'erreur si la récupération échoue.
	// Si erreur : log.Printf("[MONITOR] ERREUR lors de la récupération des liens pour la surveillance : %v", err)
	links, err :=

	for _, link := range links {
		// TODO : Pour chaque lien, vérifier son accessibilité (isUrlAccessible).
		currentState :=

		// Protéger l'accès à la map 'knownStates' car 'checkUrls' peut être exécuté concurremment
		m.mu.Lock()
		previousState, exists := m.knownStates[link.ID] // Récupère l'état précédent
		m.knownStates[link.ID] = currentState           // Met à jour l'état actuel
		m.mu.Unlock()

		// Si c'est la première vérification pour ce lien, on initialise l'état sans notifier.
		if !exists {
			log.Printf("[MONITOR] État initial pour le lien %s (%s) : %s",
				link.ShortCode, link.LongURL, formatState(currentState))
			continue
		}

		// TODO : Comparer l'état actuel avec l'état précédent.
		// Si l'état a changé, générer une fausse notification dans les logs.
		// log.Printf("[NOTIFICATION] Le lien %s (%s) est passé de %s à %s !"

	}
	log.Println("[MONITOR] Vérification de l'état des URLs terminée.")
}

// isUrlAccessible effectue une requête HTTP HEAD pour vérifier l'accessibilité d'une URL.
func (m *UrlMonitor) isUrlAccessible(url string) bool {
	// TODO Définir un timeout pour éviter de bloquer trop longtemps (5 secondes c'est bien)

	// TODO: Effectuer une requête HEAD (plus légère que GET) sur l'URL.
	// Un code de statut 2xx ou 3xx indique que l'URL est accessible.
	// Si err : log.Printf("[MONITOR] Erreur d'accès à l'URL '%s': %v", url, err)

	// TODO Assurez-vous de fermer le corps de la réponse pour libérer les ressources


	// Déterminer l'accessibilité basée sur le code de statut HTTP.
	return resp.StatusCode >= 200 && resp.StatusCode < 400 // Codes 2xx ou 3xx
}

// formatState est une fonction utilitaire pour rendre l'état plus lisible dans les logs.
func formatState(accessible bool) string {
	if accessible {
		return "ACCESSIBLE"
	}
	return "INACCESSIBLE"
}
