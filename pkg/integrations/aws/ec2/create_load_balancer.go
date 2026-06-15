package ec2

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type CreateLoadBalancer struct{}

type CreateLoadBalancerConfiguration struct {
	Region                 string   `json:"region" mapstructure:"region"`
	Name                   string   `json:"name" mapstructure:"name"`
	Type                   string   `json:"type" mapstructure:"type"`
	Scheme                 string   `json:"scheme" mapstructure:"scheme"`
	IPAddressType          string   `json:"ipAddressType" mapstructure:"ipAddressType"`
	Subnets                []string `json:"subnets" mapstructure:"subnets"`
	SecurityGroups         []string `json:"securityGroups" mapstructure:"securityGroups"`
	ListenerProtocol       string   `json:"listenerProtocol" mapstructure:"listenerProtocol"`
	ListenerPort           int      `json:"listenerPort" mapstructure:"listenerPort"`
	ListenerTargetGroup    string   `json:"listenerTargetGroup" mapstructure:"listenerTargetGroup"`
	ListenerCertificateArn string   `json:"listenerCertificateArn" mapstructure:"listenerCertificateArn"`
}

type CreateLoadBalancerNodeMetadata struct {
	Region string `json:"region" mapstructure:"region"`
	Name   string `json:"name" mapstructure:"name"`
	Type   string `json:"type" mapstructure:"type"`
	Scheme string `json:"scheme" mapstructure:"scheme"`
}

type CreateLoadBalancerExecutionMetadata struct {
	LoadBalancerARN string `json:"loadBalancerArn" mapstructure:"loadBalancerArn"`
	ListenerCreated bool   `json:"listenerCreated" mapstructure:"listenerCreated"`
	ListenerErrors  int    `json:"listenerErrors" mapstructure:"listenerErrors"`
	PollErrors      int    `json:"pollErrors" mapstructure:"pollErrors"`
	PollAttempts    int    `json:"pollAttempts" mapstructure:"pollAttempts"`
}

func (c *CreateLoadBalancer) Name() string {
	return "aws.ec2.createLoadBalancer"
}

func (c *CreateLoadBalancer) Label() string {
	return "EC2 • Create Load Balancer"
}

func (c *CreateLoadBalancer) Description() string {
	return "Create an Application, Network, or Gateway Load Balancer and wait for it to become active"
}

func (c *CreateLoadBalancer) Documentation() string {
	return `The Create Load Balancer component provisions an Elastic Load Balancer (ELBv2) and waits until it reaches **active** state before emitting.

## Use Cases

- **Traffic distribution**: Add a load balancer in front of EC2 instances or ECS services
- **Environment provisioning**: Create load balancers as part of infrastructure workflows
- **Blue/green deployments**: Stand up a new load balancer before switching traffic

## Configuration

- **Name**: Name for the load balancer (must be unique per account/region)
- **Region**: AWS region where the load balancer will be created
- **Type**: Load balancer type:
  - ` + "`application`" + ` — Layer 7; routes HTTP/HTTPS traffic
  - ` + "`network`" + ` — Layer 4; routes TCP, TLS, UDP, or TCP_UDP traffic
  - ` + "`gateway`" + ` — Layer 3; routes traffic through third-party virtual appliances using GENEVE
- **Scheme** (Application and Network only): ` + "`internet-facing`" + ` (public DNS) or ` + "`internal`" + ` (private VPC only). Not applicable for Gateway load balancers.
- **Subnets**: Subnets to attach to the load balancer. Application and Network load balancers require at least two subnets in different Availability Zones; Gateway load balancers require at least one.
- **Security Groups** (Application only): Security groups to associate with the load balancer.
- **IP Address Type** (optional): Address family for the load balancer:
  - ` + "`ipv4`" + ` — IPv4 only (default)
  - ` + "`dualstack`" + ` — IPv4 and IPv6
  - ` + "`dualstack-without-public-ipv4`" + ` — IPv6 public, IPv4 private only
- **Listener Protocol** (optional): Protocol for the default listener. Valid values depend on the load balancer type:
  - Application: ` + "`HTTP`" + `, ` + "`HTTPS`" + `
  - Network: ` + "`TCP`" + `, ` + "`TLS`" + `, ` + "`UDP`" + `, ` + "`TCP_UDP`" + `
  - Gateway: ` + "`GENEVE`" + `
- **Listener Port** (optional): Port the listener receives traffic on (1–65535). Shown when a Listener Protocol is selected.
- **Target Group** (optional): Target group to forward listener traffic to. Shown when a Listener Protocol is selected.

## Output

Emits load balancer details on the default output channel, including:
- ` + "`loadBalancerArn`" + ` — full ARN of the load balancer
- ` + "`name`" + ` — load balancer name
- ` + "`dnsName`" + ` — DNS name for routing traffic
- ` + "`scheme`" + ` — ` + "`internet-facing`" + ` or ` + "`internal`" + `
- ` + "`type`" + ` — ` + "`application`" + `, ` + "`network`" + `, or ` + "`gateway`" + `
- ` + "`state`" + ` — should be ` + "`active`" + `
- ` + "`vpcId`" + ` — VPC the load balancer is associated with
`
}

