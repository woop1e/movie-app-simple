package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"yozigo/internal/mongoplatform"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	dbname := os.Getenv("MONGODB_DATABASE")
	if dbname == "" {
		dbname = "yozigo"
	}
	p, err := mongoplatform.Connect(ctx, uri, dbname)
	if err != nil {
		log.Fatal(err)
	}
	defer p.Client.Disconnect(context.Background())
	if os.Getenv("SEED_ON_START") != "false" {
		seedCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		if err := p.Seed(seedCtx); err != nil {
			cancel()
			log.Fatal(err)
		}
		cancel()
	}
	s := &http.Server{Addr: ":" + env("PORT", "8080"), Handler: p.Handler(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		c, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		_ = s.Shutdown(c)
	}()
	log.Printf("MongoDB recommendation API listening on %s (database %s)", s.Addr, dbname)
	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
