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
	GetInstanceMetricsPayloadType           = "aws.ec2.instance.metrics"
	ManageInstancePowerStartPayloadType     = "aws.ec2.instance.power.started"
	ManageInstancePowerStopPayloadType      = "aws.ec2.instance.power.stopped"
	ManageInstancePowerRebootPayloadType    = "aws.ec2.instance.power.rebooted"
	ManageInstancePowerHibernatePayloadType = "aws.ec2.instance.power.hibernated"
	UpdateInstancePayloadType               = "aws.ec2.instance.updated"

	instancePollInterval    = 10 * time.Second
	maxInstancePollErrors   = 10
	maxInstancePollAttempts = 180
)
