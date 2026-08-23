package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/api"
	"github.com/jb843051627/lockgate-warden/internal/cache"
	"github.com/jb843051627/lockgate-warden/internal/clock"
	"github.com/jb843051627/lockgate-warden/internal/config"
	"github.com/jb843051627/lockgate-warden/internal/ingest"
	"github.com/jb843051627/lockgate-warden/internal/metrics"
	"github.com/jb843051627/lockgate-warden/internal/service"
	"github.com/jb843051627/lockgate-warden/internal/store"
)

func main() {
	cfg := config.FromEnv()
	m := metrics.New()

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	clk := clock.System{}
	c := cache.New(time.Minute)
	stopJanitor := c.StartJanitor(30 * time.Second)
	defer func() { <-stopJanitor }()

	pipe := ingest.New(cfg.QueueBuffer, m, log.Default())
	defer pipe.Close()

	svc := service.New(st, clk, c, pipe, m, service.Params{
		DedupWindow: cfg.DedupWindow,
		Staleness:   cfg.Staleness,
		WarnTTL:     cfg.WarnTTL,
	})

	srv := api.NewServer(svc)
	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.Middleware(m, srv.Handler()),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			svc.RunWatchdog()
		}
	}()

	go func() {
		log.Printf("lockgate-warden listening on %s", cfg.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
