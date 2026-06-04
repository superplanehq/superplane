package ec2

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	ec2ServiceName                    = "ec2"
	ssmServiceName                    = "ssm"
	elbServiceName                    = "elasticloadbalancing"
	cloudWatchServiceName             = "monitoring"
	ssmTargetPrefix                   = "AmazonSSM."
	ec2APIVersion                     = "2016-11-15"
	elbAPIVersion                     = "2015-12-01"
	cloudWatchAPIVersion              = "2010-08-01"
	ResourceTypeImageOS               = "ec2.imageOS"
	ResourceTypeElasticIP             = "ec2.elasticIp"
	ResourceTypeElasticIPUnassociated = "ec2.elasticIpUnassociated"
	ResourceTypeElasticIPAssociation  = "ec2.elasticIpAssociation"
	ResourceTypePublicIPv4Pool        = "ec2.publicIpv4Pool"
	ResourceTypeCustomerOwnedIPv4Pool = "ec2.customerOwnedIpv4Pool"
	ResourceTypeIpamPool              = "ec2.ipamPool"
	maxPublicImagesPerOS              = 200
	defaultUbuntuImages               = 12
	defaultDebianImages               = 8
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

type InstanceDetails struct {
	RequestID        string `json:"requestId,omitempty" mapstructure:"requestId"`
	InstanceID       string `json:"instanceId" mapstructure:"instanceId"`
	InstanceType     string `json:"instanceType" mapstructure:"instanceType"`
	ImageID          string `json:"imageId" mapstructure:"imageId"`
	State            string `json:"state" mapstructure:"state"`
	Name             string `json:"name" mapstructure:"name"`
	KeyName          string `json:"keyName,omitempty" mapstructure:"keyName"`
	LaunchTime       string `json:"launchTime,omitempty" mapstructure:"launchTime"`
	PrivateIPAddress string `json:"privateIpAddress,omitempty" mapstructure:"privateIpAddress"`
	PublicIPAddress  string `json:"publicIpAddress,omitempty" mapstructure:"publicIpAddress"`
	PrivateDNSName   string `json:"privateDnsName,omitempty" mapstructure:"privateDnsName"`
	PublicDNSName    string `json:"publicDnsName,omitempty" mapstructure:"publicDnsName"`
	SubnetID         string `json:"subnetId,omitempty" mapstructure:"subnetId"`
	VpcID            string `json:"vpcId,omitempty" mapstructure:"vpcId"`
	Region           string `json:"region" mapstructure:"region"`
}

type Subnet struct {
	SubnetID         string `json:"subnetId" mapstructure:"subnetId"`
	VpcID            string `json:"vpcId" mapstructure:"vpcId"`
	CidrBlock        string `json:"cidrBlock" mapstructure:"cidrBlock"`
	AvailabilityZone string `json:"availabilityZone" mapstructure:"availabilityZone"`
	Name             string `json:"name" mapstructure:"name"`
}

type SecurityGroup struct {
	GroupID     string `json:"groupId" mapstructure:"groupId"`
	GroupName   string `json:"groupName" mapstructure:"groupName"`
	Description string `json:"description" mapstructure:"description"`
	VpcID       string `json:"vpcId" mapstructure:"vpcId"`
}

type KeyPair struct {
	KeyName   string `json:"keyName" mapstructure:"keyName"`
	KeyPairID string `json:"keyPairId" mapstructure:"keyPairId"`
}

type RunInstancesInput struct {
	ImageID                  string
	InstanceType             string
	SubnetID                 string
	SecurityGroupIDs         []string
	KeyName                  string
	UserData                 string
	Name                     string
	AssociatePublicIPAddress bool
	RootVolume               *RootVolumeConfig
}

type RootVolumeConfig struct {
	DeviceName string
	VolumeSize int
	VolumeType string
	Iops       int
}

type InstanceTypeInfo struct {
	InstanceType string `json:"instanceType" mapstructure:"instanceType"`
	VCPUs        int    `json:"vcpus" mapstructure:"vcpus"`
	MemoryMiB    int    `json:"memoryMiB" mapstructure:"memoryMiB"`
}

type GetMetricStatisticsInput struct {
	Namespace  string
	MetricName string
	InstanceID string
	StartTime  time.Time
	EndTime    time.Time
	Period     int
	Statistic  string
}

type CloudWatchDatapoint struct {
	Timestamp string
	Average   float64
	Sum       float64
}

type RunInstancesOutput struct {
	RequestID  string `json:"requestId" mapstructure:"requestId"`
	InstanceID string `json:"instanceId" mapstructure:"instanceId"`
	Region     string `json:"region" mapstructure:"region"`
	State      string `json:"state" mapstructure:"state"`
}

type TerminateInstancesOutput struct {
	RequestID  string `json:"requestId" mapstructure:"requestId"`
	InstanceID string `json:"instanceId" mapstructure:"instanceId"`
	State      string `json:"state" mapstructure:"state"`
}

type StopInstancesOutput struct {
	RequestID  string `json:"requestId" mapstructure:"requestId"`
	InstanceID string `json:"instanceId" mapstructure:"instanceId"`
	State      string `json:"state" mapstructure:"state"`
}

type StartInstancesOutput struct {
	RequestID  string `json:"requestId" mapstructure:"requestId"`
	InstanceID string `json:"instanceId" mapstructure:"instanceId"`
	State      string `json:"state" mapstructure:"state"`
}

type AllocateAddressInput struct {
	PublicIPv4Pool        string
	CustomerOwnedIPv4Pool string
	IpamPoolID            string
	Address               string
	Tags                  []common.Tag
}

type AllocateAddressOutput struct {
	RequestID    string `json:"requestId,omitempty" mapstructure:"requestId"`
	AllocationID string `json:"allocationId" mapstructure:"allocationId"`
	PublicIP     string `json:"publicIp" mapstructure:"publicIp"`
	Domain       string `json:"domain" mapstructure:"domain"`
	Region       string `json:"region" mapstructure:"region"`
}

type AssociateAddressInput struct {
	AllocationID       string
	InstanceID         string
	NetworkInterfaceID string
	AllowReassociation bool
}

type AssociateAddressOutput struct {
	RequestID     string `json:"requestId,omitempty" mapstructure:"requestId"`
	AssociationID string `json:"associationId" mapstructure:"associationId"`
	Region        string `json:"region" mapstructure:"region"`
}

type ElasticIP struct {
	AllocationID  string `json:"allocationId" mapstructure:"allocationId"`
	AssociationID string `json:"associationId,omitempty" mapstructure:"associationId"`
	PublicIP      string `json:"publicIp" mapstructure:"publicIp"`
	InstanceID    string `json:"instanceId,omitempty" mapstructure:"instanceId"`
	Domain        string `json:"domain" mapstructure:"domain"`
}

type PublicIPv4Pool struct {
	PoolID      string `json:"poolId" mapstructure:"poolId"`
	Description string `json:"description,omitempty" mapstructure:"description"`
}

type CustomerOwnedIPv4Pool struct {
	PoolID                   string `json:"poolId" mapstructure:"poolId"`
	LocalGatewayRouteTableID string `json:"localGatewayRouteTableId,omitempty" mapstructure:"localGatewayRouteTableId"`
}

