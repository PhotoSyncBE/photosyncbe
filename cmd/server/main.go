package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"photosync-backend/internal/api"
	"photosync-backend/internal/auth"
	"photosync-backend/internal/config"
	"photosync-backend/internal/storage"
)

func main() {
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		configFile = "config.yaml"
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	jwtExpiry, err := cfg.GetJWTExpiry()
	if err != nil {
		log.Fatalf("Invalid JWT expiry: %v", err)
	}

	poolTTL, err := cfg.GetPoolTTL()
	if err != nil {
		log.Fatalf("Invalid pool TTL: %v", err)
	}

	authenticator, err := auth.NewAuthenticator(cfg)
	if err != nil {
		log.Fatalf("Failed to create authenticator: %v", err)
	}

	jwtManager := auth.NewJWTManager(
		[]byte(cfg.JWT.SecretKey),
		[]byte(cfg.JWT.EncryptionKey),
		cfg.JWT.Issuer,
		jwtExpiry,
	)

	storageBackend, err := storage.NewStorageBackend(cfg)
	if err != nil {
		log.Fatalf("Failed to create storage backend: %v", err)
	}

	pool := storage.NewGenericConnectionPool(storageBackend, poolTTL)
	defer pool.Close()

	authHandler := api.NewAuthHandler(authenticator, jwtManager)
	photoHandler := api.NewPhotoHandler(pool, storageBackend, jwtManager)

	router := api.NewRouter(authHandler, photoHandler)

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	server := &http.Server{
		Addr:         cfg.Server.Host + ":" + cfg.Server.Port,
		Handler:      router,
		TLSConfig:    tlsConfig,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Starting server on %s", server.Addr)
		if err := server.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
