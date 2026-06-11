package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourusername/nebula/internal/api"
	"github.com/yourusername/nebula/internal/config"
	"github.com/yourusername/nebula/internal/collectors"
	"github.com/yourusername/nebula/internal/collectors/bitcoin"
	"github.com/yourusername/nebula/internal/collectors/censys"
	"github.com/yourusername/nebula/internal/collectors/crtsh"
	"github.com/yourusername/nebula/internal/collectors/dns"
	"github.com/yourusername/nebula/internal/collectors/dnsdumpster"
	"github.com/yourusername/nebula/internal/collectors/emailrep"
	"github.com/yourusername/nebula/internal/collectors/ethereum"
	"github.com/yourusername/nebula/internal/collectors/geoip"
	"github.com/yourusername/nebula/internal/collectors/tron"
	"github.com/yourusername/nebula/internal/collectors/github"
	"github.com/yourusername/nebula/internal/collectors/greynoise"
	"github.com/yourusername/nebula/internal/collectors/onion"
	"github.com/yourusername/nebula/internal/collectors/pastebin"
	"github.com/yourusername/nebula/internal/collectors/searchengine"
	"github.com/yourusername/nebula/internal/collectors/shodan"
	"github.com/yourusername/nebula/internal/collectors/social"
	"github.com/yourusername/nebula/internal/collectors/solana"
	"github.com/yourusername/nebula/internal/collectors/subdomains"
	"github.com/yourusername/nebula/internal/collectors/threatintel"
	"github.com/yourusername/nebula/internal/collectors/urlscan"
	"github.com/yourusername/nebula/internal/collectors/virustotal"
	"github.com/yourusername/nebula/internal/collectors/wayback"
	"github.com/yourusername/nebula/internal/collectors/whois"
	"github.com/yourusername/nebula/internal/logger"
	"github.com/yourusername/nebula/internal/progress"
	"github.com/yourusername/nebula/internal/queue"
	"github.com/yourusername/nebula/internal/store"
	"github.com/yourusername/nebula/internal/summary"
	"github.com/yourusername/nebula/internal/workers"
)

func main() {
	log := logger.New(slog.LevelInfo)

	cfgPath := "configs/config.yaml"
	if env := os.Getenv("NEBULA_CONFIG"); env != "" {
		cfgPath = env
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	memStore := store.NewMemoryStore(5 * time.Minute)
	q := queue.New(cfg.Queue.MaxQueueSize)
	hub := progress.NewHub(100)
	reg := collectors.NewRegistry()

	registerCollectors(reg, cfg, log)

	workerPool := workers.NewPool(q, reg, hub, memStore, log, cfg.Queue.Workers)

	var summ summary.Summarizer
	if cfg.AI.Key != "" {
		summ = summary.NewGroq(cfg.AI.Key, cfg.AI.Model)
	} else {
		summ = &fallbackSummarizer{}
	}

	handler := api.NewHandler(log, memStore, reg, workerPool, hub, summ)
	mw := api.NewMiddleware(log, cfg.APIAuth.Keys, cfg.APIAuth.RequireKey,
		cfg.RateLimit.RequestsPerMin, cfg.RateLimit.Burst)
	router := api.NewRouter(handler, mw)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	workerPool.Start(ctx)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSec) * time.Second,
	}

	go func() {
		log.Info("server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down...")
	workerPool.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", "error", err)
	}
}

func registerCollectors(reg *collectors.Registry, cfg *config.Config, log *slog.Logger) {
	always := func(c collectors.Collector) { reg.Register(c) }
	whenEnabled := func(cfg config.CollectorItem, c collectors.Collector) {
		if !cfg.Enabled {
			return
		}
		if c.RequiresKey() && cfg.Key == "" {
			log.Warn("collector skipped: requires key but none provided", "collector", c.Name())
			return
		}
		reg.Register(c)
	}

	always(&dns.Collector{})
	always(whois.New())
	always(crtsh.New())
	always(subdomains.New())
	always(dnsdumpster.New())
	always(searchengine.New())
	always(social.New())
	always(wayback.New())
	always(pastebin.New())
	always(threatintel.New())
	whenEnabled(cfg.Collectors.Ethereum, ethereum.New(cfg.Collectors.Ethereum.Key))
	whenEnabled(cfg.Collectors.Tron, tron.New(cfg.Collectors.Tron.Key))
	always(bitcoin.New())
	always(solana.New())
	always(github.New(cfg.Collectors.Github.Key))

	whenEnabled(cfg.Collectors.Shodan, shodan.New(cfg.Collectors.Shodan.Key))
	whenEnabled(cfg.Collectors.Censys, censys.New(cfg.Collectors.Censys.Key))
	whenEnabled(cfg.Collectors.VirusTotal, virustotal.New(cfg.Collectors.VirusTotal.Key))
	whenEnabled(cfg.Collectors.GreyNoise, greynoise.New(cfg.Collectors.GreyNoise.Key))
	whenEnabled(cfg.Collectors.EmailRep, emailrep.New(cfg.Collectors.EmailRep.Key))
	whenEnabled(cfg.Collectors.URLScan, urlscan.New(cfg.Collectors.URLScan.Key))
	whenEnabled(cfg.Collectors.Onion, onion.New(nil))

	if cfg.Collectors.GeoIP.Enabled {
		g, err := geoip.New(cfg.GeoIP.CityDBPath, cfg.GeoIP.ASNDBPath)
		if err == nil {
			reg.Register(g)
		} else {
			fmt.Fprintf(os.Stderr, "warning: geoip init: %v\n", err)
		}
	}
}

type fallbackSummarizer struct{}

func (f *fallbackSummarizer) Summarize(_ context.Context, _ string, _ string, results []collectors.Result) (string, error) {
	if len(results) == 0 {
		return "No results found.", nil
	}
	by := make(map[string]int)
	for _, r := range results {
		by[r.Collector]++
	}
	msg := fmt.Sprintf("Found %d results across %d collectors:\n", len(results), len(by))
	for name, count := range by {
		msg += fmt.Sprintf("- %s: %d results\n", name, count)
	}
	return msg, nil
}

func (f *fallbackSummarizer) ProviderName() string { return "fallback" }
