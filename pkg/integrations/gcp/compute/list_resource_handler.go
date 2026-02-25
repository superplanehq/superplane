package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const cacheTTL = 24 * time.Hour

const (
	ResourceTypeRegion        = "region"
	ResourceTypeZone          = "zone"
	ResourceTypeMachineFamily = "machineFamily"
	ResourceTypeMachineType   = "machineType"
)

type Region struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Zones       []string `json:"zones"`
}

type Zone struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Region string `json:"region"`
}

type MachineType struct {
	Name        string `json:"name"`
	GuestCPUs   int    `json:"guestCpus"`
	MemoryMB    int    `json:"memoryMb"`
	Description string `json:"description"`
	SharedCPU   bool   `json:"isSharedCpu"`
	Family      string `json:"family,omitempty"`
}

type MachineFamily struct {
	Family string `json:"family"`
}

type regionsListResp struct {
	Items         []*regionItem `json:"items"`
	NextPageToken string        `json:"nextPageToken"`
}

type regionItem struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Zones       []string `json:"zones"`
}

type machineTypesListResp struct {
	Items         []*machineTypeItem `json:"items"`
	NextPageToken string             `json:"nextPageToken"`
}

type machineTypeItem struct {
	Name        string `json:"name"`
	GuestCpus   int64  `json:"guestCpus"`
	MemoryMb    int64  `json:"memoryMb"`
	Description string `json:"description"`
	IsSharedCPU bool   `json:"isSharedCpu"`
}

type cacheEntry struct {
	data    any
	expires time.Time
}

var (
	machineConfigCache   = make(map[string]*cacheEntry)
	machineConfigCacheMu sync.RWMutex
)

func cacheGet(key string) (any, bool) {
	machineConfigCacheMu.RLock()
	e, ok := machineConfigCache[key]
	if !ok || e == nil {
		machineConfigCacheMu.RUnlock()
		return nil, false
	}
	if time.Now().After(e.expires) {
		machineConfigCacheMu.RUnlock()
		// Lazy eviction: remove expired entry to avoid unbounded memory growth
		machineConfigCacheMu.Lock()
		if e2, ok2 := machineConfigCache[key]; ok2 && e2 != nil && time.Now().After(e2.expires) {
			delete(machineConfigCache, key)
		}
		machineConfigCacheMu.Unlock()
		return nil, false
	}
	data := e.data
	machineConfigCacheMu.RUnlock()
	return data, true
}

func cacheSet(key string, data any) {
	machineConfigCacheMu.Lock()
	defer machineConfigCacheMu.Unlock()
	machineConfigCache[key] = &cacheEntry{data: data, expires: time.Now().Add(cacheTTL)}
}
func regionFromAPI(it *regionItem) Region {
	zoneNames := make([]string, 0, len(it.Zones))
	for _, z := range it.Zones {
		if name := lastSegment(z); name != "" {
			zoneNames = append(zoneNames, name)
		}
	}
	return Region{
		Name:        it.Name,
		Description: it.Description,
		Status:      it.Status,
		Zones:       zoneNames,
	}
}

func machineTypeFromAPI(it *machineTypeItem) MachineType {
	return MachineType{
		Name:        it.Name,
		GuestCPUs:   int(it.GuestCpus),
		MemoryMB:    int(it.MemoryMb),
		Description: it.Description,
		SharedCPU:   it.IsSharedCPU,
		Family:      DeriveFamily(it.Name),
	}
}

func lastSegment(url string) string {
	if url == "" {
		return ""
	}
	i := strings.LastIndex(url, "/")
	if i < 0 {
		return url
	}
	return url[i+1:]
}

func withPageToken(path, pageToken string) string {
	if pageToken == "" {
		return path
	}
	return path + "?pageToken=" + pageToken
}

func DeriveFamily(machineType string) string {
	parts := strings.SplitN(strings.TrimSpace(machineType), "-", 2)
	if len(parts) == 0 || parts[0] == "" {
		return ""
	}
	return strings.ToUpper(parts[0])
}

