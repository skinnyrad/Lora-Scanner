package main

import (
	"context"
	"errors"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skinnyrad/Lora-Scanner/internal/api"
	"github.com/skinnyrad/Lora-Scanner/internal/frames"
	"github.com/skinnyrad/Lora-Scanner/internal/gateway"
	"github.com/skinnyrad/Lora-Scanner/internal/hub"
	"github.com/skinnyrad/Lora-Scanner/internal/serial"
	"github.com/skinnyrad/Lora-Scanner/web"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	addr := flag.String("addr", env("LORA_ADDR", ":5000"), "listen address")
	cfgPath := flag.String("config", env("LORA_CONFIG", gateway.DefaultPath()), "gateway list file")
	flag.Parse()

	gin.SetMode(gin.ReleaseMode)

	h := hub.New()
	store := frames.NewStore(h, frames.DefaultMax)
	cfg := gateway.NewConfig(*cfgPath)
	if err := cfg.Load("gateway_config.json", filepath.Join(filepath.Dir(*cfgPath), "gateway_config.json")); err != nil {
		log.Fatalf("load config: %v", err)
	}
	scraper := gateway.NewScraper(cfg, store,
		env("LORA_GATEWAY_USER", "root"), env("LORA_GATEWAY_PASS", "dragino"))

	scraper.Hub = h

	ui, err := fs.Sub(web.FS, "dist")
	if err != nil {
		log.Fatal(err)
	}
	srv := &api.Server{
		Serial: serial.NewManager(h, store), Hub: h, Frames: store,
		Config: cfg, Scraper: scraper, UI: ui,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go scraper.Run(ctx)

	hs := &http.Server{Addr: *addr, Handler: srv.Router()}
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		hs.Shutdown(sctx)
	}()
	log.Printf("Lora Scanner listening on %s (gateway list: %s)", *addr, *cfgPath)
	if err := hs.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
