package hosttags

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cloud.google.com/go/compute/metadata"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromEC2ReadsInstanceTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest/api/token":
			w.Header().Set("X-Aws-Ec2-Metadata-Token-Ttl-Seconds", "21600")
			_, _ = w.Write([]byte("token"))
		case "/latest/dynamic/instance-identity/document":
			assert.Equal(t, "token", r.Header.Get("X-aws-ec2-metadata-token"))
			_, _ = w.Write([]byte(`{"instanceId":"i-123","instanceType":"m7i.large","imageId":"ami-123","region":"us-east-1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tags, err := fromEC2(t.Context(), imds.New(imds.Options{
		Endpoint: server.URL, EnableFallback: aws.FalseTernary,
	}))
	require.NoError(t, err)
	assert.Equal(t, "i-123", tags["ec2_instance_id"])
	assert.Equal(t, "m7i.large", tags["ec2_instance_type"])
}

func TestFromGCPReadsInstanceTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Google", r.Header.Get("Metadata-Flavor"))
		values := map[string]string{
			"/computeMetadata/v1/instance/id":        "12345",
			"/computeMetadata/v1/project/project-id": "project-a",
			"/computeMetadata/v1/instance/zone":      "projects/123/zones/us-central1-b",
		}
		value, ok := values[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, value)
	}))
	defer server.Close()
	t.Setenv("GCE_METADATA_HOST", strings.TrimPrefix(server.URL, "http://"))

	tags, err := fromGCP(context.Background(), metadata.NewClient(server.Client()))
	require.NoError(t, err)
	assert.Equal(t, "12345", tags["gcp_instance_id"])
	assert.Equal(t, "project-a", tags["gcp_project_id"])
	assert.Equal(t, "us-central1-b", tags["gcp_zone"])
}
