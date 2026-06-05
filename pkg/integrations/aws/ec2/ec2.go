package ec2

import "time"

const (
	Source                   = "aws.ec2"
	DetailTypeAMIStateChange = "EC2 AMI State Change"
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
	ManageInstancePowerStartPayloadType     = "aws.ec2.instance.power.started"
	ManageInstancePowerStopPayloadType      = "aws.ec2.instance.power.stopped"
	ManageInstancePowerRebootPayloadType    = "aws.ec2.instance.power.rebooted"
	ManageInstancePowerHibernatePayloadType = "aws.ec2.instance.power.hibernated"

	CreateLoadBalancerPayloadType = "aws.ec2.loadBalancer"
	DeleteLoadBalancerPayloadType = "aws.ec2.loadBalancer.deleted"

	instancePollInterval    = 10 * time.Second
	maxInstancePollErrors   = 10
	maxInstancePollAttempts = 180

	loadBalancerPollInterval    = 15 * time.Second
	maxLoadBalancerPollErrors   = 10
	maxLoadBalancerPollAttempts = 120
)

const (
	LoadBalancerStateProvisioning   = "provisioning"
	LoadBalancerStateActive         = "active"
	LoadBalancerStateActiveImpaired = "active_impaired"
	LoadBalancerStateFailed         = "failed"
	LoadBalancerStateDeleting       = "deleting"

	LoadBalancerTypeApplication = "application"
	LoadBalancerTypeNetwork     = "network"
	LoadBalancerTypeGateway     = "gateway"

	LoadBalancerSchemeInternetFacing = "internet-facing"
	LoadBalancerSchemeInternal       = "internal"
)
