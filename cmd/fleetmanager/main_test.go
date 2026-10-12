package main

import (
	"bytes"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestPrintVersion(t *testing.T) {
	previousVersion := Version
	Version = "v1.2.3-rc.1"
	t.Cleanup(func() { Version = previousVersion })

	var output bytes.Buffer
	if !printVersion([]string{"--version"}, &output) {
		t.Fatal("expected --version to be handled")
	}
	if output.String() != "fleet-manager v1.2.3-rc.1\n" {
		t.Fatalf("version output = %q", output.String())
	}
}

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
