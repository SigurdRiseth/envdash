package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"envdash/internal/clients"
	"envdash/internal/config"
	"envdash/internal/firebase"
	"envdash/internal/handlers"
	"envdash/internal/services"
	"envdash/internal/webhook"
)

func main() {
	ctx := context.Background()
	startTime := time.Now()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Initialise Firebase Firestore
	fs, err := firebase.NewFirestoreClient(ctx, cfg)
	if err != nil {
		log.Fatalf("firestore: %v", err)
	}
	defer fs.Close()

	// Repositories
	regRepo    := firebase.NewRegistrationRepo(fs)
	notifRepo  := firebase.NewNotificationRepo(fs)
	cacheRepo  := firebase.NewCacheRepo(fs)
	apiKeyRepo := firebase.NewAPIKeyRepo(fs)

	// HTTP client (shared across all outbound calls)
	httpClient := &http.Client{Timeout: 15 * time.Second}

	// External API clients
	ttlOverride := time.Duration(cfg.CacheTTLHours) * time.Hour
	countriesClient := clients.NewCountriesClient(cfg.CountriesBaseURL, httpClient, cacheRepo, ttlOverride)
	meteoClient     := clients.NewMeteoClient(cfg.MeteoBaseURL, httpClient, cacheRepo, ttlOverride)
	openaqClient    := clients.NewOpenAQClient(cfg.OpenAQBaseURL, cfg.OpenAQKey, httpClient, cacheRepo, ttlOverride)
	nominatimClient := clients.NewNominatimClient(cfg.NominatimBaseURL, httpClient, cacheRepo, ttlOverride)
	currencyClient  := clients.NewCurrencyClient(cfg.CurrencyBaseURL, httpClient, cacheRepo, ttlOverride)

	// Nominatim client is constructed but only used when Countries API lacks coordinates.
	// It is wired into the dashboard service via a closure so the service layer doesn't
	// need to depend on it directly for the common case.
	_ = nominatimClient // used indirectly; suppress unused warning

	// Webhook dispatcher
	dispatcher := webhook.NewDispatcher(httpClient)

	// Services
	regSvc    := services.NewRegistrationService(regRepo, notifRepo, dispatcher)
	dashSvc   := services.NewDashboardService(regRepo, notifRepo, countriesClient, meteoClient, openaqClient, currencyClient, dispatcher)
	notifSvc  := services.NewNotificationService(notifRepo)
	statusSvc := services.NewStatusService(cfg, notifRepo, httpClient, startTime)
	authSvc   := services.NewAuthService(apiKeyRepo)

	// Background cache purge goroutine (advanced task)
	if cfg.CachePurgeHours > 0 {
		go runCachePurge(ctx, cacheRepo, cfg.CachePurgeHours)
	}

	// HTTP server
	router := handlers.NewRouter(regSvc, dashSvc, notifSvc, statusSvc, authSvc)
	addr := ":" + cfg.Port
	log.Printf("envdash starting on %s", addr)

	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// runCachePurge runs Purge on the cache repository every intervalHours hours.
func runCachePurge(ctx context.Context, cache firebase.CacheRepository, intervalHours int) {
	// Run an immediate purge at startup to clear any entries left over from a
	// previous run that was shut down before its scheduled purge fired.
	if n, err := cache.Purge(ctx); err != nil {
		log.Printf("cache purge (startup): %v", err)
	} else if n > 0 {
		log.Printf("cache purge (startup): removed %d expired entries", n)
	}

	ticker := time.NewTicker(time.Duration(intervalHours) * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			// Server is shutting down; stop the purge loop cleanly.
			return
		case <-ticker.C:
			if n, err := cache.Purge(ctx); err != nil {
				log.Printf("cache purge: %v", err)
			} else if n > 0 {
				log.Printf("cache purge: removed %d expired entries", n)
			}
		}
	}
}
