package createvm

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	compute "google.golang.org/api/compute/v1"

	"github.com/superplanehq/superplane/pkg/core"
)

type DiskType struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Snapshot struct {
	Name string `json:"name"`
}

type Disk struct {
	Name string `json:"name"`
}

type ResourcePolicy struct {
	Name string `json:"name"`
}

type snapshotsListResp struct {
	Items         []*snapshotItem `json:"items"`
	NextPageToken string          `json:"nextPageToken"`
}

type snapshotItem struct {
	Name string `json:"name"`
}

type disksListResp struct {
	Items         []*diskItem `json:"items"`
	NextPageToken string      `json:"nextPageToken"`
}

type diskItem struct {
	Name string `json:"name"`
}

type diskTypesListResp struct {
	Items         []*diskTypeItem `json:"items"`
	NextPageToken string          `json:"nextPageToken"`
}

type diskTypeItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type resourcePoliciesListResp struct {
	Items         []*resourcePolicyItem `json:"items"`
	NextPageToken string                `json:"nextPageToken"`
}

type resourcePolicyItem struct {
	Name string `json:"name"`
}

var (
	osStorageCache   = make(map[string]*cacheEntry)
	osStorageCacheMu sync.RWMutex
)

const osStorageCacheTTL = 24 * time.Hour

func osStorageCacheGet(key string) (any, bool) {
	osStorageCacheMu.RLock()
	e, ok := osStorageCache[key]
	if !ok || e == nil {
		osStorageCacheMu.RUnlock()
		return nil, false
	}
	if time.Now().After(e.expires) {
		osStorageCacheMu.RUnlock()
		// Lazy eviction: remove expired entry to avoid unbounded memory growth
		osStorageCacheMu.Lock()
		if e2, ok2 := osStorageCache[key]; ok2 && e2 != nil && time.Now().After(e2.expires) {
			delete(osStorageCache, key)
		}
		osStorageCacheMu.Unlock()
		return nil, false
	}
	data := e.data
	osStorageCacheMu.RUnlock()
	return data, true
}

func osStorageCacheSet(key string, data any) {
	osStorageCacheMu.Lock()
	defer osStorageCacheMu.Unlock()
	osStorageCache[key] = &cacheEntry{data: data, expires: time.Now().Add(osStorageCacheTTL)}
}

const (
	BootDiskSourcePublicImage  = "publicImage"
	BootDiskSourceCustomImage  = "customImage"
	BootDiskSourceSnapshot     = "snapshot"
	BootDiskSourceExistingDisk = "existingDisk"
)

const (
	AdditionalDiskModeNew      = "newDisk"
	AdditionalDiskModeExisting = "existingDisk"
)

func intPtr(n int) *int {
	return &n
}

func strPtr(s string) *string {
	return &s
}

const (
	DefaultDiskType   = "pd-balanced"
	DefaultDiskSizeGb = 10
)

type OSAndStorageConfig struct {
	BootDiskSourceType       string                `mapstructure:"bootDiskSourceType"`
	BootDiskOS               string                `mapstructure:"bootDiskOS"`
	BootDiskPublicImage      string                `mapstructure:"bootDiskPublicImage"`
	BootDiskCustomImage      string                `mapstructure:"bootDiskCustomImage"`
	BootDiskSnapshot         string                `mapstructure:"bootDiskSnapshot"`
	BootDiskExistingDisk     string                `mapstructure:"bootDiskExistingDisk"`
	BootDiskType             string                `mapstructure:"bootDiskType"`
	BootDiskSizeGb           int64                 `mapstructure:"bootDiskSizeGb"`
	BootDiskEncryptionKey    string                `mapstructure:"bootDiskEncryptionKey"`
	BootDiskSnapshotSchedule string                `mapstructure:"bootDiskSnapshotSchedule"`
	BootDiskAutoDelete       bool                  `mapstructure:"bootDiskAutoDelete"`
	LocalSSDCount            int64                 `mapstructure:"localSSDCount"`
	AdditionalDisks          []AdditionalDiskEntry `mapstructure:"additionalDisks"`
}

