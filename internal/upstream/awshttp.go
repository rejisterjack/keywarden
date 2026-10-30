package upstream

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/rejisterjack/keywarden/internal/fail"
	"github.com/rejisterjack/keywarden/internal/mcp"
	"github.com/rejisterjack/keywarden/internal/secret"
)

// awsMaxBody caps request and response bodies at 1 MiB.
const awsMaxBody = 1 << 20

// awsSignerTimeSkew signs slightly in the future to tolerate clock drift.
const awsSignerTimeSkew = 30 * time.Second

// AWSHTTP signs and forwards one governed call to an AWS service.
// Credentials never leave this struct: static keys or an AssumeRole session
// are used to SigV4-sign the outgoing request, and the agent only ever sees
// the response body.
type AWSHTTP struct {
	// HTTP is injectable for tests.
	HTTP *http.Client
	// Signer is injectable for tests; defaults to the SDK SigV4 signer.
	Signer interface {
		SignHTTP(ctx context.Context, credentials aws.Credentials, r *http.Request, payloadHash, service, region string, signingTime time.Time, optFns ...func(*awsSignerOptions)) error
	}
	// AssumeRole is injectable for tests; defaults to STS over HTTPS.
	AssumeRole func(ctx context.Context, keys secret.AWSSecret, cfg secret.AWSConfig) (aws.Credentials, error)
}

// awsSignerOptions mirrors aws/signer/v4.SignerOptions without importing it in
// the struct signature above; the SDK option functions remain compatible.
type awsSignerOptions = awsSignerOptionsAlias

// List returns the single request tool.
func (a AWSHTTP) List(ctx context.Context, config, secretRaw []byte) ([]mcp.Tool, error) {
	cfg, err := secret.AWS(config)
	if err != nil {
		return nil, err
	}
	if _, err := secret.AWSKeys(secretRaw); err != nil {
		return nil, err
	}
	return []mcp.Tool{{
		Name:        "request",
		Description: "Call the configured AWS " + cfg.Service + " API in " + cfg.Region + ". Credentials stay in Keywarden.",
		InputSchema: json.RawMessage(plainHTTPToolSchema),
	}}, nil
}

// Call executes one SigV4-signed request.
func (a AWSHTTP) Call(ctx context.Context, config, secretRaw, args []byte) (string, error) {
	cfg, err := secret.AWS(config)
	if err != nil {
		return "", err
	}
	keys, err := secret.AWSKeys(secretRaw)
	if err != nil {
		return "", err
	}
	var body struct {
		Method  string            `json:"method"`
		Path    string            `json:"path"`
		Headers map[string]string `json:"headers,omitempty"`
		Body    string            `json:"body,omitempty"`
	}
	if err := strictUnmarshal(args, &body); err != nil {
		return "", err
	}
	if body.Method == "" || body.Path == "" {
		return "", fail.New(400, "request requires method and path")
	}
	method := strings.ToUpper(body.Method)
	if !methodAllowed(cfg.AllowedMethods, method) {
		return "", fail.New(400, "method is not allowed for this tool")
	}
	base := cfg.BaseURL
	if base == "" {
		base = fmt.Sprintf("https://%s.%s.amazonaws.com", cfg.Service, cfg.Region)
	}
	target, err := joinPath(base, cfg.PathPrefix, body.Path)
	if err != nil {
		return "", err
	}
	creds, err := a.credentials(ctx, keys, cfg)
	if err != nil {
		return "", err
	}
	var payload io.Reader
	payloadHash := sha256.Sum256(nil)
	if body.Body != "" {
		if len(body.Body) > awsMaxBody {
			return "", fail.New(400, "request body is too large")
		}
		buf := []byte(body.Body)
		sum := sha256.Sum256(buf)
		payloadHash = sum
		payload = strings.NewReader(body.Body)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, payload)
	if err != nil {
		return "", fail.New(400, "request is invalid")
	}
	for k, v := range body.Headers {
		if strings.EqualFold(k, "Host") || strings.EqualFold(k, "Content-Length") ||
			strings.EqualFold(k, "Authorization") || strings.EqualFold(k, "X-Amz-Date") ||
			strings.EqualFold(k, "X-Amz-Security-Token") {
			return "", fail.New(400, "request headers cannot override signing headers")
		}
		req.Header.Set(k, v)
	}
	if body.Body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	signer := a.signer()
	if err := signer.SignHTTP(ctx, creds, req, hex.EncodeToString(payloadHash[:]), cfg.Service, cfg.Region, time.Now().Add(awsSignerTimeSkew)); err != nil {
		return "", fmt.Errorf("signing failed")
	}
	client := a.HTTP
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("upstream unavailable")
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, awsMaxBody+1))
	if err != nil {
		return "", fmt.Errorf("upstream unavailable")
	}
	if len(raw) > awsMaxBody {
		return "", fail.New(400, "response body is too large")
	}
	out := map[string]any{
		"status": res.StatusCode,
		"body":   string(raw),
	}
	enc, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(enc), nil
}

