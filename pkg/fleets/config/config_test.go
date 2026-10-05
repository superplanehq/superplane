package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAppliesAWSDefaultsAndRejectsMutableReleaseURL(t *testing.T) {
	body := `{
		"id":"fleet-manager",
		"superplaneUrl":"https://superplane.example",
		"installationAdminToken":"personal-token",
		"runnerReleaseBaseUrl":"https://downloads.example/runner/",
		"awsRegion":"us-east-1",
		"fleets":[{
			"id":"linux-amd64",
			"warmCapacity":1,
			"maxCapacity":2,
			"aws":{
				"ami":"ami-123",
				"architecture":"amd64",
				"subnetIds":["subnet-a"],
				"securityGroupIds":["sg-a"]
			}
		}]
	}`
	config, err := Load(writeConfig(t, body))
	if err != nil {
		t.Fatal(err)
	}
	if config.Fleets[0].AWS.InstanceType != "t3.micro" ||
		config.Fleets[0].AWS.VolumeSizeGB != 30 {
		t.Fatalf("defaults were not applied: %#v", config.Fleets[0].AWS)
	}
	if config.Fleets[0].MaxCapacity != 2 {
		t.Fatalf("maximum capacity = %d", config.Fleets[0].MaxCapacity)
	}
	if config.RunnerReleaseBaseURL != "https://downloads.example/runner" {
		t.Fatalf("runner release base URL = %q", config.RunnerReleaseBaseURL)
	}

	mutable := strings.Replace(
		body,
		"https://downloads.example/runner/",
		"https://downloads.example/runner/latest/",
		1,
	)
	if _, err := Load(writeConfig(t, mutable)); err == nil {
		t.Fatal("expected mutable artifact URL to fail")
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	if _, err := Load(writeConfig(t, `{"unknown":true}`)); err == nil {
		t.Fatal("expected unknown field to fail")
	}
}

func TestLoadRejectsSnakeCaseFields(t *testing.T) {
	_, err := Load(writeConfig(t, `{"superplane_url":"https://superplane.example"}`))
	if err == nil || !strings.Contains(err.Error(), `unknown field "superplane_url"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestLoadRejectsRemovedTaskSpecificField(t *testing.T) {
	_, err := Load(writeConfig(t, `{"fleets":[{"taskSpecific":true}]}`))
	if err == nil || !strings.Contains(err.Error(), `unknown field "taskSpecific"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestLoadFleetManagerIDAndAWSResourceTags(t *testing.T) {
	config, err := Load(writeConfig(t, `{
		"id":" fleet-manager-production ",
		"superplaneUrl":"https://superplane.example",
		"installationAdminToken":"personal-token",
		"runnerReleaseBaseUrl":"https://downloads.example/runner",
		"awsRegion":"us-east-1",
		"fleets":[{
			"id":"linux-amd64",
			"aws":{
				"ami":"ami-123",
				"architecture":"amd64",
				"subnetIds":["subnet-a"],
				"securityGroupIds":["sg-a"],
				"resourceTags":{
					"Environment":"production",
					"CostCenter":"runners"
				}
			}
		}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if config.ID != "fleet-manager-production" {
		t.Fatalf("Fleet Manager ID = %q", config.ID)
	}
	if config.Fleets[0].AWS.ResourceTags["Environment"] != "production" ||
		config.Fleets[0].AWS.ResourceTags["CostCenter"] != "runners" {
		t.Fatalf("resource tags = %#v", config.Fleets[0].AWS.ResourceTags)
	}
}

func TestLoadRejectsMissingFleetManagerID(t *testing.T) {
	t.Setenv("INSTALLATION_ADMIN_TOKEN", "personal-token")
	_, err := Load(writeConfig(t, `{
		"superplaneUrl":"http://app:8000",
		"fleets":[{
			"id":"e1-large-amd64",
			"provider":"docker",
			"docker":{
				"image":"runner:dev",
				"architecture":"amd64"
			}
		}]
	}`))
	if err == nil || !strings.Contains(err.Error(), "id is required") {
		t.Fatalf("error = %v", err)
	}
}

func TestLoadYAML(t *testing.T) {
	body := `
id: fleet-manager
superplaneUrl: http://app:8000
installationAdminToken: personal-token
fleets:
  - id: e1-large-amd64
    provider: docker
    docker:
      image: runner:dev
      architecture: amd64
      runnerApiUrl: http://app:8000
      network: superplane_default
`

	for _, extension := range []string{".yml", ".yaml"} {
		t.Run(extension, func(t *testing.T) {
			config, err := Load(writeConfigWithExtension(t, body, extension))
			if err != nil {
				t.Fatal(err)
			}
			if config.Fleets[0].Provider != ProviderDocker {
				t.Fatalf("provider = %q", config.Fleets[0].Provider)
			}
			if config.Fleets[0].Docker.Image != "runner:dev" {
				t.Fatalf("image = %q", config.Fleets[0].Docker.Image)
			}
		})
	}
}

func TestLoadEnvironmentConfiguration(t *testing.T) {
	tests := map[string]string{
		"JSON": `{
			"id":"fleet-manager",
			"superplaneUrl":"http://app:8000",
			"fleets":[{
				"id":"e1-large-amd64",
				"provider":"docker",
				"docker":{
					"image":"runner:dev",
					"architecture":"amd64"
				}
			}]
		}`,
		"YAML": `
id: fleet-manager
superplaneUrl: http://app:8000
fleets:
  - id: e1-large-amd64
    provider: docker
    docker:
      image: runner:dev
      architecture: amd64
`,
	}

	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv("FLEET_MANAGER_CONFIG", body)
			t.Setenv("INSTALLATION_ADMIN_TOKEN", "personal-token")

			config, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			if config.InstallationAdminToken != "personal-token" {
				t.Fatalf(
					"installation admin token = %q",
					config.InstallationAdminToken,
				)
			}
			if config.Fleets[0].ID != "e1-large-amd64" {
				t.Fatalf("fleet ID = %q", config.Fleets[0].ID)
			}
		})
	}
}

func TestLoadEnvironmentConfigurationAppliesInstallationAdminTokenOverride(t *testing.T) {
	t.Setenv("FLEET_MANAGER_CONFIG", `{
		"id":"fleet-manager",
		"superplaneUrl":"http://app:8000",
		"installationAdminToken":"config-token",
		"fleets":[{
			"id":"e1-large-amd64",
			"provider":"docker",
			"docker":{
				"image":"runner:dev",
				"architecture":"amd64"
			}
		}]
	}`)
	t.Setenv("INSTALLATION_ADMIN_TOKEN", "environment-token")

	config, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if config.InstallationAdminToken != "environment-token" {
		t.Fatalf(
			"installation admin token = %q",
			config.InstallationAdminToken,
		)
	}
}

func TestLoadRejectsWarmCapacityAboveMaximum(t *testing.T) {
	t.Setenv("INSTALLATION_ADMIN_TOKEN", "personal-token")
	_, err := Load(writeConfig(t, `{
		"id":"fleet-manager",
		"superplaneUrl":"http://app:8000",
		"fleets":[{
			"id":"e1-large-amd64",
			"provider":"docker",
			"warmCapacity":2,
			"maxCapacity":1,
			"docker":{
				"image":"runner:dev",
				"architecture":"amd64"
			}
		}]
	}`))
	if err == nil || !strings.Contains(
		err.Error(),
		"fleets[0].warmCapacity must not exceed maxCapacity",
	) {
		t.Fatalf("error = %v", err)
	}
}

func TestLoadDockerProviderDoesNotRequireAWSConfiguration(t *testing.T) {
	t.Setenv("INSTALLATION_ADMIN_TOKEN", "personal-token")
	config, err := Load(writeConfig(t, `{
		"id":"fleet-manager",
		"superplaneUrl":"http://app:8000",
		"installationAdminToken":"",
		"fleets":[{
			"id":"e1-large-amd64",
			"provider":"docker",
			"docker":{
				"image":"runner:dev",
				"architecture":"amd64",
				"runnerApiUrl":"http://app:8000",
				"network":"superplane_default"
			}
		}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if config.InstallationAdminToken != "personal-token" {
		t.Fatalf(
			"installation admin token = %q",
			config.InstallationAdminToken,
		)
	}
	if config.Fleets[0].Provider != ProviderDocker {
		t.Fatalf("provider = %q", config.Fleets[0].Provider)
	}
}

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	return writeConfigWithExtension(t, body, ".json")
}

func writeConfigWithExtension(t *testing.T, body, extension string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fleet-manager"+extension)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
