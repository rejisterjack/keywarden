package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rejisterjack/keywarden/internal/config"
	"github.com/rejisterjack/keywarden/internal/fixture"
	"github.com/rejisterjack/keywarden/internal/httpapi"
	"github.com/rejisterjack/keywarden/internal/limit"
	"github.com/rejisterjack/keywarden/internal/logging"
	"github.com/rejisterjack/keywarden/internal/proxy"
	"github.com/rejisterjack/keywarden/internal/seed"
	"github.com/rejisterjack/keywarden/internal/store"
	"github.com/rejisterjack/keywarden/internal/upstream"
	"github.com/rejisterjack/keywarden/internal/vault"
	"log/slog"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "serve":
		err = serve()
	case "seed":
		err = runSeed()
	case "task":
		err = task(os.Args[2:])
	case "policy":
		err = policyCmd(os.Args[2:])
	case "audit":
		err = auditCmd(os.Args[2:])
	case "-h", "--help", "help":
		usage()
		return
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `Keywarden is a least-privilege proxy for agent tool calls.

Usage:
  keywarden serve
  keywarden seed
  keywarden task create --agent <id> [--ttl 15m] [--tool <id>]
  keywarden task revoke <task-id>
  keywarden policy apply <file>
  keywarden audit verify

serve and seed read KEYWARDEN_DATABASE_URL, KEYWARDEN_ADMIN_TOKEN, and KEYWARDEN_VAULT_KEY.
The other commands call KEYWARDEN_URL (default http://127.0.0.1:8080) with KEYWARDEN_ADMIN_TOKEN.
`)
}

func serve() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logging.New(os.Stdout)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	st, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		return err
	}
	if cfg.Seed {
		if err := seed.Run(ctx, st, seed.Options{OrdersDSN: cfg.OrdersDSN, FixtureURL: cfg.FixtureURL}); err != nil {
			return err
		}
	}
	if err := st.AssertDemoSecrets(ctx); err != nil {
		return err
	}
	n, err := st.InterruptInFlight(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		log.Info("expired in-flight calls from a previous process", "count", n)
	}
	api := &httpapi.Server{
		Store:   st,
		Proxy:   &proxy.Proxy{Store: st, Up: &upstream.Dispatcher{AllowStdio: cfg.AllowStdio && !cfg.Demo}},
		Cfg:     cfg,
		Log:     log,
		Limit:   limit.New(cfg.MCPRate, cfg.MCPBurst),
		Fixture: fixture.Handler(),
	}
	return listen(ctx, cfg, api, log)
}

func listen(ctx context.Context, cfg config.Config, api *httpapi.Server, log *slog.Logger) error {
	ln, err := net.Listen("tcp", cfg.HTTPAddr)
	if err != nil {
		return err
	}
	srv := &http.Server{
		Handler:           api.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      11 * time.Minute,
		IdleTimeout:       2 * time.Minute,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}
	errCh := make(chan error, 1)
	go func() {
		err := srv.Serve(ln)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()
	log.Info("listening", "addr", cfg.HTTPAddr, "demo", cfg.Demo)
	select {
	case <-ctx.Done():
		shut, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_ = srv.Shutdown(shut)
		return <-errCh
	case err := <-errCh:
		return err
	}
}

func runSeed() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	st, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		return err
	}
	return seed.Run(ctx, st, seed.Options{OrdersDSN: cfg.OrdersDSN, FixtureURL: cfg.FixtureURL})
}

func openStore(ctx context.Context, cfg config.Config) (*store.Store, error) {
	v, err := vault.New(cfg.VaultKey)
	if err != nil {
		return nil, err
	}
	return store.Open(ctx, store.Options{
		DatabaseURL:  cfg.DatabaseURL,
		Vault:        v,
		Demo:         cfg.Demo,
		AllowStdio:   cfg.AllowStdio && !cfg.Demo,
		FixtureURL:   cfg.FixtureURL,
		OrdersDSN:    cfg.OrdersDSN,
		SingleTenant: cfg.SingleTenant,
	})
}
