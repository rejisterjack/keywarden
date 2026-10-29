package seed

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/rejisterjack/keywarden/internal/fail"
	"github.com/rejisterjack/keywarden/internal/store"
)

// Options describes the fake data a demo deployment is allowed to hold.
type Options struct {
	OrdersDSN  string
	FixtureURL string
}

const policyDocument = `{
  "version": 1,
  "rules": [
    {"id": "orders-read", "match": {"agent": "support", "tool": "orders", "name": "query"}, "when": {"sql": "read"}, "effect": "allow"},
    {"id": "orders-mutate", "match": {"agent": "support", "tool": "orders", "name": "query"}, "when": {"sql": "mutating"}, "effect": "require_approval", "timeout_seconds": 120},
    {"id": "orders-destroy", "match": {"agent": "support", "tool": "orders", "name": "query"}, "when": {"sql": "destructive"}, "effect": "deny"},
    {"id": "catalog-lookup", "match": {"agent": "support", "tool": "catalog", "name": "lookup_order"}, "effect": "allow"},
    {"id": "catalog-refund", "match": {"agent": "support", "tool": "catalog", "name": "refund_order"}, "effect": "require_approval"},
    {"id": "catalog-delete", "match": {"agent": "support", "tool": "catalog", "name": "delete_catalog"}, "effect": "deny"}
  ]
}`

// Run inserts the fictional support agent, orders database, catalog fixture, and sample policy.
// Existing rows are left in place.
func Run(ctx context.Context, st *store.Store, opt Options) error {
	if opt.OrdersDSN == "" {
		return fail.New(400, "KEYWARDEN_ORDERS_DSN is required to seed the orders tool")
	}
	if opt.FixtureURL == "" {
		return fail.New(400, "fixture URL is required to seed the catalog tool")
	}
	if err := ensureOrders(ctx, opt.OrdersDSN); err != nil {
		return err
	}
	if _, err := st.GetAgent(ctx, "support"); fail.IsStatus(err, 404) {
		if _, err := st.CreateAgent(ctx, "support", "Support"); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if _, err := st.GetTool(ctx, "orders"); fail.IsStatus(err, 404) {
		secretRaw, _ := json.Marshal(map[string]string{"dsn": opt.OrdersDSN, "kw_demo": "fake"})
		if _, err := st.CreateTool(ctx, store.CreateToolInput{
			ID: "orders", Name: "Orders database", Kind: "postgres",
			Config: json.RawMessage(`{}`), Secret: secretRaw, Seeded: true,
		}); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if _, err := st.GetTool(ctx, "catalog"); fail.IsStatus(err, 404) {
		cfg, _ := json.Marshal(map[string]string{"url": opt.FixtureURL})
		secretRaw, _ := json.Marshal(map[string]any{"kw_demo": "fake", "headers": map[string]string{}})
		if _, err := st.CreateTool(ctx, store.CreateToolInput{
			ID: "catalog", Name: "Catalog", Kind: "mcp_http",
			Config: cfg, Secret: secretRaw, Seeded: true,
		}); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if _, err := st.ActivePolicy(ctx); fail.IsStatus(err, 404) {
		if _, err := st.ApplyPolicy(ctx, []byte(policyDocument)); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

func ensureOrders(ctx context.Context, dsn string) error {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS orders (
			id integer PRIMARY KEY,
			status text NOT NULL,
			customer text NOT NULL
		)`); err != nil {
		return err
	}
	_, err = conn.Exec(ctx, `
		INSERT INTO orders (id, status, customer)
		VALUES (1042, 'shipped', 'Ada Lovelace')
		ON CONFLICT (id) DO NOTHING`)
	return err
}