type AdditionalDiskEntry struct {
	Mode         string `mapstructure:"mode"`
	Name         string `mapstructure:"name"`
	SizeGb       int64  `mapstructure:"sizeGb"`
	DiskType     string `mapstructure:"diskType"`
	ExistingDisk string `mapstructure:"existingDisk"`
	AutoDelete   bool   `mapstructure:"autoDelete"`
}

type BootDiskConfig struct {
	Name              string
	DiskType          string
	SizeGb            int64
	SourceImage       string
	SourceSnapshot    string
	SourceDisk        string
	SnapshotSchedule  string
	AutoDelete        bool
	DiskEncryptionKey string
}

type AdditionalDisk struct {
	Name       string
	SizeGb     int64
	DiskType   string
	SourceDisk string
	AutoDelete bool
}

func ListSnapshots(ctx context.Context, c Client, project string) ([]Snapshot, error) {
	project = strings.TrimSpace(project)
	if project == "" {
		project = c.ProjectID()
	}
	cacheKey := "snapshots:" + project
	if v, ok := osStorageCacheGet(cacheKey); ok {
		return v.([]Snapshot), nil
	}
	path := fmt.Sprintf("projects/%s/global/snapshots", project)
	var all []Snapshot
	var pageToken string
	for {
		body, err := c.Get(ctx, withPageToken(path, pageToken))
		if err != nil {
			return nil, err
		}
		var resp snapshotsListResp
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parse snapshots response: %w", err)
		}
		for _, it := range resp.Items {
			if it == nil {
				continue
			}
			all = append(all, Snapshot{Name: it.Name})
		}
		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}
	osStorageCacheSet(cacheKey, all)
	return all, nil
}

func ListDisks(ctx context.Context, c Client, project, zone string) ([]Disk, error) {
	project = strings.TrimSpace(project)
	zone = strings.TrimSpace(zone)
	if project == "" {
		project = c.ProjectID()
	}
	if zone == "" {
		return nil, fmt.Errorf("zone is required")
	}
	cacheKey := "disks:" + project + ":" + zone
	if v, ok := osStorageCacheGet(cacheKey); ok {
		return v.([]Disk), nil
	}
	path := fmt.Sprintf("projects/%s/zones/%s/disks", project, zone)
	var all []Disk
	var pageToken string
	for {
		body, err := c.Get(ctx, withPageToken(path, pageToken))
		if err != nil {
			return nil, err
		}
		var resp disksListResp
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parse disks response: %w", err)
		}
		for _, it := range resp.Items {
			if it == nil {
				continue
			}
			all = append(all, Disk{Name: it.Name})
		}
		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}
	osStorageCacheSet(cacheKey, all)
	return all, nil
}

func ListDiskTypes(ctx context.Context, c Client, project, zone string) ([]DiskType, error) {
	project = strings.TrimSpace(project)
	zone = strings.TrimSpace(zone)
	if project == "" {
		project = c.ProjectID()
	}
	if zone == "" {
		return nil, fmt.Errorf("zone is required")
	}
	cacheKey := "diskTypes:" + project + ":" + zone
	if v, ok := osStorageCacheGet(cacheKey); ok {
		return v.([]DiskType), nil
	}
	path := fmt.Sprintf("projects/%s/zones/%s/diskTypes", project, zone)
	var all []DiskType
	var pageToken string
	for {
		body, err := c.Get(ctx, withPageToken(path, pageToken))
		if err != nil {
			return nil, err
		}
		var resp diskTypesListResp
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parse diskTypes response: %w", err)
		}
		for _, it := range resp.Items {
			if it == nil {
				continue
			}
			all = append(all, DiskType{Name: it.Name, Description: it.Description})
		}
		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}
	osStorageCacheSet(cacheKey, all)
	return all, nil
}

