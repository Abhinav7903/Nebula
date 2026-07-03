package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Abhinav7903/nebula/internal/api"
	"github.com/Abhinav7903/nebula/internal/config"
	"github.com/Abhinav7903/nebula/internal/collectors"
	"github.com/Abhinav7903/nebula/internal/collectors/bitcoin"
	"github.com/Abhinav7903/nebula/internal/collectors/censys"
	"github.com/Abhinav7903/nebula/internal/collectors/crtsh"
	"github.com/Abhinav7903/nebula/internal/collectors/dns"
	"github.com/Abhinav7903/nebula/internal/collectors/dnsdumpster"
	"github.com/Abhinav7903/nebula/internal/collectors/emailrep"
	"github.com/Abhinav7903/nebula/internal/collectors/ethereum"
	"github.com/Abhinav7903/nebula/internal/collectors/geoip"
	"github.com/Abhinav7903/nebula/internal/collectors/tron"
	"github.com/Abhinav7903/nebula/internal/collectors/github"
	"github.com/Abhinav7903/nebula/internal/collectors/greynoise"
	"github.com/Abhinav7903/nebula/internal/collectors/onion"
	"github.com/Abhinav7903/nebula/internal/collectors/pastebin"
	"github.com/Abhinav7903/nebula/internal/collectors/searchengine"
	"github.com/Abhinav7903/nebula/internal/collectors/shodan"
	"github.com/Abhinav7903/nebula/internal/collectors/social"
	"github.com/Abhinav7903/nebula/internal/collectors/solana"
	"github.com/Abhinav7903/nebula/internal/collectors/subdomains"
	"github.com/Abhinav7903/nebula/internal/collectors/threatintel"
	"github.com/Abhinav7903/nebula/internal/collectors/urlscan"
	"github.com/Abhinav7903/nebula/internal/collectors/virustotal"
	"github.com/Abhinav7903/nebula/internal/collectors/wayback"
	"github.com/Abhinav7903/nebula/internal/collectors/whois"
	"github.com/Abhinav7903/nebula/internal/logger"
	"github.com/Abhinav7903/nebula/internal/progress"
	"github.com/Abhinav7903/nebula/internal/queue"
	"github.com/Abhinav7903/nebula/internal/store"
	"github.com/Abhinav7903/nebula/internal/summary"
	"github.com/Abhinav7903/nebula/internal/websearch"
	"github.com/Abhinav7903/nebula/internal/workers"
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

	ws := websearch.New(nil)
	if cfg.WebSearch.Enabled {
		ws.AddProvider(websearch.NewDuckDuckGoProvider(nil))
		ws.AddProvider(websearch.NewBingProvider(nil))
		ws.AddProvider(websearch.NewMojeekProvider(nil))
		ws.AddProvider(websearch.NewGoogleProvider(nil,
			cfg.WebSearch.GoogleAPIKey,
			cfg.WebSearch.GoogleEngineID,
		))
		log.Info("websearch enabled",
			"providers", len(ws.Providers()),
		)
	}

	registerCollectors(reg, cfg, log, ws)

	workerPool := workers.NewPool(q, reg, hub, memStore, log, cfg.Queue.Workers)

	var summ summary.Summarizer
	if cfg.AI.Key != "" {
		summ = summary.NewGroq(cfg.AI.Key, cfg.AI.Model)
	} else {
		summ = &fallbackSummarizer{}
	}

	handler := api.NewHandler(log, memStore, reg, workerPool, hub, summ, ws)
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

func registerCollectors(reg *collectors.Registry, cfg *config.Config, log *slog.Logger, ws *websearch.Engine) {
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
	always(searchengine.New(ws))
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
		if err := ensureGeoIPDBs(cfg.GeoIP.CityDBPath, cfg.GeoIP.ASNDBPath); err != nil {
			fmt.Fprintf(os.Stderr, "warning: geoip init: %v\n", err)
		} else {
			g, err := geoip.New(cfg.GeoIP.CityDBPath, cfg.GeoIP.ASNDBPath)
			if err == nil {
				reg.Register(g)
			} else {
				fmt.Fprintf(os.Stderr, "warning: geoip init: %v\n", err)
			}
		}
	}
}

func ensureGeoIPDBs(cityPath, asnPath string) error {
	dir := filepath.Dir(cityPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	type dbEntry struct{ path, url string }
	dbs := []dbEntry{
		{cityPath, "https://raw.githubusercontent.com/P3TERX/GeoLite.mmdb/download/GeoLite2-City.mmdb"},
		{asnPath, "https://raw.githubusercontent.com/P3TERX/GeoLite.mmdb/download/GeoLite2-ASN.mmdb"},
	}

	for _, db := range dbs {
		if _, err := os.Stat(db.path); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "downloading %s...\n", filepath.Base(db.path))
			if err := downloadFile(db.path, db.url); err != nil {
				return fmt.Errorf("download %s: %w", db.path, err)
			}
		}
	}
	return nil
}

func downloadFile(path, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
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
