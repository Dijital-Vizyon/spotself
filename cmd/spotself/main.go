package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Dijital-Vizyon/spotself/internal/spotself"
)

func main() {
	cfg := spotself.Config{
		Addr:           env("SPOTSELF_ADDR", ":8080"),
		DataDir:        env("SPOTSELF_DATA_DIR", "./data"),
		PublicURL:      env("SPOTSELF_PUBLIC_URL", "http://localhost:8080"),
		MaxUploadMB:    envInt("SPOTSELF_MAX_UPLOAD_MB", 64),
		AdminToken:     env("SPOTSELF_ADMIN_TOKEN", ""),
		AllowNoAuth:    envBool("SPOTSELF_ALLOW_NO_AUTH", false),
		MaxImagePixels: envInt("SPOTSELF_MAX_IMAGE_PIXELS", 24000000),
	}

	srv, err := spotself.NewServer(cfg)
	if err != nil {
		log.Fatalf("start spotself: %v", err)
	}

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("SpotSelf listening on %s", cfg.Addr)
	log.Fatal(httpServer.ListenAndServe())
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "TRUE" || value == "yes"
}
