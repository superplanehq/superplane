package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ResourceTypeAlertPolicy         = "alertPolicy"
	ResourceTypeNotificationChannel = "notificationChannel"
	ResourceTypeSnooze              = "snooze"
)

type snoozeListResponse struct {
	Snoozes []struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
	} `json:"snoozes"`
	NextPageToken string `json:"nextPageToken"`
}

type alertPolicyListResponse struct {
	AlertPolicies []struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
	} `json:"alertPolicies"`
	NextPageToken string `json:"nextPageToken"`
}

type notificationChannelListResponse struct {
	NotificationChannels []struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		Type        string `json:"type"`
	} `json:"notificationChannels"`
	NextPageToken string `json:"nextPageToken"`
}

// ListAlertingPolicyResources lists the alert policies in the project, keyed by
// resource name so the Get/Delete/Update components can target one directly.
func ListAlertingPolicyResources(ctx context.Context, c Client) ([]core.IntegrationResource, error) {
	project := c.ProjectID()
	if project == "" {
		return nil, nil
	}
	base := fmt.Sprintf("%s/projects/%s/alertPolicies?pageSize=500", monitoringBaseURL, project)

	var resources []core.IntegrationResource
	err := paginate(ctx, c, base, func(data []byte) (string, error) {
		var resp alertPolicyListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return "", fmt.Errorf("failed to parse alert policies response: %w", err)
		}
		for _, p := range resp.AlertPolicies {
			if p.Name == "" {
				continue
			}
			label := p.DisplayName
			if label == "" {
				label = lastSegment(p.Name)
			}
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeAlertPolicy,
				ID:   p.Name,
				Name: label,
			})
		}
		return resp.NextPageToken, nil
	})
	return resources, err
}

// ListNotificationChannelResources lists the notification channels in the
// project so they can be attached to a policy.
func ListNotificationChannelResources(ctx context.Context, c Client) ([]core.IntegrationResource, error) {
	project := c.ProjectID()
	if project == "" {
		return nil, nil
	}
	base := fmt.Sprintf("%s/projects/%s/notificationChannels?pageSize=500", monitoringBaseURL, project)

	var resources []core.IntegrationResource
	err := paginate(ctx, c, base, func(data []byte) (string, error) {
		var resp notificationChannelListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return "", fmt.Errorf("failed to parse notification channels response: %w", err)
		}
		for _, ch := range resp.NotificationChannels {
			if ch.Name == "" {
				continue
			}
			label := ch.DisplayName
			switch {
			case label != "" && ch.Type != "":
				label = fmt.Sprintf("%s (%s)", label, ch.Type)
			case label == "":
				label = lastSegment(ch.Name)
			}
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeNotificationChannel,
				ID:   ch.Name,
				Name: label,
			})
		}
		return resp.NextPageToken, nil
	})
	return resources, err
}

// ListSnoozeResources lists the snoozes in the project so the Get/Expire
// components can target one directly.
func ListSnoozeResources(ctx context.Context, c Client) ([]core.IntegrationResource, error) {
	project := c.ProjectID()
	if project == "" {
		return nil, nil
	}
	base := fmt.Sprintf("%s/projects/%s/snoozes?pageSize=500", monitoringBaseURL, project)

	var resources []core.IntegrationResource
	err := paginate(ctx, c, base, func(data []byte) (string, error) {
		var resp snoozeListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return "", fmt.Errorf("failed to parse snoozes response: %w", err)
		}
		for _, s := range resp.Snoozes {
			if s.Name == "" {
				continue
			}
			label := s.DisplayName
			if label == "" {
				label = lastSegment(s.Name)
			}
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeSnooze,
				ID:   s.Name,
				Name: label,
			})
		}
		return resp.NextPageToken, nil
	})
	return resources, err
}

// paginate walks a paginated monitoring list endpoint, invoking handle for each
// page and following nextPageToken until it is empty.
func paginate(ctx context.Context, c Client, baseURL string, handle func(data []byte) (string, error)) error {
	pageURL := baseURL
	for {
		data, err := c.GetURL(ctx, pageURL)
		if err != nil {
			return err
		}
		token, err := handle(data)
		if err != nil {
			return err
		}
		if token == "" {
			break
		}
		pageURL = baseURL + "&pageToken=" + url.QueryEscape(token)
	}
	return nil
}