// credentials resolves static keys, or assumes the configured role with them.
func (a AWSHTTP) credentials(ctx context.Context, keys secret.AWSSecret, cfg secret.AWSConfig) (aws.Credentials, error) {
	static := aws.Credentials{
		AccessKeyID:     keys.AccessKeyID,
		SecretAccessKey: keys.SecretAccessKey,
		SessionToken:    keys.SessionToken,
	}
	if keys.RoleArn == "" {
		return static, nil
	}
	assume := a.AssumeRole
	if assume == nil {
		assume = assumeRoleSTS
	}
	return assume(ctx, keys, cfg)
}

func (a AWSHTTP) signer() interface {
	SignHTTP(ctx context.Context, credentials aws.Credentials, r *http.Request, payloadHash, service, region string, signingTime time.Time, optFns ...func(*awsSignerOptions)) error
} {
	if a.Signer != nil {
		return a.Signer
	}
	return defaultSigner{}
}

// assumeRoleSTS exchanges sealed caller keys for short-lived role credentials.
func assumeRoleSTS(ctx context.Context, keys secret.AWSSecret, cfg secret.AWSConfig) (aws.Credentials, error) {
	provider := awscreds.NewStaticCredentialsProvider(keys.AccessKeyID, keys.SecretAccessKey, keys.SessionToken)
	client := sts.New(sts.Options{
		Region:      cfg.Region,
		Credentials: aws.NewCredentialsCache(provider),
	})
	input := &sts.AssumeRoleInput{RoleArn: aws.String(keys.RoleArn), RoleSessionName: aws.String("keywarden")}
	if keys.ExternalID != "" {
		input.ExternalId = aws.String(keys.ExternalID)
	}
	out, err := client.AssumeRole(ctx, input)
	if err != nil {
		return aws.Credentials{}, fmt.Errorf("role assumption failed")
	}
	creds := out.Credentials
	if creds == nil {
		return aws.Credentials{}, fmt.Errorf("role assumption returned no credentials")
	}
	return aws.Credentials{
		AccessKeyID:     aws.ToString(creds.AccessKeyId),
		SecretAccessKey: aws.ToString(creds.SecretAccessKey),
		SessionToken:    aws.ToString(creds.SessionToken),
		Expires:         aws.ToTime(creds.Expiration),
		Source:          "AssumeRole",
		CanExpire:       true,
	}, nil
}

type defaultSigner struct{}

func (defaultSigner) SignHTTP(ctx context.Context, credentials aws.Credentials, r *http.Request, payloadHash, service, region string, signingTime time.Time, optFns ...func(*awsSignerOptions)) error {
	return sdkSignHTTP(ctx, credentials, r, payloadHash, service, region, signingTime, optFns...)
}
