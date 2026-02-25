package compute

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_lastSegment(t *testing.T) {
	assert.Equal(t, "", lastSegment(""))
	assert.Equal(t, "e2-medium", lastSegment("zones/us-central1-a/machineTypes/e2-medium"))
	assert.Equal(t, "name", lastSegment("a/b/c/name"))
	assert.Equal(t, "only", lastSegment("only"))
}

func Test_DeriveFamily(t *testing.T) {
	assert.Equal(t, "E2", DeriveFamily("e2-medium"))
	assert.Equal(t, "N2", DeriveFamily("n2-standard-4"))
	assert.Equal(t, "C2", DeriveFamily("c2-standard-8"))
	assert.Equal(t, "", DeriveFamily(""))
	assert.Equal(t, "", DeriveFamily("   "))
	assert.Equal(t, "N2", DeriveFamily("  n2-standard-4  "))
}

func Test_zoneToRegion(t *testing.T) {
	assert.Equal(t, "us-central1", zoneToRegion("us-central1-a"))
	assert.Equal(t, "europe-west1", zoneToRegion("europe-west1-b"))
	assert.Equal(t, "", zoneToRegion(""))
	assert.Equal(t, "", zoneToRegion("   "))
	assert.Equal(t, "us-east1", zoneToRegion("us-east1-c"))
	t.Run("no hyphen returns zone as-is", func(t *testing.T) {
		assert.Equal(t, "region", zoneToRegion("region"))
	})
}

func Test_FormatMachineTypeSummary(t *testing.T) {
	assert.Equal(t, "", FormatMachineTypeSummary(nil))
	t.Run("formats vCPU and memory", func(t *testing.T) {
		mt := &MachineType{GuestCPUs: 4, MemoryMB: 16384}
		assert.Equal(t, "4 vCPU, 16 GB memory", FormatMachineTypeSummary(mt))
	})
	t.Run("small memory rounds to 1 GB", func(t *testing.T) {
		mt := &MachineType{GuestCPUs: 1, MemoryMB: 512}
		assert.Equal(t, "1 vCPU, 1 GB memory", FormatMachineTypeSummary(mt))
	})
	t.Run("thousands with commas", func(t *testing.T) {
		mt := &MachineType{GuestCPUs: 288, MemoryMB: 1105920} // 1080 GB
		assert.Equal(t, "288 vCPU, 1,080 GB memory", FormatMachineTypeSummary(mt))
	})
}

type mockOSClient struct {
	projectID string
	get       func(ctx context.Context, path string) ([]byte, error)
}

func (m *mockOSClient) Get(ctx context.Context, path string) ([]byte, error) {
	if m.get != nil {
		return m.get(ctx, path)
	}
	return nil, errors.New("not implemented")
}

func (m *mockOSClient) Post(ctx context.Context, path string, body any) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (m *mockOSClient) GetURL(ctx context.Context, fullURL string) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (m *mockOSClient) ProjectID() string {
	return m.projectID
}
func Test_isPublicImageProject(t *testing.T) {
	assert.True(t, isPublicImageProject("debian-cloud"))
	assert.True(t, isPublicImageProject("ubuntu-os-cloud"))
	assert.True(t, isPublicImageProject("windows-cloud"))
	assert.True(t, isPublicImageProject("rocky-linux-cloud"))
	assert.True(t, isPublicImageProject("deeplearning-platform-release"))
	assert.True(t, isPublicImageProject("ubuntu-os-pro-cloud"))
	assert.True(t, isPublicImageProject("suse-byos-cloud"))
	assert.True(t, isPublicImageProject("rhel-cloud"))
	assert.False(t, isPublicImageProject("my-custom-project"))
	assert.False(t, isPublicImageProject(""))
}

func Test_withMaxResults(t *testing.T) {
	t.Run("maxResults <= 0 returns path with only pageToken", func(t *testing.T) {
		assert.Equal(t, "path", withMaxResults("path", 0, ""))
		assert.Equal(t, "path?pageToken=token", withMaxResults("path", 0, "token"))
	})
	t.Run("maxResults > 0 adds query", func(t *testing.T) {
		assert.Equal(t, "path?maxResults=100", withMaxResults("path", 100, ""))
		assert.Equal(t, "path?maxResults=50&pageToken=next", withMaxResults("path", 50, "next"))
	})
	t.Run("existing query uses &", func(t *testing.T) {
		assert.Equal(t, "path?foo=bar&maxResults=100", withMaxResults("path?foo=bar", 100, ""))
		assert.Equal(t, "path?foo=bar&maxResults=100&pageToken=t", withMaxResults("path?foo=bar", 100, "t"))
	})
}