func formatIntWithCommas(n int) string {
	if n < 0 {
		return "-" + formatIntWithCommas(-n)
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return formatIntWithCommas(n/1000) + "," + fmt.Sprintf("%03d", n%1000)
}

func FormatMachineTypeSummary(mt *MachineType) string {
	if mt == nil {
		return ""
	}
	memoryGB := mt.MemoryMB / 1024
	if memoryGB < 1 && mt.MemoryMB > 0 {
		memoryGB = 1
	}
	return fmt.Sprintf("%s vCPU, %s GB memory", formatIntWithCommas(mt.GuestCPUs), formatIntWithCommas(memoryGB))
}

const hoursPerMonth = 730

const (
	defaultVCPUHourUSDStandard     = 0.033
	defaultMemoryGBHourUSDStandard = 0.004
	defaultVCPUHourUSDSpot         = 0.0099
	defaultMemoryGBHourUSDSpot     = 0.0012
)

type regionRates struct {
	VCPUHourUSD     float64
	MemoryGBHourUSD float64
}

var (
	defaultStandardRates = regionRates{
		VCPUHourUSD: defaultVCPUHourUSDStandard, MemoryGBHourUSD: defaultMemoryGBHourUSDStandard,
	}
	defaultSpotRates = regionRates{
		VCPUHourUSD: defaultVCPUHourUSDSpot, MemoryGBHourUSD: defaultMemoryGBHourUSDSpot,
	}
	regionalStandardRates = map[string]regionRates{
		"us-central1":     {VCPUHourUSD: defaultVCPUHourUSDStandard, MemoryGBHourUSD: defaultMemoryGBHourUSDStandard},
		"us-east1":        {VCPUHourUSD: defaultVCPUHourUSDStandard, MemoryGBHourUSD: defaultMemoryGBHourUSDStandard},
		"us-west1":        {VCPUHourUSD: defaultVCPUHourUSDStandard, MemoryGBHourUSD: defaultMemoryGBHourUSDStandard},
		"europe-west1":    {VCPUHourUSD: 0.036, MemoryGBHourUSD: 0.0048},
		"asia-east1":      {VCPUHourUSD: 0.036, MemoryGBHourUSD: 0.0048},
		"asia-northeast1": {VCPUHourUSD: 0.037, MemoryGBHourUSD: 0.005},
	}
	regionalSpotRates = map[string]regionRates{
		"us-central1":     {VCPUHourUSD: defaultVCPUHourUSDSpot, MemoryGBHourUSD: defaultMemoryGBHourUSDSpot},
		"us-east1":        {VCPUHourUSD: defaultVCPUHourUSDSpot, MemoryGBHourUSD: defaultMemoryGBHourUSDSpot},
		"us-west1":        {VCPUHourUSD: defaultVCPUHourUSDSpot, MemoryGBHourUSD: defaultMemoryGBHourUSDSpot},
		"europe-west1":    {VCPUHourUSD: 0.0108, MemoryGBHourUSD: 0.00144},
		"asia-east1":      {VCPUHourUSD: 0.0108, MemoryGBHourUSD: 0.00144},
		"asia-northeast1": {VCPUHourUSD: 0.0111, MemoryGBHourUSD: 0.0015},
	}
)

func zoneToRegion(zone string) string {
	zone = strings.TrimSpace(zone)
	if zone == "" {
		return ""
	}
	if i := strings.LastIndex(zone, "-"); i > 0 {
		return zone[:i]
	}
	return zone
}

func getRates(region string, isSpot bool) regionRates {
	if isSpot {
		if r, ok := regionalSpotRates[region]; ok {
			return r
		}
		return defaultSpotRates
	}
	if r, ok := regionalStandardRates[region]; ok {
		return r
	}
	return defaultStandardRates
}

func monthlyEstimateFromMachineType(mt *MachineType, zone, provisioningModel string) float64 {
	if mt == nil {
		return 0
	}
	region := zoneToRegion(zone)
	isSpot := provisioningModel == string(ProvisioningSpot)
	rates := getRates(region, isSpot)
	memoryGB := float64(mt.MemoryMB) / 1024
	if memoryGB < 0 {
		memoryGB = 0
	}
	hourlyUSD := rates.VCPUHourUSD*float64(mt.GuestCPUs) + rates.MemoryGBHourUSD*memoryGB
	monthlyUSD := hourlyUSD * hoursPerMonth
	return float64(int(monthlyUSD*100+0.5)) / 100
}

func formatMonthlyEstimate(monthlyUSD float64) string {
	if monthlyUSD <= 0 {
		return ""
	}
	return fmt.Sprintf(" — ~US$%s/mo", formatFloatWithCommas(monthlyUSD))
}

func formatFloatWithCommas(x float64) string {
	s := fmt.Sprintf("%.2f", x)
	parts := strings.Split(s, ".")
	if len(parts) != 2 {
		return s
	}
	intPart := parts[0]
	if len(intPart) <= 3 {
		return s
	}
	var b strings.Builder
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteString(",")
		}
		b.WriteRune(c)
	}
	b.WriteString(".")
	b.WriteString(parts[1])
	return b.String()
}
func ListRegions(ctx context.Context, c Client) ([]Region, error) {
	cacheKey := "regions:" + c.ProjectID()
	if v, ok := cacheGet(cacheKey); ok {
		return v.([]Region), nil
	}

	path := fmt.Sprintf("projects/%s/regions", c.ProjectID())
	var all []Region
	var pageToken string
	for {
		body, err := c.Get(ctx, withPageToken(path, pageToken))
		if err != nil {
			return nil, err
		}
		var resp regionsListResp
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parse regions response: %w", err)
		}
		for _, it := range resp.Items {
			if it == nil {
				continue
			}
			all = append(all, regionFromAPI(it))
		}
		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}
	cacheSet(cacheKey, all)
	return all, nil
}

