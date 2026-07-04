package ec2

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ListElasticIPs(t *testing.T) {
	t.Run("missing region -> error", func(t *testing.T) {
		_, err := ListElasticIPs(core.ListResourcesContext{
			Integration: elasticIPIntegration(),
			Parameters:  map[string]string{},
		}, ResourceTypeElasticIP)
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("returns only VPC Elastic IPs, excluding classic standard-domain addresses", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(describeAddressesXML()),
			},
		}

		resources, err := ListElasticIPs(core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: elasticIPIntegration(),
			Parameters:  map[string]string{"region": "us-east-1"},
		}, ResourceTypeElasticIP)

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, ResourceTypeElasticIP, resources[0].Type)
		assert.Equal(t, "eipalloc-abc123", resources[0].ID)
		assert.Equal(t, "203.0.113.10 (eipalloc-abc123)", resources[0].Name)
		assert.Equal(t, "eipalloc-def456", resources[1].ID)
		assert.Equal(t, "198.51.100.5 (eipalloc-def456)", resources[1].Name)

		require.Len(t, httpContext.Requests, 1)
		body, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Action=DescribeAddresses")
	})
}

func Test__ListUnassociatedElasticIPs(t *testing.T) {
	t.Run("missing region -> error", func(t *testing.T) {
		_, err := ListUnassociatedElasticIPs(core.ListResourcesContext{
			Integration: elasticIPIntegration(),
			Parameters:  map[string]string{},
		}, ResourceTypeElasticIPUnassociated)
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("returns only unassociated VPC Elastic IPs, excluding classic standard-domain addresses", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(describeAddressesXML()),
			},
		}

		resources, err := ListUnassociatedElasticIPs(core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: elasticIPIntegration(),
			Parameters:  map[string]string{"region": "us-east-1"},
		}, ResourceTypeElasticIPUnassociated)

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, ResourceTypeElasticIPUnassociated, resources[0].Type)
		assert.Equal(t, "eipalloc-def456", resources[0].ID)
		assert.Equal(t, "198.51.100.5 (eipalloc-def456)", resources[0].Name)
	})
}

func Test__ListElasticIPAssociations(t *testing.T) {
	t.Run("missing region -> error", func(t *testing.T) {
		_, err := ListElasticIPAssociations(core.ListResourcesContext{
			Integration: elasticIPIntegration(),
			Parameters:  map[string]string{},
		}, ResourceTypeElasticIPAssociation)
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("returns only associated VPC Elastic IPs, excluding classic standard-domain addresses", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(describeAddressesXML()),
			},
		}

		resources, err := ListElasticIPAssociations(core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: elasticIPIntegration(),
			Parameters:  map[string]string{"region": "us-east-1"},
		}, ResourceTypeElasticIPAssociation)

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, ResourceTypeElasticIPAssociation, resources[0].Type)
		assert.Equal(t, "eipassoc-xyz789", resources[0].ID)
		assert.Equal(t, "203.0.113.10 → i-abc123 (eipassoc-xyz789)", resources[0].Name)
	})
}

func Test__Client__ListAddresses(t *testing.T) {
	creds, err := common.CredentialsFromInstallation(elasticIPIntegration())
	require.NoError(t, err)

	client := NewClient(
		&contexts.HTTPContext{Responses: []*http.Response{okResponse(describeAddressesXML())}},
		creds,
		"us-east-1",
	)

	addresses, err := client.ListAddresses()
	require.NoError(t, err)
	require.Len(t, addresses, 3)
	assert.Equal(t, "eipalloc-abc123", addresses[0].AllocationID)
	assert.Equal(t, "eipassoc-xyz789", addresses[0].AssociationID)
	assert.Equal(t, "203.0.113.10", addresses[0].PublicIP)
	assert.Equal(t, "i-abc123", addresses[0].InstanceID)
	assert.Equal(t, "vpc", addresses[0].Domain)
	assert.Equal(t, "", addresses[1].AssociationID)
	assert.Equal(t, "eipalloc-classic", addresses[2].AllocationID)
	assert.Equal(t, "standard", addresses[2].Domain)
}

func describeAddressesXML() string {
	return `
		<DescribeAddressesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
			<requestId>req-describe-addresses</requestId>
			<addressesSet>
				<item>
					<publicIp>203.0.113.10</publicIp>
					<allocationId>eipalloc-abc123</allocationId>
					<associationId>eipassoc-xyz789</associationId>
					<instanceId>i-abc123</instanceId>
					<domain>vpc</domain>
				</item>
				<item>
					<publicIp>198.51.100.5</publicIp>
					<allocationId>eipalloc-def456</allocationId>
					<domain>vpc</domain>
				</item>
				<item>
					<publicIp>192.0.2.100</publicIp>
					<allocationId>eipalloc-classic</allocationId>
					<domain>standard</domain>
				</item>
			</addressesSet>
		</DescribeAddressesResponse>`
}

func Test__elasticIPResourceName(t *testing.T) {
	assert.Equal(t, "eipalloc-abc", elasticIPResourceName(ElasticIP{AllocationID: "eipalloc-abc"}))
	assert.Equal(t, "203.0.113.10 (eipalloc-abc)", elasticIPResourceName(ElasticIP{
		AllocationID: "eipalloc-abc",
		PublicIP:     "203.0.113.10",
	}))
}

