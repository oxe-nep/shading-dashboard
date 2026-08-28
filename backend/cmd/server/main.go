package main

import (
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/oxe-nep/shading-dashboard/internal/api"
	"github.com/oxe-nep/shading-dashboard/internal/config"
	switchdrv "github.com/oxe-nep/shading-dashboard/internal/switch"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	configPath := envOr("CONFIG_PATH", "data/config.json")
	port := envIntOr("PORT", 8080)
	corsOrigin := envOr("CORS_ORIGIN", "*")

	store, err := config.NewStore(configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	manager := switchdrv.NewManager(store)

	hub := api.NewHub(manager)
	manager.SetUpdateHandler(hub.Broadcast)

	server := api.NewServer(store, manager, hub)
	manager.Start()

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{corsOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))
	r.Mount("/", server.Routes())

	addr := ":" + strconv.Itoa(port)
	log.Info().Str("addr", addr).Msg("shading-dashboard backend listening")
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal().Err(err).Msg("server stopped")
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
