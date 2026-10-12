package hosttags

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
)

const discoveryTimeout = time.Second

func FromEC2(ctx context.Context) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(ctx, discoveryTimeout)
	defer cancel()
	return fromEC2(ctx, imds.New(imds.Options{EnableFallback: aws.FalseTernary}))
}

func fromEC2(ctx context.Context, client *imds.Client) (map[string]string, error) {
	identity, err := client.GetInstanceIdentityDocument(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("read EC2 instance identity: %w", err)
	}
	if identity.InstanceID == "" {
		return nil, fmt.Errorf("EC2 instance identity has no instance ID")
	}
	return map[string]string{
		"ec2_instance_id":   identity.InstanceID,
		"ec2_instance_type": identity.InstanceType,
		"ec2_ami_id":        identity.ImageID,
		"ec2_region":        identity.Region,
	}, nil
}

func FromGCP(ctx context.Context) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(ctx, discoveryTimeout)
	defer cancel()
	return fromGCP(ctx, metadata.NewClient(nil))
}

func fromGCP(ctx context.Context, client *metadata.Client) (map[string]string, error) {
	instanceID, err := client.InstanceIDWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("read Google Cloud instance ID: %w", err)
	}
	if instanceID == "" {
		return nil, fmt.Errorf("Google Cloud metadata has no instance ID")
	}
	tags := map[string]string{"gcp_instance_id": instanceID}
	if projectID, err := client.ProjectIDWithContext(ctx); err == nil {
		tags["gcp_project_id"] = projectID
	}
	if zone, err := client.ZoneWithContext(ctx); err == nil {
		tags["gcp_zone"] = zone
	}
	return tags, nil
}