func Test__ListPublicIPv4Pools(t *testing.T) {
	t.Run("returns BYOIP pools", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{okResponse(describePublicIpv4PoolsXML())},
		}

		resources, err := ListPublicIPv4Pools(core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: elasticIPIntegration(),
			Parameters:  map[string]string{"region": "us-east-1"},
		}, ResourceTypePublicIPv4Pool)

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "ipv4pool-ec2-abc123", resources[0].ID)
		assert.Equal(t, "My BYOIP pool (ipv4pool-ec2-abc123)", resources[0].Name)
	})
}

func Test__ListCustomerOwnedIPv4Pools(t *testing.T) {
	t.Run("returns customer-owned pools", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{okResponse(describeCoipPoolsXML())},
		}

		resources, err := ListCustomerOwnedIPv4Pools(core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: elasticIPIntegration(),
			Parameters:  map[string]string{"region": "us-east-1"},
		}, ResourceTypeCustomerOwnedIPv4Pool)

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "cpool-abc123", resources[0].ID)
		assert.Equal(t, "cpool-abc123 (lgw-rtb-abc123)", resources[0].Name)
	})
}

func Test__ListIpamPools(t *testing.T) {
	t.Run("returns EC2-compatible public IPv4 IPAM pools", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{okResponse(describeIpamPoolsXML())},
		}

		resources, err := ListIpamPools(core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: elasticIPIntegration(),
			Parameters:  map[string]string{"region": "us-east-1"},
		}, ResourceTypeIpamPool)

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "ipam-pool-abc123", resources[0].ID)
		assert.Equal(t, "Public EIP pool (ipam-pool-abc123)", resources[0].Name)
	})
}

func Test__isIpamPoolForElasticIP(t *testing.T) {
	region := "us-east-1"

	assert.True(t, isIpamPoolForElasticIP(xmlIpamPool{
		AddressFamily:          "ipv4",
		PubliclyAdvertisable:   true,
		Locale:                 "us-east-1",
		AllocationResourceType: "ec2",
	}, region))

	assert.False(t, isIpamPoolForElasticIP(xmlIpamPool{
		AddressFamily:        "ipv6",
		PubliclyAdvertisable: true,
		Locale:               "us-east-1",
	}, region))

	assert.False(t, isIpamPoolForElasticIP(xmlIpamPool{
		AddressFamily:        "ipv4",
		PubliclyAdvertisable: false,
		Locale:               "us-east-1",
	}, region))

	assert.False(t, isIpamPoolForElasticIP(xmlIpamPool{
		AddressFamily:        "ipv4",
		PubliclyAdvertisable: true,
		Locale:               "eu-west-1",
	}, region))
}

func describePublicIpv4PoolsXML() string {
	return `
		<DescribePublicIpv4PoolsResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
			<requestId>req-public-pools</requestId>
			<publicIpv4PoolSet>
				<item>
					<poolId>ipv4pool-ec2-abc123</poolId>
					<description>My BYOIP pool</description>
				</item>
			</publicIpv4PoolSet>
		</DescribePublicIpv4PoolsResponse>`
}

func describeCoipPoolsXML() string {
	return `
		<DescribeCoipPoolsResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
			<requestId>req-coip-pools</requestId>
			<coipPoolSet>
				<item>
					<poolId>cpool-abc123</poolId>
					<localGatewayRouteTableId>lgw-rtb-abc123</localGatewayRouteTableId>
				</item>
			</coipPoolSet>
		</DescribeCoipPoolsResponse>`
}

func describeIpamPoolsXML() string {
	return `
		<DescribeIpamPoolsResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
			<requestId>req-ipam-pools</requestId>
			<ipamPoolSet>
				<item>
					<ipamPoolId>ipam-pool-abc123</ipamPoolId>
					<description>Public EIP pool</description>
					<addressFamily>ipv4</addressFamily>
					<publiclyAdvertisable>true</publiclyAdvertisable>
					<locale>us-east-1</locale>
					<allocationResourceType>ec2</allocationResourceType>
				</item>
				<item>
					<ipamPoolId>ipam-pool-private</ipamPoolId>
					<description>Private pool</description>
					<addressFamily>ipv4</addressFamily>
					<publiclyAdvertisable>false</publiclyAdvertisable>
					<locale>us-east-1</locale>
				</item>
			</ipamPoolSet>
		</DescribeIpamPoolsResponse>`
}

func Test__elasticIPAssociationResourceName(t *testing.T) {
	assert.Equal(t, "eipassoc-xyz", elasticIPAssociationResourceName(ElasticIP{AssociationID: "eipassoc-xyz"}))
	assert.Equal(t, "203.0.113.10 → i-abc123 (eipassoc-xyz)", elasticIPAssociationResourceName(ElasticIP{
		AssociationID: "eipassoc-xyz",
		PublicIP:      "203.0.113.10",
		InstanceID:    "i-abc123",
	}))
}
