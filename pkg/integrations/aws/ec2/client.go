package ec2

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	serviceName = "ec2"
	apiVersion  = "2016-11-15"
)

type Client struct {
	http        core.HTTPContext
	region      string
	credentials *aws.Credentials
	signer      *v4.Signer
}

type Instance struct {
	InstanceID   string `json:"instanceId" mapstructure:"instanceId"`
	InstanceType string `json:"instanceType" mapstructure:"instanceType"`
	State        string `json:"state" mapstructure:"state"`
	Name         string `json:"name" mapstructure:"name"`
}

type CreateImageInput struct {
	InstanceID  string
	Name        string
	Description string
	NoReboot    bool
}

type CreateImageOutput struct {
	RequestID  string `json:"requestId" mapstructure:"requestId"`
	ImageID    string `json:"imageId" mapstructure:"imageId"`
	InstanceID string `json:"instanceId" mapstructure:"instanceId"`
	Name       string `json:"name" mapstructure:"name"`
	Region     string `json:"region" mapstructure:"region"`
	State      string `json:"state" mapstructure:"state"`
}

type Image struct {
	RequestID          string `json:"requestId" mapstructure:"requestId"`
	ImageID            string `json:"imageId" mapstructure:"imageId"`
	Name               string `json:"name" mapstructure:"name"`
	Description        string `json:"description" mapstructure:"description"`
	State              string `json:"state" mapstructure:"state"`
	CreationDate       string `json:"creationDate" mapstructure:"creationDate"`
	OwnerID            string `json:"ownerId" mapstructure:"ownerId"`
	Architecture       string `json:"architecture" mapstructure:"architecture"`
	ImageType          string `json:"imageType" mapstructure:"imageType"`
	RootDeviceType     string `json:"rootDeviceType" mapstructure:"rootDeviceType"`
	RootDeviceName     string `json:"rootDeviceName" mapstructure:"rootDeviceName"`
	VirtualizationType string `json:"virtualizationType" mapstructure:"virtualizationType"`
	Hypervisor         string `json:"hypervisor" mapstructure:"hypervisor"`
	Region             string `json:"region" mapstructure:"region"`
}

func NewClient(httpCtx core.HTTPContext, credentials *aws.Credentials, region string) *Client {
	return &Client{
		http:        httpCtx,
		region:      region,
		credentials: credentials,
		signer:      v4.NewSigner(),
	}
}

func (c *Client) CreateImage(input CreateImageInput) (*CreateImageOutput, error) {
	params := url.Values{}
	params.Set("InstanceId", strings.TrimSpace(input.InstanceID))
	params.Set("Name", strings.TrimSpace(input.Name))
	params.Set("NoReboot", fmt.Sprintf("%t", input.NoReboot))

	description := strings.TrimSpace(input.Description)
	if description != "" {
		params.Set("Description", description)
	}

	response := createImageResponse{}
	if err := c.postForm("CreateImage", params, &response); err != nil {
		return nil, err
	}

	if strings.TrimSpace(response.ImageID) == "" {
		return nil, fmt.Errorf("response did not include image ID")
	}

	return &CreateImageOutput{
		RequestID:  response.RequestID,
		ImageID:    response.ImageID,
		InstanceID: strings.TrimSpace(input.InstanceID),
		Name:       strings.TrimSpace(input.Name),
		Region:     c.region,
		State:      ImageStatePending,
	}, nil
}

func (c *Client) DescribeImage(imageID string) (*Image, error) {
	params := url.Values{}
	params.Set("ImageId.1", strings.TrimSpace(imageID))

	response := describeImagesResponse{}
	if err := c.postForm("DescribeImages", params, &response); err != nil {
		return nil, err
	}

	if len(response.Images) == 0 {
		return nil, fmt.Errorf("image not found: %s", imageID)
	}

	image := response.Images[0]
	return &Image{
		ImageID:            strings.TrimSpace(image.ImageID),
		Name:               strings.TrimSpace(image.Name),
		Description:        strings.TrimSpace(image.Description),
		State:              strings.TrimSpace(image.State),
		CreationDate:       strings.TrimSpace(image.CreationDate),
		OwnerID:            strings.TrimSpace(image.OwnerID),
		Architecture:       strings.TrimSpace(image.Architecture),
		ImageType:          strings.TrimSpace(image.ImageType),
		RootDeviceType:     strings.TrimSpace(image.RootDeviceType),
		RootDeviceName:     strings.TrimSpace(image.RootDeviceName),
		VirtualizationType: strings.TrimSpace(image.VirtualizationType),
		Hypervisor:         strings.TrimSpace(image.Hypervisor),
		Region:             c.region,
	}, nil
}

