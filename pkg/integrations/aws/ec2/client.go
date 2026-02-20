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

type CopyImageInput struct {
	SourceImageID string
	SourceRegion  string
	Name          string
	Description   string
}

type CopyImageOutput struct {
	RequestID     string `json:"requestId" mapstructure:"requestId"`
	ImageID       string `json:"imageId" mapstructure:"imageId"`
	SourceImageID string `json:"sourceImageId" mapstructure:"sourceImageId"`
	SourceRegion  string `json:"sourceRegion" mapstructure:"sourceRegion"`
	Name          string `json:"name" mapstructure:"name"`
	Description   string `json:"description" mapstructure:"description"`
	Region        string `json:"region" mapstructure:"region"`
	State         string `json:"state" mapstructure:"state"`
}

type EnableImageDeprecationOutput struct {
	RequestID   string `json:"requestId" mapstructure:"requestId"`
	ImageID     string `json:"imageId" mapstructure:"imageId"`
	Region      string `json:"region" mapstructure:"region"`
	DeprecateAt string `json:"deprecateAt" mapstructure:"deprecateAt"`
}

type Image struct {
	RequestID           string                    `json:"requestId" mapstructure:"requestId"`
	ImageID             string                    `json:"imageId" mapstructure:"imageId"`
	ImageLocation       string                    `json:"imageLocation" mapstructure:"imageLocation"`
	Public              bool                      `json:"public" mapstructure:"public"`
	Name                string                    `json:"name" mapstructure:"name"`
	Description         string                    `json:"description" mapstructure:"description"`
	State               string                    `json:"state" mapstructure:"state"`
	CreationDate        string                    `json:"creationDate" mapstructure:"creationDate"`
	OwnerID             string                    `json:"ownerId" mapstructure:"ownerId"`
	PlatformDetails     string                    `json:"platformDetails" mapstructure:"platformDetails"`
	UsageOperation      string                    `json:"usageOperation" mapstructure:"usageOperation"`
	BlockDeviceMappings []ImageBlockDeviceMapping `json:"blockDeviceMappings" mapstructure:"blockDeviceMappings"`
	EnaSupport          bool                      `json:"enaSupport" mapstructure:"enaSupport"`
	SriovNetSupport     string                    `json:"sriovNetSupport" mapstructure:"sriovNetSupport"`
	BootMode            string                    `json:"bootMode" mapstructure:"bootMode"`
	ImdsSupport         string                    `json:"imdsSupport" mapstructure:"imdsSupport"`
	Architecture        string                    `json:"architecture" mapstructure:"architecture"`
	ImageType           string                    `json:"imageType" mapstructure:"imageType"`
	RootDeviceType      string                    `json:"rootDeviceType" mapstructure:"rootDeviceType"`
	RootDeviceName      string                    `json:"rootDeviceName" mapstructure:"rootDeviceName"`
	VirtualizationType  string                    `json:"virtualizationType" mapstructure:"virtualizationType"`
	Hypervisor          string                    `json:"hypervisor" mapstructure:"hypervisor"`
	Region              string                    `json:"region" mapstructure:"region"`
}

type ImageBlockDeviceMapping struct {
	DeviceName string          `json:"deviceName" mapstructure:"deviceName"`
	Ebs        ImageEbsDetails `json:"ebs" mapstructure:"ebs"`
}

