package secret

import "testing"

func TestDemoRejectsForeignUpstream(t *testing.T) {
	fixture := "http://127.0.0.1:8080/demo/mcp"
	orders := "postgres://orders_owner:fake-orders-password@postgres:5432/orders"
	err := Validate("mcp_http", []byte(`{"url":"https://evil.example/mcp"}`), []byte(`{"kw_demo":"fake"}`), true, fixture, orders)
	if err == nil {
		t.Fatal("expected foreign MCP url to be rejected")
	}
	err = Validate("postgres", []byte(`{}`), []byte(`{"dsn":"postgres://keywarden:keywarden@postgres:5432/keywarden","kw_demo":"fake"}`), true, fixture, orders)
	if err == nil {
		t.Fatal("expected the control-plane database to be rejected")
	}
	err = Validate("mcp_http", []byte(`{"url":"http://127.0.0.1:8080/demo/mcp"}`), []byte(`{"kw_demo":"fake"}`), true, fixture, orders)
	if err != nil {
		t.Fatal(err)
	}
	err = Validate("postgres", []byte(`{}`), []byte(`{"dsn":"postgres://orders_owner:other@postgres:5432/orders?sslmode=disable","kw_demo":"fake"}`), true, fixture, orders)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDemoRequiresFakeMarker(t *testing.T) {
	err := Validate("mcp_http", []byte(`{"url":"http://127.0.0.1:8080/demo/mcp"}`), []byte(`{"headers":{"Authorization":"Bearer real"}}`), true, "http://127.0.0.1:8080/demo/mcp", "")
	if err == nil {
		t.Fatal("expected a real secret to be rejected")
	}
}