func (c *Client) ListInstances() ([]Instance, error) {
	instances := []Instance{}
	nextToken := ""

	for {
		params := url.Values{}
		params.Set("MaxResults", "100")
		params.Set("Filter.1.Name", "instance-state-name")
		params.Set("Filter.1.Value.1", "running")
		params.Set("Filter.1.Value.2", "stopped")

		if nextToken != "" {
			params.Set("NextToken", nextToken)
		}

		response := describeInstancesResponse{}
		if err := c.postForm("DescribeInstances", params, &response); err != nil {
			return nil, err
		}

		for _, reservation := range response.Reservations {
			for _, instance := range reservation.Instances {
				instances = append(instances, Instance{
					InstanceID:   instance.InstanceID,
					InstanceType: instance.InstanceType,
					State:        instance.State.Name,
					Name:         nameTag(instance.Tags),
				})
			}
		}

		nextToken = strings.TrimSpace(response.NextToken)
		if nextToken == "" {
			break
		}
	}

	return instances, nil
}

func (c *Client) postForm(action string, params url.Values, out any) error {
	if params == nil {
		params = url.Values{}
	}

	params.Set("Action", action)
	params.Set("Version", apiVersion)

	body := []byte(params.Encode())
	endpoint := fmt.Sprintf("https://ec2.%s.amazonaws.com/", c.region)
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	if err := c.signRequest(req, body); err != nil {
		return err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := parseError(responseBody); awsErr != nil {
			return awsErr
		}
		return fmt.Errorf("EC2 API request failed with %d: %s", res.StatusCode, string(responseBody))
	}

	if out == nil {
		return nil
	}

	if err := xml.Unmarshal(responseBody, out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

func (c *Client) signRequest(req *http.Request, payload []byte) error {
	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	return c.signer.SignHTTP(context.Background(), *c.credentials, req, payloadHash, serviceName, c.region, time.Now())
}

func nameTag(tags []xmlTag) string {
	for _, tag := range tags {
		if tag.Key == "Name" {
			return tag.Value
		}
	}

	return ""
}

func parseError(body []byte) *common.Error {
	var errResp struct {
		Errors struct {
			Error struct {
				Code    string `xml:"Code"`
				Message string `xml:"Message"`
			} `xml:"Error"`
		} `xml:"Errors"`
	}

	if err := xml.Unmarshal(body, &errResp); err == nil {
		if strings.TrimSpace(errResp.Errors.Error.Code) != "" || strings.TrimSpace(errResp.Errors.Error.Message) != "" {
			return &common.Error{
				Code:    strings.TrimSpace(errResp.Errors.Error.Code),
				Message: strings.TrimSpace(errResp.Errors.Error.Message),
			}
		}
	}

	return nil
}

type createImageResponse struct {
	RequestID string `xml:"requestId"`
	ImageID   string `xml:"imageId"`
}

type describeInstancesResponse struct {
	Reservations []xmlReservation `xml:"reservationSet>item"`
	NextToken    string           `xml:"nextToken"`
}

type describeImagesResponse struct {
	RequestID string     `xml:"requestId"`
	Images    []xmlImage `xml:"imagesSet>item"`
}

type xmlReservation struct {
	Instances []xmlInstance `xml:"instancesSet>item"`
}

type xmlImage struct {
	ImageID            string `xml:"imageId"`
	Name               string `xml:"name"`
	Description        string `xml:"description"`
	State              string `xml:"imageState"`
	CreationDate       string `xml:"creationDate"`
	OwnerID            string `xml:"ownerId"`
	Architecture       string `xml:"architecture"`
	ImageType          string `xml:"imageType"`
	RootDeviceType     string `xml:"rootDeviceType"`
	RootDeviceName     string `xml:"rootDeviceName"`
	VirtualizationType string `xml:"virtualizationType"`
	Hypervisor         string `xml:"hypervisor"`
}

type xmlInstance struct {
	InstanceID   string   `xml:"instanceId"`
	InstanceType string   `xml:"instanceType"`
	State        xmlState `xml:"instanceState"`
	Tags         []xmlTag `xml:"tagSet>item"`
}

type xmlState struct {
	Name string `xml:"name"`
}

type xmlTag struct {
	Key   string `xml:"key"`
	Value string `xml:"value"`
}