type IpamPool struct {
	PoolID                 string `json:"ipamPoolId" mapstructure:"ipamPoolId"`
	Description            string `json:"description,omitempty" mapstructure:"description"`
	AddressFamily          string `json:"addressFamily" mapstructure:"addressFamily"`
	PubliclyAdvertisable   bool   `json:"publiclyAdvertisable" mapstructure:"publiclyAdvertisable"`
	Locale                 string `json:"locale,omitempty" mapstructure:"locale"`
	AllocationResourceType string `json:"allocationResourceType,omitempty" mapstructure:"allocationResourceType"`
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
					State:        instance.stateName(),
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

func (c *Client) RunInstances(input RunInstancesInput) (*RunInstancesOutput, error) {
	params := url.Values{}
	params.Set("MinCount", "1")
	params.Set("MaxCount", "1")
	params.Set("ImageId", strings.TrimSpace(input.ImageID))
	params.Set("InstanceType", strings.TrimSpace(input.InstanceType))
	params.Set("NetworkInterface.1.DeviceIndex", "0")
	params.Set("NetworkInterface.1.SubnetId", strings.TrimSpace(input.SubnetID))

	params.Set("NetworkInterface.1.AssociatePublicIpAddress", fmt.Sprintf("%t", input.AssociatePublicIPAddress))

	securityGroupIndex := 1
	for _, securityGroupID := range input.SecurityGroupIDs {
		trimmed := strings.TrimSpace(securityGroupID)
		if trimmed == "" {
			continue
		}
		params.Set(fmt.Sprintf("NetworkInterface.1.SecurityGroupId.%d", securityGroupIndex), trimmed)
		securityGroupIndex++
	}

	keyName := strings.TrimSpace(input.KeyName)
	if keyName != "" {
		params.Set("KeyName", keyName)
	}

	userData := strings.TrimSpace(input.UserData)
	if userData != "" {
		params.Set("UserData", base64.StdEncoding.EncodeToString([]byte(userData)))
	}

	name := strings.TrimSpace(input.Name)
	if name != "" {
		params.Set("TagSpecification.1.ResourceType", "instance")
		params.Set("TagSpecification.1.Tag.1.Key", "Name")
		params.Set("TagSpecification.1.Tag.1.Value", name)
	}

	if input.RootVolume != nil {
		deviceName := strings.TrimSpace(input.RootVolume.DeviceName)
		if deviceName == "" {
			deviceName = "/dev/xvda"
		}

		params.Set("BlockDeviceMapping.1.DeviceName", deviceName)
		params.Set("BlockDeviceMapping.1.Ebs.DeleteOnTermination", "true")
		if input.RootVolume.VolumeSize > 0 {
			params.Set("BlockDeviceMapping.1.Ebs.VolumeSize", fmt.Sprintf("%d", input.RootVolume.VolumeSize))
		}
		volumeType := strings.TrimSpace(input.RootVolume.VolumeType)
		if volumeType != "" {
			params.Set("BlockDeviceMapping.1.Ebs.VolumeType", volumeType)
		}
		if input.RootVolume.Iops > 0 {
			params.Set("BlockDeviceMapping.1.Ebs.Iops", fmt.Sprintf("%d", input.RootVolume.Iops))
		}
	}

	response := runInstancesResponse{}
	if err := c.postForm("RunInstances", params, &response); err != nil {
		return nil, err
	}

	if len(response.Instances) == 0 || strings.TrimSpace(response.Instances[0].InstanceID) == "" {
		return nil, fmt.Errorf("response did not include instance ID")
	}

	instance := response.Instances[0]
	return &RunInstancesOutput{
		RequestID:  response.RequestID,
		InstanceID: instance.InstanceID,
		Region:     c.region,
		State:      instance.stateName(),
	}, nil
}

func (c *Client) DescribeInstance(instanceID string) (*InstanceDetails, error) {
	params := url.Values{}
	params.Set("InstanceId.1", strings.TrimSpace(instanceID))

	response := describeInstancesResponse{}
	if err := c.postForm("DescribeInstances", params, &response); err != nil {
		return nil, err
	}

	for _, reservation := range response.Reservations {
		for _, instance := range reservation.Instances {
			if instance.InstanceID == strings.TrimSpace(instanceID) {
				return instanceDetailsFromXML(instance, c.region, response.RequestID), nil
			}
		}
	}

	return nil, &common.Error{
		Code:    "InvalidInstanceID.NotFound",
		Message: fmt.Sprintf("instance not found: %s", instanceID),
	}
}

func (c *Client) TerminateInstances(instanceIDs ...string) (*TerminateInstancesOutput, error) {
	params := url.Values{}
	index := 1
	for _, instanceID := range instanceIDs {
		trimmed := strings.TrimSpace(instanceID)
		if trimmed == "" {
			continue
		}
		params.Set(fmt.Sprintf("InstanceId.%d", index), trimmed)
		index++
	}

	if index == 1 {
		return nil, fmt.Errorf("at least one instance ID is required")
	}

	response := terminateInstancesResponse{}
	if err := c.postForm("TerminateInstances", params, &response); err != nil {
		return nil, err
	}

	if len(response.Instances) == 0 || strings.TrimSpace(response.Instances[0].InstanceID) == "" {
		return nil, fmt.Errorf("response did not include instance ID")
	}

	instance := response.Instances[0]
	return &TerminateInstancesOutput{
		RequestID:  response.RequestID,
		InstanceID: instance.InstanceID,
		State:      instance.stateName(),
	}, nil
}

func (c *Client) StopInstances(instanceID string) (*StopInstancesOutput, error) {
	return c.stopInstances(instanceID, false)
}

func (c *Client) HibernateInstances(instanceID string) (*StopInstancesOutput, error) {
	return c.stopInstances(instanceID, true)
}

func (c *Client) stopInstances(instanceID string, hibernate bool) (*StopInstancesOutput, error) {
	params := url.Values{}
	params.Set("InstanceId.1", strings.TrimSpace(instanceID))
	if hibernate {
		params.Set("Hibernate", "true")
	}

	response := stopInstancesResponse{}
	if err := c.postForm("StopInstances", params, &response); err != nil {
		return nil, err
	}

	if len(response.Instances) == 0 {
		return nil, fmt.Errorf("response did not include instance ID")
	}

	instance := response.Instances[0]
	return &StopInstancesOutput{
		RequestID:  response.RequestID,
		InstanceID: instance.InstanceID,
		State:      instance.CurrentState.Name,
	}, nil
}

func (c *Client) RebootInstances(instanceID string) error {
	params := url.Values{}
	params.Set("InstanceId.1", strings.TrimSpace(instanceID))

	response := struct {
		RequestID string `xml:"requestId"`
		Return    bool   `xml:"return"`
	}{}
	if err := c.postForm("RebootInstances", params, &response); err != nil {
		return err
	}

	if !response.Return {
		return fmt.Errorf("RebootInstances returned false")
	}

	return nil
}

func (c *Client) ModifyInstanceType(instanceID, instanceType string) error {
	params := url.Values{}
	params.Set("InstanceId", strings.TrimSpace(instanceID))
	params.Set("InstanceType.Value", strings.TrimSpace(instanceType))

	response := modifyInstanceAttributeResponse{}
	if err := c.postForm("ModifyInstanceAttribute", params, &response); err != nil {
		return err
	}

	if !response.Return {
		return fmt.Errorf("ModifyInstanceAttribute returned false")
	}

	return nil
}

func (c *Client) ModifySecurityGroups(instanceID string, groupIDs []string) error {
	params := url.Values{}
	params.Set("InstanceId", strings.TrimSpace(instanceID))

	index := 1
	for _, id := range groupIDs {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		params.Set(fmt.Sprintf("GroupId.%d", index), trimmed)
		index++
	}

	response := modifyInstanceAttributeResponse{}
	if err := c.postForm("ModifyInstanceAttribute", params, &response); err != nil {
		return err
	}

	if !response.Return {
		return fmt.Errorf("ModifyInstanceAttribute for security groups returned false")
	}

	return nil
}

// GetMetricStatistics retrieves CloudWatch metric data points for an EC2 instance.
// Statistic should be "Average" for CPU/memory metrics, "Sum" for network metrics.
func (c *Client) GetMetricStatistics(input GetMetricStatisticsInput) ([]CloudWatchDatapoint, error) {
	params := url.Values{}
	params.Set("Namespace", input.Namespace)
	params.Set("MetricName", input.MetricName)
	params.Set("Dimensions.member.1.Name", "InstanceId")
	params.Set("Dimensions.member.1.Value", strings.TrimSpace(input.InstanceID))
	params.Set("StartTime", input.StartTime.UTC().Format(time.RFC3339))
	params.Set("EndTime", input.EndTime.UTC().Format(time.RFC3339))
	params.Set("Period", fmt.Sprintf("%d", input.Period))
	params.Set("Statistics.member.1", input.Statistic)

	response := getMetricStatisticsResponse{}
	if err := c.postCloudWatchForm("GetMetricStatistics", params, &response); err != nil {
		return nil, err
	}

	points := make([]CloudWatchDatapoint, 0, len(response.Datapoints))
	for _, dp := range response.Datapoints {
		points = append(points, CloudWatchDatapoint{
			Timestamp: dp.Timestamp,
			Average:   dp.Average,
			Sum:       dp.Sum,
		})
	}

	return points, nil
}

func (c *Client) postCloudWatchForm(action string, params url.Values, out any) error {
	return c.postSignedForm(cloudWatchServiceName, cloudWatchAPIVersion, action, params, out)
}

func (c *Client) StartInstances(instanceID string) (*StartInstancesOutput, error) {
	params := url.Values{}
	params.Set("InstanceId.1", strings.TrimSpace(instanceID))

	response := startInstancesResponse{}
	if err := c.postForm("StartInstances", params, &response); err != nil {
		return nil, err
	}

	if len(response.Instances) == 0 {
		return nil, fmt.Errorf("response did not include instance ID")
	}

	instance := response.Instances[0]
	return &StartInstancesOutput{
		RequestID:  response.RequestID,
		InstanceID: instance.InstanceID,
		State:      instance.CurrentState.Name,
	}, nil
}

func (c *Client) AllocateAddress(input AllocateAddressInput) (*AllocateAddressOutput, error) {
	params := url.Values{}
	params.Set("Domain", "vpc")

	if pool := strings.TrimSpace(input.PublicIPv4Pool); pool != "" {
		params.Set("PublicIpv4Pool", pool)
	}
	if pool := strings.TrimSpace(input.CustomerOwnedIPv4Pool); pool != "" {
		params.Set("CustomerOwnedIpv4Pool", pool)
	}
	if pool := strings.TrimSpace(input.IpamPoolID); pool != "" {
		params.Set("IpamPoolId", pool)
	}
	if address := strings.TrimSpace(input.Address); address != "" {
		params.Set("Address", address)
	}
	if len(input.Tags) > 0 {
		params.Set("TagSpecification.1.ResourceType", "elastic-ip")
		for i, tag := range input.Tags {
			prefix := fmt.Sprintf("TagSpecification.1.Tag.%d.", i+1)
			params.Set(prefix+"Key", tag.Key)
			params.Set(prefix+"Value", tag.Value)
		}
	}

	response := allocateAddressResponse{}
	if err := c.postForm("AllocateAddress", params, &response); err != nil {
		return nil, err
	}

	if strings.TrimSpace(response.AllocationID) == "" {
		return nil, fmt.Errorf("response did not include allocation ID")
	}

	return &AllocateAddressOutput{
		RequestID:    response.RequestID,
		AllocationID: response.AllocationID,
		PublicIP:     response.PublicIP,
		Domain:       response.Domain,
		Region:       c.region,
	}, nil
}

func (c *Client) ReleaseAddress(allocationID string) error {
	params := url.Values{}
	params.Set("AllocationId", strings.TrimSpace(allocationID))

	response := releaseAddressResponse{}
	if err := c.postForm("ReleaseAddress", params, &response); err != nil {
		return err
	}

	if !response.Return {
		return fmt.Errorf("ReleaseAddress returned false")
	}

	return nil
}

func (c *Client) AssociateAddress(input AssociateAddressInput) (*AssociateAddressOutput, error) {
	params := url.Values{}
	params.Set("AllocationId", strings.TrimSpace(input.AllocationID))

	if input.AllowReassociation {
		params.Set("AllowReassociation", "true")
	}

	instanceID := strings.TrimSpace(input.InstanceID)
	networkInterfaceID := strings.TrimSpace(input.NetworkInterfaceID)
	if instanceID != "" {
		params.Set("InstanceId", instanceID)
	}
	if networkInterfaceID != "" {
		params.Set("NetworkInterfaceId", networkInterfaceID)
	}

	response := associateAddressResponse{}
	if err := c.postForm("AssociateAddress", params, &response); err != nil {
		return nil, err
	}

	if strings.TrimSpace(response.AssociationID) == "" {
		return nil, fmt.Errorf("response did not include association ID")
	}

	return &AssociateAddressOutput{
		RequestID:     response.RequestID,
		AssociationID: response.AssociationID,
		Region:        c.region,
	}, nil
}

func (c *Client) DisassociateAddress(associationID string) error {
	params := url.Values{}
	params.Set("AssociationId", strings.TrimSpace(associationID))

	response := disassociateAddressResponse{}
	if err := c.postForm("DisassociateAddress", params, &response); err != nil {
		return err
	}

	if !response.Return {
		return fmt.Errorf("DisassociateAddress returned false")
	}

	return nil
}

func (c *Client) ListAddresses() ([]ElasticIP, error) {
	response := describeAddressesResponse{}
	if err := c.postForm("DescribeAddresses", url.Values{}, &response); err != nil {
		return nil, err
	}

	addresses := make([]ElasticIP, 0, len(response.Addresses))
	for _, address := range response.Addresses {
		allocationID := strings.TrimSpace(address.AllocationID)
		if allocationID == "" {
			continue
		}

		addresses = append(addresses, ElasticIP{
			AllocationID:  allocationID,
			AssociationID: strings.TrimSpace(address.AssociationID),
			PublicIP:      strings.TrimSpace(address.PublicIP),
			InstanceID:    strings.TrimSpace(address.InstanceID),
			Domain:        strings.TrimSpace(address.Domain),
		})
	}

	return addresses, nil
}

func (c *Client) ListPublicIPv4Pools() ([]PublicIPv4Pool, error) {
	pools := []PublicIPv4Pool{}
	nextToken := ""

	for {
		params := url.Values{}
		params.Set("MaxResults", "10")
		if nextToken != "" {
			params.Set("NextToken", nextToken)
		}

		response := describePublicIpv4PoolsResponse{}
		if err := c.postForm("DescribePublicIpv4Pools", params, &response); err != nil {
			return nil, err
		}

		for _, pool := range response.Pools {
			poolID := strings.TrimSpace(pool.PoolID)
			if poolID == "" {
				continue
			}

			pools = append(pools, PublicIPv4Pool{
				PoolID:      poolID,
				Description: strings.TrimSpace(pool.Description),
			})
		}

		nextToken = strings.TrimSpace(response.NextToken)
		if nextToken == "" {
			break
		}
	}

	return pools, nil
}

func (c *Client) ListCustomerOwnedIPv4Pools() ([]CustomerOwnedIPv4Pool, error) {
	pools := []CustomerOwnedIPv4Pool{}
	nextToken := ""

	for {
		params := url.Values{}
		params.Set("MaxResults", "100")
		if nextToken != "" {
			params.Set("NextToken", nextToken)
		}

		response := describeCoipPoolsResponse{}
		if err := c.postForm("DescribeCoipPools", params, &response); err != nil {
			return nil, err
		}

		for _, pool := range response.Pools {
			poolID := strings.TrimSpace(pool.PoolID)
			if poolID == "" {
				continue
			}

			pools = append(pools, CustomerOwnedIPv4Pool{
				PoolID:                   poolID,
				LocalGatewayRouteTableID: strings.TrimSpace(pool.LocalGatewayRouteTableID),
			})
		}

		nextToken = strings.TrimSpace(response.NextToken)
		if nextToken == "" {
			break
		}
	}

	return pools, nil
}

func (c *Client) ListIpamPoolsForElasticIP() ([]IpamPool, error) {
	pools := []IpamPool{}
	nextToken := ""

	for {
		params := url.Values{}
		params.Set("MaxResults", "100")
		if nextToken != "" {
			params.Set("NextToken", nextToken)
		}

		response := describeIpamPoolsResponse{}
		if err := c.postForm("DescribeIpamPools", params, &response); err != nil {
			return nil, err
		}

		for _, pool := range response.Pools {
			if !isIpamPoolForElasticIP(pool, c.region) {
				continue
			}

			poolID := strings.TrimSpace(pool.IpamPoolID)
			if poolID == "" {
				continue
			}

			pools = append(pools, IpamPool{
				PoolID:                 poolID,
				Description:            strings.TrimSpace(pool.Description),
				AddressFamily:          strings.TrimSpace(pool.AddressFamily),
				PubliclyAdvertisable:   pool.PubliclyAdvertisable,
				Locale:                 strings.TrimSpace(pool.Locale),
				AllocationResourceType: strings.TrimSpace(pool.AllocationResourceType),
			})
		}

		nextToken = strings.TrimSpace(response.NextToken)
		if nextToken == "" {
			break
		}
	}

	return pools, nil
}

func isIpamPoolForElasticIP(pool xmlIpamPool, region string) bool {
	if !strings.EqualFold(strings.TrimSpace(pool.AddressFamily), "ipv4") {
		return false
	}
	if !pool.PubliclyAdvertisable {
		return false
	}

	locale := strings.TrimSpace(pool.Locale)
	if locale != "" && locale != region {
		return false
	}

	allocationType := strings.TrimSpace(pool.AllocationResourceType)
	if allocationType != "" && !strings.EqualFold(allocationType, "ec2") {
		return false
	}

	return true
}

func (c *Client) ListSubnets() ([]Subnet, error) {
	subnets := []Subnet{}
	nextToken := ""

	for {
		params := url.Values{}
		params.Set("MaxResults", "100")
		if nextToken != "" {
			params.Set("NextToken", nextToken)
		}

		response := describeSubnetsResponse{}
		if err := c.postForm("DescribeSubnets", params, &response); err != nil {
			return nil, err
		}

		for _, subnet := range response.Subnets {
			subnets = append(subnets, Subnet{
				SubnetID:         subnet.SubnetID,
				VpcID:            subnet.VpcID,
				CidrBlock:        subnet.CidrBlock,
				AvailabilityZone: subnet.AvailabilityZone,
				Name:             nameTag(subnet.Tags),
			})
		}

		nextToken = strings.TrimSpace(response.NextToken)
		if nextToken == "" {
			break
		}
	}

	return subnets, nil
}

func (c *Client) ListSecurityGroups() ([]SecurityGroup, error) {
	return c.listSecurityGroups(url.Values{})
}

func (c *Client) ListSecurityGroupsByVPC(vpcID string) ([]SecurityGroup, error) {
	params := url.Values{}
	params.Set("Filter.1.Name", "vpc-id")
	params.Set("Filter.1.Value.1", strings.TrimSpace(vpcID))
	return c.listSecurityGroups(params)
}

func (c *Client) listSecurityGroups(baseParams url.Values) ([]SecurityGroup, error) {
	securityGroups := []SecurityGroup{}
	nextToken := ""

	for {
		params := url.Values{}
		for k, vs := range baseParams {
			for _, v := range vs {
				params.Add(k, v)
			}
		}
		params.Set("MaxResults", "100")
		if nextToken != "" {
			params.Set("NextToken", nextToken)
		}

		response := describeSecurityGroupsResponse{}
		if err := c.postForm("DescribeSecurityGroups", params, &response); err != nil {
			return nil, err
		}

		for _, group := range response.SecurityGroups {
			securityGroups = append(securityGroups, SecurityGroup{
				GroupID:     group.GroupID,
				GroupName:   group.GroupName,
				Description: group.Description,
				VpcID:       group.VpcID,
			})
		}

		nextToken = strings.TrimSpace(response.NextToken)
		if nextToken == "" {
			break
		}
	}

	return securityGroups, nil
}

func (c *Client) DescribeSubnet(subnetID string) (*Subnet, error) {
	trimmed := strings.TrimSpace(subnetID)
	if trimmed == "" {
		return nil, fmt.Errorf("subnet ID is required")
	}

	params := url.Values{}
	params.Set("Filter.1.Name", "subnet-id")
	params.Set("Filter.1.Value.1", trimmed)

	response := describeSubnetsResponse{}
	if err := c.postForm("DescribeSubnets", params, &response); err != nil {
		return nil, err
	}

	if len(response.Subnets) == 0 {
		return nil, fmt.Errorf("subnet not found: %s", trimmed)
	}

	subnet := response.Subnets[0]
	return &Subnet{
		SubnetID:         subnet.SubnetID,
		VpcID:            subnet.VpcID,
		CidrBlock:        subnet.CidrBlock,
		AvailabilityZone: subnet.AvailabilityZone,
		Name:             nameTag(subnet.Tags),
	}, nil
}

func (c *Client) CreateSecurityGroup(groupName, description, vpcID string) (string, error) {
	trimmedName := strings.TrimSpace(groupName)
	trimmedDescription := strings.TrimSpace(description)
	trimmedVpcID := strings.TrimSpace(vpcID)

	if trimmedName == "" {
		return "", fmt.Errorf("security group name is required")
	}
	if trimmedDescription == "" {
		return "", fmt.Errorf("security group description is required")
	}
	if trimmedVpcID == "" {
		return "", fmt.Errorf("VPC ID is required")
	}

	params := url.Values{}
	params.Set("GroupName", trimmedName)
	params.Set("GroupDescription", trimmedDescription)
	params.Set("VpcId", trimmedVpcID)

	response := createSecurityGroupResponse{}
	if err := c.postForm("CreateSecurityGroup", params, &response); err != nil {
		return "", err
	}

	groupID := strings.TrimSpace(response.GroupID)
	if groupID == "" {
		return "", fmt.Errorf("response did not include security group ID")
	}

	return groupID, nil
}

func (c *Client) EnsureSecurityGroupIngressRules(groupID string, rules []SecurityGroupIngressRule) error {
	trimmedGroupID := strings.TrimSpace(groupID)
	if trimmedGroupID == "" {
		return fmt.Errorf("security group ID is required")
	}
	if len(rules) == 0 {
		return nil
	}

	params := url.Values{}
	params.Set("GroupId", trimmedGroupID)

	for index, rule := range rules {
		prefix := fmt.Sprintf("IpPermissions.%d", index+1)
		params.Set(prefix+".IpProtocol", rule.Protocol)
		params.Set(prefix+".FromPort", strconv.Itoa(rule.FromPort))
		params.Set(prefix+".ToPort", strconv.Itoa(rule.ToPort))
		params.Set(prefix+".IpRanges.1.CidrIp", rule.CidrIPv4)
	}

	if err := c.postForm("AuthorizeSecurityGroupIngress", params, &authorizeSecurityGroupIngressResponse{}); err != nil {
		if IsSecurityGroupRuleDuplicate(err) {
			return nil
		}
		return err
	}

	return nil
}

func (c *Client) ListKeyPairs() ([]KeyPair, error) {
	response := describeKeyPairsResponse{}
	if err := c.postForm("DescribeKeyPairs", url.Values{}, &response); err != nil {
		return nil, err
	}

	keyPairs := make([]KeyPair, 0, len(response.KeyPairs))
	for _, keyPair := range response.KeyPairs {
		keyPairs = append(keyPairs, KeyPair{
			KeyName:   keyPair.KeyName,
			KeyPairID: keyPair.KeyPairID,
		})
	}

	return keyPairs, nil
}

func IsInstanceNotFound(err error) bool {
	var awsErr *common.Error
	return errors.As(err, &awsErr) && awsErr.Code == "InvalidInstanceID.NotFound"
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

func (c *Client) ListPublicImages(imageOS string) ([]Image, error) {
	definition, ok := imageOSDefinitions[strings.TrimSpace(imageOS)]
	if !ok {
		return nil, fmt.Errorf("unsupported image OS: %s", imageOS)
	}

	imagesByID := map[string]Image{}
	c.tryLoadPublicImagesFromSSM(imageOS, definition, imagesByID)

	describeAttempted := false
	relevantCount := countRelevantPublicImages(imageOS, imagesByID)
	if !definition.skipDescribeWhenSSMResolved(relevantCount) {
		if err := c.loadPublicImagesFromDescribe(imageOS, definition, imagesByID); err != nil {
			return nil, err
		}
		describeAttempted = true
	}

	images := collectRelevantPublicImages(imageOS, imagesByID)
	if len(images) == 0 && !describeAttempted {
		if err := c.loadPublicImagesFromDescribe(imageOS, definition, imagesByID); err != nil {
			return nil, err
		}
		images = collectRelevantPublicImages(imageOS, imagesByID)
	}

	sort.Slice(images, func(i, j int) bool {
		return images[i].CreationDate > images[j].CreationDate
	})

	if len(images) > definition.publicImageLimit() {
		images = images[:definition.publicImageLimit()]
	}

	return images, nil
}

func (c *Client) tryLoadPublicImagesFromSSM(imageOS string, definition imageOSDefinition, imagesByID map[string]Image) {
	if len(definition.SSMParameterNames) > 0 {
		amiIDs, err := c.listPublicAMIIDsFromSSMNames(definition.SSMParameterNames)
		if err == nil && len(amiIDs) > 0 {
			_ = c.mergePublicImagesByIDs(imageOS, imagesByID, amiIDs)
		}
	}

	if len(definition.SSMParameterPaths) > 0 {
		amiIDs, err := c.listPublicAMIIDsFromSSM(definition.SSMParameterPaths)
		if err == nil && len(amiIDs) > 0 {
			_ = c.mergePublicImagesByIDs(imageOS, imagesByID, amiIDs)
		}
	}
}

func (c *Client) loadPublicImagesFromDescribe(imageOS string, definition imageOSDefinition, imagesByID map[string]Image) error {
	for _, owner := range definition.Owners {
		for _, nameFilter := range definition.NameFilters {
			params := publicImageDescribeParams(owner, definition, nameFilter)

			matches, err := c.describeImagesPaginated(params)
			if err != nil {
				return err
			}

			for _, image := range matches {
				parsed := imageFromXML(image)
				if parsed.ImageID == "" || !isRelevantPublicImage(imageOS, *parsed) {
					continue
				}
				parsed.Region = c.region
				imagesByID[parsed.ImageID] = *parsed
			}
		}
	}

	return nil
}

func countRelevantPublicImages(imageOS string, imagesByID map[string]Image) int {
	count := 0
	for _, image := range imagesByID {
		if isAvailablePublicImage(imageOS, image) {
			count++
		}
	}

	return count
}

func collectRelevantPublicImages(imageOS string, imagesByID map[string]Image) []Image {
	images := make([]Image, 0, len(imagesByID))
	for _, image := range imagesByID {
		if !isAvailablePublicImage(imageOS, image) {
			continue
		}
		images = append(images, image)
	}

	return images
}

func isAvailablePublicImage(imageOS string, image Image) bool {
	if strings.TrimSpace(image.ImageID) == "" {
		return false
	}

	if state := strings.TrimSpace(image.State); state != "" && state != ImageStateAvailable {
		return false
	}

	return isRelevantPublicImage(imageOS, image)
}

func publicImageDescribeParams(owner string, definition imageOSDefinition, nameFilter string) url.Values {
	params := url.Values{}
	params.Set("MaxResults", "100")
	params.Set("Owner.1", owner)

	filterIndex := 1
	addFilter := func(name, value string) {
		params.Set(fmt.Sprintf("Filter.%d.Name", filterIndex), name)
		params.Set(fmt.Sprintf("Filter.%d.Value.1", filterIndex), value)
		filterIndex++
	}

	addFilter("state", ImageStateAvailable)
	addFilter("name", nameFilter)
	addFilter("root-device-type", "ebs")
	addFilter("virtualization-type", "hvm")
	if strings.TrimSpace(definition.Platform) != "" {
		addFilter("platform", definition.Platform)
	}

	return params
}

func (c *Client) describeImagesByIDs(imageIDs []string) ([]xmlImage, error) {
	if len(imageIDs) == 0 {
		return nil, nil
	}

	params := url.Values{}
	params.Set("MaxResults", "100")
	for index, imageID := range imageIDs {
		params.Set(fmt.Sprintf("ImageId.%d", index+1), imageID)
	}

	return c.describeImagesPaginated(params)
}

func (c *Client) mergePublicImagesByIDs(imageOS string, imagesByID map[string]Image, amiIDs []string) error {
	matches, err := c.describeImagesByIDs(amiIDs)
	if err != nil {
		return err
	}

	for _, image := range matches {
		parsed := imageFromXML(image)
		if !isAvailablePublicImage(imageOS, *parsed) {
			continue
		}
		parsed.Region = c.region
		imagesByID[parsed.ImageID] = *parsed
	}

	return nil
}

func (c *Client) listPublicAMIIDsFromSSMNames(names []string) ([]string, error) {
	amiIDs := []string{}
	seen := map[string]struct{}{}

	const batchSize = 10
	for start := 0; start < len(names); start += batchSize {
		end := start + batchSize
		if end > len(names) {
			end = len(names)
		}

		batchNames := make([]string, 0, end-start)
		for _, name := range names[start:end] {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			batchNames = append(batchNames, name)
		}
		if len(batchNames) == 0 {
			continue
		}

		response := getParametersResponse{}
		if err := c.postSSMJSON("GetParameters", map[string]any{
			"Names":          batchNames,
			"WithDecryption": false,
		}, &response); err != nil {
			return nil, err
		}

		for _, parameter := range response.Parameters {
			amiID := strings.TrimSpace(parameter.Value)
			if !strings.HasPrefix(amiID, "ami-") {
				continue
			}
			if _, ok := seen[amiID]; ok {
				continue
			}
			seen[amiID] = struct{}{}
			amiIDs = append(amiIDs, amiID)
		}
	}

	return amiIDs, nil
}

func (c *Client) listPublicAMIIDsFromSSM(paths []string) ([]string, error) {
	amiIDs := []string{}
	seen := map[string]struct{}{}

	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		nextToken := ""
		for {
			payload := map[string]any{
				"Path":           path,
				"Recursive":      true,
				"WithDecryption": false,
				"MaxResults":     10,
			}
			if nextToken != "" {
				payload["NextToken"] = nextToken
			}

			response := getParametersByPathResponse{}
			if err := c.postSSMJSON("GetParametersByPath", payload, &response); err != nil {
				return nil, err
			}

			for _, parameter := range response.Parameters {
				name := strings.TrimSpace(parameter.Name)
				if name == "" || strings.Contains(name, "arm64") {
					continue
				}

				amiID := strings.TrimSpace(parameter.Value)
				if !strings.HasPrefix(amiID, "ami-") {
					continue
				}

				if _, ok := seen[amiID]; ok {
					continue
				}

				seen[amiID] = struct{}{}
				amiIDs = append(amiIDs, amiID)
			}

			nextToken = strings.TrimSpace(response.NextToken)
			if nextToken == "" {
				break
			}
		}
	}

	return amiIDs, nil
}

func (c *Client) describeImagesPaginated(params url.Values) ([]xmlImage, error) {
	images := []xmlImage{}
	nextToken := ""

	for {
		pageParams := url.Values{}
		for key, values := range params {
			for _, value := range values {
				pageParams.Add(key, value)
			}
		}
		if nextToken != "" {
			pageParams.Set("NextToken", nextToken)
		}

		response := describeImagesResponse{}
		if err := c.postForm("DescribeImages", pageParams, &response); err != nil {
			return nil, err
		}

		images = append(images, response.Images...)
		nextToken = strings.TrimSpace(response.NextToken)
		if nextToken == "" {
			break
		}
	}

	return images, nil
}

func (c *Client) ListInstanceTypes() ([]InstanceTypeInfo, error) {
	instanceTypes := []InstanceTypeInfo{}
	nextToken := ""

	for {
		params := url.Values{}
		params.Set("MaxResults", "100")
		params.Set("Filter.1.Name", "current-generation")
		params.Set("Filter.1.Value.1", "true")
		if nextToken != "" {
			params.Set("NextToken", nextToken)
		}

		response := describeInstanceTypesResponse{}
		if err := c.postForm("DescribeInstanceTypes", params, &response); err != nil {
			return nil, err
		}

		for _, item := range response.InstanceTypes {
			instanceTypes = append(instanceTypes, InstanceTypeInfo{
				InstanceType: item.InstanceType,
				VCPUs:        item.VCPUInfo.vcpus(),
				MemoryMiB:    item.MemoryInfo.memoryMiB(),
			})
		}

		nextToken = strings.TrimSpace(response.NextToken)
		if nextToken == "" {
			break
		}
	}

	sort.Slice(instanceTypes, func(i, j int) bool {
		return instanceTypes[i].InstanceType < instanceTypes[j].InstanceType
	})

	return instanceTypes, nil
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
	return c.postSignedForm(ec2ServiceName, ec2APIVersion, action, params, out)
}

func (c *Client) postSSMJSON(action string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal SSM request: %w", err)
	}

	endpoint := fmt.Sprintf("https://%s.%s.amazonaws.com/", ssmServiceName, c.region)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build SSM request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", ssmTargetPrefix+action)
	if err := c.signRequest(req, body, ssmServiceName); err != nil {
		return err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("SSM request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read SSM response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := common.ParseError(responseBody); awsErr != nil {
			return awsErr
		}
		return fmt.Errorf("SSM API request failed with %d: %s", res.StatusCode, string(responseBody))
	}

	if out == nil {
		return nil
	}

	if err := json.Unmarshal(responseBody, out); err != nil {
		return fmt.Errorf("failed to decode SSM response: %w", err)
	}

	return nil
}

