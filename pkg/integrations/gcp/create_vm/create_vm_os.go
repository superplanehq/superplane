package createvm

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ResourceTypePublicImages      = "publicImages"
	ResourceTypeCustomImages      = "customImages"
	ResourceTypeSnapshots         = "snapshots"
	ResourceTypeDisks             = "disks"
	ResourceTypeDiskTypes         = "diskTypes"
	ResourceTypeSnapshotSchedules = "snapshotSchedules"
)

type Image struct {
	Name        string `json:"name"`
	Family      string `json:"family"`
	Description string `json:"description"`
	SelfLink    string `json:"selfLink"`
}

type imagesListResp struct {
	Items         []*imageItem `json:"items"`
	NextPageToken string       `json:"nextPageToken"`
}

type imageItem struct {
	Name        string `json:"name"`
	Family      string `json:"family"`
	Description string `json:"description"`
	SelfLink    string `json:"selfLink"`
}

func CreateVMOSAndStorageConfigFields() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "bootDiskSourceType",
			Label:       "Boot disk source",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Source for boot disk: public image, custom image, snapshot, or existing disk.",
			Default:     BootDiskSourcePublicImage,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Public image", Value: BootDiskSourcePublicImage},
						{Label: "Custom image", Value: BootDiskSourceCustomImage},
						{Label: "Snapshot", Value: BootDiskSourceSnapshot},
						{Label: "Existing disk", Value: BootDiskSourceExistingDisk},
					},
				},
			},
		},
		{
			Name:        "bootDiskOS",
			Label:       "Operating system",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Select the OS (e.g. Debian, Ubuntu). Then pick a version below.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: publicImageOSOptions,
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "bootDiskSourceType", Values: []string{BootDiskSourcePublicImage}},
			},
		},
		{
			Name:        "bootDiskPublicImage",
			Label:       "Version",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Select the image version for the chosen operating system.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypePublicImages,
					Parameters: []configuration.ParameterRef{
						{Name: "project", ValueFrom: &configuration.ParameterValueFrom{Field: "bootDiskOS"}},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "bootDiskSourceType", Values: []string{BootDiskSourcePublicImage}},
			},
		},
		{
			Name:        "bootDiskCustomImage",
			Label:       "Custom image",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Select a custom image from your project.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeCustomImages,
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "bootDiskSourceType", Values: []string{BootDiskSourceCustomImage}},
			},
		},
		{
			Name:        "bootDiskSnapshot",
			Label:       "Snapshot",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Select a snapshot to create the boot disk from.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeSnapshots,
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "bootDiskSourceType", Values: []string{BootDiskSourceSnapshot}},
			},
		},
		{
			Name:        "bootDiskExistingDisk",
			Label:       "Existing disk",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Select an existing disk in the same zone to use as the boot disk.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeDisks,
					Parameters: []configuration.ParameterRef{
						{Name: "zone", ValueFrom: &configuration.ParameterValueFrom{Field: "zone"}},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "bootDiskSourceType", Values: []string{BootDiskSourceExistingDisk}},
			},
		},
		{
			Name:        "bootDiskType",
			Label:       "Boot disk type",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Balanced, SSD, or Standard persistent disk.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeDiskTypes,
					Parameters: []configuration.ParameterRef{
						{Name: "zone", ValueFrom: &configuration.ParameterValueFrom{Field: "zone"}},
						{Name: "bootDiskOnly", Value: strPtr("true")},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "bootDiskSourceType", Values: []string{BootDiskSourcePublicImage, BootDiskSourceCustomImage, BootDiskSourceSnapshot}},
			},
		},
		{
			Name:        "bootDiskSizeGb",
			Label:       "Boot disk size (GB)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Boot disk size in GB. Provision between 10 and 65536 GB.",
			Default:     DefaultDiskSizeGb,
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{Min: intPtr(10), Max: intPtr(65536)},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "bootDiskSourceType", Values: []string{BootDiskSourcePublicImage, BootDiskSourceCustomImage, BootDiskSourceSnapshot}},
			},
		},
		{
			Name:        "bootDiskEncryptionKey",
			Label:       "Disk encryption key (optional)",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Cloud KMS key resource name for customer-managed encryption (CMEK). Leave empty for Google-managed encryption.",
			Placeholder: "e.g. projects/my-project/locations/region/keyRings/ring/cryptoKeys/key",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "bootDiskSourceType", Values: []string{BootDiskSourcePublicImage, BootDiskSourceCustomImage, BootDiskSourceSnapshot}},
			},
		},
		{
			Name:        "bootDiskSnapshotSchedule",
			Label:       "Snapshot schedule",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional resource policy for automatic snapshot schedule.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeSnapshotSchedules,
					Parameters: []configuration.ParameterRef{
						{Name: "region", ValueFrom: &configuration.ParameterValueFrom{Field: "region"}},
					},
				},
			},
		},
		{
			Name:        "bootDiskAutoDelete",
			Label:       "Delete boot disk on termination",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Delete the boot disk when the instance is deleted.",
			Default:     true,
		},
		{
			Name:        "localSSDCount",
			Label:       "Local SSD count",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Number of local SSDs to attach (0–8). Each local SSD is ~375 GB, NVME interface.",
			Default:     0,
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{Min: intPtr(0), Max: intPtr(8)},
			},
		},
		{
			Name:        "additionalDisks",
			Label:       "Additional disks",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Add disks: create new disks or attach existing ones from the same zone.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Disk",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "mode",
								Label:       "Type",
								Type:        configuration.FieldTypeSelect,
								Required:    true,
								Description: "Add a new disk or attach an existing disk.",
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "Add new disk", Value: AdditionalDiskModeNew},
											{Label: "Attach existing disk", Value: AdditionalDiskModeExisting},
										},
									},
								},
							},
							{
								Name:        "name",
								Label:       "Name",
								Type:        configuration.FieldTypeString,
								Required:    false,
								Description: "Device / disk name (e.g. data-disk-1).",
								Placeholder: "e.g. data-disk-1",
								VisibilityConditions: []configuration.VisibilityCondition{
									{Field: "mode", Values: []string{AdditionalDiskModeNew}},
								},
							},
							{
								Name:        "sizeGb",
								Label:       "Size (GB)",
								Type:        configuration.FieldTypeNumber,
								Required:    false,
								Description: "Size of the new disk in GB. Not used for Local SSD (fixed size per disk).",
								Default:     10,
								TypeOptions: &configuration.TypeOptions{
									Number: &configuration.NumberTypeOptions{Min: intPtr(1), Max: intPtr(65536)},
								},
								VisibilityConditions: []configuration.VisibilityCondition{
									{Field: "mode", Values: []string{AdditionalDiskModeNew}},
									{Field: "diskType", Values: []string{"pd-balanced", "pd-ssd", "pd-standard", "hyperdisk-balanced", "hyperdisk-throughput"}},
								},
							},
							{
								Name:        "diskType",
								Label:       "Disk type",
								Type:        configuration.FieldTypeSelect,
								Required:    false,
								Description: "Disk type for the new disk. Resolved per zone when creating the VM.",
								Default:     DefaultDiskType,
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "Balanced persistent disk", Value: "pd-balanced"},
											{Label: "SSD persistent disk", Value: "pd-ssd"},
											{Label: "Standard persistent disk", Value: "pd-standard"},
											{Label: "Hyperdisk Balanced", Value: "hyperdisk-balanced"},
											{Label: "Hyperdisk Throughput", Value: "hyperdisk-throughput"},
											{Label: "Local SSD", Value: "local-ssd"},
										},
									},
								},
								VisibilityConditions: []configuration.VisibilityCondition{
									{Field: "mode", Values: []string{AdditionalDiskModeNew}},
								},
							},
							{
								Name:        "existingDisk",
								Label:       "Existing disk",
								Type:        configuration.FieldTypeIntegrationResource,
								Required:    false,
								Description: "Select an existing disk in the same zone to attach.",
								TypeOptions: &configuration.TypeOptions{
									Resource: &configuration.ResourceTypeOptions{
										Type: ResourceTypeDisks,
										Parameters: []configuration.ParameterRef{
											{Name: "zone", ValueFrom: &configuration.ParameterValueFrom{Field: "zone"}},
										},
									},
								},
								VisibilityConditions: []configuration.VisibilityCondition{
									{Field: "mode", Values: []string{AdditionalDiskModeExisting}},
								},
							},
							{
								Name:        "autoDelete",
								Label:       "Delete on termination",
								Type:        configuration.FieldTypeBool,
								Required:    false,
								Description: "Delete this disk when the instance is terminated.",
								Default:     true,
							},
						},
					},
				},
			},
		},
	}
}

