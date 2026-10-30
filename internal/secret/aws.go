package secret

import (
	"bytes"
	"net/url"
	"strings"

	"github.com/rejisterjack/keywarden/internal/fail"
)

// AWSConfig is the public configuration of an aws_http tool.
type AWSConfig struct {
	Service        string   `json:"service"`
	Region         string   `json:"region"`
	BaseURL        string   `json:"base_url,omitempty"`
	AllowedMethods []string `json:"allowed_methods,omitempty"`
	PathPrefix     string   `json:"path_prefix,omitempty"`
}

// AWSSecret is the sealed credential set of an aws_http tool. Either static
// keys or an AssumeRole caller: role_arn requires the caller keys.
type AWSSecret struct {
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
	SessionToken    string `json:"session_token,omitempty"`
	RoleArn         string `json:"role_arn,omitempty"`
	ExternalID      string `json:"external_id,omitempty"`
}

var awsServices = map[string]bool{
	"s3": true, "sns": true, "sqs": true, "sts": true, "dynamodb": true,
	"execute-api": true, "logs": true, "events": true, "lambda": true,
	"secretsmanager": true, "ssm": true, "kms": true, "iam": true,
	"ec2": true, "ecs": true, "rds": true, "cloudwatch": true,
}

var httpMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true, "HEAD": true,
}

// validatePlainHTTP checks an http tool. The base URL must be a public http or
// https origin without credentials, and the secret may only carry headers.
func validatePlainHTTP(config []byte, sec Secret) error {
	cfg, err := decodePlainHTTP(config)
	if err != nil {
		return err
	}
	if err := validateBaseURL(cfg.BaseURL, "http config base_url"); err != nil {
		return err
	}
	if err := validateMethodsAndPrefix(cfg.AllowedMethods, cfg.PathPrefix); err != nil {
		return err
	}
	if sec.DSN != "" {
		return fail.New(400, "http secret does not take dsn")
	}
	if len(sec.Env) > 0 {
		return fail.New(400, "http secret does not take env")
	}
	return nil
}

func decodePlainHTTP(config []byte) (PlainHTTPConfig, error) {
	var cfg PlainHTTPConfig
	if len(bytes.TrimSpace(config)) == 0 {
		return cfg, fail.New(400, "config is required")
	}
	dec := strictDecoder(config)
	if err := dec.Decode(&cfg); err != nil {
		return cfg, fail.New(400, "config must be a JSON object")
	}
	return cfg, nil
}

// ValidateAWS checks an aws_http tool config and secret.
func ValidateAWS(config []byte, sec Secret) error {
	var cfg AWSConfig
	if len(bytes.TrimSpace(config)) == 0 {
		return fail.New(400, "config is required")
	}
	dec := strictDecoder(config)
	if err := dec.Decode(&cfg); err != nil {
		return fail.New(400, "config must be a JSON object")
	}
	if !awsServices[strings.ToLower(cfg.Service)] {
		return fail.New(400, "aws config service is not in the supported set")
	}
	if !awsRegion(cfg.Region) {
		return fail.New(400, "aws config region is invalid")
	}
	if cfg.BaseURL != "" {
		if err := validateBaseURL(cfg.BaseURL, "aws config base_url"); err != nil {
			return err
		}
	}
	if err := validateMethodsAndPrefix(cfg.AllowedMethods, cfg.PathPrefix); err != nil {
		return err
	}
	keys := AWSSecret{
		AccessKeyID:     sec.AccessKeyID,
		SecretAccessKey: sec.SecretAccessKey,
		SessionToken:    sec.SessionToken,
		RoleArn:         sec.RoleArn,
		ExternalID:      sec.ExternalID,
	}
	if keys.AccessKeyID == "" || keys.SecretAccessKey == "" {
		return fail.New(400, "aws secret requires access_key_id and secret_access_key")
	}
	if keys.RoleArn != "" && !strings.HasPrefix(keys.RoleArn, "arn:aws:iam::") {
		return fail.New(400, "aws secret role_arn must be an IAM arn")
	}
	if len(sec.Headers) > 0 {
		return fail.New(400, "aws secret does not take headers")
	}
	if sec.DSN != "" {
		return fail.New(400, "aws secret does not take dsn")
	}
	if len(sec.Env) > 0 {
		return fail.New(400, "aws secret does not take env")
	}
	return nil
}

// AWSKeys decodes the sealed aws secret for the upstream signer.
func AWSKeys(raw []byte) (AWSSecret, error) {
	sec, err := Parse(raw)
	if err != nil {
		return AWSSecret{}, err
	}
	return AWSSecret{
		AccessKeyID:     sec.AccessKeyID,
		SecretAccessKey: sec.SecretAccessKey,
		SessionToken:    sec.SessionToken,
		RoleArn:         sec.RoleArn,
		ExternalID:      sec.ExternalID,
	}, nil
}

// PlainHTTP decodes an http tool's public config and sealed secret headers.
func PlainHTTP(config, secretRaw []byte) (PlainHTTPConfig, Secret, error) {
	cfg, err := decodePlainHTTP(config)
	if err != nil {
		return PlainHTTPConfig{}, Secret{}, err
	}
	sec, err := Parse(secretRaw)
	if err != nil {
		return PlainHTTPConfig{}, Secret{}, err
	}
	return cfg, sec, nil
}

// AWS decodes an aws_http tool's public config.
func AWS(config []byte) (AWSConfig, error) {
	var cfg AWSConfig
	if len(bytes.TrimSpace(config)) == 0 {
		return cfg, fail.New(400, "config is required")
	}
	dec := strictDecoder(config)
	if err := dec.Decode(&cfg); err != nil {
		return cfg, fail.New(400, "config must be a JSON object")
	}
	return cfg, nil
}

func validateBaseURL(raw, label string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fail.New(400, label+" must be http or https")
	}
	if u.User != nil {
		return fail.New(400, label+" must not contain credentials")
	}
	host := strings.ToLower(u.Hostname())
	if host == "169.254.169.254" || host == "metadata.google.internal" {
		return fail.New(400, label+" must not point at a cloud metadata address")
	}
	if u.Path != "" && u.Path != "/" {
		return fail.New(400, label+" must be an origin without a path")
	}
	return nil
}

func validateMethodsAndPrefix(methods []string, prefix string) error {
	if len(methods) == 0 {
		return fail.New(400, "config allowed_methods is required")
	}
	if len(methods) > 8 {
		return fail.New(400, "config allowed_methods has too many entries")
	}
	seen := map[string]bool{}
	for _, m := range methods {
		upper := strings.ToUpper(m)
		if !httpMethods[upper] {
			return fail.New(400, "config allowed_methods has an unsupported method")
		}
		if seen[upper] {
			return fail.New(400, "config allowed_methods repeats a method")
		}
		seen[upper] = true
	}
	if prefix != "" {
		if !strings.HasPrefix(prefix, "/") || strings.Contains(prefix, "?") || strings.Contains(prefix, "#") ||
			strings.Contains(prefix, "..") || len(prefix) > 256 {
			return fail.New(400, "config path_prefix must be a simple path prefix")
		}
	}
	return nil
}

func awsRegion(region string) bool {
	if len(region) < 3 || len(region) > 20 {
		return false
	}
	for _, r := range region {
		if !(r == '-' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}
