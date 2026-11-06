package upstream

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rejisterjack/keywarden/internal/fail"
	"github.com/rejisterjack/keywarden/internal/mcp"
	"github.com/rejisterjack/keywarden/internal/secret"
)

// GCPHTTP forwards one governed call to a Google Cloud HTTPS API with a sealed
// Bearer access token. It is a basic HTTPS+auth helper, not Cloud IAM.
type GCPHTTP struct {
	// HTTP is injectable for tests.
	HTTP *http.Client
}

// AzureHTTP forwards one governed call to an Azure HTTPS API with a sealed
// Bearer access token. It is a basic HTTPS+auth helper, not Entra/ARM IAM.
type AzureHTTP struct {
	// HTTP is injectable for tests.
	HTTP *http.Client
}

// List returns the single request tool.
func (g GCPHTTP) List(ctx context.Context, config, secretRaw []byte) ([]mcp.Tool, error) {
	return cloudList("GCP", config, secretRaw)
}

// Call executes one Bearer-authenticated request.
func (g GCPHTTP) Call(ctx context.Context, config, secretRaw, args []byte) (string, error) {
	return cloudCall(ctx, "gcp", PlainHTTP{HTTP: g.HTTP}, config, secretRaw, args)
}

// List returns the single request tool.
func (a AzureHTTP) List(ctx context.Context, config, secretRaw []byte) ([]mcp.Tool, error) {
	return cloudList("Azure", config, secretRaw)
}

// Call executes one Bearer-authenticated request.
func (a AzureHTTP) Call(ctx context.Context, config, secretRaw, args []byte) (string, error) {
	return cloudCall(ctx, "azure", PlainHTTP{HTTP: a.HTTP}, config, secretRaw, args)
}

func cloudList(label string, config, secretRaw []byte) ([]mcp.Tool, error) {
	if _, _, err := secret.CloudHTTP(config, secretRaw); err != nil {
		return nil, err
	}
	return []mcp.Tool{{
		Name:        "request",
		Description: "Call the configured " + label + " HTTPS API. Credentials stay in Keywarden.",
		InputSchema: json.RawMessage(plainHTTPToolSchema),
	}}, nil
}

func cloudCall(ctx context.Context, kind string, plain PlainHTTP, config, secretRaw, args []byte) (string, error) {
	cfg, sec, err := secret.CloudHTTP(config, secretRaw)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(sec.AccessToken) == "" {
		return "", fail.New(500, kind+" tool secret is invalid")
	}
	headers := map[string]string{}
	for k, v := range sec.Headers {
		headers[k] = v
	}
	headers["Authorization"] = "Bearer " + sec.AccessToken
	plainCfg, err := json.Marshal(secret.PlainHTTPConfig{
		BaseURL:        cfg.BaseURL,
		AllowedMethods: cfg.AllowedMethods,
		PathPrefix:     cfg.PathPrefix,
	})
	if err != nil {
		return "", err
	}
	plainSec, err := json.Marshal(map[string]any{"headers": headers})
	if err != nil {
		return "", err
	}
	return plain.Call(ctx, plainCfg, plainSec, args)
}
