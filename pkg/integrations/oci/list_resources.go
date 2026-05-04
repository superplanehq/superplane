package oci

import (
	"fmt"
	"sort"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ResourceTypeCompartment         = "compartment"
	ResourceTypeAvailabilityDomain  = "availabilityDomain"
	ResourceTypeShape               = "shape"
	ResourceTypeImageOS             = "imageOS"
	ResourceTypeImage               = "image"
	ResourceTypeCustomImage         = "customImage"
	ResourceTypeVCN                 = "vcn"
	ResourceTypeSubnet              = "subnet"
	ResourceTypeBlockVolume         = "blockVolume"
	ResourceTypeBootVolumeVPU       = "bootVolumeVPU"
	ResourceTypeImageSource         = "imageSource"
	ResourceTypeSourceImageType     = "sourceImageType"
	ResourceTypeObjectNamespace     = "objectStorageNamespace"
	ResourceTypeObjectBucket        = "objectStorageBucket"
	ResourceTypeObject              = "objectStorageObject"
	ResourceTypeInstance            = "instance"
	ResourceTypeFunctionApplication = "functionApplication"
	ResourceTypeFunction            = "function"
	ResourceTypeContainerRepository = "containerRepository"
	ResourceTypeContainerImage      = "containerImage"
)

func (o *OCI) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case ResourceTypeCompartment:
		return listCompartments(ctx)
	case ResourceTypeAvailabilityDomain:
		return listAvailabilityDomains(ctx)
	case ResourceTypeShape:
		return listShapes(ctx)
	case ResourceTypeImageOS:
		return listImageOperatingSystems(ctx)
	case ResourceTypeImage:
		return listImages(ctx)
	case ResourceTypeCustomImage:
		return listCustomImages(ctx)
	case ResourceTypeVCN:
		return listVCNs(ctx)
	case ResourceTypeSubnet:
		return listSubnets(ctx)
	case ResourceTypeBlockVolume:
		return listBlockVolumes(ctx)
	case ResourceTypeBootVolumeVPU:
		return staticResources(ResourceTypeBootVolumeVPU, []core.IntegrationResource{
			{Type: ResourceTypeBootVolumeVPU, Name: "Lower Cost (0 VPUs/GB)", ID: "0"},
			{Type: ResourceTypeBootVolumeVPU, Name: "Balanced (10 VPUs/GB)", ID: "10"},
			{Type: ResourceTypeBootVolumeVPU, Name: "Higher Performance (20 VPUs/GB)", ID: "20"},
			{Type: ResourceTypeBootVolumeVPU, Name: "Ultra High Performance (30 VPUs/GB)", ID: "30"},
		}), nil
	case ResourceTypeImageSource:
		return staticResources(ResourceTypeImageSource, []core.IntegrationResource{
			{Type: ResourceTypeImageSource, Name: "Instance", ID: createImageSourceInstance},
			{Type: ResourceTypeImageSource, Name: "Object Storage URL", ID: createImageSourceObjectStorageURI},
			{Type: ResourceTypeImageSource, Name: "Object Storage Object", ID: createImageSourceObjectStorageObject},
		}), nil
	case ResourceTypeSourceImageType:
		return staticResources(ResourceTypeSourceImageType, []core.IntegrationResource{
			{Type: ResourceTypeSourceImageType, Name: "QCOW2", ID: "QCOW2"},
			{Type: ResourceTypeSourceImageType, Name: "VMDK", ID: "VMDK"},
			{Type: ResourceTypeSourceImageType, Name: "OCI", ID: "OCI"},
		}), nil
	case ResourceTypeObjectNamespace:
		return listObjectStorageNamespaces(ctx)
	case ResourceTypeObjectBucket:
		return listObjectStorageBuckets(ctx)
	case ResourceTypeObject:
		return listObjectStorageObjects(ctx)
	case ResourceTypeInstance:
		return listInstances(ctx)
	case ResourceTypeFunctionApplication:
		return listFunctionApplications(ctx)
	case ResourceTypeFunction:
		return listFunctions(ctx)
	case ResourceTypeContainerRepository:
		return listContainerRepositories(ctx)
	case ResourceTypeContainerImage:
		return listContainerImages(ctx)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

func staticResources(resourceType string, resources []core.IntegrationResource) []core.IntegrationResource {
	for i := range resources {
		resources[i].Type = resourceType
	}
	return resources
}

func listCompartments(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI client: %w", err)
	}

	compartments, err := client.ListCompartments()
	if err != nil {
		return nil, fmt.Errorf("failed to list compartments: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(compartments)+1)

	// Always include the root (tenancy) compartment first.
	resources = append(resources, core.IntegrationResource{
		Type: ResourceTypeCompartment,
		Name: "Root (tenancy)",
		ID:   client.tenancyOCID,
	})

	for _, c := range compartments {
		if c.LifecycleState != "ACTIVE" {
			continue
		}
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeCompartment,
			Name: c.Name,
			ID:   c.ID,
		})
	}

	return resources, nil
}