func Test_imageItemToImage(t *testing.T) {
	assert.Equal(t, Image{}, imageItemToImage(nil))
	it := &imageItem{
		Name:        "debian-12",
		Family:      "debian-12",
		Description: "Debian 12",
		SelfLink:    "https://www.googleapis.com/.../debian-12",
	}
	img := imageItemToImage(it)
	assert.Equal(t, "debian-12", img.Name)
	assert.Equal(t, "debian-12", img.Family)
	assert.Equal(t, "Debian 12", img.Description)
	assert.Equal(t, "https://www.googleapis.com/.../debian-12", img.SelfLink)
}

func Test_imageSelfLinkOrName(t *testing.T) {
	assert.Equal(t, "my-image", imageSelfLinkOrName(Image{Name: "my-image"}))
	assert.Equal(t, "https://self/link", imageSelfLinkOrName(Image{Name: "n", SelfLink: "https://self/link"}))
	assert.Equal(t, "", imageSelfLinkOrName(Image{}))
}

func Test_ListPublicImages(t *testing.T) {
	ctx := context.Background()

	t.Run("empty project returns nil nil", func(t *testing.T) {
		c := &mockOSClient{projectID: "my-project"}
		list, err := ListPublicImages(ctx, c, "")
		require.NoError(t, err)
		assert.Nil(t, list)
	})

	t.Run("whitespace project returns nil nil", func(t *testing.T) {
		c := &mockOSClient{projectID: "my-project"}
		list, err := ListPublicImages(ctx, c, "   ")
		require.NoError(t, err)
		assert.Nil(t, list)
	})

	t.Run("non-public project returns nil nil", func(t *testing.T) {
		c := &mockOSClient{projectID: "my-project"}
		list, err := ListPublicImages(ctx, c, "my-private-project")
		require.NoError(t, err)
		assert.Nil(t, list)
	})

	t.Run("client error returns error", func(t *testing.T) {
		c := &mockOSClient{
			projectID: "my-project",
			get: func(_ context.Context, path string) ([]byte, error) {
				return nil, errors.New("network error")
			},
		}
		list, err := ListPublicImages(ctx, c, "debian-cloud")
		require.Error(t, err)
		assert.Nil(t, list)
		assert.Contains(t, err.Error(), "list public images")
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		c := &mockOSClient{
			projectID: "my-project",
			get: func(_ context.Context, path string) ([]byte, error) {
				return []byte("not json"), nil
			},
		}
		list, err := ListPublicImages(ctx, c, "debian-cloud")
		require.Error(t, err)
		assert.Nil(t, list)
		assert.Contains(t, err.Error(), "parse images")
	})

	t.Run("success returns images", func(t *testing.T) {
		resp := imagesListResp{
			Items: []*imageItem{
				{Name: "debian-12", Family: "debian-12", SelfLink: "https://.../debian-12"},
				{Name: "debian-11", Family: "debian-11", SelfLink: "https://.../debian-11"},
			},
		}
		body, _ := json.Marshal(resp)
		c := &mockOSClient{
			projectID: "my-project",
			get: func(_ context.Context, path string) ([]byte, error) {
				return body, nil
			},
		}
		list, err := ListPublicImages(ctx, c, "debian-cloud")
		require.NoError(t, err)
		require.Len(t, list, 2)
		assert.Equal(t, "debian-12", list[0].Name)
		assert.Equal(t, "debian-11", list[1].Name)
	})

	t.Run("ubuntu images sorted with modern LTS first", func(t *testing.T) {
		resp := imagesListResp{
			Items: []*imageItem{
				{Name: "ubuntu-1204-precise-v20150625", Family: "ubuntu-1204-precise", SelfLink: "https://.../ubuntu-1204-precise"},
				{Name: "ubuntu-2204-jammy-v20240101", Family: "ubuntu-2204-lts", SelfLink: "https://.../ubuntu-2204-jammy"},
				{Name: "ubuntu-2404-noble-v20240601", Family: "ubuntu-2404-lts", SelfLink: "https://.../ubuntu-2404-noble"},
			},
		}
		body, _ := json.Marshal(resp)
		c := &mockOSClient{
			projectID: "my-project",
			get: func(_ context.Context, path string) ([]byte, error) {
				return body, nil
			},
		}
		list, err := ListPublicImages(ctx, c, "ubuntu-os-cloud")
		require.NoError(t, err)
		require.Len(t, list, 3)
		assert.Equal(t, "ubuntu-2404-noble-v20240601", list[0].Name, "ubuntu-24 first")
		assert.Equal(t, "ubuntu-2204-jammy-v20240101", list[1].Name, "ubuntu-22 second")
		assert.Equal(t, "ubuntu-1204-precise-v20150625", list[2].Name, "ubuntu-12 last")
	})

	t.Run("pagination fetches all pages", func(t *testing.T) {
		page1 := imagesListResp{
			Items:         []*imageItem{{Name: "img-1", Family: "f1", SelfLink: "https://.../img-1"}},
			NextPageToken: "next",
		}
		page2 := imagesListResp{Items: []*imageItem{{Name: "img-2", Family: "f2", SelfLink: "https://.../img-2"}}}
		body1, _ := json.Marshal(page1)
		body2, _ := json.Marshal(page2)
		callCount := 0
		c := &mockOSClient{
			projectID: "my-project",
			get: func(_ context.Context, path string) ([]byte, error) {
				callCount++
				if callCount == 1 {
					return body1, nil
				}
				return body2, nil
			},
		}
		list, err := ListPublicImages(ctx, c, "rhel-cloud")
		require.NoError(t, err)
		require.Len(t, list, 2)
		// sortPublicImagesForProject sorts by name descending when ranks are equal, so img-2 before img-1
		assert.Equal(t, "img-2", list[0].Name)
		assert.Equal(t, "img-1", list[1].Name)
		assert.Equal(t, 2, callCount)
	})
}

