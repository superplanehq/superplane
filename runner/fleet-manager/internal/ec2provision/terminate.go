package ec2provision

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// TerminateInstance requests EC2 termination for the given instance id (e.g. i-0123…).
func (l *Launcher) TerminateInstance(ctx context.Context, instanceID string) error {
	id := strings.TrimSpace(instanceID)
	if id == "" {
		return fmt.Errorf("empty instance id")
	}
	_, err := l.Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{id},
	})
	return err
}