func (c *CreateLoadBalancer) Icon() string {
	return "aws"
}

func (c *CreateLoadBalancer) Color() string {
	return "gray"
}

func (c *CreateLoadBalancer) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

var lbTypeOptions = []configuration.FieldOption{
	{Label: "Application", Value: LoadBalancerTypeApplication},
	{Label: "Network", Value: LoadBalancerTypeNetwork},
	{Label: "Gateway", Value: LoadBalancerTypeGateway},
}

var lbSchemeOptions = []configuration.FieldOption{
	{Label: "Internet-facing", Value: LoadBalancerSchemeInternetFacing},
	{Label: "Internal", Value: LoadBalancerSchemeInternal},
}

var lbIPAddressTypeOptions = []configuration.FieldOption{
	{Label: "IPv4", Value: LoadBalancerIPAddressTypeIPv4},
	{Label: "Dual-stack (IPv4 + IPv6)", Value: LoadBalancerIPAddressTypeDualStack},
	{Label: "Dual-stack without public IPv4", Value: LoadBalancerIPAddressTypeDualStackWithoutPublicIP},
}

var albListenerProtocolOptions = []configuration.FieldOption{
	{Label: "HTTP", Value: ListenerProtocolHTTP},
	{Label: "HTTPS", Value: ListenerProtocolHTTPS},
}

var nlbListenerProtocolOptions = []configuration.FieldOption{
	{Label: "TCP", Value: ListenerProtocolTCP},
	{Label: "TLS", Value: ListenerProtocolTLS},
	{Label: "UDP", Value: ListenerProtocolUDP},
	{Label: "TCP/UDP", Value: ListenerProtocolTCPUDP},
}

var gwlbListenerProtocolOptions = []configuration.FieldOption{
	{Label: "GENEVE", Value: ListenerProtocolGENEVE},
}

var allListenerProtocolOptions = append(
	append(albListenerProtocolOptions, nlbListenerProtocolOptions...),
	gwlbListenerProtocolOptions...,
)