func Test_GetImageFromFamily(t *testing.T) {
	ctx := context.Background()

	t.Run("empty family returns error", func(t *testing.T) {
		c := &mockOSClient{projectID: "my-project"}
		img, err := GetImageFromFamily(ctx, c, "debian-cloud", "")
		require.Error(t, err)
		assert.Nil(t, img)
		assert.Contains(t, err.Error(), "family is required")
	})

	t.Run("whitespace family returns error", func(t *testing.T) {
		c := &mockOSClient{projectID: "my-project"}
		img, err := GetImageFromFamily(ctx, c, "debian-cloud", "   ")
		require.Error(t, err)
		assert.Nil(t, img)
	})

	t.Run("client error returned", func(t *testing.T) {
		c := &mockOSClient{
			projectID: "my-project",
			get: func(_ context.Context, path string) ([]byte, error) {
				return nil, errors.New("get failed")
			},
		}
		img, err := GetImageFromFamily(ctx, c, "debian-cloud", "debian-12")
		require.Error(t, err)
		assert.Nil(t, img)
	})

	t.Run("success returns image", func(t *testing.T) {
		it := imageItem{Name: "debian-12-20240101", Family: "debian-12", SelfLink: "https://.../debian-12"}
		body, _ := json.Marshal(it)
		c := &mockOSClient{
			projectID: "my-project",
			get: func(_ context.Context, path string) ([]byte, error) {
				return body, nil
			},
		}
		img, err := GetImageFromFamily(ctx, c, "debian-cloud", "debian-12")
		require.NoError(t, err)
		require.NotNil(t, img)
		assert.Equal(t, "debian-12-20240101", img.Name)
		assert.Equal(t, "debian-12", img.Family)
	})

	t.Run("empty project uses client ProjectID", func(t *testing.T) {
		it := imageItem{Name: "img", Family: "fam"}
		body, _ := json.Marshal(it)
		c := &mockOSClient{
			projectID: "default-project",
			get: func(_ context.Context, path string) ([]byte, error) {
				assert.Contains(t, path, "projects/default-project/global/images/family/fam")
				return body, nil
			},
		}
		img, err := GetImageFromFamily(ctx, c, "", "fam")
		require.NoError(t, err)
		require.NotNil(t, img)
	})
}

