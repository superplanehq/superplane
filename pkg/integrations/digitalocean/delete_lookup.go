package digitalocean

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type dropletDeleteTarget struct {
	ID   int
	Name string
}

type appDeleteTarget struct {
	ID   string
	Name string
}

func resolveDropletDeleteTarget(client *Client, droplet string) (dropletDeleteTarget, error) {
	value := strings.TrimSpace(droplet)
	if value == "" {
		return dropletDeleteTarget{}, errors.New("droplet is required")
	}

	id, err := parseDropletID(value)
	if err == nil {
		return dropletDeleteTarget{ID: id}, nil
	}

	if !errors.Is(err, errInvalidDropletID) {
		return dropletDeleteTarget{}, fmt.Errorf("invalid droplet ID %q: %w", droplet, err)
	}

	return resolveDropletDeleteTargetByName(client, value)
}

func resolveDropletDeleteExecutionTarget(client *Client, droplet string, metadata core.MetadataReader) (dropletDeleteTarget, error) {
	if target, ok := dropletDeleteTargetFromMetadata(droplet, metadata); ok {
		return target, nil
	}

	return resolveDropletDeleteTarget(client, droplet)
}

func dropletDeleteTargetFromMetadata(droplet string, metadata core.MetadataReader) (dropletDeleteTarget, bool) {
	if metadata == nil {
		return dropletDeleteTarget{}, false
	}

	value := strings.TrimSpace(droplet)
	var existing DropletNodeMetadata
	if err := mapstructure.Decode(metadata.Get(), &existing); err != nil || existing.DropletID == 0 {
		return dropletDeleteTarget{}, false
	}

	if existing.DropletName != "" && existing.DropletName == value {
		return dropletDeleteTarget{ID: existing.DropletID, Name: existing.DropletName}, true
	}

	if fmt.Sprintf("%d", existing.DropletID) == value {
		return dropletDeleteTarget{ID: existing.DropletID, Name: existing.DropletName}, true
	}

	return dropletDeleteTarget{}, false
}

func resolveDropletDeleteTargetByName(client *Client, name string) (dropletDeleteTarget, error) {
	droplets, err := client.ListDropletsByName(name)
	if err != nil {
		return dropletDeleteTarget{}, fmt.Errorf("failed to list droplets for name lookup: %w", err)
	}

	matches := make([]Droplet, 0, 1)
	for _, droplet := range droplets {
		if droplet.Name == name {
			matches = append(matches, droplet)
		}
	}

	switch len(matches) {
	case 0:
		return dropletDeleteTarget{}, fmt.Errorf("droplet named %q was not found", name)
	case 1:
		return dropletDeleteTarget{ID: matches[0].ID, Name: matches[0].Name}, nil
	default:
		return dropletDeleteTarget{}, fmt.Errorf("multiple droplets named %q found; use the droplet ID", name)
	}
}

func resolveDropletDeleteMetadata(ctx core.SetupContext, droplet string) error {
	value := strings.TrimSpace(droplet)
	if strings.Contains(value, "{{") {
		return ctx.Metadata.Set(DropletNodeMetadata{DropletName: value})
	}

	var existing DropletNodeMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &existing); err == nil {
		if existing.DropletName != "" && existing.DropletName == value {
			return nil
		}
		if existing.DropletID != 0 && fmt.Sprintf("%d", existing.DropletID) == value && existing.DropletName != "" {
			return nil
		}
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client for metadata resolution: %w", err)
	}

	target, err := resolveDropletDeleteTarget(client, value)
	if err != nil {
		return err
	}

	if target.Name == "" {
		droplet, err := client.GetDroplet(target.ID)
		if err != nil {
			return fmt.Errorf("failed to fetch droplet %d for metadata: %w", target.ID, err)
		}
		target.Name = droplet.Name
	}

	return ctx.Metadata.Set(DropletNodeMetadata{
		DropletID:   target.ID,
		DropletName: target.Name,
	})
}

func dropletDeletedPayload(target dropletDeleteTarget) map[string]any {
	payload := map[string]any{"dropletId": target.ID}
	if target.Name != "" {
		payload["dropletName"] = target.Name
	}
	return payload
}

func resolveAppDeleteTarget(client *Client, app string) (appDeleteTarget, error) {
	value := strings.TrimSpace(app)
	if value == "" {
		return appDeleteTarget{}, errors.New("app is required")
	}

	if _, err := uuid.Parse(value); err == nil {
		return appDeleteTarget{ID: value}, nil
	}

	return resolveAppDeleteTargetByName(client, value)
}

func resolveAppDeleteExecutionTarget(client *Client, app string, metadata core.MetadataReader) (appDeleteTarget, error) {
	if target, ok := appDeleteTargetFromMetadata(app, metadata); ok {
		return target, nil
	}

	return resolveAppDeleteTarget(client, app)
}

func appDeleteTargetFromMetadata(app string, metadata core.MetadataReader) (appDeleteTarget, bool) {
	if metadata == nil {
		return appDeleteTarget{}, false
	}

	value := strings.TrimSpace(app)
	var existing AppNodeMetadata
	if err := mapstructure.Decode(metadata.Get(), &existing); err != nil || existing.AppID == "" {
		return appDeleteTarget{}, false
	}

	if existing.AppName != "" && existing.AppName == value {
		return appDeleteTarget{ID: existing.AppID, Name: existing.AppName}, true
	}

	if existing.AppID == value {
		return appDeleteTarget{ID: existing.AppID, Name: existing.AppName}, true
	}

	return appDeleteTarget{}, false
}

func resolveAppDeleteTargetByName(client *Client, name string) (appDeleteTarget, error) {
	apps, err := client.ListApps()
	if err != nil {
		return appDeleteTarget{}, fmt.Errorf("failed to list apps for name lookup: %w", err)
	}

	matches := make([]App, 0, 1)
	for _, app := range apps {
		if getAppName(app) == name {
			matches = append(matches, app)
		}
	}

	switch len(matches) {
	case 0:
		return appDeleteTarget{}, fmt.Errorf("app named %q was not found", name)
	case 1:
		return appDeleteTarget{ID: matches[0].ID, Name: getAppName(matches[0])}, nil
	default:
		return appDeleteTarget{}, fmt.Errorf("multiple apps named %q found; use the app ID", name)
	}
}

func resolveAppDeleteMetadata(ctx core.SetupContext, app string) error {
	value := strings.TrimSpace(app)
	if strings.Contains(value, "{{") {
		return ctx.Metadata.Set(AppNodeMetadata{AppName: value})
	}

	var existing AppNodeMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &existing); err == nil {
		if existing.AppName != "" && existing.AppName == value {
			return nil
		}
		if existing.AppID != "" && existing.AppID == value && existing.AppName != "" {
			return nil
		}
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client for metadata resolution: %w", err)
	}

	target, err := resolveAppDeleteTarget(client, value)
	if err != nil {
		return err
	}

	if target.Name == "" {
		app, err := client.GetApp(target.ID)
		if err != nil {
			return fmt.Errorf("failed to fetch app %q for metadata: %w", target.ID, err)
		}
		target.Name = getAppName(*app)
	}

	return ctx.Metadata.Set(AppNodeMetadata{
		AppID:   target.ID,
		AppName: target.Name,
	})
}

func appDeletedPayload(target appDeleteTarget) map[string]any {
	payload := map[string]any{"appId": target.ID}
	if target.Name != "" {
		payload["appName"] = target.Name
	}
	return payload
}

func getAppName(app App) string {
	if app.Spec != nil && app.Spec.Name != "" {
		return app.Spec.Name
	}

	return app.ID
}
