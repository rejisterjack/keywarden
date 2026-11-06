package secret

import (
	"bytes"
	"strings"

	"github.com/rejisterjack/keywarden/internal/fail"
)

// CloudHTTPConfig is the public configuration of a gcp_http or azure_http tool.
// Same shape as PlainHTTPConfig: HTTPS origin, method allow-list, path prefix.
type CloudHTTPConfig struct {
	BaseURL        string   `json:"base_url"`
	AllowedMethods []string `json:"allowed_methods,omitempty"`
	PathPrefix     string   `json:"path_prefix,omitempty"`
}

// validateGCPHTTP checks a gcp_http tool. Auth is a sealed Bearer access token
// (and optional extra headers). This is not Cloud IAM or Workload Identity.
func validateGCPHTTP(config []byte, sec Secret) error {
	return validateCloudHTTP("gcp", config, sec)
}

// validateAzureHTTP checks an azure_http tool. Auth is a sealed Bearer access
// token (and optional extra headers). This is not Entra ID / ARM IAM.
func validateAzureHTTP(config []byte, sec Secret) error {
	return validateCloudHTTP("azure", config, sec)
}

func validateCloudHTTP(kind string, config []byte, sec Secret) error {
	cfg, err := decodeCloudHTTP(config)
	if err != nil {
		return err
	}
	if err := validateBaseURL(cfg.BaseURL, kind+" config base_url"); err != nil {
		return err
	}
	if err := validateMethodsAndPrefix(cfg.AllowedMethods, cfg.PathPrefix); err != nil {
		return err
	}
	if strings.TrimSpace(sec.AccessToken) == "" {
		return fail.New(400, kind+" secret requires access_token")
	}
	if len(sec.AccessToken) > 16*1024 {
		return fail.New(400, kind+" secret access_token is too long")
	}
	if sec.DSN != "" {
		return fail.New(400, kind+" secret does not take dsn")
	}
	if len(sec.Env) > 0 {
		return fail.New(400, kind+" secret does not take env")
	}
	if sec.AccessKeyID != "" || sec.SecretAccessKey != "" || sec.RoleArn != "" {
		return fail.New(400, kind+" secret does not take aws credentials")
	}
	for k := range sec.Headers {
		if strings.EqualFold(k, "Authorization") {
			return fail.New(400, kind+" secret headers cannot set Authorization; use access_token")
		}
	}
	return nil
}

func decodeCloudHTTP(config []byte) (CloudHTTPConfig, error) {
	var cfg CloudHTTPConfig
	if len(bytes.TrimSpace(config)) == 0 {
		return cfg, fail.New(400, "config is required")
	}
	dec := strictDecoder(config)
	if err := dec.Decode(&cfg); err != nil {
		return cfg, fail.New(400, "config must be a JSON object")
	}
	return cfg, nil
}

// CloudHTTP decodes a gcp_http or azure_http tool's public config and sealed secret.
func CloudHTTP(config, secretRaw []byte) (CloudHTTPConfig, Secret, error) {
	cfg, err := decodeCloudHTTP(config)
	if err != nil {
		return CloudHTTPConfig{}, Secret{}, err
	}
	sec, err := Parse(secretRaw)
	if err != nil {
		return CloudHTTPConfig{}, Secret{}, err
	}
	return cfg, sec, nil
}
