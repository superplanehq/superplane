package ec2provision

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type managedInstance struct {
	id        string
	launchUTC time.Time
	state     string
	privateIP string
}

// SweepUnhealthy terminates running managed instances that fail GET /healthz on the private IP
// after the boot grace period.
func (l *Launcher) SweepUnhealthy(ctx context.Context) ([]string, error) {
	instances, err := l.listManagedInstances(ctx)
	if err != nil {
		return nil, err
	}
	grace := time.Duration(l.Config.BootGraceSec) * time.Second
	port := l.Config.RunnerHealthPort
	if port <= 0 {
		port = 9090
	}
	client := &http.Client{Timeout: 5 * time.Second}

	var terminate []string
	for _, inst := range instances {
		if inst.state != string(types.InstanceStateNameRunning) {
			continue
		}
		if time.Since(inst.launchUTC) < grace {
			continue
		}
		if inst.privateIP == "" {
			terminate = append(terminate, inst.id)
			continue
		}
		url := fmt.Sprintf("http://%s:%d/healthz", inst.privateIP, port)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			terminate = append(terminate, inst.id)
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			terminate = append(terminate, inst.id)
			continue
		}
		ok := resp.StatusCode == http.StatusOK
		_ = resp.Body.Close()
		if !ok {
			terminate = append(terminate, inst.id)
		}
	}
	if len(terminate) == 0 {
		return nil, nil
	}
	_, err = l.Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{InstanceIds: terminate})
	if err != nil {
		return nil, fmt.Errorf("terminate unhealthy: %w", err)
	}
	if l.Log != nil {
		l.Log.Info("ec2 terminated unhealthy runners", slog.Int("count", len(terminate)), slog.Any("instance_ids", terminate))
	}
	return terminate, nil
}

func (l *Launcher) listManagedInstances(ctx context.Context) ([]managedInstance, error) {
	in := &ec2.DescribeInstancesInput{
		Filters: managedDescribeFilters(l.Config.RunnerFleetID, []string{"pending", "running"}),
	}
	pager := ec2.NewDescribeInstancesPaginator(l.Client, in)
	var out []managedInstance
	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, rv := range page.Reservations {
			for _, inst := range rv.Instances {
				if inst.InstanceId == nil || inst.LaunchTime == nil {
					continue
				}
				row := managedInstance{
					id:        aws.ToString(inst.InstanceId),
					launchUTC: *inst.LaunchTime,
					state:     string(inst.State.Name),
				}
				if inst.PrivateIpAddress != nil {
					row.privateIP = aws.ToString(inst.PrivateIpAddress)
				}
				out = append(out, row)
			}
		}
	}
	return out, nil
}
