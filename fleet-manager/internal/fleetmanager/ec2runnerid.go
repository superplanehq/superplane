package fleetmanager

import "regexp"

// ec2ManagedRunnerID identifies runner_id values fleet-manager treats as EC2 instance ids when
// terminating disposable workers after a task (avoid passing arbitrary caller strings to AWS).
var ec2InstanceIDPattern = regexp.MustCompile(`^i-[0-9a-f]{8,32}$`)

func isEC2InstanceID(s string) bool {
	return ec2InstanceIDPattern.MatchString(s)
}
