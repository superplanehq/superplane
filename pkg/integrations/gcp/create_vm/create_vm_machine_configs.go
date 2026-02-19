package createvm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

func CreateVMMachineConfigFields() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "instanceName",
			Label:       "Instance name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Start with a letter; use only a-z, 0-9, and hyphens; end with a letter or digit. 1 to 63 characters length.",
			Placeholder: "e.g. my-vm-01",
		},
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "GCP project ID for the VM. Leave empty to use the integration's project.",
			Placeholder: "Leave empty to use integration project",
		},
		{
			Name:        "region",
			Label:       "Region",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "GCP region (e.g. us-central1). Used to filter zones.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeRegion,
				},
			},
		},
		{
			Name:        "zone",
			Label:       "Zone",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "GCP zone within the selected region (e.g. us-central1-a).",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeZone,
					Parameters: []configuration.ParameterRef{
						{Name: "region", ValueFrom: &configuration.ParameterValueFrom{Field: "region"}},
					},
				},
			},
		},
		{
			Name:        "machineFamily",
			Label:       "Machine family",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional. Filter machine types by family (e.g. E2). Leave empty to see all.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeMachineFamily,
					Parameters: []configuration.ParameterRef{
						{Name: "zone", ValueFrom: &configuration.ParameterValueFrom{Field: "zone"}},
					},
				},
			},
		},
		{
			Name:        "machineType",
			Label:       "Machine type",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Machine type for the VM (e.g. e2-medium, n2-standard-4).",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeMachineType,
					Parameters: []configuration.ParameterRef{
						{Name: "zone", ValueFrom: &configuration.ParameterValueFrom{Field: "zone"}},
						{Name: "machineFamily", ValueFrom: &configuration.ParameterValueFrom{Field: "machineFamily"}},
					},
				},
			},
		},
		{
			Name:        "provisioningModel",
			Label:       "Provisioning model",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Standard (on-demand) or Spot (preemptible).",
			Default:     string(ProvisioningStandard),
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Standard", Value: string(ProvisioningStandard)},
						{Label: "Spot", Value: string(ProvisioningSpot)},
					},
				},
			},
		},
	}
}

const cacheTTL = 24 * time.Hour

const (
	ResourceTypeRegion        = "region"
	ResourceTypeZone          = "zone"
	ResourceTypeMachineFamily = "machineFamily"
	ResourceTypeMachineType   = "machineType"
)

type ProvisioningModel string

const (
	ProvisioningStandard ProvisioningModel = "STANDARD"
	ProvisioningSpot     ProvisioningModel = "SPOT"
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
