package ec2provision

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestManagedRunInstancesTags_IncludesFleetIDPartitionKey(t *testing.T) {
	tags := managedRunInstancesTags("aws-arm64")

	got := map[string]string{}
	for _, tg := range tags {
		got[aws.ToString(tg.Key)] = aws.ToString(tg.Value)
	}

	if got["Name"] != "superplane-runner" {
		t.Errorf("Name tag = %q, want superplane-runner", got["Name"])
	}
	if got[TagKeyManaged] != "true" {
		t.Errorf("%s = %q, want true", TagKeyManaged, got[TagKeyManaged])
	}
	if got[TagKeyFleetID] != "aws-arm64" {
		t.Errorf("%s = %q, want aws-arm64", TagKeyFleetID, got[TagKeyFleetID])
	}
	if len(tags) != 3 {
		t.Errorf("expected exactly 3 tags, got %d (%v)", len(tags), got)
	}
}

func TestManagedDescribeFilters_ScopesByManagedAndFleetIDAndStates(t *testing.T) {
	filters := managedDescribeFilters("aws-amd64", []string{"pending", "running"})

	got := map[string][]string{}
	for _, f := range filters {
		got[aws.ToString(f.Name)] = f.Values
	}

	if v := got["tag:"+TagKeyManaged]; len(v) != 1 || v[0] != "true" {
		t.Errorf("tag:%s = %v, want [true]", TagKeyManaged, v)
	}
	if v := got["tag:"+TagKeyFleetID]; len(v) != 1 || v[0] != "aws-amd64" {
		t.Errorf("tag:%s = %v, want [aws-amd64]", TagKeyFleetID, v)
	}
	if v := got["instance-state-name"]; len(v) != 2 || v[0] != "pending" || v[1] != "running" {
		t.Errorf("instance-state-name = %v, want [pending running]", v)
	}
	if len(filters) != 3 {
		t.Errorf("expected exactly 3 filters, got %d", len(filters))
	}
}

func TestManagedDescribeFilters_DifferentFleetIDsProduceDifferentFilters(t *testing.T) {
	// Two pools inside one fleet-manager process must produce mutually-exclusive
	// Describe filters so they don't reconcile each other's instances.
	a := managedDescribeFilters("aws-amd64", []string{"pending", "running"})
	b := managedDescribeFilters("aws-arm64", []string{"pending", "running"})

	fleetTag := func(fs []types.Filter) string {
		for _, f := range fs {
			if aws.ToString(f.Name) == "tag:"+TagKeyFleetID && len(f.Values) == 1 {
				return f.Values[0]
			}
		}
		return ""
	}
	if got := fleetTag(a); got != "aws-amd64" {
		t.Errorf("pool A fleet-id filter = %q, want aws-amd64", got)
	}
	if got := fleetTag(b); got != "aws-arm64" {
		t.Errorf("pool B fleet-id filter = %q, want aws-arm64", got)
	}
}