func ListZones(ctx context.Context, c Client, region string) ([]Zone, error) {
	region = strings.TrimSpace(region)
	regions, err := ListRegions(ctx, c)
	if err != nil {
		return nil, err
	}
	var out []Zone
	for _, r := range regions {
		if region != "" && r.Name != region {
			continue
		}
		for _, zoneName := range r.Zones {
			out = append(out, Zone{
				Name:   zoneName,
				Status: r.Status,
				Region: r.Name,
			})
		}
	}
	return out, nil
}

func ListMachineTypes(ctx context.Context, c Client, zone string) ([]MachineType, error) {
	zone = strings.TrimSpace(zone)
	cacheKey := "machineTypes:" + c.ProjectID() + ":" + zone
	if v, ok := cacheGet(cacheKey); ok {
		return v.([]MachineType), nil
	}

	path := fmt.Sprintf("projects/%s/zones/%s/machineTypes", c.ProjectID(), zone)
	var all []MachineType
	var pageToken string
	for {
		body, err := c.Get(ctx, withPageToken(path, pageToken))
		if err != nil {
			return nil, err
		}
		var resp machineTypesListResp
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parse machineTypes response: %w", err)
		}
		for _, it := range resp.Items {
			if it == nil {
				continue
			}
			all = append(all, machineTypeFromAPI(it))
		}
		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}
	cacheSet(cacheKey, all)
	return all, nil
}

func GetMachineType(ctx context.Context, c Client, zone, machineType string) (*MachineType, error) {
	zone = strings.TrimSpace(zone)
	machineType = strings.TrimSpace(machineType)
	path := fmt.Sprintf("projects/%s/zones/%s/machineTypes/%s", c.ProjectID(), zone, machineType)
	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var it machineTypeItem
	if err := json.Unmarshal(body, &it); err != nil {
		return nil, fmt.Errorf("parse machineType response: %w", err)
	}
	mt := machineTypeFromAPI(&it)
	return &mt, nil
}

func ListMachineFamilies(ctx context.Context, c Client, zone string) ([]MachineFamily, error) {
	types, err := ListMachineTypes(ctx, c, zone)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	var out []MachineFamily
	for _, mt := range types {
		if mt.Family == "" {
			continue
		}
		if _, ok := seen[mt.Family]; ok {
			continue
		}
		seen[mt.Family] = struct{}{}
		out = append(out, MachineFamily{Family: mt.Family})
	}
	return out, nil
}

