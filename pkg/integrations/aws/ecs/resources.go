package ecs

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const describeTasksMaxBatchSize = 100

func ListClusters(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := ctx.Parameters["region"]
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	clusters, err := client.ListClusters()
	if err != nil {
		return nil, fmt.Errorf("failed to list ECS clusters: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(clusters))
	for _, cluster := range clusters {
		name := cluster.ClusterName
		if name == "" {
			name = clusterNameFromArn(cluster.ClusterArn)
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: name,
			ID:   cluster.ClusterArn,
		})
	}

	return resources, nil
}

func ListServices(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := ctx.Parameters["region"]
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	cluster := ctx.Parameters["cluster"]
	if cluster == "" {
		return nil, fmt.Errorf("cluster is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	serviceArns, err := client.ListServices(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to list ECS services: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(serviceArns))
	for _, arn := range serviceArns {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: serviceNameFromArn(arn),
			ID:   arn,
		})
	}

	return resources, nil
}

func ListTaskDefinitions(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := ctx.Parameters["region"]
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	taskDefinitionArns, err := client.ListTaskDefinitions()
	if err != nil {
		return nil, fmt.Errorf("failed to list ECS task definitions: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(taskDefinitionArns))
	for _, arn := range taskDefinitionArns {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: taskDefinitionNameFromArn(arn),
			ID:   arn,
		})
	}

	return resources, nil
}

func ListTasks(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := ctx.Parameters["region"]
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	cluster := ctx.Parameters["cluster"]
	if cluster == "" {
		return nil, fmt.Errorf("cluster is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	taskArns, err := client.ListTasks(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to list ECS tasks: %w", err)
	}
	if len(taskArns) == 0 {
		return []core.IntegrationResource{}, nil
	}

	taskResourceNames := listTaskResourceNamesByDescribe(cluster, taskArns, client.DescribeTasks)

	resources := make([]core.IntegrationResource, 0, len(taskArns))
	for _, arn := range taskArns {
		name := taskResourceNames[arn]
		if name == "" {
			name = taskIDFromArn(arn)
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: name,
			ID:   arn,
		})
	}

	return resources, nil
}

func listTaskResourceNamesByDescribe(
	cluster string,
	taskArns []string,
	describeTasks func(string, []string) (*DescribeTasksResponse, error),
) map[string]string {
	taskResourceNames := make(map[string]string, len(taskArns))

	for _, batch := range batchTaskARNs(taskArns, describeTasksMaxBatchSize) {
		describeResponse, err := describeTasks(cluster, batch)
		if err != nil {
			continue
		}

		for _, task := range describeResponse.Tasks {
			if task.TaskArn == "" {
				continue
			}

			taskResourceNames[task.TaskArn] = formatTaskResourceName(task)
		}
	}

	return taskResourceNames
}

func batchTaskARNs(taskArns []string, batchSize int) [][]string {
	if len(taskArns) == 0 {
		return [][]string{}
	}

	if batchSize <= 0 {
		return [][]string{taskArns}
	}

	batches := make([][]string, 0, (len(taskArns)+batchSize-1)/batchSize)
	for start := 0; start < len(taskArns); start += batchSize {
		end := start + batchSize
		if end > len(taskArns) {
			end = len(taskArns)
		}

		batches = append(batches, taskArns[start:end])
	}

	return batches
}

func formatTaskResourceName(task Task) string {
	taskID := taskIDFromArn(task.TaskArn)
	taskDefinition := taskDefinitionNameFromArn(task.TaskDefinitionArn)
	status := task.LastStatus

	if taskDefinition != "" && status != "" && taskID != "" {
		return fmt.Sprintf("%s (%s) %s", taskDefinition, status, taskID)
	}

	if taskDefinition != "" && taskID != "" {
		return fmt.Sprintf("%s %s", taskDefinition, taskID)
	}

	if taskID != "" {
		return taskID
	}

	return task.TaskArn
}
