package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"envdash/internal/clients"
	"envdash/internal/config"
	"envdash/internal/firebase"
	"envdash/internal/handlers"
	"envdash/internal/services"
	"envdash/internal/webhook"
)

func main() {
	// Use a JSON handler so log output is machine-parseable in production.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx := context.Background()
	startTime := time.Now()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	// Initialise Firebase Firestore
	fs, err := firebase.NewFirestoreClient(ctx, cfg)
	if err != nil {
		logger.Error("failed to connect to Firestore", "err", err)
		os.Exit(1)
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
	dispatcher := webhook.NewDispatcher(httpClient, logger)

	// Services
	regSvc    := services.NewRegistrationService(regRepo, notifRepo, dispatcher)
	dashSvc   := services.NewDashboardService(regRepo, notifRepo, countriesClient, meteoClient, openaqClient, currencyClient, dispatcher, logger)
	notifSvc  := services.NewNotificationService(notifRepo)
	statusSvc := services.NewStatusService(cfg, notifRepo, httpClient, startTime)
	authSvc   := services.NewAuthService(apiKeyRepo)

	// Background cache purge goroutine (advanced task)
	if cfg.CachePurgeHours > 0 {
		go runCachePurge(ctx, cacheRepo, cfg.CachePurgeHours, logger)
	}

	// HTTP server
	router := handlers.NewRouter(regSvc, dashSvc, notifSvc, statusSvc, authSvc, logger)
	addr := ":" + cfg.Port
	logger.Info("envdash starting", "addr", addr)

	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

// runCachePurge runs Purge on the cache repository every intervalHours hours.
func runCachePurge(ctx context.Context, cache firebase.CacheRepository, intervalHours int, logger *slog.Logger) {
	// Run an immediate purge at startup to clear any entries left over from a
	// previous run that was shut down before its scheduled purge fired.
	if n, err := cache.Purge(ctx); err != nil {
		logger.Error("cache purge failed", "phase", "startup", "err", err)
	} else if n > 0 {
		logger.Info("cache purge complete", "phase", "startup", "removed", n)
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
				logger.Error("cache purge failed", "err", err)
			} else if n > 0 {
				logger.Info("cache purge complete", "removed", n)
			}
		}
	}
}
