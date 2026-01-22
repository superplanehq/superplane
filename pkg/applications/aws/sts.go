package aws

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

type stsCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      time.Time
}

type assumeRoleResponse struct {
	Result assumeRoleResult `xml:"AssumeRoleWithWebIdentityResult"`
}

type assumeRoleResult struct {
	Credentials stsCredentialsResponse `xml:"Credentials"`
}

type stsCredentialsResponse struct {
	AccessKeyID     string `xml:"AccessKeyId"`
	SecretAccessKey string `xml:"SecretAccessKey"`
	SessionToken    string `xml:"SessionToken"`
	Expiration      string `xml:"Expiration"`
}

func assumeRoleWithWebIdentity(httpCtx core.HTTPContext, region string, roleArn string, sessionName string, token string, durationSeconds int) (stsCredentials, error) {
	endpoint := stsEndpoint(region)

	values := url.Values{}
	values.Set("Action", "AssumeRoleWithWebIdentity")
	values.Set("Version", "2011-06-15")
	values.Set("RoleArn", roleArn)
	values.Set("RoleSessionName", sessionName)
	values.Set("WebIdentityToken", token)
	if durationSeconds > 0 {
		values.Set("DurationSeconds", strconv.Itoa(durationSeconds))
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return stsCredentials{}, fmt.Errorf("error building STS request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/xml")

	res, err := httpCtx.Do(req)
	if err != nil {
		return stsCredentials{}, fmt.Errorf("error executing STS request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return stsCredentials{}, fmt.Errorf("error reading STS response: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return stsCredentials{}, fmt.Errorf("STS request failed with %d: %s", res.StatusCode, string(body))
	}

	var response assumeRoleResponse
	if err := xml.Unmarshal(body, &response); err != nil {
		return stsCredentials{}, fmt.Errorf("error parsing STS response: %w", err)
	}

	expiration, err := time.Parse(time.RFC3339, strings.TrimSpace(response.Result.Credentials.Expiration))
	if err != nil {
		return stsCredentials{}, fmt.Errorf("error parsing STS expiration: %w", err)
	}

	credentials := stsCredentials{
		AccessKeyID:     response.Result.Credentials.AccessKeyID,
		SecretAccessKey: response.Result.Credentials.SecretAccessKey,
		SessionToken:    response.Result.Credentials.SessionToken,
		Expiration:      expiration,
	}

	if credentials.AccessKeyID == "" || credentials.SecretAccessKey == "" || credentials.SessionToken == "" {
		return stsCredentials{}, fmt.Errorf("STS response missing credentials")
	}

	return credentials, nil
}

func stsEndpoint(region string) string {
	region = strings.TrimSpace(region)
	if region == "" {
		return "https://sts.amazonaws.com"
	}

	if strings.HasPrefix(region, "http://") || strings.HasPrefix(region, "https://") {
		return region
	}

	return fmt.Sprintf("https://sts.%s.amazonaws.com", region)
}
