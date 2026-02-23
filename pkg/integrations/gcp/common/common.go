package common

const (
	SecretNameServiceAccountKey = "serviceAccountKey"
	SecretNameAccessToken       = "accessToken"
	ScopeCloudPlatform          = "https://www.googleapis.com/auth/cloud-platform"
)

var RequiredJSONKeys = []string{"type", "project_id", "private_key_id", "private_key", "client_email", "client_id"}

const (
	AuthMethodServiceAccountKey = "serviceAccountKey"
	AuthMethodWIF               = "workloadIdentityFederation"
)

type Metadata struct {
	ProjectID            string `json:"projectId"`
	ClientEmail          string `json:"clientEmail"`
	AuthMethod           string `json:"authMethod"`
	AccessTokenExpiresAt string `json:"accessTokenExpiresAt"`
	PubSubTopic          string `json:"pubsubTopic,omitempty"`
	PubSubSubscription   string `json:"pubsubSubscription,omitempty"`
}