func (c *CreateLoadBalancer) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Name for the load balancer",
			Placeholder: "my-load-balancer",
		},
		{
			Name:     "region",
			Label:    "Region",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "us-east-1",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: common.AllRegions,
				},
			},
		},
		{
			Name:     "type",
			Label:    "Type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  LoadBalancerTypeApplication,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: lbTypeOptions,
				},
			},
		},
		{
			Name:     "scheme",
			Label:    "Scheme",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			Default:  LoadBalancerSchemeInternetFacing,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{LoadBalancerTypeApplication, LoadBalancerTypeNetwork}},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: lbSchemeOptions,
				},
			},
		},
		{
			Name:        "subnets",
			Label:       "Subnets",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Subnets across at least two Availability Zones",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "ec2.subnet",
					Multi: true,
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
					},
				},
			},
		},
		{
			Name:        "securityGroups",
			Label:       "Security Groups",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Security groups to associate (Application Load Balancers only)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
				{Field: "type", Values: []string{LoadBalancerTypeApplication}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "ec2.securityGroup",
					Multi: true,
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
					},
				},
			},
		},
		{
			Name:     "ipAddressType",
			Label:    "IP Address Type",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			Default:  LoadBalancerIPAddressTypeIPv4,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: lbIPAddressTypeOptions,
				},
			},
		},
		{
			Name:        "listenerProtocol",
			Label:       "Listener Protocol",
			Description: "Protocol for the default listener — ALB: HTTP/HTTPS · NLB: TCP/TLS/UDP/TCP_UDP · GWLB: GENEVE",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: allListenerProtocolOptions,
				},
			},
		},
		{
			Name:        "listenerPort",
			Label:       "Listener Port",
			Description: "Port the listener receives traffic on (1–65535)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "listenerProtocol", Values: []string{"*"}},
			},
		},
		{
			Name:        "listenerTargetGroup",
			Label:       "Target Group",
			Description: "Target group to forward traffic to",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "listenerProtocol", Values: []string{"*"}},
				{Field: "region", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "ec2.targetGroup",
					Multi: false,
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
					},
				},
			},
		},
		{
			Name:        "listenerCertificateArn",
			Label:       "Listener Certificate ARN",
			Description: "ACM certificate ARN required for HTTPS and TLS listeners",
			Type:        configuration.FieldTypeString,
			Required:    false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "listenerProtocol", Values: []string{ListenerProtocolHTTPS, ListenerProtocolTLS}},
			},
		},
	}
}

func (c *CreateLoadBalancer) Setup(ctx core.SetupContext) error {
	config := CreateLoadBalancerConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}

	if _, err := requireName(config.Name); err != nil {
		return err
	}

	lbType := strings.TrimSpace(config.Type)
	minSubnets := minSubnetsForALBNLB
	if lbType == LoadBalancerTypeGateway {
		minSubnets = minSubnetsForGWLB
	}
	if err := validateLoadBalancerSubnets(ctx, region, config.Subnets, minSubnets); err != nil {
		return err
	}

	if err := validateListenerConfig(config); err != nil {
		return err
	}

	return ctx.Metadata.Set(CreateLoadBalancerNodeMetadata{
		Region: region,
		Name:   strings.TrimSpace(config.Name),
		Type:   lbType,
		Scheme: strings.TrimSpace(config.Scheme),
	})
}

func (c *CreateLoadBalancer) Execute(ctx core.ExecutionContext) error {
	config := CreateLoadBalancerConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}

	name, err := requireName(config.Name)
	if err != nil {
		return err
	}

	lbType := strings.TrimSpace(config.Type)
	if lbType == "" {
		lbType = LoadBalancerTypeApplication
	}

	scheme := strings.TrimSpace(config.Scheme)
	if scheme == "" && lbType != LoadBalancerTypeGateway {
		scheme = LoadBalancerSchemeInternetFacing
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	var securityGroups []string
	if lbType == LoadBalancerTypeApplication {
		securityGroups = config.SecurityGroups
	}

	client := NewClient(ctx.HTTP, creds, region)
	output, err := client.CreateLoadBalancer(CreateLoadBalancerInput{
		Name:           name,
		Type:           lbType,
		Scheme:         scheme,
		IPAddressType:  strings.TrimSpace(config.IPAddressType),
		SubnetIDs:      config.Subnets,
		SecurityGroups: securityGroups,
	})
	if err != nil {
		return fmt.Errorf("failed to create load balancer: %w", err)
	}

	if err := ctx.Metadata.Set(CreateLoadBalancerExecutionMetadata{
		LoadBalancerARN: output.LoadBalancerARN,
	}); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, loadBalancerPollInterval)
}

func (c *CreateLoadBalancer) Hooks() []core.Hook {
	return []core.Hook{
		{Name: "poll", Type: core.HookTypeInternal},
	}
}

func (c *CreateLoadBalancer) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name != "poll" {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("unknown action: %s", ctx.Name))
	}

	return c.poll(ctx)
}