func ListRegionResources(ctx context.Context, c Client) ([]core.IntegrationResource, error) {
	list, err := ListRegions(ctx, c)
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(list))
	for _, r := range list {
		out = append(out, core.IntegrationResource{Type: ResourceTypeRegion, Name: r.Name, ID: r.Name})
	}
	return out, nil
}

func ListZoneResources(ctx context.Context, c Client, region string) ([]core.IntegrationResource, error) {
	list, err := ListZones(ctx, c, region)
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(list))
	for _, z := range list {
		out = append(out, core.IntegrationResource{Type: ResourceTypeZone, Name: z.Name, ID: z.Name})
	}
	return out, nil
}

func ListMachineFamilyResources(ctx context.Context, c Client, zone string) ([]core.IntegrationResource, error) {
	if strings.TrimSpace(zone) == "" {
		return []core.IntegrationResource{}, nil
	}
	list, err := ListMachineFamilies(ctx, c, zone)
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(list))
	for _, f := range list {
		out = append(out, core.IntegrationResource{Type: ResourceTypeMachineFamily, Name: f.Family, ID: f.Family})
	}
	return out, nil
}

func ListMachineTypeResources(ctx context.Context, c Client, zone, machineFamily string) ([]core.IntegrationResource, error) {
	if strings.TrimSpace(zone) == "" {
		return []core.IntegrationResource{}, nil
	}
	list, err := ListMachineTypes(ctx, c, zone)
	if err != nil {
		return nil, err
	}
	machineFamily = strings.TrimSpace(machineFamily)
	out := make([]core.IntegrationResource, 0, len(list))
	for _, mt := range list {
		if machineFamily != "" && mt.Family != machineFamily {
			continue
		}
		summary := FormatMachineTypeSummary(&mt)
		name := mt.Name
		if summary != "" {
			name = fmt.Sprintf("%s (%s)", mt.Name, summary)
		}
		if monthly := monthlyEstimateFromMachineType(&mt, zone, string(ProvisioningStandard)); monthly > 0 {
			name += formatMonthlyEstimate(monthly)
		}
		out = append(out, core.IntegrationResource{Type: ResourceTypeMachineType, Name: name, ID: mt.Name})
	}
	return out, nil
}

// ubuntuLTSFamilyOrder defines sort order for Ubuntu LTS families (modern first).
var ubuntuLTSFamilyOrder = []string{"ubuntu-24", "ubuntu-22", "ubuntu-20"}

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

// publicImageProjects lists all GCP public image project IDs (all images shown to user).
var publicImageProjects = []string{
	"centos-cloud",
	"cos-cloud",
	"debian-cloud",
	"deeplearning-platform-release",
	"fedora-cloud",
	"opensuse-cloud",
	"oracle-linux-cloud",
	"rhel-cloud",
	"rhel-sap-cloud",
	"rocky-linux-accelerator-cloud",
	"rocky-linux-cloud",
	"suse-byos-cloud",
	"suse-cloud",
	"suse-sap-cloud",
	"ubuntu-os-accelerator-images",
	"ubuntu-os-cloud",
	"ubuntu-os-pro-cloud",
	"windows-cloud",
	"windows-sql-cloud",
}

// maxPublicImagesPerPage is the GCP API page size when listing public images (max 500).
const maxPublicImagesPerPage = 500

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

// sortPublicImagesForProject sorts images so modern Ubuntu LTS appear first (ubuntu-os-cloud and related).
func sortPublicImagesForProject(images []Image) {
	slices.SortFunc(images, func(a, b Image) int {
		rankA := ubuntuImageSortRank(a)
		rankB := ubuntuImageSortRank(b)
		if rankA != rankB {
			return rankA - rankB
		}
		return strings.Compare(b.Name, a.Name)
	})
}

