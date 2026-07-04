package ec2

import (
	"time"

	"github.com/superplanehq/superplane/pkg/configuration"
)

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
	alarmNamespaceEC2     = "AWS/EC2"
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

	CreateAlarmPayloadType = "aws.ec2.alarm"
	GetAlarmPayloadType    = "aws.ec2.alarm"
	UpdateAlarmPayloadType = "aws.ec2.alarm"
	DeleteAlarmPayloadType = "aws.ec2.alarm.deleted"

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

var EC2MetricOptions = []configuration.FieldOption{
	{Label: "CPU Utilization", Value: "CPUUtilization"},
	{Label: "CPU Credit Usage", Value: "CPUCreditUsage"},
	{Label: "CPU Credit Balance", Value: "CPUCreditBalance"},
	{Label: "Disk Read Bytes", Value: "DiskReadBytes"},
	{Label: "Disk Read Ops", Value: "DiskReadOps"},
	{Label: "Disk Write Bytes", Value: "DiskWriteBytes"},
	{Label: "Disk Write Ops", Value: "DiskWriteOps"},
	{Label: "Network In", Value: "NetworkIn"},
	{Label: "Network Out", Value: "NetworkOut"},
	{Label: "Network Packets In", Value: "NetworkPacketsIn"},
	{Label: "Network Packets Out", Value: "NetworkPacketsOut"},
	{Label: "Status Check Failed", Value: "StatusCheckFailed"},
	{Label: "Status Check Failed (Instance)", Value: "StatusCheckFailed_Instance"},
	{Label: "Status Check Failed (System)", Value: "StatusCheckFailed_System"},
	{Label: "Status Check Failed (Attached EBS)", Value: "StatusCheckFailed_AttachedEBS"},
}

// AlarmEC2ActionOptions are EC2 automation actions that CloudWatch can trigger
// when an alarm enters the ALARM state. Each value maps to
// arn:aws:automate:<region>:ec2:<value>.
var AlarmEC2ActionOptions = []configuration.FieldOption{
	{Label: "Recover", Value: "recover"},
	{Label: "Reboot", Value: "reboot"},
	{Label: "Stop", Value: "stop"},
	{Label: "Terminate", Value: "terminate"},
}

var AlarmStatisticOptions = []configuration.FieldOption{
	{Label: "Average", Value: "Average"},
	{Label: "Sum", Value: "Sum"},
	{Label: "Minimum", Value: "Minimum"},
	{Label: "Maximum", Value: "Maximum"},
	{Label: "Sample Count", Value: "SampleCount"},
}

var AlarmComparisonOperatorOptions = []configuration.FieldOption{
	{Label: "Greater Than Threshold", Value: "GreaterThanThreshold"},
	{Label: "Greater Than Or Equal To Threshold", Value: "GreaterThanOrEqualToThreshold"},
	{Label: "Less Than Threshold", Value: "LessThanThreshold"},
	{Label: "Less Than Or Equal To Threshold", Value: "LessThanOrEqualToThreshold"},
}

var AlarmTreatMissingDataOptions = []configuration.FieldOption{
	{Label: "Missing", Value: "missing"},
	{Label: "Ignore", Value: "ignore"},
	{Label: "Breaching", Value: "breaching"},
	{Label: "Not Breaching", Value: "notBreaching"},
}
