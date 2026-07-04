package compute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

// Image deprecation states accepted by the Compute Engine images.deprecate API.
const (
	ImageStateActive     = "ACTIVE"
	ImageStateDeprecated = "DEPRECATED"
	ImageStateObsolete   = "OBSOLETE"
	ImageStateDeleted    = "DELETED"
)

// ImageStateNoChange is the sentinel used by the Update Image deprecation-state
// select to mean "leave the current deprecation state untouched". A select
// option must not use an empty string value (the frontend's Radix-based select
// throws on empty item values), so this explicit sentinel is used instead.
const ImageStateNoChange = "NO_CHANGE"

// ImageNodeMetadata is persisted on the node so the collapsed UI can show the
// targeted image name.
type ImageNodeMetadata struct {
	ImageName string `json:"imageName" mapstructure:"imageName"`
}

// parseImagePath extracts (project, name) from an image value. Compute Engine
// images are global resources, so the accepted forms are:
//   - a full selfLink URL containing projects/<project>/global/images/<name>
//   - a relative path global/images/<name> or projects/<project>/global/images/<name>
//   - a bare image name (no slash), in which case project is empty
//
// The project segment is optional — relative dropdown values and bare names
// carry no project, but chained selfLinks do, and the caller must verify it
// matches the integration's bound project before issuing a mutating call.
func parseImagePath(value string) (project, name string, err error) {
	s := strings.TrimSpace(value)
	if s == "" {
		return "", "", errors.New("image is required")
	}

	if idx := strings.Index(s, "projects/"); idx >= 0 {
		rest := s[idx+len("projects/"):]
		if slash := strings.Index(rest, "/"); slash > 0 {
			project = rest[:slash]
		}
	}

	const marker = "global/images/"
	if idx := strings.Index(s, marker); idx >= 0 {
		name = s[idx+len(marker):]
	} else if !strings.Contains(s, "/") {
		// Bare image name.
		name = s
	} else {
		return "", "", fmt.Errorf("image %q must be a name or a path like global/images/<name> or a GCE selfLink URL", value)
	}

	if q := strings.IndexAny(name, "/?#"); q >= 0 {
		name = name[:q]
	}
	if name == "" {
		return "", "", fmt.Errorf("image %q is missing a name", value)
	}
	return project, name, nil
}

type imageGetResp struct {
	Id                uint64            `json:"id,string"`
	Name              string            `json:"name"`
	SelfLink          string            `json:"selfLink"`
	Family            string            `json:"family"`
	Description       string            `json:"description"`
	Status            string            `json:"status"`
	DiskSizeGb        int64             `json:"diskSizeGb,string"`
	ArchiveSizeBytes  int64             `json:"archiveSizeBytes,string"`
	SourceDisk        string            `json:"sourceDisk"`
	CreationTimestamp string            `json:"creationTimestamp"`
	StorageLocations  []string          `json:"storageLocations"`
	Labels            map[string]string `json:"labels"`
	LabelFingerprint  string            `json:"labelFingerprint"`
	Deprecated        *struct {
		State       string `json:"state"`
		Replacement string `json:"replacement"`
	} `json:"deprecated"`
}

// GetImage reads a global image by name.
func GetImage(ctx context.Context, client Client, project, name string) ([]byte, error) {
	if project == "" {
		project = client.ProjectID()
	}
	path := fmt.Sprintf("projects/%s/global/images/%s", project, name)
	return client.Get(ctx, path)
}

