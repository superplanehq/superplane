package dockerprovider

import (
	"context"
	"errors"
	"log/slog"
	"slices"
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
			[]byte("container-1\trunner-1\te1-large-amd64\trunning\tcoolify\n"),
			[]byte("container-1\trunner-1\te1-large-amd64\trunning\tcoolify\n"),
			[]byte("container-2\n"),
			nil,
		},
	}
	p, err := newProvider(Config{
		FleetManagerID: "coolify",
		Image:          "runner:dev",
		RunnerAPIURL:   "http://app:8000",
		Network:        "superplane_default",
		Volumes:        []string{"/var/run/docker.sock:/var/run/docker.sock"},
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
	ownedArgs := strings.Join(commands.calls[0].args, " ")
	for _, expected := range []string{
		"label=superplane.managed-runner=true",
		"label=superplane.fleet-id=e1-large-amd64",
		"label=superplane.fleet-manager-id=coolify",
	} {
		if !strings.Contains(ownedArgs, expected) {
			t.Fatalf("owned list args %q do not contain %q", ownedArgs, expected)
		}
	}
	if strings.Contains(ownedArgs, "network=") {
		t.Fatalf("owned list args %q must not filter network", ownedArgs)
	}
	networkArgs := strings.Join(commands.calls[1].args, " ")
	if !strings.Contains(networkArgs, "network=superplane_default") {
		t.Fatalf("network list args %q do not contain the runner network", networkArgs)
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
	createArgs := strings.Join(commands.calls[2].args, " ")
	for _, expected := range []string{
		"--network superplane_default",
		"--label superplane.fleet-manager-id=coolify",
		"runner:dev",
		"--url http://app:8000",
		"--registration-token one-time-token",
		"--env RUNNER_TAGS=fleet_manager_id=manager-a",
	} {
		if !strings.Contains(createArgs, expected) {
			t.Fatalf("create args %q do not contain %q", createArgs, expected)
		}
	}
	if strings.Contains(createArgs, "--tag ") {
		t.Fatalf("create args pass an unsupported flag to older runners: %q", createArgs)
	}
	for _, option := range []string{"--init", "--user", "--entrypoint"} {
		if slices.Contains(commands.calls[1].args, option) {
			t.Fatalf("default runner unexpectedly overrides %s: %#v", option, commands.calls[1].args)
		}
	}

	if err := p.Delete(context.Background(), created); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(commands.calls[3].args, " "); got != "rm -f container-2" {
		t.Fatalf("delete args = %q", got)
	}
}

func TestProviderReportsDockerErrors(t *testing.T) {
	commands := &fakeCommandRunner{
		outputs: [][]byte{[]byte("daemon unavailable")},
		errors:  []error{errors.New("exit status 1")},
	}
	p, err := newProvider(Config{
		FleetManagerID: "coolify",
		Image:          "runner:dev",
	}, commands, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.List(context.Background(), "fleet"); err == nil ||
		!strings.Contains(err.Error(), "daemon unavailable") {
		t.Fatalf("List error = %v", err)
	}
	listArgs := strings.Join(commands.calls[0].args, " ")
	if !strings.Contains(listArgs, "label=superplane.fleet-manager-id=coolify") {
		t.Fatalf("list args %q must filter fleet-manager-id without a network", listArgs)
	}
}

func TestListAdoptsUnlabeledRunnersOnThisNetwork(t *testing.T) {
	commands := &fakeCommandRunner{
		outputs: [][]byte{
			[]byte("ours\trunner-ours\te1-large-amd64\trunning\tcoolify\n"),
			[]byte(strings.Join([]string{
				"ours\trunner-ours\te1-large-amd64\trunning\tcoolify",
				"legacy\trunner-legacy\te1-large-amd64\texited\t",
				"other\trunner-other\te1-large-amd64\trunning\tlocal",
			}, "\n") + "\n"),
		},
	}
	p, err := newProvider(Config{
		FleetManagerID: "coolify",
		Image:          "runner:dev",
		Network:        "coolify_runners",
	}, commands, slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	resources, err := p.List(context.Background(), "e1-large-amd64")
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 2 {
		t.Fatalf("resources = %#v", resources)
	}
	if resources[0].RunnerID != "runner-ours" || resources[1].RunnerID != "runner-legacy" {
		t.Fatalf("resources = %#v", resources)
	}
}

func TestListFindsOwnedRunnersOutsideConfiguredNetwork(t *testing.T) {
	commands := &fakeCommandRunner{
		outputs: [][]byte{
			[]byte("old-net\trunner-old\te1-large-amd64\texited\tcoolify\n"),
			[]byte(""),
		},
	}
	p, err := newProvider(Config{
		FleetManagerID: "coolify",
		Image:          "runner:dev",
		Network:        "coolify_runners",
	}, commands, slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	resources, err := p.List(context.Background(), "e1-large-amd64")
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 || resources[0].RunnerID != "runner-old" {
		t.Fatalf("resources = %#v", resources)
	}
}

func TestNewRequiresFleetManagerID(t *testing.T) {
	if _, err := newProvider(Config{Image: "runner:dev"}, &fakeCommandRunner{}, nil); err == nil ||
		!strings.Contains(err.Error(), "Fleet Manager ID is required") {
		t.Fatalf("New error = %v", err)
	}
}

func TestProviderRejectsBlankEntrypointExecutable(t *testing.T) {
	for _, executable := range []string{"", "  "} {
		_, err := New(Config{FleetManagerID: "coolify", Image: "runner:dev", Entrypoint: []string{executable, "argument"}}, nil)
		if err == nil || !strings.Contains(err.Error(), "entrypoint executable is required") {
			t.Fatalf("blank executable %q: %v", executable, err)
		}
	}
}

func TestProviderCreatesRunnerWithProcessOptions(t *testing.T) {
	commands := &fakeCommandRunner{outputs: [][]byte{[]byte("container-options\n")}}
	p, err := newProvider(Config{
		FleetManagerID: "coolify",
		Image:          "runner:dev",
		Init:           true,
		User:           "1000:1000",
		Entrypoint:     []string{"/bin/sh", "-c", `exec /usr/local/bin/runner "$@"`, "", "argument with spaces"},
	}, commands, nil)
	if err != nil {
		t.Fatal(err)
	}
	bootstrap, err := p.BuildBootstrap(provider.RunnerBootstrap{
		RunnerAPIURL: "http://app:8000", RegistrationToken: "one-time-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Create(t.Context(), provider.CreateRequest{
		RunnerID: "runner-options", FleetID: "fleet-a", RunnerVersion: "dev", Bootstrap: bootstrap,
	}); err != nil {
		t.Fatal(err)
	}
	args := commands.calls[0].args
	imageIndex := slices.Index(args, "runner:dev")
	if imageIndex < 0 {
		t.Fatalf("runner image missing from Docker arguments: %#v", args)
	}
	options := args[:imageIndex]
	if !slices.Contains(options, "--init") {
		t.Fatalf("init missing from Docker options: %#v", options)
	}
	for flag, want := range map[string]string{"--user": "1000:1000", "--entrypoint": "/bin/sh"} {
		index := slices.Index(options, flag)
		if index < 0 || index+1 >= len(options) || options[index+1] != want {
			t.Fatalf("Docker option %s: %#v", flag, options)
		}
	}
	wantCommand := []string{"-c", `exec /usr/local/bin/runner "$@"`, "", "argument with spaces", "--url", "http://app:8000", "--registration-token", "one-time-token"}
	if !slices.Equal(wantCommand, args[imageIndex+1:]) {
		t.Fatalf("runner command = %#v, want %#v", args[imageIndex+1:], wantCommand)
	}
}