func ubuntuImageSortRank(img Image) int {
	family := strings.ToLower(img.Family)
	name := strings.ToLower(img.Name)
	source := family
	if source == "" {
		source = name
	}
	for i, prefix := range ubuntuLTSFamilyOrder {
		if strings.Contains(source, prefix) {
			return i
		}
	}
	return len(ubuntuLTSFamilyOrder)
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
	var all []Image
	var pageToken string
	for {
		body, err := c.Get(ctx, withMaxResults(path, maxPublicImagesPerPage, pageToken))
		if err != nil {
			return nil, fmt.Errorf("list public images for %s: %w", project, err)
		}
		var resp imagesListResp
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parse images response: %w", err)
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
	sortPublicImagesForProject(all)
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
	if strings.TrimSpace(zone) == "" {
		return []core.IntegrationResource{}, nil
	}
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
	if strings.TrimSpace(zone) == "" {
		return []core.IntegrationResource{}, nil
	}
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
	if strings.TrimSpace(region) == "" {
		return []core.IntegrationResource{}, nil
	}
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

const (
	ResourceTypeNetwork    = "network"
	ResourceTypeSubnetwork = "subnetwork"
	ResourceTypeAddress    = "address"
	ResourceTypeFirewall   = "firewall"
)
const AddressTypeExternal = "EXTERNAL"

type Network struct {
	Name     string `json:"name"`
	SelfLink string `json:"selfLink"`
}

type Subnetwork struct {
	Name     string `json:"name"`
	Region   string `json:"region"`
	SelfLink string `json:"selfLink"`
}

type Address struct {
	Name        string `json:"name"`
	Address     string `json:"address"`
	Region      string `json:"region"`
	SelfLink    string `json:"selfLink"`
	Status      string `json:"status"`
	AddressType string `json:"addressType"`
}

type Firewall struct {
	Name     string `json:"name"`
	SelfLink string `json:"selfLink"`
	Network  string `json:"network,omitempty"`
}

type networksListResp struct {
	Items         []*networkItem `json:"items"`
	NextPageToken string         `json:"nextPageToken"`
}

type networkItem struct {
	Name     string `json:"name"`
	SelfLink string `json:"selfLink"`
}

type subnetworksListResp struct {
	Items         []*subnetworkItem `json:"items"`
	NextPageToken string            `json:"nextPageToken"`
}

type subnetworkItem struct {
	Name     string `json:"name"`
	Region   string `json:"region"`
	SelfLink string `json:"selfLink"`
}

type addressesListResp struct {
	Items         []*addressItem `json:"items"`
	NextPageToken string         `json:"nextPageToken"`
}

type addressItem struct {
	Name        string `json:"name"`
	Address     string `json:"address"`
	Region      string `json:"region"`
	SelfLink    string `json:"selfLink"`
	Status      string `json:"status"`
	AddressType string `json:"addressType"`
}

type firewallsListResp struct {
	Items         []*firewallItem `json:"items"`
	NextPageToken string          `json:"nextPageToken"`
}

type firewallItem struct {
	Name       string   `json:"name"`
	SelfLink   string   `json:"selfLink"`
	Network    string   `json:"network"`
	TargetTags []string `json:"targetTags"`
}

func ensureProject(project string, c Client) string {
	if project == "" {
		return c.ProjectID()
	}
	return project
}

func ensureNonEmptyRegion(region, errMsg string) (string, error) {
	r := strings.TrimSpace(region)
	if r == "" {
		return "", fmt.Errorf("%s", errMsg)
	}
	return r, nil
}

func defaultRegion(itemRegion, fallback string) string {
	if strings.TrimSpace(itemRegion) != "" {
		return itemRegion
	}
	return fallback
}
func ListNetworks(ctx context.Context, c Client, project string) ([]Network, error) {
	project = ensureProject(project, c)
	path := fmt.Sprintf("projects/%s/global/networks", project)
	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var resp networksListResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse networks list: %w", err)
	}
	out := make([]Network, 0, len(resp.Items))
	for _, n := range resp.Items {
		if n == nil {
			continue
		}
		out = append(out, Network{Name: n.Name, SelfLink: n.SelfLink})
	}
	return out, nil
}

func ListSubnetworks(ctx context.Context, c Client, project, region string) ([]Subnetwork, error) {
	project = ensureProject(project, c)
	region, err := ensureNonEmptyRegion(region, "region is required for listing subnetworks")
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("projects/%s/regions/%s/subnetworks", project, region)
	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var resp subnetworksListResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse subnetworks list: %w", err)
	}
	out := make([]Subnetwork, 0, len(resp.Items))
	for _, s := range resp.Items {
		if s == nil {
			continue
		}
		out = append(out, Subnetwork{
			Name:     s.Name,
			Region:   defaultRegion(s.Region, region),
			SelfLink: s.SelfLink,
		})
	}
	return out, nil
}

