package aws

import (
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/codeartifact"
	"github.com/superplanehq/superplane/pkg/integrations/aws/codepipeline"
	"github.com/superplanehq/superplane/pkg/integrations/aws/ec2"
	"github.com/superplanehq/superplane/pkg/integrations/aws/ecr"
	"github.com/superplanehq/superplane/pkg/integrations/aws/ecs"
	"github.com/superplanehq/superplane/pkg/integrations/aws/lambda"
	"github.com/superplanehq/superplane/pkg/integrations/aws/prometheus"
	"github.com/superplanehq/superplane/pkg/integrations/aws/route53"
	"github.com/superplanehq/superplane/pkg/integrations/aws/s3"
	"github.com/superplanehq/superplane/pkg/integrations/aws/sns"
	"github.com/superplanehq/superplane/pkg/integrations/aws/sqs"
)

func (a *AWS) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "lambda.function":
		return lambda.ListFunctions(ctx, resourceType)

	case "ecr.repository":
		return ecr.ListRepositories(ctx, resourceType)

	case "ecs.cluster":
		return ecs.ListClusters(ctx, resourceType)

	case "ecs.service":
		return ecs.ListServices(ctx, resourceType)

	case "ecs.taskDefinition":
		return ecs.ListTaskDefinitions(ctx, resourceType)

	case "ecs.task":
		return ecs.ListTasks(ctx, resourceType)

	case "ec2.instance":
		return ec2.ListInstances(ctx, resourceType)

	case "ec2.loadBalancer":
		return ec2.ListLoadBalancers(ctx, resourceType)

	case "ec2.targetGroup":
		return ec2.ListTargetGroups(ctx, resourceType)

	case "ec2.image":
		return ec2.ListImages(ctx, resourceType)

	case "ec2.imageOS":
		return ec2.ListImageOperatingSystems(ctx, resourceType)

	case "ec2.instanceType":
		return ec2.ListInstanceTypes(ctx, resourceType)

	case "ec2.subnet":
		return ec2.ListSubnets(ctx, resourceType)

	case "ec2.securityGroup":
		return ec2.ListSecurityGroups(ctx, resourceType)

	case "ec2.keyPair":
		return ec2.ListKeyPairs(ctx, resourceType)

	case "ec2.alarm":
		return ec2.ListAlarms(ctx, resourceType)

	case "ec2.instanceAlarm":
		return ec2.ListInstanceAlarms(ctx, resourceType)

	case "ec2.elasticIp":
		return ec2.ListElasticIPs(ctx, resourceType)

	case "ec2.elasticIpUnassociated":
		return ec2.ListUnassociatedElasticIPs(ctx, resourceType)

	case "ec2.elasticIpAssociation":
		return ec2.ListElasticIPAssociations(ctx, resourceType)

	case "ec2.publicIpv4Pool":
		return ec2.ListPublicIPv4Pools(ctx, resourceType)

	case "ec2.customerOwnedIpv4Pool":
		return ec2.ListCustomerOwnedIPv4Pools(ctx, resourceType)

	case "ec2.ipamPool":
		return ec2.ListIpamPools(ctx, resourceType)

	case "codeartifact.repository":
		return codeartifact.ListRepositories(ctx, resourceType)

	case "codeartifact.domain":
		return codeartifact.ListDomains(ctx, resourceType)

	case "codepipeline.pipeline":
		return codepipeline.ListPipelines(ctx, resourceType)

	case "codepipeline.stage":
		return codepipeline.ListStages(ctx, resourceType)

	case "codepipeline.pipelineExecution":
		return codepipeline.ListPipelineExecutions(ctx, resourceType)

	case "sqs.queue":
		return sqs.ListQueues(ctx, resourceType)

	case "s3.bucket":
		return s3.ListBuckets(ctx, resourceType)

	case "prometheus.workspace":
		return prometheus.ListWorkspaces(ctx, resourceType)

	case "route53.hostedZone":
		return route53.ListHostedZones(ctx, resourceType)

	case "sns.topic":
		return sns.ListTopics(ctx, resourceType)

	case "sns.subscription":
		return sns.ListSubscriptions(ctx, resourceType)

	default:
		return []core.IntegrationResource{}, nil
	}
}