type ImageEbsDetails struct {
	DeleteOnTermination bool   `json:"deleteOnTermination" mapstructure:"deleteOnTermination"`
	Iops                int    `json:"iops" mapstructure:"iops"`
	SnapshotID          string `json:"snapshotId" mapstructure:"snapshotId"`
	VolumeSize          int    `json:"volumeSize" mapstructure:"volumeSize"`
	VolumeType          string `json:"volumeType" mapstructure:"volumeType"`
	Throughput          int    `json:"throughput" mapstructure:"throughput"`
	Encrypted           bool   `json:"encrypted" mapstructure:"encrypted"`
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

	if input.NoReboot {
		params.Set("NoReboot", fmt.Sprintf("%t", input.NoReboot))
	}

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

func (c *Client) CopyImage(input CopyImageInput) (*CopyImageOutput, error) {
	params := url.Values{}
	params.Set("SourceImageId", strings.TrimSpace(input.SourceImageID))
	params.Set("SourceRegion", strings.TrimSpace(input.SourceRegion))
	params.Set("Name", strings.TrimSpace(input.Name))

	description := strings.TrimSpace(input.Description)
	if description != "" {
		params.Set("Description", description)
	}

	response := copyImageResponse{}
	if err := c.postForm("CopyImage", params, &response); err != nil {
		return nil, err
	}

	if strings.TrimSpace(response.ImageID) == "" {
		return nil, fmt.Errorf("response did not include image ID")
	}

	return &CopyImageOutput{
		RequestID:     response.RequestID,
		ImageID:       response.ImageID,
		SourceImageID: strings.TrimSpace(input.SourceImageID),
		SourceRegion:  strings.TrimSpace(input.SourceRegion),
		Name:          strings.TrimSpace(input.Name),
		Description:   description,
		Region:        c.region,
		State:         ImageStatePending,
	}, nil
}

func (c *Client) DeregisterImage(imageID string) (string, error) {
	return c.runImageBooleanAction("DeregisterImage", imageID, nil)
}

func (c *Client) DeleteSnapshot(snapshotID string) (string, error) {
	params := url.Values{}
	params.Set("SnapshotId", strings.TrimSpace(snapshotID))

	response := imageActionResponse{}
	if err := c.postForm("DeleteSnapshot", params, &response); err != nil {
		return "", err
	}

	if !response.Return {
		return "", fmt.Errorf("DeleteSnapshot returned unsuccessful response")
	}

	return strings.TrimSpace(response.RequestID), nil
}

func (c *Client) EnableImage(imageID string) (string, error) {
	return c.runImageBooleanAction("EnableImage", imageID, nil)
}

func (c *Client) DisableImage(imageID string) (string, error) {
	return c.runImageBooleanAction("DisableImage", imageID, nil)
}

func (c *Client) EnableImageDeprecation(imageID, deprecateAt string) (*EnableImageDeprecationOutput, error) {
	params := url.Values{}
	params.Set("DeprecateAt", strings.TrimSpace(deprecateAt))

	requestID, err := c.runImageBooleanAction("EnableImageDeprecation", imageID, params)
	if err != nil {
		return nil, err
	}

	return &EnableImageDeprecationOutput{
		RequestID:   requestID,
		ImageID:     strings.TrimSpace(imageID),
		Region:      c.region,
		DeprecateAt: strings.TrimSpace(deprecateAt),
	}, nil
}

func (c *Client) DisableImageDeprecation(imageID string) (string, error) {
	return c.runImageBooleanAction("DisableImageDeprecation", imageID, nil)
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
	return imageFromXML(image), nil
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

func (c *Client) ListImages(ownerID string, includeDisabled bool) ([]Image, error) {
	images := []Image{}
	nextToken := ""
	trimmedOwnerID := strings.TrimSpace(ownerID)

	for {
		params := url.Values{}
		params.Set("MaxResults", "100")
		params.Set("Owner.1", trimmedOwnerID)
		if includeDisabled {
			params.Set("IncludeDisabled", "true")
		}

		if nextToken != "" {
			params.Set("NextToken", nextToken)
		}

		response := describeImagesResponse{}
		if err := c.postForm("DescribeImages", params, &response); err != nil {
			return nil, err
		}

		for _, image := range response.Images {
			images = append(images, *imageFromXML(image))
		}

		nextToken = strings.TrimSpace(response.NextToken)
		if nextToken == "" {
			break
		}
	}

	return images, nil
}

func (c *Client) runImageBooleanAction(action, imageID string, additionalParams url.Values) (string, error) {
	params := additionalParams
	if params == nil {
		params = url.Values{}
	}
	params.Set("ImageId", strings.TrimSpace(imageID))

	response := imageActionResponse{}
	if err := c.postForm(action, params, &response); err != nil {
		return "", err
	}

	if !response.Return {
		return "", fmt.Errorf("%s returned unsuccessful response", action)
	}

	return strings.TrimSpace(response.RequestID), nil
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

type copyImageResponse struct {
	RequestID string `xml:"requestId"`
	ImageID   string `xml:"imageId"`
}

type imageActionResponse struct {
	RequestID string `xml:"requestId"`
	Return    bool   `xml:"return"`
}

type describeInstancesResponse struct {
	Reservations []xmlReservation `xml:"reservationSet>item"`
	NextToken    string           `xml:"nextToken"`
}

type describeImagesResponse struct {
	RequestID string     `xml:"requestId"`
	Images    []xmlImage `xml:"imagesSet>item"`
	NextToken string     `xml:"nextToken"`
}

type xmlReservation struct {
	Instances []xmlInstance `xml:"instancesSet>item"`
}

type xmlImage struct {
	ImageID            string                  `xml:"imageId"`
	ImageLocation      string                  `xml:"imageLocation"`
	Public             bool                    `xml:"isPublic"`
	Name               string                  `xml:"name"`
	Description        string                  `xml:"description"`
	State              string                  `xml:"imageState"`
	CreationDate       string                  `xml:"creationDate"`
	OwnerID            string                  `xml:"ownerId"`
	PlatformDetails    string                  `xml:"platformDetails"`
	UsageOperation     string                  `xml:"usageOperation"`
	BlockDeviceMapping []xmlBlockDeviceMapping `xml:"blockDeviceMapping>item"`
	EnaSupport         bool                    `xml:"enaSupport"`
	SriovNetSupport    string                  `xml:"sriovNetSupport"`
	BootMode           string                  `xml:"bootMode"`
	ImdsSupport        string                  `xml:"imdsSupport"`
	Architecture       string                  `xml:"architecture"`
	ImageType          string                  `xml:"imageType"`
	RootDeviceType     string                  `xml:"rootDeviceType"`
	RootDeviceName     string                  `xml:"rootDeviceName"`
	VirtualizationType string                  `xml:"virtualizationType"`
	Hypervisor         string                  `xml:"hypervisor"`
}

type xmlBlockDeviceMapping struct {
	DeviceName string            `xml:"deviceName"`
	Ebs        xmlEbsBlockDevice `xml:"ebs"`
}

type xmlEbsBlockDevice struct {
	DeleteOnTermination bool   `xml:"deleteOnTermination"`
	Iops                int    `xml:"iops"`
	SnapshotID          string `xml:"snapshotId"`
	VolumeSize          int    `xml:"volumeSize"`
	VolumeType          string `xml:"volumeType"`
	Throughput          int    `xml:"throughput"`
	Encrypted           bool   `xml:"encrypted"`
}

func imageFromXML(image xmlImage) *Image {
	if image.OwnerID == "" {
		image.OwnerID = ownerIDFromImageLocation(image.ImageLocation)
	}

	blockDeviceMappings := make([]ImageBlockDeviceMapping, 0, len(image.BlockDeviceMapping))
	for _, mapping := range image.BlockDeviceMapping {
		blockDeviceMappings = append(blockDeviceMappings, ImageBlockDeviceMapping{
			DeviceName: strings.TrimSpace(mapping.DeviceName),
			Ebs: ImageEbsDetails{
				DeleteOnTermination: mapping.Ebs.DeleteOnTermination,
				Iops:                mapping.Ebs.Iops,
				SnapshotID:          strings.TrimSpace(mapping.Ebs.SnapshotID),
				VolumeSize:          mapping.Ebs.VolumeSize,
				VolumeType:          strings.TrimSpace(mapping.Ebs.VolumeType),
				Throughput:          mapping.Ebs.Throughput,
				Encrypted:           mapping.Ebs.Encrypted,
			},
		})
	}

	return &Image{
		ImageID:             image.ImageID,
		ImageLocation:       image.ImageLocation,
		Public:              image.Public,
		Name:                image.Name,
		Description:         image.Description,
		State:               image.State,
		CreationDate:        image.CreationDate,
		OwnerID:             image.OwnerID,
		PlatformDetails:     image.PlatformDetails,
		UsageOperation:      image.UsageOperation,
		BlockDeviceMappings: blockDeviceMappings,
		EnaSupport:          image.EnaSupport,
		SriovNetSupport:     image.SriovNetSupport,
		BootMode:            image.BootMode,
		ImdsSupport:         image.ImdsSupport,
		Architecture:        image.Architecture,
		ImageType:           image.ImageType,
		RootDeviceType:      image.RootDeviceType,
		RootDeviceName:      image.RootDeviceName,
		VirtualizationType:  image.VirtualizationType,
		Hypervisor:          image.Hypervisor,
	}
}

func ownerIDFromImageLocation(imageLocation string) string {
	prefix, _, ok := strings.Cut(strings.TrimSpace(imageLocation), "/")
	if !ok || len(prefix) != 12 {
		return ""
	}

	for _, c := range prefix {
		if c < '0' || c > '9' {
			return ""
		}
	}

	return prefix
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