func listAvailabilityDomains(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI client: %w", err)
	}

	// Use the compartmentId parameter if provided, otherwise fall back to the tenancy OCID.
	compartmentID := ctx.Parameters["compartmentId"]
	if compartmentID == "" {
		compartmentID = client.tenancyOCID
	}

	ads, err := client.ListAvailabilityDomains(compartmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list availability domains: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(ads))
	for _, ad := range ads {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeAvailabilityDomain,
			Name: ad.Name,
			ID:   ad.Name,
		})
	}

	return resources, nil
}

func listShapes(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI client: %w", err)
	}

	compartmentID := ctx.Parameters["compartmentId"]
	if compartmentID == "" {
		compartmentID = client.tenancyOCID
	}

	shapes, err := client.ListShapes(compartmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list shapes: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(shapes))
	for _, s := range shapes {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeShape,
			Name: s.Shape,
			ID:   s.Shape,
		})
	}

	return resources, nil
}

func listImageOperatingSystems(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI client: %w", err)
	}

	images, err := client.ListImages(client.tenancyOCID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	names := map[string]struct{}{}
	for _, img := range images {
		if img.LifecycleState != "AVAILABLE" || img.OperatingSystem == "" {
			continue
		}
		names[img.OperatingSystem] = struct{}{}
	}

	sorted := make([]string, 0, len(names))
	for name := range names {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)

	resources := make([]core.IntegrationResource, 0, len(sorted))
	for _, name := range sorted {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeImageOS,
			Name: name,
			ID:   name,
		})
	}

	return resources, nil
}

func listImages(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return listImageResources(ctx, ResourceTypeImage, func(_ Image) bool { return true })
}

func listCustomImages(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return listImageResources(ctx, ResourceTypeCustomImage, isCustomImage)
}

func listImageResources(ctx core.ListResourcesContext, resourceType string, include func(Image) bool) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI client: %w", err)
	}

	// The ListImages API requires a compartment ID, but since images are global resources,
	// we can use the tenancy OCID as the compartment ID to list all images accessible to the tenancy.
	images, err := client.ListImages(client.tenancyOCID, ctx.Parameters["imageOs"])
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(images))
	for _, img := range images {
		if img.LifecycleState != "AVAILABLE" {
			continue
		}
		if !include(img) {
			continue
		}
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: img.DisplayName,
			ID:   img.ID,
		})
	}

	return resources, nil
}

func listObjectStorageNamespaces(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI client: %w", err)
	}

	namespaceName, err := client.GetObjectStorageNamespace()
	if err != nil {
		return nil, fmt.Errorf("failed to get object storage namespace: %w", err)
	}
	if namespaceName == "" {
		return nil, nil
	}

	return []core.IntegrationResource{{
		Type: ResourceTypeObjectNamespace,
		Name: namespaceName,
		ID:   namespaceName,
	}}, nil
}

func listObjectStorageBuckets(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI client: %w", err)
	}

	namespaceName := ctx.Parameters["namespaceName"]
	if namespaceName == "" {
		namespaceName, err = client.GetObjectStorageNamespace()
		if err != nil {
			return nil, fmt.Errorf("failed to get object storage namespace: %w", err)
		}
	}
	if namespaceName == "" {
		return nil, nil
	}

	compartmentID := ctx.Parameters["compartmentId"]
	if compartmentID == "" {
		compartmentID = client.tenancyOCID
	}

	buckets, err := client.ListBuckets(namespaceName, compartmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(buckets))
	for _, bucket := range buckets {
		if bucket.Name == "" {
			continue
		}
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeObjectBucket,
			Name: bucket.Name,
			ID:   bucket.Name,
		})
	}

	return resources, nil
}

func listObjectStorageObjects(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI client: %w", err)
	}

	namespaceName := ctx.Parameters["namespaceName"]
	if namespaceName == "" {
		namespaceName, err = client.GetObjectStorageNamespace()
		if err != nil {
			return nil, fmt.Errorf("failed to get object storage namespace: %w", err)
		}
	}
	bucketName := ctx.Parameters["bucketName"]
	if namespaceName == "" || bucketName == "" {
		return nil, nil
	}

	objects, err := client.ListObjects(namespaceName, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(objects))
	for _, object := range objects {
		if object.Name == "" {
			continue
		}
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeObject,
			Name: object.Name,
			ID:   object.Name,
		})
	}

	return resources, nil
}

func listInstances(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI client: %w", err)
	}

	compartmentID := ctx.Parameters["compartmentId"]
	if compartmentID == "" {
		compartmentID = client.tenancyOCID
	}

	instances, err := client.ListInstances(compartmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(instances))
	for _, instance := range instances {
		if instance.LifecycleState != "RUNNING" && instance.LifecycleState != "STOPPED" {
			continue
		}

		name := instance.DisplayName
		if name == "" {
			name = instance.ID
		}

		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeInstance,
			Name: name,
			ID:   instance.ID,
		})
	}

	return resources, nil
}

