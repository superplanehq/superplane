package main

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestNewEC2ClientUsesFleetRegion(t *testing.T) {
	base := aws.Config{Region: "us-east-1"}
	east := newEC2Client(base, "us-east-1")
	west := newEC2Client(base, "us-west-2")

	if east == west {
		t.Fatal("expected each fleet to receive its own EC2 client")
	}
	if east.Options().Region != "us-east-1" {
		t.Fatalf("east client region = %q", east.Options().Region)
	}
	if west.Options().Region != "us-west-2" {
		t.Fatalf("west client region = %q", west.Options().Region)
	}
	if base.Region != "us-east-1" {
		t.Fatalf("base AWS config region changed to %q", base.Region)
	}
}
