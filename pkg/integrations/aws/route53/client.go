package route53

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	serviceName = "route53"
	region      = "us-east-1"
	endpoint    = "https://route53.amazonaws.com/2013-04-01"
	xmlNS       = "https://route53.amazonaws.com/doc/2013-04-01/"
)

type Client struct {
	http        core.HTTPContext
	credentials *aws.Credentials
	signer      *v4.Signer
}

func NewClient(httpCtx core.HTTPContext, credentials *aws.Credentials) *Client {
	return &Client{
		http:        httpCtx,
		credentials: credentials,
		signer:      v4.NewSigner(),
	}
}

// ChangeResourceRecordSets creates, updates, or deletes DNS records in a hosted zone.
func (c *Client) ChangeResourceRecordSets(hostedZoneID, action string, recordSet ResourceRecordSet) (*ChangeInfo, error) {
	records := make([]ResourceRecord, len(recordSet.Values))
	for i, v := range recordSet.Values {
		records[i] = ResourceRecord{Value: v}
	}

	request := changeResourceRecordSetsRequest{
		XMLNS: xmlNS,
		ChangeBatch: changeBatch{
			Changes: []change{
				{
					Action: action,
					ResourceRecordSet: xmlResourceRecordSet{
						Name:            recordSet.Name,
						Type:            recordSet.Type,
						TTL:             recordSet.TTL,
						ResourceRecords: xmlResourceRecords{Records: records},
					},
				},
			},
		},
	}

	body, err := xml.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	body = append([]byte(xml.Header), body...)
	url := fmt.Sprintf("%s/hostedzone/%s/rrset", endpoint, normalizeHostedZoneID(hostedZoneID))

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "text/xml")

	if err := c.signRequest(req, body); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := parseError(responseBody); awsErr != nil {
			return nil, awsErr
		}
		return nil, fmt.Errorf("Route 53 API request failed with %d: %s", res.StatusCode, string(responseBody))
	}

	var response changeResourceRecordSetsResponse
	if err := xml.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &ChangeInfo{
		ID:          response.ChangeInfo.ID,
		Status:      response.ChangeInfo.Status,
		SubmittedAt: response.ChangeInfo.SubmittedAt,
	}, nil
}

// GetChange returns the current status of a change request.
// changeID is the ID returned by ChangeResourceRecordSets (e.g. /change/C0123456789ABCDEF).
func (c *Client) GetChange(changeID string) (*ChangeInfo, error) {
	changeID = strings.TrimSpace(changeID)
	if changeID == "" {
		return nil, fmt.Errorf("change ID is required")
	}
	if !strings.HasPrefix(changeID, "/") {
		changeID = "/" + changeID
	}
	url := endpoint + changeID

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build get change request: %w", err)
	}

	if err := c.signRequest(req, []byte{}); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get change request failed: %w", err)
	}

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read get change response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := parseError(body); awsErr != nil {
			return nil, awsErr
		}
		return nil, fmt.Errorf("get change failed with %d: %s", res.StatusCode, string(body))
	}

	var response getChangeResponse
	if err := xml.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode get change response: %w", err)
	}

	return &ChangeInfo{
		ID:          response.ChangeInfo.ID,
		Status:      response.ChangeInfo.Status,
		SubmittedAt: response.ChangeInfo.SubmittedAt,
	}, nil
}

// ListHostedZones returns all hosted zones in the account.
func (c *Client) ListHostedZones() ([]HostedZoneSummary, error) {
	var zones []HostedZoneSummary
	marker := ""

	for {
		url := fmt.Sprintf("%s/hostedzone?maxitems=100", endpoint)
		if strings.TrimSpace(marker) != "" {
			url += "&marker=" + marker
		}

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to build list hosted zones request: %w", err)
		}

		if err := c.signRequest(req, []byte{}); err != nil {
			return nil, err
		}

		res, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("list hosted zones request failed: %w", err)
		}

		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read list hosted zones response: %w", err)
		}

		if res.StatusCode < 200 || res.StatusCode >= 300 {
			if awsErr := parseError(body); awsErr != nil {
				return nil, awsErr
			}
			return nil, fmt.Errorf("list hosted zones failed with %d: %s", res.StatusCode, string(body))
		}

		var response listHostedZonesResponse
		if err := xml.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to decode list hosted zones response: %w", err)
		}

		for _, hz := range response.HostedZones {
			zones = append(zones, HostedZoneSummary{
				ID:   hz.ID,
				Name: hz.Name,
			})
		}

		if !response.IsTruncated {
			break
		}
		marker = response.NextMarker
	}

	return zones, nil
}