var publicImageProjects = []string{
	"debian-cloud",
	"ubuntu-os-cloud",
	"windows-cloud",
	"windows-sql-cloud",
	"centos-cloud",
	"cos-cloud",
	"opensuse-cloud",
	"oracle-linux-cloud",
	"fedora-cloud",
	"rocky-linux-cloud",
}

const maxPublicImagesSingleProject = 100

func isPublicImageProject(project string) bool {
	return slices.Contains(publicImageProjects, project)
}

func withMaxResults(path string, maxResults int, pageToken string) string {
	if maxResults <= 0 {
		return withPageToken(path, pageToken)
	}
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	p := path + sep + fmt.Sprintf("maxResults=%d", maxResults)
	if pageToken != "" {
		p += "&pageToken=" + pageToken
	}
	return p
}

func imageItemToImage(it *imageItem) Image {
	if it == nil {
		return Image{}
	}
	return Image{
		Name:        it.Name,
		Family:      it.Family,
		Description: it.Description,
		SelfLink:    it.SelfLink,
	}
}

func imageSelfLinkOrName(img Image) string {
	if img.SelfLink != "" {
		return img.SelfLink
	}
	return img.Name
}

var publicImageOSOptions = []configuration.FieldOption{
	{Label: "Debian", Value: "debian-cloud"},
	{Label: "Ubuntu", Value: "ubuntu-os-cloud"},
	{Label: "Windows Server", Value: "windows-cloud"},
	{Label: "SQL Server on Windows", Value: "windows-sql-cloud"},
	{Label: "CentOS", Value: "centos-cloud"},
	{Label: "Container-Optimized OS", Value: "cos-cloud"},
	{Label: "openSUSE", Value: "opensuse-cloud"},
	{Label: "Oracle Linux", Value: "oracle-linux-cloud"},
	{Label: "Fedora", Value: "fedora-cloud"},
	{Label: "Rocky Linux", Value: "rocky-linux-cloud"},
}

