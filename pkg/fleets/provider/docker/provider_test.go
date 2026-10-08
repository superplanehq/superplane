package dockerprovider

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/superplanehq/superplane/pkg/fleets/provider"
)

type commandCall struct {
	args []string
}

type fakeCommandRunner struct {
	calls   []commandCall
	outputs [][]byte
	errors  []error
}

func (f *fakeCommandRunner) Run(
	_ context.Context,
	args ...string,
) ([]byte, error) {
	f.calls = append(f.calls, commandCall{args: append([]string(nil), args...)})
	index := len(f.calls) - 1
	var output []byte
	if index < len(f.outputs) {
		output = f.outputs[index]
	}
	var err error
	if index < len(f.errors) {
		err = f.errors[index]
	}
	return output, err
}

func TestProviderLifecycle(t *testing.T) {
	commands := &fakeCommandRunner{
		outputs: [][]byte{
			[]byte("container-1\trunner-1\te1-large-amd64\trunning\n"),
			[]byte("container-2\n"),
			nil,
		},
	}
	p, err := newProvider(Config{
		Image:        "runner:dev",
		RunnerAPIURL: "http://app:8000",
		Network:      "superplane_default",
		Volumes:      []string{"/var/run/docker.sock:/var/run/docker.sock"},
	}, commands, slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	resources, err := p.List(context.Background(), "e1-large-amd64")
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 || resources[0].RunnerID != "runner-1" {
		t.Fatalf("resources = %#v", resources)
	}

	encoded, err := p.BuildBootstrap(provider.RunnerBootstrap{
		RunnerAPIURL:      "https://public.example",
		RegistrationToken: "one-time-token",
		Tags:              map[string]string{"fleet_manager_id": "manager-a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	created, err := p.Create(context.Background(), provider.CreateRequest{
		RunnerID:      "runner-2",
		FleetID:       "e1-large-amd64",
		RunnerVersion: "dev",
		Bootstrap:     encoded,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != "container-2" {
		t.Fatalf("created ID = %q", created.ID)
	}
	createArgs := strings.Join(commands.calls[1].args, " ")
	for _, expected := range []string{
		"--network superplane_default",
		"runner:dev",
		"--url http://app:8000",
		"--registration-token one-time-token",
		"--tag fleet_manager_id=manager-a",
	} {
		if !strings.Contains(createArgs, expected) {
			t.Fatalf("create args %q do not contain %q", createArgs, expected)
		}
	}

	if err := p.Delete(context.Background(), created); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(commands.calls[2].args, " "); got != "rm -f container-2" {
		t.Fatalf("delete args = %q", got)
	}
}

func TestProviderReportsDockerErrors(t *testing.T) {
	commands := &fakeCommandRunner{
		outputs: [][]byte{[]byte("daemon unavailable")},
		errors:  []error{errors.New("exit status 1")},
	}
	p, err := newProvider(Config{Image: "runner:dev"}, commands, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.List(context.Background(), "fleet"); err == nil ||
		!strings.Contains(err.Error(), "daemon unavailable") {
		t.Fatalf("List error = %v", err)
	}
}
