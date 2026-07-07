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
	"strings"
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
	case "-h", "--help", "help":
		printRootHelp()
	case "serve":
		runServe(args)
	case "openapi":
		runOpenAPI(args)
	case "version":
		fmt.Println(version.Version)
	default:
		if len(cmd) > 0 && cmd[0] == '-' {
			runServe(os.Args[1:])
			return
		}
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		printRootHelp()
		os.Exit(2)
	}
}

func printRootHelp() {
	fmt.Fprintf(os.Stdout, `Cerebro web admin for Elasticsearch and OpenSearch.

Usage:
  cerebro [command]

Commands:
  serve      Run the web application
  openapi    Generate the OpenAPI document
  version    Print the version

Run "cerebro [command] -h" for command-specific flags.
`)
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	configPath := fs.String("config", "conf/application.yaml", "path to config file")
	_ = fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}
	configureLogging(cfg.Logging)
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
		Cfg:     cfg,
		Client:  client,
		History: store,
		Auth:    authMod,
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if cfg.Server.Secret == "" || cfg.Server.Secret == "change-me" || cfg.Server.Secret == "dev-secret-change-me" {
		slog.Warn("server.secret is empty or set to a default placeholder — set APPLICATION_SECRET to a strong, random value before exposing this instance")
	}
	authProviders := enabledAuthProviders(cfg.Auth)
	if authProviders == "disabled" {
		slog.Warn("authentication is disabled — anyone reaching this port can manage the configured Elasticsearch clusters")
		if cfg.Server.CSRFEnabled {
			slog.Warn("csrf is enabled without authentication — csrf protects browsers from cross-site requests but does not restrict direct API clients")
		}
	}
	slog.Info("cerebro starting", "addr", fmt.Sprintf(":%d", cfg.Server.Port), "scheme", srv.Scheme(), "auth", authProviders, "hosts", len(cfg.Hosts))
	if err := srv.Run(ctx); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func configureLogging(cfg config.Logging) {
	level := new(slog.LevelVar)
	switch cfg.Level {
	case "debug":
		level.Set(slog.LevelDebug)
	case "warn":
		level.Set(slog.LevelWarn)
	case "error":
		level.Set(slog.LevelError)
	default:
		level.Set(slog.LevelInfo)
	}
	opts := &slog.HandlerOptions{Level: level}
	if cfg.Format == "json" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, opts)))
		return
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, opts)))
}

func runOpenAPI(args []string) {
	fs := flag.NewFlagSet("openapi", flag.ExitOnError)
	configPath := fs.String("config", "conf/application.yaml", "path to config file (used to bootstrap routes)")
	_ = fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		// Tolerate missing config — generate spec with defaults.
		cfg = &config.Config{Server: config.Server{Port: 9000, BasePath: "/", Secret: "x"}}
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

func enabledAuthProviders(cfg config.Auth) string {
	providers := []string{}
	if cfg.Basic.Enabled {
		providers = append(providers, "basic")
	}
	if cfg.LDAP.Enabled {
		providers = append(providers, "ldap")
	}
	if cfg.Proxy.Enabled {
		providers = append(providers, "proxy")
	}
	if cfg.EntraID.Enabled {
		providers = append(providers, "entra_id")
	}
	if len(providers) == 0 {
		return "disabled"
	}
	return strings.Join(providers, ",")
}