func (c *CreateLoadBalancer) poll(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata CreateLoadBalancerExecutionMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode metadata: %v", err))
	}
	if metadata.LoadBalancerARN == "" {
		return ctx.ExecutionState.Fail("error", "poll metadata is missing loadBalancerArn: execution state may be corrupted")
	}

	config := CreateLoadBalancerConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get AWS credentials: %v", err))
	}

	client := NewClient(ctx.HTTP, creds, region)
	lb, err := client.DescribeLoadBalancer(metadata.LoadBalancerARN)
	if err != nil {
		metadata.PollErrors++
		ctx.Logger.Warnf("failed to describe load balancer %s (attempt %d/%d): %v",
			metadata.LoadBalancerARN, metadata.PollErrors, maxLoadBalancerPollErrors, err)
		if metadata.PollErrors >= maxLoadBalancerPollErrors {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("giving up polling load balancer %s after %d consecutive errors: %v",
				metadata.LoadBalancerARN, maxLoadBalancerPollErrors, err))
		}
		if err := ctx.Metadata.Set(metadata); err != nil {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to save poll error count: %v", err))
		}
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, loadBalancerPollInterval)
	}

	metadata.PollErrors = 0
	metadata.PollAttempts++
	if err := ctx.Metadata.Set(metadata); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to save poll attempt count: %v", err))
	}

	switch lb.State {
	case LoadBalancerStateActive, LoadBalancerStateActiveImpaired:
		if !metadata.ListenerCreated {
			if err := c.maybeCreateListener(client, metadata.LoadBalancerARN, config, ctx); err != nil {
				metadata.ListenerErrors++
				ctx.Logger.Warnf("failed to create listener for load balancer %s (attempt %d/%d): %v",
					metadata.LoadBalancerARN, metadata.ListenerErrors, maxLoadBalancerListenerErrors, err)
				if metadata.ListenerErrors >= maxLoadBalancerListenerErrors {
					return ctx.ExecutionState.Fail("error", fmt.Sprintf("giving up creating listener for load balancer %s after %d consecutive errors: %v",
						metadata.LoadBalancerARN, maxLoadBalancerListenerErrors, err))
				}
				if err := ctx.Metadata.Set(metadata); err != nil {
					return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to save listener error count: %v", err))
				}
				return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, loadBalancerPollInterval)
			}
			metadata.ListenerCreated = true
			metadata.ListenerErrors = 0
			if err := ctx.Metadata.Set(metadata); err != nil {
				return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to save listener created state: %v", err))
			}
		}
		return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, CreateLoadBalancerPayloadType, []any{
			map[string]any{
				"loadBalancerArn": lb.LoadBalancerARN,
				"name":            lb.Name,
				"dnsName":         lb.DNSName,
				"scheme":          lb.Scheme,
				"type":            lb.Type,
				"state":           lb.State,
				"vpcId":           lb.VpcID,
				"region":          lb.Region,
			},
		})
	case LoadBalancerStateFailed:
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("load balancer %s entered failed state", metadata.LoadBalancerARN))
	}

	if metadata.PollAttempts >= maxLoadBalancerPollAttempts {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("timed out waiting for load balancer %s to become active after %d poll attempts (state: %s)",
			metadata.LoadBalancerARN, metadata.PollAttempts, lb.State))
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, loadBalancerPollInterval)
}

