package ec2

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
