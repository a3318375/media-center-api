package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"media-center/internal/httpapi"
	"media-center/internal/store"
)

func main() {
	dataDir := os.Getenv("MEDIA_CENTER_DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	db, err := store.Open(filepath.Clean(dataDir))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":18080"
	}
	server := &http.Server{Addr: addr, Handler: httpapi.NewRouter(db), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}

	stopped := make(chan os.Signal, 1)
	signal.Notify(stopped, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stopped
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("server shutdown: %v", err)
		}
	}()

	log.Printf("media-center API listening on %s (data: %s)", addr, dataDir)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
