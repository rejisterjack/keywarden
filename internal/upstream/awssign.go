package upstream

import (
	"context"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

// awsSignerOptionsAlias mirrors the SDK signer options so the injectable
// signer interface stays narrow without exporting SDK types.
type awsSignerOptionsAlias = v4.SignerOptions

var sdkSigner = v4.NewSigner()

func sdkSignHTTP(ctx context.Context, credentials aws.Credentials, r *http.Request, payloadHash, service, region string, signingTime time.Time, optFns ...func(*awsSignerOptions)) error {
	return sdkSigner.SignHTTP(ctx, credentials, r, payloadHash, service, region, signingTime, optFns...)
}
