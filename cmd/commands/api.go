package commands

import (
	"context"
	"go-ldap-sso/config"
	"go-ldap-sso/db"
	"go-ldap-sso/internal/handler"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

func RunAPI(cfg *config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbConn := db.NewDatabase(cfg)
	defer dbConn.Close()

	authHandler, err := handler.NewAuthHandler(cfg, dbConn)
	if err != nil {
		log.Fatalf("Failed to initialize auth handler: %v", err)
	}

	server := handler.SetupRoutes(authHandler)

	go func() {
		log.Println("🚀 Server running on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("🛑 Received shutdown signal, gracefully shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)
	} else {
		log.Println("✅ Server gracefully stopped")
	}

	return nil
}

func RunIDP(cfg *config.Config) error {

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/simulated-idp-metadata-lldap.xml", fs)

	log.Println("🟢 Simulated IdP metadata server running at http://localhost:6000")
	if err := http.ListenAndServe(":6000", nil); err != nil {
		log.Fatalf("failed to start metadata server: %v", err)
		return err
	}

	return nil
}