func ListSnapshotSchedules(ctx context.Context, c Client, project, region string) ([]ResourcePolicy, error) {
	project = strings.TrimSpace(project)
	region = strings.TrimSpace(region)
	if project == "" {
		project = c.ProjectID()
	}
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}
	cacheKey := "resourcePolicies:" + project + ":" + region
	if v, ok := osStorageCacheGet(cacheKey); ok {
		return v.([]ResourcePolicy), nil
	}
	path := fmt.Sprintf("projects/%s/regions/%s/resourcePolicies", project, region)
	var all []ResourcePolicy
	var pageToken string
	for {
		body, err := c.Get(ctx, withPageToken(path, pageToken))
		if err != nil {
			return nil, err
		}
		var resp resourcePoliciesListResp
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parse resourcePolicies response: %w", err)
		}
		for _, it := range resp.Items {
			if it == nil {
				continue
			}
			all = append(all, ResourcePolicy{Name: it.Name})
		}
		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}
	osStorageCacheSet(cacheKey, all)
	return all, nil
}

var allowedBootDiskTypes = []string{"pd-balanced", "pd-ssd", "pd-standard"}

func isAllowedBootDiskType(name string) bool {
	return slices.Contains(allowedBootDiskTypes, name)
}

func ListSnapshotResources(ctx context.Context, c Client, project string) ([]core.IntegrationResource, error) {
	list, err := ListSnapshots(ctx, c, project)
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(list))
	for _, s := range list {
		out = append(out, core.IntegrationResource{Type: ResourceTypeSnapshots, Name: s.Name, ID: s.Name})
	}
	return out, nil
}

func ListDiskResources(ctx context.Context, c Client, project, zone string) ([]core.IntegrationResource, error) {
	list, err := ListDisks(ctx, c, project, zone)
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(list))
	for _, d := range list {
		out = append(out, core.IntegrationResource{Type: ResourceTypeDisks, Name: d.Name, ID: d.Name})
	}
	return out, nil
}

func ListDiskTypeResources(ctx context.Context, c Client, project, zone string, bootDiskOnly bool) ([]core.IntegrationResource, error) {
	list, err := ListDiskTypes(ctx, c, project, zone)
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(list))
	for _, dt := range list {
		if bootDiskOnly && !isAllowedBootDiskType(dt.Name) {
			continue
		}
		displayName := dt.Description
		if displayName == "" {
			displayName = dt.Name
		}
		out = append(out, core.IntegrationResource{Type: ResourceTypeDiskTypes, Name: displayName, ID: dt.Name})
	}
	return out, nil
}

func ListSnapshotScheduleResources(ctx context.Context, c Client, project, region string) ([]core.IntegrationResource, error) {
	list, err := ListSnapshotSchedules(ctx, c, project, region)
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(list))
	for _, p := range list {
		out = append(out, core.IntegrationResource{Type: ResourceTypeSnapshotSchedules, Name: p.Name, ID: p.Name})
	}
	return out, nil
}

func BuildBootDisk(project, zone string, config BootDiskConfig) *compute.AttachedDisk {
	diskType := strings.TrimSpace(config.DiskType)
	if diskType == "" {
		diskType = DefaultDiskType
	}
	sizeGb := config.SizeGb
	if sizeGb < 1 {
		sizeGb = DefaultDiskSizeGb
	}
	autoDelete := config.AutoDelete
	diskEncryptionKey := buildDiskEncryptionKey(config.DiskEncryptionKey)

	if strings.TrimSpace(config.SourceDisk) != "" {
		att := &compute.AttachedDisk{
			Source:     config.SourceDisk,
			Boot:       true,
			AutoDelete: autoDelete,
			DeviceName: strings.TrimSpace(config.Name),
		}
		if diskEncryptionKey != nil {
			att.DiskEncryptionKey = diskEncryptionKey
		}
		return att
	}

	params := &compute.AttachedDiskInitializeParams{
		DiskName:   strings.TrimSpace(config.Name),
		DiskSizeGb: sizeGb,
		DiskType:   resolveDiskTypeURL(project, zone, diskType),
	}
	if config.SourceImage != "" {
		params.SourceImage = strings.TrimSpace(config.SourceImage)
	}
	if config.SourceSnapshot != "" {
		params.SourceSnapshot = strings.TrimSpace(config.SourceSnapshot)
	}
	if strings.TrimSpace(config.SnapshotSchedule) != "" {
		params.ResourcePolicies = []string{strings.TrimSpace(config.SnapshotSchedule)}
	}

	att := &compute.AttachedDisk{
		Boot:             true,
		AutoDelete:       autoDelete,
		InitializeParams: params,
		DeviceName:       strings.TrimSpace(config.Name),
	}
	if diskEncryptionKey != nil {
		att.DiskEncryptionKey = diskEncryptionKey
	}
	return att
}

