package fleets

import (
	"fmt"
	"os"
	"strings"
)

const firstRunFleetID = "e1-large-amd64"

func FirstRunFleetID() string {
	return firstRunFleetID
}

func developmentAppEnv() bool {
	return strings.TrimSpace(os.Getenv("APP_ENV")) == "development"
}

// FirstRunManagerYAML is the Fleet Manager configuration an operator pastes
// after owner setup. Development omits the self-host AWS placeholders because
// local Compose already runs Docker Fleet Manager with its own token.
func FirstRunManagerYAML(superplaneURL, token string) string {
	superplaneURL = strings.TrimRight(strings.TrimSpace(superplaneURL), "/")
	if developmentAppEnv() {
		return firstRunDevManagerYAML(superplaneURL)
	}

	token = strings.TrimSpace(token)
	return fmt.Sprintf(`id: self-host
superplaneUrl: %s
installationAdminToken: %s
runnerReleaseBaseUrl: <RUNNER_RELEASE_BASE_URL>
fleets:
  - id: %s
    warmCapacity: 0
    maxCapacity: 1
    aws:
      region: <REGION>
      ami: <PACKER_IMAGE_ID>
      instanceType: <INSTANCE_TYPE>
      architecture: amd64
      subnetIds:
        - <SUBNET_ID>
      securityGroupIds:
        - <SECURITY_GROUP_ID>
      iamInstanceProfile: <INSTANCE_PROFILE>
      keyName: ""
      volumeSizeGb: 30
`, superplaneURL, token, firstRunFleetID)
}

func firstRunDevManagerYAML(superplaneURL string) string {
	return fmt.Sprintf(`id: local
superplaneUrl: %s
fleets:
  - id: %s
    provider: docker
    maxCapacity: 10
    docker:
      image: superplane-runner-local:dev
      architecture: amd64
      runnerApiUrl: %s
`, superplaneURL, firstRunFleetID, superplaneURL)
}
