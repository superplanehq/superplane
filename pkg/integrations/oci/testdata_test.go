package oci

import (
	"io"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/test/support/contexts"
)

const testOCIPrivateKey = `-----BEGIN PRIVATE KEY-----
MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQDBaw/KsqBmsyjy
jyVsbvmYE5OPKi7wmqJ44t/OnUGr1MqmPD/fAsPE0H9we7iSP7j0UApNGmVFPKA9
2Fc4jUlEAudiKqVhPQPFKpI3+tiO55hYp8EjZwks21u035dxTnQwHtGfOxYaltOW
xKgAPZqJUCxxZfG7EEDa+igPRv0vnM7wSM6EwKP5ysRF0N6NXXskM0Vj08Z5jNgN
Y3DXPaPHZClh0eltv3hlh/lh3+PN8nNWvACSHtBtgwk9tGEjc5SE86rDuBMAw2nk
0gKO9JIQgWzuMzOHeqexx/SmWMMcHmeteKR01+uQjGtSruo36mXLg8/1/RdNla5k
UQuVqlgnAgMBAAECggEAN9e6z/mBsRUSAfINSoDB5EPmqwNxWPs0ZHWQS32Aq+U8
ewFTKYaJUiYmXSoDUIpAUp1XVAqGaZaG50QybnvwRsgV2PRaGjh9Aax4Wdw9MQkx
pYsNirShZAeTMyYI+eg+SHRlbjUnfRbF0TZHEQa4OuPgaP8XazxWlUJ7VWnYCKob
QysttNOZJKTzikVhSIZvDW1RyK5nbpv44ZDJS8+ZQBbTPFrp+GzxpKDPWc3KVuO3
7jQGRUuF2YwcjbcDH2271b0fn19Abm6DLDLhfGSFPkXuISbeQ+eNSmvXZfHZhzLj
dJefJIsntB+/VvJxyOdDrOCCPhUXWaN9Stch7ziIXQKBgQDoQge7U0CFI+mj98cp
miuuGZYM1Fk8+hGnw/O9Gvpyt7aXkteIx26JjJZiNPcxvbMgJOtKu7Qdcf5cPfsA
XAanokVteNvgBQqKSPqqqKpW5r8QeSKPKtrbMgtmy3KZT6KxbaH6xCBpnzvZ6gT5
EgvUuqFipPDSrgZqFnNs4duY+wKBgQDVMKIAqh6NYeVRX6XS7s0el8iWUE52hAq+
kD27gb+aIt5i/KidJj1Qbbc0Mr69BtmsQsnrNtaRjMkx6nfgc6ODfYpxKHux/xyx
fYVKa5+5AuZg04PatNpkju2wckPC5GCrw6LLRLuyy0+Pf6BfmUas/TXBDMlirtK7
pqJiOiStxQKBgGUiA23dNXYECkN8q/uAh06bE4xolqcHmNJ9b8/DRJTZTCe6KCIF
/SrlzcHboFvHZ40ypkX3b9l2frS5xGcGq1spPKQLgWqNp2ZJmuTe5rVKap4IsTS4
C25w3ygWpML/Oy+ZNnQUHK0BSjV8QkgWRJKP5aAnhDmoz2A4gHBD9LQrAoGAPUDh
6yr16E1uY/kFXhu618VonreoM6kwpRwwgIWBFbpbBzntAGoSR9+eOeMypoEnXbU6
6tgwwlUfIbZqhxTysD8L3gNxtuzDw8N63q0ZkUDiDIP5aId6EFZ4uK+8BG010WQ+
jATNoUuFKofS/mS9x8pg/Xy9CBuO9Nel5G8sRrkCgYBH3i9kkjwiGEkEC9yDuaSN
e8GQFwd4MCKwzDB+mjqxgTKNlpzTcfmqI5fGSFFY//XLxaIJxeMqg6nVIfEg+/+t
lT4hHZpmGCUnXTBwAKd5DzCxYFMA9mOIsXzw30qzT20XtGAjBKOfOnzQCdJl0ZYr
BhAK7kaeOgKerVxbFJ57Yw==
-----END PRIVATE KEY-----`

const testInstanceID = "ocid1.instance.oc1.eu-frankfurt-1.testinstance"
const testCompartmentID = "ocid1.tenancy.oc1..testtenancy"

func ociIntegrationContext() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		Configuration: map[string]any{
			"tenancyOcid": "ocid1.tenancy.oc1..testtenancy",
			"userOcid":    "ocid1.user.oc1..testuser",
			"fingerprint": "12:34:56:78:90:ab:cd:ef",
			"privateKey":  testOCIPrivateKey,
			"region":      "eu-frankfurt-1",
		},
		Metadata: IntegrationMetadata{TopicID: "ocid1.onstopic.oc1.eu-frankfurt-1.testtopic"},
	}
}

func ociMockResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{},
		Request:    &http.Request{},
	}
}

func ociInstanceBody(state string) string {
	return `{
		"id": "` + testInstanceID + `",
		"displayName": "test-instance",
		"lifecycleState": "` + state + `",
		"shape": "VM.Standard.E2.1.Micro",
		"availabilityDomain": "XXXX:eu-frankfurt-1-AD-1",
		"compartmentId": "` + testCompartmentID + `",
		"region": "eu-frankfurt-1",
		"timeCreated": "2026-04-22T20:31:25.145Z"
	}`
}

func ociLogger() *logrus.Entry {
	logger := logrus.New()
	logger.Out = io.Discard
	return logrus.NewEntry(logger)
}
