package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lmenezes/cerebro/internal/auth"
	"github.com/lmenezes/cerebro/internal/config"
	"github.com/lmenezes/cerebro/internal/elastic"
	"github.com/lmenezes/cerebro/internal/history"
	"github.com/lmenezes/cerebro/internal/server"
	"github.com/lmenezes/cerebro/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		runServe(os.Args[1:])
		return
	}
	cmd := os.Args[1]
	args := os.Args[2:]
	switch cmd {
	case "serve":
		runServe(args)
	case "openapi":
		runOpenAPI(args)
	case "version":
		fmt.Println(version.Version)
	default:
		runServe(os.Args[1:])
	}
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	configPath := fs.String("config", "conf/application.yaml", "path to config file")
	publicDir := fs.String("public", "public", "directory with static frontend assets")
	_ = fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}
	authMod, err := auth.NewModule(cfg)
	if err != nil {
		slog.Error("init auth", "err", err)
		os.Exit(1)
	}
	hc := &http.Client{Timeout: 60 * time.Second}
	client, err := elastic.NewHTTPClientWithConfig(hc, cfg.ES)
	if err != nil {
		slog.Error("init elasticsearch client", "err", err)
		os.Exit(1)
	}

	store, err := history.Open(cfg.Data.Path, cfg.Rest.HistorySize)
	if err != nil {
		slog.Error("open history db", "err", err)
		os.Exit(1)
	}
	defer store.Close()

	srv := server.New(server.Options{
		Cfg:       cfg,
		Client:    client,
		History:   store,
		Auth:      authMod,
		PublicDir: *publicDir,
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if cfg.Server.Secret == "" || cfg.Server.Secret == "change-me" || cfg.Server.Secret == "dev-secret-change-me" {
		slog.Warn("server.secret is empty or set to a default placeholder — set APPLICATION_SECRET to a strong, random value before exposing this instance")
	}
	if cfg.Auth.Type == "disabled" {
		slog.Warn("authentication is disabled — anyone reaching this port can manage the configured Elasticsearch clusters")
	}
	slog.Info("cerebro starting", "addr", fmt.Sprintf(":%d", cfg.Server.Port), "auth", cfg.Auth.Type, "hosts", len(cfg.Hosts))
	if err := srv.Run(ctx); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func runOpenAPI(args []string) {
	fs := flag.NewFlagSet("openapi", flag.ExitOnError)
	configPath := fs.String("config", "conf/application.yaml", "path to config file (used to bootstrap routes)")
	_ = fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		// Tolerate missing config — generate spec with defaults.
		cfg = &config.Config{Server: config.Server{Port: 9000, BasePath: "/", Secret: "x"}, Auth: config.Auth{Type: "disabled"}}
	}
	authMod, err := auth.NewModule(cfg)
	if err != nil {
		slog.Error("init auth", "err", err)
		os.Exit(1)
	}
	client, err := elastic.NewHTTPClientWithConfig(nil, cfg.ES)
	if err != nil {
		slog.Error("init elasticsearch client", "err", err)
		os.Exit(1)
	}
	srv := server.New(server.Options{
		Cfg:    cfg,
		Client: client,
		Auth:   authMod,
	})
	out, _ := json.MarshalIndent(srv.HumaAPI().OpenAPI(), "", "  ")
	fmt.Println(string(out))
}