func ListAddresses(ctx context.Context, c Client, project, region string) ([]Address, error) {
	project = ensureProject(project, c)
	region, err := ensureNonEmptyRegion(region, "region is required for listing addresses")
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("projects/%s/regions/%s/addresses", project, region)
	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var resp addressesListResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse addresses list: %w", err)
	}
	out := make([]Address, 0, len(resp.Items))
	for _, a := range resp.Items {
		if a == nil {
			continue
		}
		out = append(out, Address{
			Name:        a.Name,
			Address:     a.Address,
			Region:      defaultRegion(a.Region, region),
			SelfLink:    a.SelfLink,
			Status:      a.Status,
			AddressType: a.AddressType,
		})
	}
	return out, nil
}

func ListFirewalls(ctx context.Context, c Client, project string) ([]Firewall, error) {
	project = ensureProject(project, c)
	path := fmt.Sprintf("projects/%s/global/firewalls", project)
	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var resp firewallsListResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse firewalls list: %w", err)
	}
	out := make([]Firewall, 0, len(resp.Items))
	for _, f := range resp.Items {
		if f == nil {
			continue
		}
		out = append(out, Firewall{Name: f.Name, SelfLink: f.SelfLink, Network: f.Network})
	}
	return out, nil
}

func ListNetworkResources(ctx context.Context, c Client, project string) ([]core.IntegrationResource, error) {
	list, err := ListNetworks(ctx, c, project)
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(list))
	for _, n := range list {
		out = append(out, core.IntegrationResource{Type: ResourceTypeNetwork, Name: n.Name, ID: n.SelfLink})
	}
	return out, nil
}

func ListSubnetworkResources(ctx context.Context, c Client, project, region string) ([]core.IntegrationResource, error) {
	if strings.TrimSpace(region) == "" {
		return []core.IntegrationResource{}, nil
	}
	list, err := ListSubnetworks(ctx, c, project, region)
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(list))
	for _, s := range list {
		label := s.Name
		if s.Region != "" {
			label = fmt.Sprintf("%s (%s)", s.Name, s.Region)
		}
		out = append(out, core.IntegrationResource{Type: ResourceTypeSubnetwork, Name: label, ID: s.SelfLink})
	}
	return out, nil
}

func ListAddressResources(ctx context.Context, c Client, project, region string) ([]core.IntegrationResource, error) {
	if strings.TrimSpace(region) == "" {
		return []core.IntegrationResource{}, nil
	}
	list, err := ListAddresses(ctx, c, project, region)
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(list))
	for _, a := range list {
		if a.AddressType != AddressTypeExternal {
			continue
		}
		label := a.Name
		if a.Address != "" {
			label = fmt.Sprintf("%s (%s)", a.Name, a.Address)
		}
		id := a.SelfLink
		if a.Address != "" {
			id = a.Address
		}
		out = append(out, core.IntegrationResource{Type: ResourceTypeAddress, Name: label, ID: id})
	}
	return out, nil
}

func ListFirewallResources(ctx context.Context, c Client, project string) ([]core.IntegrationResource, error) {
	list, err := ListFirewalls(ctx, c, project)
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(list))
	for _, f := range list {
		label := f.Name
		if f.Network != "" {
			label = fmt.Sprintf("%s (%s)", f.Name, lastSegment(f.Network))
		}
		id := f.SelfLink
		if id == "" {
			id = f.Name
		}
		out = append(out, core.IntegrationResource{Type: ResourceTypeFirewall, Name: label, ID: id})
	}
	return out, nil
}