func (c *Client) signRequest(req *http.Request, payload []byte) error {
	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	return c.signer.SignHTTP(context.Background(), *c.credentials, req, payloadHash, serviceName, region, time.Now())
}

// parseError extracts a user-facing error from Route53 API error responses.
// It handles both the standard ErrorResponse > Error > Code/Message format and
// the InvalidChangeBatch format (root <InvalidChangeBatch> with <Messages><Message>).
func parseError(body []byte) *common.Error {
	// Standard ErrorResponse format (e.g. AccessDenied, InvalidInput).
	var errResp struct {
		Error struct {
			Code    string `xml:"Code"`
			Message string `xml:"Message"`
		} `xml:"Error"`
	}
	if err := xml.Unmarshal(body, &errResp); err == nil && (errResp.Error.Code != "" || errResp.Error.Message != "") {
		return &common.Error{
			Code:    strings.TrimSpace(errResp.Error.Code),
			Message: strings.TrimSpace(errResp.Error.Message),
		}
	}

	// InvalidChangeBatch format (e.g. "RRSet with DNS name X is not permitted in zone Y").
	var invalidBatch struct {
		Messages []string `xml:"Messages>Message"`
	}
	if err := xml.Unmarshal(body, &invalidBatch); err == nil && len(invalidBatch.Messages) > 0 {
		msg := strings.TrimSpace(strings.Join(invalidBatch.Messages, "; "))
		return &common.Error{
			Code:    "InvalidChangeBatch",
			Message: msg,
		}
	}

	return nil
}

/*
 * normalizeHostedZoneID strips the /hostedzone/ prefix if present.
 */
func normalizeHostedZoneID(id string) string {
	return strings.TrimPrefix(strings.TrimSpace(id), "/hostedzone/")
}

// XML request types for ChangeResourceRecordSets.
type changeResourceRecordSetsRequest struct {
	XMLName     xml.Name    `xml:"ChangeResourceRecordSetsRequest"`
	XMLNS       string      `xml:"xmlns,attr"`
	ChangeBatch changeBatch `xml:"ChangeBatch"`
}

type changeBatch struct {
	Changes []change `xml:"Changes>Change"`
}

type change struct {
	Action            string               `xml:"Action"`
	ResourceRecordSet xmlResourceRecordSet `xml:"ResourceRecordSet"`
}

type xmlResourceRecordSet struct {
	Name            string             `xml:"Name"`
	Type            string             `xml:"Type"`
	TTL             int                `xml:"TTL"`
	ResourceRecords xmlResourceRecords `xml:"ResourceRecords"`
}

type xmlResourceRecords struct {
	Records []ResourceRecord `xml:"ResourceRecord"`
}

type ResourceRecord struct {
	Value string `xml:"Value"`
}

// XML response types for ChangeResourceRecordSets.
type changeResourceRecordSetsResponse struct {
	XMLName    xml.Name      `xml:"ChangeResourceRecordSetsResponse"`
	ChangeInfo xmlChangeInfo `xml:"ChangeInfo"`
}

type xmlChangeInfo struct {
	ID          string `xml:"Id"`
	Status      string `xml:"Status"`
	SubmittedAt string `xml:"SubmittedAt"`
}

// XML response types for ListHostedZones.
type listHostedZonesResponse struct {
	XMLName     xml.Name        `xml:"ListHostedZonesResponse"`
	HostedZones []xmlHostedZone `xml:"HostedZones>HostedZone"`
	IsTruncated bool            `xml:"IsTruncated"`
	MaxItems    int             `xml:"MaxItems"`
	NextMarker  string          `xml:"NextMarker"`
}

type xmlHostedZone struct {
	ID   string `xml:"Id"`
	Name string `xml:"Name"`
}

// XML response type for GetChange.
type getChangeResponse struct {
	XMLName    xml.Name      `xml:"GetChangeResponse"`
	ChangeInfo xmlChangeInfo `xml:"ChangeInfo"`
}
