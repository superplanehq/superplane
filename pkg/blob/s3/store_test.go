package s3

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
)

func TestPutGetAndSignedGetURL(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")

	const payload = "artifact-bytes"
	var sawPut bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			sawPut = true
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != payload {
				http.Error(w, "bad body", http.StatusBadRequest)
				return
			}
			if r.Header.Get("X-Amz-Date") == "" && r.URL.Query().Get("X-Amz-Date") == "" {
				http.Error(w, "missing signature date", http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			w.Header().Set("Content-Type", "text/plain")
			_, _ = io.WriteString(w, payload)
		default:
			http.Error(w, r.Method, http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(srv.Close)

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion("us-east-1"))
	require.NoError(t, err)
	client := awss3.NewFromConfig(cfg, func(o *awss3.Options) {
		o.BaseEndpoint = aws.String(srv.URL)
		o.UsePathStyle = true
	})
	store := &Store{
		bucket:    "artifacts",
		client:    client,
		presigner: awss3.NewPresignClient(client),
	}

	err = store.Put(context.Background(), "runs/1", strings.NewReader(payload), blob.PutOptions{ContentType: "text/plain"})
	require.NoError(t, err)
	assert.True(t, sawPut)

	rc, err := store.Get(context.Background(), "runs/1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = rc.Close() })
	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, payload, string(got))

	signed, err := store.SignedGetURL(context.Background(), "runs/1", time.Hour)
	require.NoError(t, err)
	assert.Contains(t, signed, "runs/1")
	assert.Contains(t, signed, blob.SignedURLMarkerParam+"="+blob.SignedURLMarkerValue)
	assert.Contains(t, signed, "X-Amz-Signature=")
}