func (c *Client) postSignedForm(service, version, action string, params url.Values, out any) error {
	if params == nil {
		params = url.Values{}
	}

	params.Set("Action", action)
	params.Set("Version", version)

	body := []byte(params.Encode())
	endpoint := fmt.Sprintf("https://%s.%s.amazonaws.com/", service, c.region)
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	if err := c.signRequest(req, body, service); err != nil {
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
		return fmt.Errorf("%s API request failed with %d: %s", strings.ToUpper(service), res.StatusCode, string(responseBody))
	}

	if out == nil {
		return nil
	}

	if err := xml.Unmarshal(responseBody, out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

func (c *Client) signRequest(req *http.Request, payload []byte, service string) error {
	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	return c.signer.SignHTTP(context.Background(), *c.credentials, req, payloadHash, service, c.region, time.Now())
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
	RequestID    string           `xml:"requestId"`
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
	InstanceID       string   `xml:"instanceId"`
	InstanceType     string   `xml:"instanceType"`
	ImageID          string   `xml:"imageId"`
	KeyName          string   `xml:"keyName"`
	LaunchTime       string   `xml:"launchTime"`
	PrivateDNSName   string   `xml:"privateDnsName"`
	PrivateIPAddress string   `xml:"privateIpAddress"`
	PublicDNSName    string   `xml:"dnsName"`
	PublicIPAddress  string   `xml:"ipAddress"`
	SubnetID         string   `xml:"subnetId"`
	VpcID            string   `xml:"vpcId"`
	State            xmlState `xml:"instanceState"`
	CurrentState     xmlState `xml:"currentState"`
	Tags             []xmlTag `xml:"tagSet>item"`
}

func (instance xmlInstance) stateName() string {
	if instance.State.Name != "" {
		return instance.State.Name
	}

	return instance.CurrentState.Name
}

type xmlState struct {
	Name string `xml:"name"`
}

type xmlTag struct {
	Key   string `xml:"key"`
	Value string `xml:"value"`
}

type runInstancesResponse struct {
	RequestID string        `xml:"requestId"`
	Instances []xmlInstance `xml:"instancesSet>item"`
}

type terminateInstancesResponse struct {
	RequestID string        `xml:"requestId"`
	Instances []xmlInstance `xml:"instancesSet>item"`
}

type stopInstancesResponse struct {
	RequestID string                   `xml:"requestId"`
	Instances []xmlInstanceStateChange `xml:"instancesSet>item"`
}

type startInstancesResponse struct {
	RequestID string                   `xml:"requestId"`
	Instances []xmlInstanceStateChange `xml:"instancesSet>item"`
}

type allocateAddressResponse struct {
	RequestID    string `xml:"requestId"`
	PublicIP     string `xml:"publicIp"`
	Domain       string `xml:"domain"`
	AllocationID string `xml:"allocationId"`
}

type releaseAddressResponse struct {
	RequestID string `xml:"requestId"`
	Return    bool   `xml:"return"`
}

type associateAddressResponse struct {
	RequestID     string `xml:"requestId"`
	AssociationID string `xml:"associationId"`
}

type disassociateAddressResponse struct {
	RequestID string `xml:"requestId"`
	Return    bool   `xml:"return"`
}

type describeAddressesResponse struct {
	RequestID string       `xml:"requestId"`
	Addresses []xmlAddress `xml:"addressesSet>item"`
}

type xmlAddress struct {
	PublicIP      string `xml:"publicIp"`
	AllocationID  string `xml:"allocationId"`
	AssociationID string `xml:"associationId"`
	InstanceID    string `xml:"instanceId"`
	Domain        string `xml:"domain"`
}

type describePublicIpv4PoolsResponse struct {
	RequestID string              `xml:"requestId"`
	Pools     []xmlPublicIpv4Pool `xml:"publicIpv4PoolSet>item"`
	NextToken string              `xml:"nextToken"`
}

type xmlPublicIpv4Pool struct {
	PoolID      string `xml:"poolId"`
	Description string `xml:"description"`
}

type describeCoipPoolsResponse struct {
	RequestID string        `xml:"requestId"`
	Pools     []xmlCoipPool `xml:"coipPoolSet>item"`
	NextToken string        `xml:"nextToken"`
}

type xmlCoipPool struct {
	PoolID                   string `xml:"poolId"`
	LocalGatewayRouteTableID string `xml:"localGatewayRouteTableId"`
}

type describeIpamPoolsResponse struct {
	RequestID string        `xml:"requestId"`
	Pools     []xmlIpamPool `xml:"ipamPoolSet>item"`
	NextToken string        `xml:"nextToken"`
}

type xmlIpamPool struct {
	IpamPoolID             string `xml:"ipamPoolId"`
	Description            string `xml:"description"`
	AddressFamily          string `xml:"addressFamily"`
	PubliclyAdvertisable   bool   `xml:"publiclyAdvertisable"`
	Locale                 string `xml:"locale"`
	AllocationResourceType string `xml:"allocationResourceType"`
}

type xmlInstanceStateChange struct {
	InstanceID   string `xml:"instanceId"`
	CurrentState struct {
		Name string `xml:"name"`
	} `xml:"currentState"`
	PreviousState struct {
		Name string `xml:"name"`
	} `xml:"previousState"`
}

type describeSubnetsResponse struct {
	Subnets   []xmlSubnet `xml:"subnetSet>item"`
	NextToken string      `xml:"nextToken"`
}

type xmlSubnet struct {
	SubnetID         string   `xml:"subnetId"`
	VpcID            string   `xml:"vpcId"`
	CidrBlock        string   `xml:"cidrBlock"`
	AvailabilityZone string   `xml:"availabilityZone"`
	Tags             []xmlTag `xml:"tagSet>item"`
}

type describeSecurityGroupsResponse struct {
	SecurityGroups []xmlSecurityGroup `xml:"securityGroupInfo>item"`
	NextToken      string             `xml:"nextToken"`
}

type xmlSecurityGroup struct {
	GroupID     string `xml:"groupId"`
	GroupName   string `xml:"groupName"`
	Description string `xml:"groupDescription"`
	VpcID       string `xml:"vpcId"`
}

type createSecurityGroupResponse struct {
	RequestID string `xml:"requestId"`
	GroupID   string `xml:"groupId"`
}

type authorizeSecurityGroupIngressResponse struct {
	RequestID string `xml:"requestId"`
	Return    bool   `xml:"return"`
}

type modifyInstanceAttributeResponse struct {
	RequestID string `xml:"requestId"`
	Return    bool   `xml:"return"`
}

type getMetricStatisticsResponse struct {
	Datapoints []xmlCloudWatchDatapoint `xml:"GetMetricStatisticsResult>Datapoints>member"`
}

type xmlCloudWatchDatapoint struct {
	Timestamp string  `xml:"Timestamp"`
	Average   float64 `xml:"Average"`
	Sum       float64 `xml:"Sum"`
}

type describeKeyPairsResponse struct {
	KeyPairs []xmlKeyPair `xml:"keySet>item"`
}

type xmlKeyPair struct {
	KeyName   string `xml:"keyName"`
	KeyPairID string `xml:"keyPairId"`
}

type describeInstanceTypesResponse struct {
	InstanceTypes []xmlInstanceType `xml:"instanceTypeSet>item"`
	NextToken     string            `xml:"nextToken"`
}

type getParametersByPathResponse struct {
	Parameters []ssmParameter `json:"Parameters"`
	NextToken  string         `json:"NextToken"`
}

type getParametersResponse struct {
	Parameters []ssmParameter `json:"Parameters"`
}

type ssmParameter struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

type xmlInstanceType struct {
	InstanceType string        `xml:"instanceType"`
	VCPUInfo     xmlVCPUInfo   `xml:"vCpuInfo"`
	MemoryInfo   xmlMemoryInfo `xml:"memoryInfo"`
}

type xmlVCPUInfo struct {
	DefaultVCPUs          int `xml:"defaultVCpus"`
	DefaultVcpus          int `xml:"defaultVcpus"`
	DefaultCores          int `xml:"defaultCores"`
	DefaultThreadsPerCore int `xml:"defaultThreadsPerCore"`
}

func (info xmlVCPUInfo) vcpus() int {
	if info.DefaultVCPUs > 0 {
		return info.DefaultVCPUs
	}

	if info.DefaultVcpus > 0 {
		return info.DefaultVcpus
	}

	if info.DefaultCores > 0 && info.DefaultThreadsPerCore > 0 {
		return info.DefaultCores * info.DefaultThreadsPerCore
	}

	return info.DefaultCores
}

type xmlMemoryInfo struct {
	SizeInMiB int `xml:"sizeInMiB"`
	SizeInMib int `xml:"sizeInMib"`
}

func (info xmlMemoryInfo) memoryMiB() int {
	if info.SizeInMiB > 0 {
		return info.SizeInMiB
	}

	return info.SizeInMib
}

func instanceDetailsFromXML(instance xmlInstance, region, requestID string) *InstanceDetails {
	return &InstanceDetails{
		RequestID:        requestID,
		InstanceID:       instance.InstanceID,
		InstanceType:     instance.InstanceType,
		ImageID:          instance.ImageID,
		State:            instance.stateName(),
		Name:             nameTag(instance.Tags),
		KeyName:          instance.KeyName,
		LaunchTime:       instance.LaunchTime,
		PrivateIPAddress: instance.PrivateIPAddress,
		PublicIPAddress:  instance.PublicIPAddress,
		PrivateDNSName:   instance.PrivateDNSName,
		PublicDNSName:    instance.PublicDNSName,
		SubnetID:         instance.SubnetID,
		VpcID:            instance.VpcID,
		Region:           region,
	}
}

func instanceDetailsToMap(instance *InstanceDetails) map[string]any {
	if instance == nil {
		return map[string]any{}
	}

	return map[string]any{
		"instanceId":       instance.InstanceID,
		"instanceType":     instance.InstanceType,
		"imageId":          instance.ImageID,
		"state":            instance.State,
		"name":             instance.Name,
		"keyName":          instance.KeyName,
		"launchTime":       instance.LaunchTime,
		"privateIpAddress": instance.PrivateIPAddress,
		"publicIpAddress":  instance.PublicIPAddress,
		"privateDnsName":   instance.PrivateDNSName,
		"publicDnsName":    instance.PublicDNSName,
		"subnetId":         instance.SubnetID,
		"vpcId":            instance.VpcID,
		"region":           instance.Region,
	}
}

type imageOSDefinition struct {
	Label                     string
	Owners                    []string
	NameFilters               []string
	Platform                  string
	SSMParameterNames         []string
	SSMParameterPaths         []string
	MaxPublicImages           int
	SkipDescribeIfSSMResolved bool
}

func (definition imageOSDefinition) publicImageLimit() int {
	if definition.MaxPublicImages > 0 {
		return definition.MaxPublicImages
	}

	return maxPublicImagesPerOS
}

func (definition imageOSDefinition) skipDescribeWhenSSMResolved(resolvedImages int) bool {
	if !definition.SkipDescribeIfSSMResolved {
		return false
	}

	return resolvedImages > 0
}

func isRelevantPublicImage(imageOS string, image Image) bool {
	name := strings.ToLower(strings.TrimSpace(image.Name))
	architecture := strings.TrimSpace(image.Architecture)
	if architecture != "" && architecture != "x86_64" {
		return false
	}

	switch strings.TrimSpace(imageOS) {
	case "ubuntu":
		if strings.Contains(name, "minimal") || strings.Contains(name, "daily") || strings.Contains(name, "/pro-") || strings.Contains(name, "-eks-") {
			return false
		}
		if !strings.Contains(name, "-server-") && !strings.Contains(name, "server") {
			return false
		}
		for _, release := range []string{"resolute", "noble", "jammy", "26.04", "24.04", "22.04"} {
			if strings.Contains(name, release) {
				return true
			}
		}
		return false
	case "debian":
		for _, release := range []string{"debian-12", "debian-11", "bookworm", "bullseye"} {
			if strings.Contains(name, release) {
				return true
			}
		}
		return false
	default:
		return true
	}
}

var imageOSDefinitions = map[string]imageOSDefinition{
	"amazon_linux": {
		Label:  "Amazon Linux",
		Owners: []string{"amazon"},
		NameFilters: []string{
			"al2023-ami-*-x86_64",
			"al2023-ami-minimal-*-x86_64",
			"amzn2-ami-hvm-*-x86_64-gp2",
			"amzn2-ami-hvm-*-x86_64",
		},
		SSMParameterPaths: []string{
			"/aws/service/ami-amazon-linux-latest/",
		},
	},
	"ubuntu": {
		Label:  "Ubuntu",
		Owners: []string{"099720109477"},
		SSMParameterNames: []string{
			"/aws/service/canonical/ubuntu/server/resolute/stable/current/amd64/hvm/ebs-gp3/ami-id",
			"/aws/service/canonical/ubuntu/server/26.04/stable/current/amd64/hvm/ebs-gp3/ami-id",
			"/aws/service/canonical/ubuntu/server/noble/stable/current/amd64/hvm/ebs-gp3/ami-id",
			"/aws/service/canonical/ubuntu/server/jammy/stable/current/amd64/hvm/ebs-gp3/ami-id",
		},
		NameFilters: []string{
			"ubuntu/images/*/ubuntu-resolute-*-amd64-server-*",
			"ubuntu/images/*/ubuntu-noble-*-amd64-server-*",
			"ubuntu/images/*/ubuntu-jammy-*-amd64-server-*",
		},
		MaxPublicImages:           defaultUbuntuImages,
		SkipDescribeIfSSMResolved: true,
	},
	"red_hat": {
		Label:  "Red Hat",
		Owners: []string{"309956199498"},
		NameFilters: []string{
			"RHEL-*-x86_64-*",
			"RHEL-*",
		},
	},
	"suse": {
		Label:  "SUSE Linux",
		Owners: []string{"013124491280"},
		NameFilters: []string{
			"suse-sles-*-x86_64*",
			"suse-sles-*",
			"sles-*-x86_64*",
			"sles-*",
		},
	},
	"debian": {
		Label:  "Debian",
		Owners: []string{"136693071363"},
		SSMParameterNames: []string{
			"/aws/service/debian/release/12/latest/amd64",
			"/aws/service/debian/release/bookworm/latest/amd64",
			"/aws/service/debian/release/11/latest/amd64",
			"/aws/service/debian/release/bullseye/latest/amd64",
		},
		NameFilters: []string{
			"debian-12-amd64-*",
			"debian-11-amd64-*",
		},
		MaxPublicImages:           defaultDebianImages,
		SkipDescribeIfSSMResolved: true,
	},
}

func ListImageOperatingSystems(_ core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	keys := make([]string, 0, len(imageOSDefinitions))
	for key := range imageOSDefinitions {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	resources := make([]core.IntegrationResource, 0, len(keys))
	for _, key := range keys {
		definition := imageOSDefinitions[key]
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: definition.Label,
			ID:   key,
		})
	}

	return resources, nil
}

func imageOSLabel(imageOS string) string {
	definition, ok := imageOSDefinitions[strings.TrimSpace(imageOS)]
	if !ok {
		return strings.TrimSpace(imageOS)
	}

	return definition.Label
}

func requireImageOS(value string) (string, error) {
	imageOS := strings.TrimSpace(value)
	if imageOS == "" {
		return "", fmt.Errorf("operating system is required")
	}

	if _, ok := imageOSDefinitions[imageOS]; !ok {
		return "", fmt.Errorf("unsupported operating system: %s", imageOS)
	}

	return imageOS, nil
}

func publicImageResourceName(image Image) string {
	name := strings.TrimSpace(image.Name)
	if name == "" {
		return image.ImageID
	}

	architecture := strings.TrimSpace(image.Architecture)
	if architecture == "" {
		return fmt.Sprintf("%s (%s)", name, image.ImageID)
	}

	return fmt.Sprintf("%s (%s, %s)", name, image.ImageID, architecture)
}

// ── CloudWatch alarm types and methods ──────────────────────────────────────

type PutMetricAlarmInput struct {
	AlarmName          string
	AlarmDescription   string
	InstanceID         string
	MetricName         string
	Statistic          string
	Period             int
	EvaluationPeriods  int
	Threshold          float64
	ComparisonOperator string
	TreatMissingData   string
	// AlarmActions is a list of ARNs to invoke when the alarm enters ALARM state.
	// Entries may be SNS topic ARNs or EC2 automation ARNs
	// (arn:aws:automate:<region>:ec2:recover|reboot|stop|terminate).
	AlarmActions []string
}

type MetricAlarm struct {
	AlarmName          string           `json:"alarmName" mapstructure:"alarmName"`
	AlarmArn           string           `json:"alarmArn" mapstructure:"alarmArn"`
	AlarmDescription   string           `json:"alarmDescription" mapstructure:"alarmDescription"`
	Namespace          string           `json:"namespace" mapstructure:"namespace"`
	MetricName         string           `json:"metricName" mapstructure:"metricName"`
	Statistic          string           `json:"statistic" mapstructure:"statistic"`
	Period             int              `json:"period" mapstructure:"period"`
	EvaluationPeriods  int              `json:"evaluationPeriods" mapstructure:"evaluationPeriods"`
	Threshold          float64          `json:"threshold" mapstructure:"threshold"`
	ComparisonOperator string           `json:"comparisonOperator" mapstructure:"comparisonOperator"`
	StateValue         string           `json:"stateValue" mapstructure:"stateValue"`
	StateReason        string           `json:"stateReason" mapstructure:"stateReason"`
	TreatMissingData   string           `json:"treatMissingData" mapstructure:"treatMissingData"`
	Dimensions         []AlarmDimension `json:"dimensions" mapstructure:"dimensions"`
	Region             string           `json:"region" mapstructure:"region"`
}

type AlarmDimension struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

type describeAlarmsResponse struct {
	XMLName xml.Name             `xml:"DescribeAlarmsResponse"`
	Result  describeAlarmsResult `xml:"DescribeAlarmsResult"`
}

type describeAlarmsResult struct {
	MetricAlarms []xmlMetricAlarm `xml:"MetricAlarms>member"`
	NextToken    string           `xml:"NextToken"`
}

type xmlMetricAlarm struct {
	AlarmName          string              `xml:"AlarmName"`
	AlarmArn           string              `xml:"AlarmArn"`
	AlarmDescription   string              `xml:"AlarmDescription"`
	Namespace          string              `xml:"Namespace"`
	MetricName         string              `xml:"MetricName"`
	Statistic          string              `xml:"Statistic"`
	Period             int                 `xml:"Period"`
	EvaluationPeriods  int                 `xml:"EvaluationPeriods"`
	Threshold          float64             `xml:"Threshold"`
	ComparisonOperator string              `xml:"ComparisonOperator"`
	StateValue         string              `xml:"StateValue"`
	StateReason        string              `xml:"StateReason"`
	TreatMissingData   string              `xml:"TreatMissingData"`
	Dimensions         []xmlAlarmDimension `xml:"Dimensions>member"`
}

type xmlAlarmDimension struct {
	Name  string `xml:"Name"`
	Value string `xml:"Value"`
}

func (c *Client) PutMetricAlarm(input PutMetricAlarmInput) error {
	params := url.Values{}
	params.Set("AlarmName", strings.TrimSpace(input.AlarmName))
	params.Set("Namespace", alarmNamespaceEC2)
	params.Set("MetricName", strings.TrimSpace(input.MetricName))
	params.Set("Dimensions.member.1.Name", "InstanceId")
	params.Set("Dimensions.member.1.Value", strings.TrimSpace(input.InstanceID))

	statistic := strings.TrimSpace(input.Statistic)
	if statistic == "" {
		statistic = "Average"
	}
	params.Set("Statistic", statistic)
	params.Set("ComparisonOperator", strings.TrimSpace(input.ComparisonOperator))
	params.Set("Threshold", strconv.FormatFloat(input.Threshold, 'f', -1, 64))

	period := input.Period
	if period <= 0 {
		period = 300
	}
	params.Set("Period", strconv.Itoa(period))

	evaluationPeriods := input.EvaluationPeriods
	if evaluationPeriods <= 0 {
		evaluationPeriods = 1
	}
	params.Set("EvaluationPeriods", strconv.Itoa(evaluationPeriods))

	description := strings.TrimSpace(input.AlarmDescription)
	if description != "" {
		params.Set("AlarmDescription", description)
	}

	treatMissing := strings.TrimSpace(input.TreatMissingData)
	if treatMissing != "" {
		params.Set("TreatMissingData", treatMissing)
	}

	for i, arn := range input.AlarmActions {
		arn = strings.TrimSpace(arn)
		if arn != "" {
			params.Set(fmt.Sprintf("AlarmActions.member.%d", i+1), arn)
		}
	}

	return c.postSignedForm(monitoringServiceName, monitoringAPIVersion, "PutMetricAlarm", params, nil)
}

func (c *Client) DescribeAlarm(alarmName string) (*MetricAlarm, error) {
	params := url.Values{}
	params.Set("AlarmNames.member.1", strings.TrimSpace(alarmName))

	response := describeAlarmsResponse{}
	if err := c.postSignedForm(monitoringServiceName, monitoringAPIVersion, "DescribeAlarms", params, &response); err != nil {
		return nil, err
	}

	if len(response.Result.MetricAlarms) == 0 {
		return nil, fmt.Errorf("alarm not found: %s", alarmName)
	}

	return alarmFromXML(response.Result.MetricAlarms[0], c.region), nil
}

func (c *Client) ListAlarms() ([]MetricAlarm, error) {
	return c.listAlarms(url.Values{})
}

func (c *Client) ListAlarmsForInstance(instanceID string) ([]MetricAlarm, error) {
	all, err := c.listAlarms(url.Values{})
	if err != nil {
		return nil, err
	}

	target := strings.TrimSpace(instanceID)
	filtered := make([]MetricAlarm, 0, len(all))
	for _, alarm := range all {
		for _, dim := range alarm.Dimensions {
			if dim.Name == "InstanceId" && dim.Value == target {
				filtered = append(filtered, alarm)
				break
			}
		}
	}

	return filtered, nil
}

func (c *Client) listAlarms(base url.Values) ([]MetricAlarm, error) {
	alarms := []MetricAlarm{}
	nextToken := ""

	for {
		params := url.Values{}
		for k, vs := range base {
			params[k] = vs
		}
		params.Set("MaxRecords", "100")

		if nextToken != "" {
			params.Set("NextToken", nextToken)
		}

		response := describeAlarmsResponse{}
		if err := c.postSignedForm(monitoringServiceName, monitoringAPIVersion, "DescribeAlarms", params, &response); err != nil {
			return nil, err
		}

		for _, xmlAlarm := range response.Result.MetricAlarms {
			alarms = append(alarms, *alarmFromXML(xmlAlarm, c.region))
		}

		nextToken = strings.TrimSpace(response.Result.NextToken)
		if nextToken == "" {
			break
		}
	}

	return alarms, nil
}

func alarmFromXML(x xmlMetricAlarm, region string) *MetricAlarm {
	dimensions := make([]AlarmDimension, 0, len(x.Dimensions))
	for _, d := range x.Dimensions {
		dimensions = append(dimensions, AlarmDimension{Name: d.Name, Value: d.Value})
	}

	return &MetricAlarm{
		AlarmName:          x.AlarmName,
		AlarmArn:           x.AlarmArn,
		AlarmDescription:   x.AlarmDescription,
		Namespace:          x.Namespace,
		MetricName:         x.MetricName,
		Statistic:          x.Statistic,
		Period:             x.Period,
		EvaluationPeriods:  x.EvaluationPeriods,
		Threshold:          x.Threshold,
		ComparisonOperator: x.ComparisonOperator,
		StateValue:         x.StateValue,
		StateReason:        x.StateReason,
		TreatMissingData:   x.TreatMissingData,
		Dimensions:         dimensions,
		Region:             region,
	}
}

func alarmToMap(alarm *MetricAlarm) map[string]any {
	if alarm == nil {
		return map[string]any{}
	}

	dims := make([]map[string]any, 0, len(alarm.Dimensions))
	for _, d := range alarm.Dimensions {
		dims = append(dims, map[string]any{"name": d.Name, "value": d.Value})
	}

	return map[string]any{
		"alarmName":          alarm.AlarmName,
		"alarmArn":           alarm.AlarmArn,
		"alarmDescription":   alarm.AlarmDescription,
		"namespace":          alarm.Namespace,
		"metricName":         alarm.MetricName,
		"statistic":          alarm.Statistic,
		"period":             alarm.Period,
		"evaluationPeriods":  alarm.EvaluationPeriods,
		"threshold":          alarm.Threshold,
		"comparisonOperator": alarm.ComparisonOperator,
		"stateValue":         alarm.StateValue,
		"stateReason":        alarm.StateReason,
		"treatMissingData":   alarm.TreatMissingData,
		"dimensions":         dims,
		"region":             alarm.Region,
	}
}

type LoadBalancer struct {
	LoadBalancerARN string `json:"loadBalancerArn" mapstructure:"loadBalancerArn"`
	Name            string `json:"name" mapstructure:"name"`
	DNSName         string `json:"dnsName" mapstructure:"dnsName"`
	Scheme          string `json:"scheme" mapstructure:"scheme"`
	Type            string `json:"type" mapstructure:"type"`
	State           string `json:"state" mapstructure:"state"`
	VpcID           string `json:"vpcId" mapstructure:"vpcId"`
	Region          string `json:"region" mapstructure:"region"`
}

type CreateLoadBalancerInput struct {
	Name           string
	Type           string
	Scheme         string
	IPAddressType  string
	SubnetIDs      []string
	SecurityGroups []string
}

type CreateLoadBalancerOutput struct {
	RequestID       string `json:"requestId" mapstructure:"requestId"`
	LoadBalancerARN string `json:"loadBalancerArn" mapstructure:"loadBalancerArn"`
	Name            string `json:"name" mapstructure:"name"`
	DNSName         string `json:"dnsName" mapstructure:"dnsName"`
	Scheme          string `json:"scheme" mapstructure:"scheme"`
	Type            string `json:"type" mapstructure:"type"`
	State           string `json:"state" mapstructure:"state"`
	VpcID           string `json:"vpcId" mapstructure:"vpcId"`
	Region          string `json:"region" mapstructure:"region"`
}

type DeleteLoadBalancerOutput struct {
	RequestID       string `json:"requestId" mapstructure:"requestId"`
	LoadBalancerARN string `json:"loadBalancerArn" mapstructure:"loadBalancerArn"`
	Region          string `json:"region" mapstructure:"region"`
}

// XML parse helpers for ELBv2 responses

type xmlLoadBalancerState struct {
	Code string `xml:"Code"`
}

type xmlLoadBalancerMember struct {
	LoadBalancerARN  string               `xml:"LoadBalancerArn"`
	LoadBalancerName string               `xml:"LoadBalancerName"`
	DNSName          string               `xml:"DNSName"`
	Scheme           string               `xml:"Scheme"`
	Type             string               `xml:"Type"`
	State            xmlLoadBalancerState `xml:"State"`
	VpcID            string               `xml:"VpcId"`
}

type xmlLoadBalancerMembers struct {
	Members []xmlLoadBalancerMember `xml:"member"`
}

type xmlCreateLoadBalancerResult struct {
	LoadBalancers xmlLoadBalancerMembers `xml:"LoadBalancers"`
}

type xmlCreateLoadBalancerResponse struct {
	RequestID string                      `xml:"ResponseMetadata>RequestId"`
	Result    xmlCreateLoadBalancerResult `xml:"CreateLoadBalancerResult"`
}

type xmlDescribeLoadBalancersResult struct {
	LoadBalancers xmlLoadBalancerMembers `xml:"LoadBalancers"`
	NextMarker    string                 `xml:"NextMarker"`
}

type xmlDescribeLoadBalancersResponse struct {
	RequestID string                         `xml:"ResponseMetadata>RequestId"`
	Result    xmlDescribeLoadBalancersResult `xml:"DescribeLoadBalancersResult"`
}

type xmlDeleteLoadBalancerResponse struct {
	RequestID string `xml:"ResponseMetadata>RequestId"`
}

func parseELBError(body []byte) *common.Error {
	var errResp struct {
		Error struct {
			Code    string `xml:"Code"`
			Message string `xml:"Message"`
		} `xml:"Error"`
		RequestID string `xml:"RequestId"`
	}

	if err := xml.Unmarshal(body, &errResp); err == nil {
		if strings.TrimSpace(errResp.Error.Code) != "" || strings.TrimSpace(errResp.Error.Message) != "" {
			return &common.Error{
				Code:    strings.TrimSpace(errResp.Error.Code),
				Message: strings.TrimSpace(errResp.Error.Message),
			}
		}
	}

	return nil
}

func (c *Client) postELBForm(action string, params url.Values, out any) error {
	if params == nil {
		params = url.Values{}
	}

	params.Set("Action", action)
	params.Set("Version", elbAPIVersion)

	body := []byte(params.Encode())
	endpoint := fmt.Sprintf("https://%s.%s.amazonaws.com/", elbServiceName, c.region)
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to build ELB request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	if err := c.signRequest(req, body, elbServiceName); err != nil {
		return err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("ELB request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read ELB response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := parseELBError(responseBody); awsErr != nil {
			return awsErr
		}
		return fmt.Errorf("ELB API request failed with %d: %s", res.StatusCode, string(responseBody))
	}

	if out == nil {
		return nil
	}

	if err := xml.Unmarshal(responseBody, out); err != nil {
		return fmt.Errorf("failed to decode ELB response: %w", err)
	}

	return nil
}

func (c *Client) CreateLoadBalancer(input CreateLoadBalancerInput) (*CreateLoadBalancerOutput, error) {
	params := url.Values{}
	params.Set("Name", strings.TrimSpace(input.Name))
	params.Set("Type", strings.TrimSpace(input.Type))
	if strings.TrimSpace(input.Type) != LoadBalancerTypeGateway {
		params.Set("Scheme", strings.TrimSpace(input.Scheme))
	}

	if ip := strings.TrimSpace(input.IPAddressType); ip != "" {
		params.Set("IpAddressType", ip)
	}

	subnetIndex := 1
	for _, subnetID := range input.SubnetIDs {
		trimmed := strings.TrimSpace(subnetID)
		if trimmed == "" {
			continue
		}
		params.Set(fmt.Sprintf("Subnets.member.%d", subnetIndex), trimmed)
		subnetIndex++
	}

	sgIndex := 1
	for _, sgID := range input.SecurityGroups {
		trimmed := strings.TrimSpace(sgID)
		if trimmed == "" {
			continue
		}
		params.Set(fmt.Sprintf("SecurityGroups.member.%d", sgIndex), trimmed)
		sgIndex++
	}

	response := xmlCreateLoadBalancerResponse{}
	if err := c.postELBForm("CreateLoadBalancer", params, &response); err != nil {
		return nil, err
	}

	members := response.Result.LoadBalancers.Members
	if len(members) == 0 {
		return nil, fmt.Errorf("response did not include load balancer details")
	}

	lb := members[0]
	if strings.TrimSpace(lb.LoadBalancerARN) == "" {
		return nil, fmt.Errorf("response did not include load balancer ARN")
	}

	return &CreateLoadBalancerOutput{
		RequestID:       response.RequestID,
		LoadBalancerARN: lb.LoadBalancerARN,
		Name:            lb.LoadBalancerName,
		DNSName:         lb.DNSName,
		Scheme:          lb.Scheme,
		Type:            lb.Type,
		State:           lb.State.Code,
		VpcID:           lb.VpcID,
		Region:          c.region,
	}, nil
}

func (c *Client) DeleteLoadBalancer(loadBalancerARN string) (*DeleteLoadBalancerOutput, error) {
	params := url.Values{}
	params.Set("LoadBalancerArn", strings.TrimSpace(loadBalancerARN))

	response := xmlDeleteLoadBalancerResponse{}
	if err := c.postELBForm("DeleteLoadBalancer", params, &response); err != nil {
		return nil, err
	}

	return &DeleteLoadBalancerOutput{
		RequestID:       response.RequestID,
		LoadBalancerARN: strings.TrimSpace(loadBalancerARN),
		Region:          c.region,
	}, nil
}

func (c *Client) DescribeLoadBalancer(loadBalancerARN string) (*LoadBalancer, error) {
	params := url.Values{}
	params.Set("LoadBalancerArns.member.1", strings.TrimSpace(loadBalancerARN))

	response := xmlDescribeLoadBalancersResponse{}
	if err := c.postELBForm("DescribeLoadBalancers", params, &response); err != nil {
		return nil, err
	}

	members := response.Result.LoadBalancers.Members
	if len(members) == 0 {
		return nil, &common.Error{
			Code:    "LoadBalancerNotFound",
			Message: fmt.Sprintf("load balancer not found: %s", loadBalancerARN),
		}
	}

	lb := members[0]
	return &LoadBalancer{
		LoadBalancerARN: lb.LoadBalancerARN,
		Name:            lb.LoadBalancerName,
		DNSName:         lb.DNSName,
		Scheme:          lb.Scheme,
		Type:            lb.Type,
		State:           lb.State.Code,
		VpcID:           lb.VpcID,
		Region:          c.region,
	}, nil
}

func (c *Client) ListLoadBalancers() ([]LoadBalancer, error) {
	loadBalancers := []LoadBalancer{}
	marker := ""

	for {
		params := url.Values{}
		params.Set("PageSize", "100")
		if marker != "" {
			params.Set("Marker", marker)
		}

		response := xmlDescribeLoadBalancersResponse{}
		if err := c.postELBForm("DescribeLoadBalancers", params, &response); err != nil {
			return nil, err
		}

		for _, lb := range response.Result.LoadBalancers.Members {
			loadBalancers = append(loadBalancers, LoadBalancer{
				LoadBalancerARN: lb.LoadBalancerARN,
				Name:            lb.LoadBalancerName,
				DNSName:         lb.DNSName,
				Scheme:          lb.Scheme,
				Type:            lb.Type,
				State:           lb.State.Code,
				VpcID:           lb.VpcID,
				Region:          c.region,
			})
		}

		// ELBv2 DescribeLoadBalancers uses a Marker for pagination
		// The marker field lives inside <DescribeLoadBalancersResult><NextMarker>
		// We break when there are no more results
		nextMarker := strings.TrimSpace(response.Result.NextMarker)
		if nextMarker == "" {
			break
		}
		marker = nextMarker
	}

	return loadBalancers, nil
}

func IsLoadBalancerNotFound(err error) bool {
	var awsErr *common.Error
	return errors.As(err, &awsErr) && awsErr.Code == "LoadBalancerNotFound"
}

type TargetGroup struct {
	TargetGroupARN string `json:"targetGroupArn" mapstructure:"targetGroupArn"`
	Name           string `json:"name" mapstructure:"name"`
	Protocol       string `json:"protocol" mapstructure:"protocol"`
	Port           int    `json:"port" mapstructure:"port"`
	TargetType     string `json:"targetType" mapstructure:"targetType"`
	VpcID          string `json:"vpcId" mapstructure:"vpcId"`
}

type CreateListenerInput struct {
	LoadBalancerARN string
	Protocol        string
	Port            int
	TargetGroupARN  string
	CertificateARN  string
}

type CreateListenerOutput struct {
	ListenerARN string `json:"listenerArn" mapstructure:"listenerArn"`
	Protocol    string `json:"protocol" mapstructure:"protocol"`
	Port        int    `json:"port" mapstructure:"port"`
}

type xmlTargetGroupMember struct {
	TargetGroupARN  string `xml:"TargetGroupArn"`
	TargetGroupName string `xml:"TargetGroupName"`
	Protocol        string `xml:"Protocol"`
	Port            int    `xml:"Port"`
	TargetType      string `xml:"TargetType"`
	VpcID           string `xml:"VpcId"`
}

type xmlTargetGroupMembers struct {
	Members []xmlTargetGroupMember `xml:"member"`
}

type xmlDescribeTargetGroupsResult struct {
	TargetGroups xmlTargetGroupMembers `xml:"TargetGroups"`
	NextMarker   string                `xml:"NextMarker"`
}

type xmlDescribeTargetGroupsResponse struct {
	RequestID string                        `xml:"ResponseMetadata>RequestId"`
	Result    xmlDescribeTargetGroupsResult `xml:"DescribeTargetGroupsResult"`
}

type xmlListenerMember struct {
	ListenerARN string `xml:"ListenerArn"`
	Protocol    string `xml:"Protocol"`
	Port        int    `xml:"Port"`
}

type xmlCreateListenerResult struct {
	Listeners struct {
		Members []xmlListenerMember `xml:"member"`
	} `xml:"Listeners"`
}

type xmlCreateListenerResponse struct {
	RequestID string                  `xml:"ResponseMetadata>RequestId"`
	Result    xmlCreateListenerResult `xml:"CreateListenerResult"`
}

func (c *Client) CreateListener(input CreateListenerInput) (*CreateListenerOutput, error) {
	params := url.Values{}
	params.Set("LoadBalancerArn", strings.TrimSpace(input.LoadBalancerARN))
	params.Set("Protocol", strings.TrimSpace(input.Protocol))
	params.Set("Port", fmt.Sprintf("%d", input.Port))
	params.Set("DefaultActions.member.1.Type", "forward")
	params.Set("DefaultActions.member.1.TargetGroupArn", strings.TrimSpace(input.TargetGroupARN))
	if cert := strings.TrimSpace(input.CertificateARN); cert != "" {
		params.Set("Certificates.member.1.CertificateArn", cert)
	}

	response := xmlCreateListenerResponse{}
	if err := c.postELBForm("CreateListener", params, &response); err != nil {
		return nil, err
	}

	members := response.Result.Listeners.Members
	if len(members) == 0 {
		return nil, fmt.Errorf("response did not include listener details")
	}

	l := members[0]
	return &CreateListenerOutput{
		ListenerARN: l.ListenerARN,
		Protocol:    l.Protocol,
		Port:        l.Port,
	}, nil
}

func (c *Client) ListTargetGroups() ([]TargetGroup, error) {
	targetGroups := []TargetGroup{}
	marker := ""

	for {
		params := url.Values{}
		params.Set("PageSize", "100")
		if marker != "" {
			params.Set("Marker", marker)
		}

		response := xmlDescribeTargetGroupsResponse{}
		if err := c.postELBForm("DescribeTargetGroups", params, &response); err != nil {
			return nil, err
		}

		for _, tg := range response.Result.TargetGroups.Members {
			targetGroups = append(targetGroups, TargetGroup{
				TargetGroupARN: tg.TargetGroupARN,
				Name:           tg.TargetGroupName,
				Protocol:       tg.Protocol,
				Port:           tg.Port,
				TargetType:     tg.TargetType,
				VpcID:          tg.VpcID,
			})
		}

		nextMarker := strings.TrimSpace(response.Result.NextMarker)
		if nextMarker == "" {
			break
		}
		marker = nextMarker
	}

	return targetGroups, nil
}