// ImagePayloadFromGetResponse converts an images.get response body into the flat
// payload emitted by the image components.
func ImagePayloadFromGetResponse(body []byte) (map[string]any, error) {
	var img imageGetResp
	if err := json.Unmarshal(body, &img); err != nil {
		return nil, fmt.Errorf("parse image response: %w", err)
	}
	payload := map[string]any{
		"imageId":           fmt.Sprintf("%d", img.Id),
		"name":              img.Name,
		"selfLink":          img.SelfLink,
		"family":            img.Family,
		"status":            img.Status,
		"diskSizeGb":        img.DiskSizeGb,
		"creationTimestamp": img.CreationTimestamp,
	}
	if img.SourceDisk != "" {
		payload["sourceDisk"] = lastSegment(img.SourceDisk)
	}
	if len(img.StorageLocations) > 0 {
		payload["storageLocations"] = img.StorageLocations
	}
	if len(img.Labels) > 0 {
		payload["labels"] = img.Labels
	}
	state := ImageStateActive
	if img.Deprecated != nil && img.Deprecated.State != "" {
		state = img.Deprecated.State
		if img.Deprecated.Replacement != "" {
			payload["replacement"] = img.Deprecated.Replacement
		}
	}
	payload["deprecationState"] = state
	return payload, nil
}

// WaitForGlobalOperation polls a global (project-scoped) operation until it is
// DONE, mirroring WaitForZoneOperation for global resources like images.
func WaitForGlobalOperation(ctx context.Context, client Client, project, operationName string) error {
	path := fmt.Sprintf("projects/%s/global/operations/%s", project, operationName)
	deadline := time.Now().Add(defaultOperationWaitTimeout)
	ticker := time.NewTicker(operationPollInterval)
	defer ticker.Stop()
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for operation %s", operationName)
		}
		body, err := client.Get(ctx, path)
		if err != nil {
			return err
		}
		var op zoneOperationResp
		if err := json.Unmarshal(body, &op); err != nil {
			return fmt.Errorf("parse operation response: %w", err)
		}
		switch op.Status {
		case opStatusDone:
			if op.Error != nil && len(op.Error.Errors) > 0 {
				msg := op.Error.Errors[0].Message
				if msg == "" {
					msg = op.Error.Errors[0].Code
				}
				return fmt.Errorf("operation failed: %s", msg)
			}
			return nil
		case opStatusPending, opStatusRunning:
		default:
			return fmt.Errorf("unexpected operation status: %s", op.Status)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// operationNameFromResponse extracts the operation name from a global operation
// response body, returning a descriptive error when it is absent.
func operationNameFromResponse(body []byte, what string) (string, error) {
	var opResp struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &opResp); err != nil {
		return "", fmt.Errorf("parse %s operation response: %w", what, err)
	}
	if opResp.Name == "" {
		return "", fmt.Errorf("%s operation response missing operation name", what)
	}
	return lastSegment(opResp.Name), nil
}

// resolveImageNodeMetadata stores the targeted image name on the node so the
// collapsed UI can display something meaningful. Update/Delete Image share it.
func resolveImageNodeMetadata(ctx core.SetupContext, imageValue string) error {
	if strings.Contains(imageValue, "{{") {
		return ctx.Metadata.Set(ImageNodeMetadata{ImageName: imageValue})
	}
	_, name, err := parseImagePath(imageValue)
	if err != nil {
		return err
	}
	return ctx.Metadata.Set(ImageNodeMetadata{ImageName: name})
}

// imageLabelsFromEntries converts the labels list field into the GCP labels map,
// dropping entries with empty keys and de-duplicating on first write.
func imageLabelsFromEntries(entries []LabelEntry) map[string]string {
	if len(entries) == 0 {
		return nil
	}
	out := make(map[string]string)
	for _, e := range entries {
		k := strings.TrimSpace(e.Key)
		if k == "" {
			continue
		}
		if _, exists := out[k]; exists {
			continue
		}
		out[k] = strings.TrimSpace(e.Value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// mergeImageLabels overlays the supplied updates onto the image's existing
// labels. Keys present in updates are added or overwritten; existing keys that
// are not listed are preserved. GCP's images.setLabels replaces the entire set,
// so the merge is performed client-side before the call.
func mergeImageLabels(existing, updates map[string]string) map[string]string {
	merged := make(map[string]string, len(existing)+len(updates))
	for k, v := range existing {
		merged[k] = v
	}
	for k, v := range updates {
		merged[k] = v
	}
	return merged
}
