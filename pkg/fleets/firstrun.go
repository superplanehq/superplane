package fleets

import (
	"fmt"
	"strings"
)

const firstRunFleetID = "e1-large-amd64"

func FirstRunFleetID() string {
	return firstRunFleetID
}

// FirstRunManagerYAML is the Fleet Manager configuration an operator pastes
// after owner setup. Azure and Packer values stay placeholders until that
// stack supplies a ConfigMap.
func FirstRunManagerYAML(superplaneURL, token string) string {
	superplaneURL = strings.TrimRight(strings.TrimSpace(superplaneURL), "/")
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
