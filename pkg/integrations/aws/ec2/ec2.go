package ec2

import "time"

const (
	Source                   = "aws.ec2"
	DetailTypeAMIStateChange = "EC2 AMI State Change"
)

const (
	CloudWatchSource                     = "aws.cloudwatch"
	DetailTypeCloudWatchAlarmStateChange = "CloudWatch Alarm State Change"
)

const (
	monitoringServiceName = "monitoring"
	monitoringAPIVersion  = "2010-08-01"
)

const (
	ImageStatePending      = "pending"
	ImageStateAvailable    = "available"
	ImageStateFailed       = "failed"
	ImageStateDeregistered = "deregistered"
	ImageStateDisabled     = "disabled"
)

const (
	InstanceStatePending      = "pending"
	InstanceStateRunning      = "running"
	InstanceStateShuttingDown = "shutting-down"
	InstanceStateTerminated   = "terminated"
	InstanceStateStopping     = "stopping"
	InstanceStateStopped      = "stopped"
)

const (
	CreateInstancePayloadType               = "aws.ec2.instance"
	DeleteInstancePayloadType               = "aws.ec2.instance.deleted"
	GetInstancePayloadType                  = "aws.ec2.instance"
	GetInstanceMetricsPayloadType           = "aws.ec2.instance.metrics"
	ManageInstancePowerStartPayloadType     = "aws.ec2.instance.power.started"
	ManageInstancePowerStopPayloadType      = "aws.ec2.instance.power.stopped"
	ManageInstancePowerRebootPayloadType    = "aws.ec2.instance.power.rebooted"
	ManageInstancePowerHibernatePayloadType = "aws.ec2.instance.power.hibernated"
	UpdateInstancePayloadType               = "aws.ec2.instance.updated"
	AllocateElasticIPPayloadType            = "aws.ec2.elastic-ip.allocated"
	ReleaseElasticIPPayloadType             = "aws.ec2.elastic-ip.released"
	ManageElasticIPAssociatePayloadType     = "aws.ec2.elastic-ip.associated"
	ManageElasticIPDisassociatePayloadType  = "aws.ec2.elastic-ip.disassociated"

	CreateLoadBalancerPayloadType = "aws.ec2.loadBalancer"
	DeleteLoadBalancerPayloadType = "aws.ec2.loadBalancer.deleted"

	instancePollInterval    = 10 * time.Second
	maxInstancePollErrors   = 10
	maxInstancePollAttempts = 180

	loadBalancerPollInterval      = 10 * time.Second
	maxLoadBalancerPollErrors     = 10
	maxLoadBalancerPollAttempts   = 120
	maxLoadBalancerListenerErrors = 5
)

const (
	LoadBalancerStateProvisioning   = "provisioning"
	LoadBalancerStateActive         = "active"
	LoadBalancerStateActiveImpaired = "active_impaired"
	LoadBalancerStateFailed         = "failed"
	LoadBalancerStateDeleted        = "deleted"

	LoadBalancerTypeApplication = "application"
	LoadBalancerTypeNetwork     = "network"
	LoadBalancerTypeGateway     = "gateway"

	LoadBalancerSchemeInternetFacing = "internet-facing"
	LoadBalancerSchemeInternal       = "internal"

	LoadBalancerIPAddressTypeIPv4                     = "ipv4"
	LoadBalancerIPAddressTypeDualStack                = "dualstack"
	LoadBalancerIPAddressTypeDualStackWithoutPublicIP = "dualstack-without-public-ipv4"

	ListenerProtocolHTTP   = "HTTP"
	ListenerProtocolHTTPS  = "HTTPS"
	ListenerProtocolTCP    = "TCP"
	ListenerProtocolTLS    = "TLS"
	ListenerProtocolUDP    = "UDP"
	ListenerProtocolTCPUDP = "TCP_UDP"
	ListenerProtocolGENEVE = "GENEVE"

	minSubnetsForALBNLB = 2
	minSubnetsForGWLB   = 1
)