func (c *CreateLoadBalancer) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateLoadBalancer) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateLoadBalancer) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateLoadBalancer) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func validateLoadBalancerSubnets(ctx core.SetupContext, region string, subnets []string, minSubnets int) error {
	subnetIDs := make([]string, 0, len(subnets))
	for _, subnetID := range subnets {
		if trimmed := strings.TrimSpace(subnetID); trimmed != "" {
			subnetIDs = append(subnetIDs, trimmed)
		}
	}

	if len(subnetIDs) < minSubnets {
		if minSubnets >= minSubnetsForALBNLB {
			return fmt.Errorf("at least %d subnet(s) in different Availability Zones must be specified", minSubnets)
		}
		return fmt.Errorf("at least %d subnet(s) must be specified", minSubnets)
	}

	if minSubnets < minSubnetsForALBNLB {
		return nil
	}

	if ctx.HTTP == nil || ctx.Integration == nil {
		return nil
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, region)
	subnetsByAZ := make(map[string]string, len(subnetIDs))
	for _, subnetID := range subnetIDs {
		subnet, err := client.DescribeSubnet(subnetID)
		if err != nil {
			return fmt.Errorf("failed to describe subnet %s: %w", subnetID, err)
		}

		az := strings.TrimSpace(subnet.AvailabilityZone)
		if az == "" {
			return fmt.Errorf("subnet %s does not have an availability zone", subnetID)
		}

		if existingSubnetID, ok := subnetsByAZ[az]; ok {
			return fmt.Errorf("subnets must be in different Availability Zones: %s and %s are both in %s", existingSubnetID, subnetID, az)
		}
		subnetsByAZ[az] = subnetID
	}

	return nil
}

func validateListenerConfig(config CreateLoadBalancerConfiguration) error {
	protocol := strings.TrimSpace(config.ListenerProtocol)
	targetGroup := strings.TrimSpace(config.ListenerTargetGroup)
	if protocol == "" && targetGroup == "" {
		return nil
	}

	if protocol != "" && targetGroup == "" {
		return fmt.Errorf("listenerTargetGroup is required when listenerProtocol is specified")
	}

	if targetGroup != "" && protocol == "" {
		return fmt.Errorf("listenerProtocol is required when listenerTargetGroup is specified")
	}

	if config.ListenerPort <= 0 || config.ListenerPort > 65535 {
		return fmt.Errorf("listener port must be between 1 and 65535")
	}

	lbType := strings.TrimSpace(config.Type)
	if err := validateProtocolForType(protocol, lbType); err != nil {
		return err
	}

	if protocol == ListenerProtocolHTTPS || protocol == ListenerProtocolTLS {
		if strings.TrimSpace(config.ListenerCertificateArn) == "" {
			return fmt.Errorf("listenerCertificateArn is required for %s listeners", protocol)
		}
	}
	return nil
}

func validateProtocolForType(protocol, lbType string) error {
	if protocol == "" {
		return nil
	}
	switch lbType {
	case LoadBalancerTypeApplication, "":
		valid := []string{ListenerProtocolHTTP, ListenerProtocolHTTPS}
		if !slices.Contains(valid, protocol) {
			return fmt.Errorf("protocol %s is not valid for application load balancers; valid protocols: HTTP, HTTPS", protocol)
		}
	case LoadBalancerTypeNetwork:
		valid := []string{ListenerProtocolTCP, ListenerProtocolTLS, ListenerProtocolUDP, ListenerProtocolTCPUDP}
		if !slices.Contains(valid, protocol) {
			return fmt.Errorf("protocol %s is not valid for network load balancers; valid protocols: TCP, TLS, UDP, TCP_UDP", protocol)
		}
	case LoadBalancerTypeGateway:
		if protocol != ListenerProtocolGENEVE {
			return fmt.Errorf("protocol %s is not valid for gateway load balancers; valid protocol: GENEVE", protocol)
		}
	}
	return nil
}

func (c *CreateLoadBalancer) maybeCreateListener(client *Client, lbARN string, config CreateLoadBalancerConfiguration, ctx core.ActionHookContext) error {
	protocol := strings.TrimSpace(config.ListenerProtocol)
	targetGroup := strings.TrimSpace(config.ListenerTargetGroup)
	if protocol == "" || targetGroup == "" {
		return nil
	}

	_, err := client.CreateListener(CreateListenerInput{
		LoadBalancerARN: lbARN,
		Protocol:        protocol,
		Port:            config.ListenerPort,
		TargetGroupARN:  targetGroup,
		CertificateARN:  strings.TrimSpace(config.ListenerCertificateArn),
	})
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	ctx.Logger.Infof("created %s listener on port %d for load balancer %s", protocol, config.ListenerPort, lbARN)
	return nil
}