func listVCNs(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI client: %w", err)
	}

	compartmentID := ctx.Parameters["compartmentId"]
	if compartmentID == "" {
		compartmentID = client.tenancyOCID
	}

	vcns, err := client.ListVCNs(compartmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list VCNs: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(vcns))
	for _, v := range vcns {
		if v.LifecycleState != "AVAILABLE" {
			continue
		}
		name := v.DisplayName
		if v.CIDRBlock != "" {
			name = v.DisplayName + " (" + v.CIDRBlock + ")"
		}
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeVCN,
			Name: name,
			ID:   v.ID,
		})
	}

	return resources, nil
}

func listSubnets(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI client: %w", err)
	}

	compartmentID := ctx.Parameters["compartmentId"]
	if compartmentID == "" {
		compartmentID = client.tenancyOCID
	}
	vcnID := ctx.Parameters["vcnId"]

	subnets, err := client.ListSubnets(compartmentID, vcnID)
	if err != nil {
		return nil, fmt.Errorf("failed to list subnets: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(subnets))
	for _, sn := range subnets {
		if sn.LifecycleState != "AVAILABLE" {
			continue
		}
		name := sn.DisplayName
		if sn.CIDRBlock != "" {
			name = sn.DisplayName + " (" + sn.CIDRBlock + ")"
		}
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeSubnet,
			Name: name,
			ID:   sn.ID,
		})
	}

	return resources, nil
}

func listBlockVolumes(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI client: %w", err)
	}

	compartmentID := ctx.Parameters["compartmentId"]
	if compartmentID == "" {
		compartmentID = client.tenancyOCID
	}

	volumes, err := client.ListBlockVolumes(compartmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list block volumes: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(volumes))
	for _, v := range volumes {
		if v.LifecycleState != "AVAILABLE" {
			continue
		}
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeBlockVolume,
			Name: v.DisplayName,
			ID:   v.ID,
		})
	}

	return resources, nil
}

func listFunctionApplications(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI client: %w", err)
	}

	compartmentID := ctx.Parameters["compartmentId"]
	if compartmentID == "" {
		compartmentID = client.tenancyOCID
	}

	apps, err := client.ListApplications(compartmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list function applications: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(apps))
	for _, app := range apps {
		if app.LifecycleState != "ACTIVE" {
			continue
		}
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeFunctionApplication,
			Name: app.DisplayName,
			ID:   app.ID,
		})
	}

	return resources, nil
}

func listContainerRepositories(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI client: %w", err)
	}

	compartmentID := ctx.Parameters["compartmentId"]
	if compartmentID == "" {
		compartmentID = client.tenancyOCID
	}

	repos, err := client.ListContainerRepositories(compartmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list container repositories: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(repos))
	for _, r := range repos {
		if r.LifecycleState != "AVAILABLE" {
			continue
		}
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeContainerRepository,
			Name: r.DisplayName,
			ID:   r.ID,
		})
	}

	return resources, nil
}

func listContainerImages(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI client: %w", err)
	}

	compartmentID := ctx.Parameters["compartmentId"]
	if compartmentID == "" {
		compartmentID = client.tenancyOCID
	}
	repositoryID := ctx.Parameters["repositoryId"]
	if repositoryID == "" {
		return nil, fmt.Errorf("repositoryId parameter is required to list container images")
	}

	// Get the OCIR namespace so we can construct the full image URI.
	// OCI Functions requires the image in the format:
	//   <region>.ocir.io/<namespace>/<repositoryName>:<version>
	namespace, err := client.GetOCIRNamespace(compartmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get OCIR namespace: %w", err)
	}

	images, err := client.ListContainerImages(compartmentID, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to list container images: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(images))
	for _, img := range images {
		if img.LifecycleState != "AVAILABLE" {
			continue
		}
		// Skip untagged images: an empty Version would produce an invalid URI ending with ':'.
		if img.Version == "" {
			continue
		}
		// Construct the full OCIR image URI required by OCI Functions.
		// The format is: <region-key>.ocir.io/<namespace>/<repositoryName>:<version>
		fullImageURI := fmt.Sprintf("%s/%s/%s:%s",
			client.ocirRegistryHost(), namespace, img.RepositoryName, img.Version)
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeContainerImage,
			Name: fullImageURI,
			ID:   fullImageURI,
		})
	}

	return resources, nil
}

func listFunctions(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI client: %w", err)
	}

	applicationID := ctx.Parameters["applicationId"]
	if applicationID == "" {
		return nil, fmt.Errorf("applicationId parameter is required to list functions")
	}

	fns, err := client.ListFunctions(applicationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list functions: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(fns))
	for _, fn := range fns {
		if fn.LifecycleState != "ACTIVE" {
			continue
		}
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeFunction,
			Name: fn.DisplayName,
			ID:   fn.ID,
		})
	}

	return resources, nil
}
