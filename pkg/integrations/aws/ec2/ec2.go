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
	CreateInstancePayloadType      = "aws.ec2.instance"
	DeleteInstancePayloadType      = "aws.ec2.instance.deleted"
	GetInstancePayloadType         = "aws.ec2.instance"
	ManageInstancePowerPayloadType = "aws.ec2.instance"

	instancePollInterval    = 10 * time.Second
	maxInstancePollErrors   = 10
	maxInstancePollAttempts = 180
)
