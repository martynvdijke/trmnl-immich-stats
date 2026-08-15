package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

// Version is set at build time via -ldflags "-X main.Version=...".
var Version = "dev"

type config struct {
	immichURL    string
	immichAPIKey string
	port         string
}

func loadConfig() config {
	cfg := config{
		immichURL:    os.Getenv("IMMICH_URL"),
		immichAPIKey: os.Getenv("IMMICH_API_KEY"),
		port:         os.Getenv("PORT"),
	}
	if cfg.immichURL == "" {
		log.Fatal("IMMICH_URL environment variable is required")
	}
	if cfg.immichAPIKey == "" {
		log.Fatal("IMMICH_API_KEY environment variable is required")
	}
	if cfg.port == "" {
		cfg.port = "8080"
	}
	return cfg
}

func main() {
	cfg := loadConfig()
	client := newImmichClient(cfg.immichURL, cfg.immichAPIKey)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("GET /api/trmnl/stats", client.handleStats)

	addr := ":" + cfg.port
	log.Printf("trmnl-immich-stats %s listening on %s", Version, addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	log.Printf("error: %s (status %d)", msg, status)
	writeJSON(w, status, map[string]string{"error": msg})
}
