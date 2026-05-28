package ec2provision

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// ManagedRunnerSummary is a minimal row for admin listing.
type ManagedRunnerSummary struct {
	InstanceID string `json:"instance_id"`
	State      string `json:"state"`
	PrivateIP  string `json:"private_ip,omitempty"`
	PublicIP   string `json:"public_ip,omitempty"`
}

// ListManagedRunners returns pending/running instances tagged as fleet-managed runners
// for this launcher's pool (scoped by TagKeyFleetID).
func (l *Launcher) ListManagedRunners(ctx context.Context) ([]ManagedRunnerSummary, error) {
	in := &ec2.DescribeInstancesInput{
		Filters: managedDescribeFilters(l.Config.RunnerFleetID, []string{"pending", "running", "stopping", "shutting-down"}),
	}
	pager := ec2.NewDescribeInstancesPaginator(l.Client, in)
	var out []ManagedRunnerSummary
	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, rv := range page.Reservations {
			for _, inst := range rv.Instances {
				s := ManagedRunnerSummary{
					InstanceID: aws.ToString(inst.InstanceId),
					State:      string(inst.State.Name),
				}
				if inst.PrivateIpAddress != nil {
					s.PrivateIP = aws.ToString(inst.PrivateIpAddress)
				}
				if inst.PublicIpAddress != nil {
					s.PublicIP = aws.ToString(inst.PublicIpAddress)
				}
				out = append(out, s)
			}
		}
	}
	return out, nil
}

// ManagedInstanceConsoleOutput returns decoded EC2 console output for a managed runner (boot / cloud-init).
func (l *Launcher) ManagedInstanceConsoleOutput(ctx context.Context, instanceID string) (string, error) {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return "", fmt.Errorf("instance_id query parameter required")
	}
	if !strings.HasPrefix(instanceID, "i-") || len(instanceID) < 10 {
		return "", fmt.Errorf("invalid instance id")
	}

	desc, err := l.Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return "", err
	}
	ok := false
	for _, res := range desc.Reservations {
		for _, inst := range res.Instances {
			if aws.ToString(inst.InstanceId) != instanceID {
				continue
			}
			for _, t := range inst.Tags {
				if aws.ToString(t.Key) == TagKeyManaged && aws.ToString(t.Value) == "true" {
					ok = true
					break
				}
			}
		}
	}
	if !ok {
		return "", fmt.Errorf("instance not found or not a managed runner")
	}

	co, err := l.Client.GetConsoleOutput(ctx, &ec2.GetConsoleOutputInput{
		InstanceId: aws.String(instanceID),
		Latest:     aws.Bool(true),
	})
	if err != nil {
		return "", err
	}
	raw := aws.ToString(co.Output)
	if raw == "" {
		return "(no console output yet — instance may still be booting)", nil
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", fmt.Errorf("decode console output: %w", err)
	}
	return string(decoded), nil
}