func ListPublicImages(ctx context.Context, c Client, project string) ([]Image, error) {
	project = strings.TrimSpace(project)
	if project == "" {
		return nil, nil
	}
	if !isPublicImageProject(project) {
		return nil, nil
	}
	cacheKey := "publicImages:" + project
	if v, ok := osStorageCacheGet(cacheKey); ok {
		return v.([]Image), nil
	}
	path := fmt.Sprintf("projects/%s/global/images", project)
	body, err := c.Get(ctx, withMaxResults(path, maxPublicImagesSingleProject, ""))
	if err != nil {
		return nil, fmt.Errorf("list public images for %s: %w", project, err)
	}
	var resp imagesListResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse images response: %w", err)
	}
	var all []Image
	for _, it := range resp.Items {
		if it == nil {
			continue
		}
		all = append(all, imageItemToImage(it))
	}
	osStorageCacheSet(cacheKey, all)
	return all, nil
}

func GetImageFromFamily(ctx context.Context, c Client, project, family string) (*Image, error) {
	project = strings.TrimSpace(project)
	family = strings.TrimSpace(family)
	if project == "" {
		project = c.ProjectID()
	}
	if family == "" {
		return nil, fmt.Errorf("family is required")
	}
	path := fmt.Sprintf("projects/%s/global/images/family/%s", project, family)
	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var it imageItem
	if err := json.Unmarshal(body, &it); err != nil {
		return nil, fmt.Errorf("parse image family response: %w", err)
	}
	img := imageItemToImage(&it)
	return &img, nil
}

func ListCustomImages(ctx context.Context, c Client, project string) ([]Image, error) {
	project = strings.TrimSpace(project)
	if project == "" {
		project = c.ProjectID()
	}
	cacheKey := "customImages:" + project
	if v, ok := osStorageCacheGet(cacheKey); ok {
		return v.([]Image), nil
	}
	path := fmt.Sprintf("projects/%s/global/images", project)
	var all []Image
	var pageToken string
	for {
		body, err := c.Get(ctx, withPageToken(path, pageToken))
		if err != nil {
			return nil, err
		}
		var resp imagesListResp
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parse custom images response: %w", err)
		}
		for _, it := range resp.Items {
			if it == nil {
				continue
			}
			all = append(all, imageItemToImage(it))
		}
		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}
	osStorageCacheSet(cacheKey, all)
	return all, nil
}

func ListPublicImageResources(ctx context.Context, c Client, project string) ([]core.IntegrationResource, error) {
	list, err := ListPublicImages(ctx, c, project)
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(list))
	for _, img := range list {
		name := img.Name
		if img.Family != "" {
			name = fmt.Sprintf("%s (%s)", img.Name, img.Family)
		}
		out = append(out, core.IntegrationResource{Type: ResourceTypePublicImages, Name: name, ID: imageSelfLinkOrName(img)})
	}
	return out, nil
}

func ListCustomImageResources(ctx context.Context, c Client, project string) ([]core.IntegrationResource, error) {
	list, err := ListCustomImages(ctx, c, project)
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(list))
	for _, img := range list {
		out = append(out, core.IntegrationResource{Type: ResourceTypeCustomImages, Name: img.Name, ID: imageSelfLinkOrName(img)})
	}
	return out, nil
}