func Test_ListCustomImages(t *testing.T) {
	ctx := context.Background()

	t.Run("client error returned", func(t *testing.T) {
		c := &mockOSClient{
			projectID: "my-project",
			get: func(_ context.Context, path string) ([]byte, error) {
				return nil, errors.New("get failed")
			},
		}
		list, err := ListCustomImages(ctx, c, "my-project")
		require.Error(t, err)
		assert.Nil(t, list)
	})

	t.Run("success returns images", func(t *testing.T) {
		resp := imagesListResp{
			Items: []*imageItem{
				{Name: "custom-img-1", SelfLink: "https://.../custom-img-1"},
			},
		}
		body, _ := json.Marshal(resp)
		c := &mockOSClient{
			projectID: "my-project",
			get: func(_ context.Context, path string) ([]byte, error) {
				return body, nil
			},
		}
		list, err := ListCustomImages(ctx, c, "my-project")
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, "custom-img-1", list[0].Name)
	})

	t.Run("empty project uses client ProjectID", func(t *testing.T) {
		resp := imagesListResp{Items: []*imageItem{}}
		body, _ := json.Marshal(resp)
		c := &mockOSClient{
			projectID: "default-proj",
			get: func(_ context.Context, path string) ([]byte, error) {
				assert.Contains(t, path, "projects/default-proj/global/images")
				return body, nil
			},
		}
		list, err := ListCustomImages(ctx, c, "")
		require.NoError(t, err)
		assert.Empty(t, list)
	})
}

func Test_ListPublicImageResources(t *testing.T) {
	ctx := context.Background()

	t.Run("delegates to ListPublicImages and formats", func(t *testing.T) {
		resp := imagesListResp{
			Items: []*imageItem{
				{Name: "win-2022", Family: "windows-2022", SelfLink: "https://.../win-2022"},
			},
		}
		body, _ := json.Marshal(resp)
		c := &mockOSClient{
			projectID: "my-project",
			get: func(_ context.Context, path string) ([]byte, error) {
				return body, nil
			},
		}
		resources, err := ListPublicImageResources(ctx, c, "windows-cloud")
		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, ResourceTypePublicImages, resources[0].Type)
		assert.Equal(t, "win-2022 (windows-2022)", resources[0].Name)
		assert.Equal(t, "https://.../win-2022", resources[0].ID)
	})

	t.Run("image without family uses name only", func(t *testing.T) {
		resp := imagesListResp{
			Items: []*imageItem{
				{Name: "centos-9", Family: "", SelfLink: ""},
			},
		}
		body, _ := json.Marshal(resp)
		c := &mockOSClient{
			projectID: "my-project",
			get: func(_ context.Context, path string) ([]byte, error) {
				return body, nil
			},
		}
		resources, err := ListPublicImageResources(ctx, c, "centos-cloud")
		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "centos-9", resources[0].Name)
		assert.Equal(t, "centos-9", resources[0].ID)
	})

	t.Run("ListPublicImages error propagated", func(t *testing.T) {
		c := &mockOSClient{
			projectID: "my-project",
			get: func(_ context.Context, path string) ([]byte, error) {
				return nil, errors.New("api error")
			},
		}
		_, err := ListPublicImageResources(ctx, c, "rocky-linux-cloud")
		require.Error(t, err)
	})
}

func Test_ListCustomImageResources(t *testing.T) {
	ctx := context.Background()

	t.Run("delegates to ListCustomImages and formats", func(t *testing.T) {
		resp := imagesListResp{
			Items: []*imageItem{
				{Name: "my-custom-image", SelfLink: "https://.../my-custom-image"},
			},
		}
		body, _ := json.Marshal(resp)
		c := &mockOSClient{
			projectID: "my-project",
			get: func(_ context.Context, path string) ([]byte, error) {
				return body, nil
			},
		}
		resources, err := ListCustomImageResources(ctx, c, "custom-images-project")
		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, ResourceTypeCustomImages, resources[0].Type)
		assert.Equal(t, "my-custom-image", resources[0].Name)
		assert.Equal(t, "https://.../my-custom-image", resources[0].ID)
	})

	t.Run("ListCustomImages error propagated", func(t *testing.T) {
		c := &mockOSClient{
			projectID: "my-project",
			get: func(_ context.Context, path string) ([]byte, error) {
				return nil, errors.New("api error")
			},
		}
		_, err := ListCustomImageResources(ctx, c, "error-test-project")
		require.Error(t, err)
	})
}
func Test_isAllowedBootDiskType(t *testing.T) {
	assert.True(t, isAllowedBootDiskType("pd-balanced"))
	assert.True(t, isAllowedBootDiskType("pd-ssd"))
	assert.True(t, isAllowedBootDiskType("pd-standard"))
	assert.False(t, isAllowedBootDiskType("local-ssd"))
	assert.False(t, isAllowedBootDiskType(""))
}