func buildDiskEncryptionKey(kmsKeyName string) *compute.CustomerEncryptionKey {
	kmsKeyName = strings.TrimSpace(kmsKeyName)
	if kmsKeyName == "" {
		return nil
	}
	return &compute.CustomerEncryptionKey{KmsKeyName: kmsKeyName}
}

func BuildAdditionalDisks(project, zone string, disks []AdditionalDisk) []*compute.AttachedDisk {
	if len(disks) == 0 {
		return nil
	}
	out := make([]*compute.AttachedDisk, 0, len(disks))
	for _, d := range disks {
		att := buildOneAdditionalDisk(project, zone, d)
		if att != nil {
			out = append(out, att)
		}
	}
	return out
}

func buildOneAdditionalDisk(project, zone string, d AdditionalDisk) *compute.AttachedDisk {
	name := strings.TrimSpace(d.Name)
	if strings.TrimSpace(d.SourceDisk) != "" {
		return &compute.AttachedDisk{
			Source:     d.SourceDisk,
			Boot:       false,
			AutoDelete: d.AutoDelete,
			DeviceName: name,
		}
	}
	diskType := strings.TrimSpace(d.DiskType)
	if diskType == "" {
		diskType = DefaultDiskType
	}
	isLocalSSD := diskType == "local-ssd"
	params := &compute.AttachedDiskInitializeParams{
		DiskName: name,
		DiskType: resolveDiskTypeURL(project, zone, diskType),
	}
	if !isLocalSSD {
		sizeGb := d.SizeGb
		if sizeGb < 1 {
			sizeGb = DefaultDiskSizeGb
		}
		params.DiskSizeGb = sizeGb
	}
	att := &compute.AttachedDisk{
		Boot:             false,
		AutoDelete:       d.AutoDelete,
		InitializeParams: params,
		DeviceName:       name,
	}
	if isLocalSSD {
		att.Interface = "NVME"
		att.Type = "SCRATCH"
	}
	return att
}

func BuildLocalSSDDisks(project, zone string, count int) []*compute.AttachedDisk {
	if count <= 0 {
		return nil
	}
	if count > 8 {
		count = 8
	}
	out := make([]*compute.AttachedDisk, 0, count)
	diskTypeURL := resolveDiskTypeURL(project, zone, "local-ssd")
	for i := 0; i < count; i++ {
		deviceName := fmt.Sprintf("local-ssd-%d", i)
		out = append(out, &compute.AttachedDisk{
			Type:             "SCRATCH",
			Interface:        "NVME",
			AutoDelete:       true,
			DeviceName:       deviceName,
			InitializeParams: &compute.AttachedDiskInitializeParams{DiskType: diskTypeURL},
		})
	}
	return out
}

func resolveDiskTypeURL(project, zone, diskType string) string {
	if strings.Contains(diskType, "/") {
		return diskType
	}
	if project == "" || zone == "" {
		return diskType
	}
	return fmt.Sprintf("projects/%s/zones/%s/diskTypes/%s", project, zone, diskType)
}
