package config

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAppliesAWSDefaultsAndRejectsMutableArtifacts(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	body := `{
		"superplane_url":"https://superplane.example",
		"installation_admin_token":"personal-token",
		"artifact_manifest_url_template":"https://downloads.example/runner/v{version}/manifest.json",
		"artifact_signing_public_key":"` + base64.StdEncoding.EncodeToString(publicKey) + `",
		"aws_region":"us-east-1",
		"fleets":[{
			"id":"linux-amd64",
			"warm_capacity":1,
			"task_specific":true,
			"aws":{
				"ami":"ami-123",
				"architecture":"amd64",
				"subnet_ids":["subnet-a"],
				"security_group_ids":["sg-a"]
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

	mutable := strings.Replace(
		body,
		"https://downloads.example/runner/v{version}/manifest.json",
		"https://downloads.example/runner/latest/{version}/manifest.json",
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

func TestLoadDockerProviderDoesNotRequireAWSConfiguration(t *testing.T) {
	t.Setenv("INSTALLATION_ADMIN_TOKEN", "personal-token")
	config, err := Load(writeConfig(t, `{
		"superplane_url":"http://app:8000",
		"installation_admin_token":"",
		"fleets":[{
			"id":"e1-large-amd64",
			"provider":"docker",
			"task_specific":true,
			"docker":{
				"image":"runner:dev",
				"architecture":"amd64",
				"runner_api_url":"http://app:8000",
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
	path := filepath.Join(t.TempDir(), "fleet-manager.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
