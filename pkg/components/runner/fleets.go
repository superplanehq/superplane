package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// ComponentName is the registry / canvas component key for Runner.
const ComponentName = "runner"

const (
	configurationFieldMachineType = "machine_type"
)

// BrokerFleet is a runner pool registered on the task-broker (machine profile).
type BrokerFleet struct {
	ID          string `json:"id"`
	Provisioner string `json:"provisioner,omitempty"`
	Arch        string `json:"arch,omitempty"`
	Size        string `json:"size,omitempty"`
}

// FormatFleetOptionLabel builds a select label from broker fleet catalog metadata.
func FormatFleetOptionLabel(f BrokerFleet) string {
	id := strings.TrimSpace(f.ID)
	if id == "" {
		return ""
	}

	size := strings.TrimSpace(f.Size)
	arch := strings.TrimSpace(f.Arch)
	switch {
	case size != "" && arch != "":
		return fmt.Sprintf("%s (%s, %s)", id, size, arch)
	case size != "":
		return fmt.Sprintf("%s (%s)", id, size)
	case arch != "":
		return fmt.Sprintf("%s (%s)", id, arch)
	default:
		return id
	}
}

func requireMachineType(machineType string) (string, error) {
	fleet := strings.TrimSpace(machineType)
	if fleet == "" {
		return "", fmt.Errorf("machine type is required")
	}
	return fleet, nil
}

// ListFleets returns machine profiles from GET /v1/fleets on the task-broker.
func (b *BrokerClient) ListFleets() ([]BrokerFleet, error) {
	httpCtx, cancel := context.WithTimeout(context.Background(), brokerHTTPTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(httpCtx, http.MethodGet, b.baseURL+"/v1/fleets", nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+b.authToken)

	resp, err := b.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("broker request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("broker rejected list fleets: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out []BrokerFleet
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("unmarshal list fleets response: %w", err)
	}
	if out == nil {
		return []BrokerFleet{}, nil
	}
	return out, nil
}

// EnrichRunnerConfigurationFields loads fleet options for the Machine type select field.
func EnrichRunnerConfigurationFields(httpClient core.HTTPContext, fields []configuration.Field) []configuration.Field {
	broker, err := NewBrokerClient(httpClient)
	if err != nil {
		log.WithError(err).Debug("runner: machine type options unavailable (task broker not configured)")
		return fields
	}

	fleets, err := broker.ListFleets()
	if err != nil {
		log.WithError(err).Warn("runner: failed to list fleets from task-broker for machine type options")
		return fields
	}
	if len(fleets) == 0 {
		log.Warn("runner: task-broker returned no fleets for machine type options")
		return fields
	}

	options := make([]configuration.FieldOption, 0, len(fleets))
	for _, f := range fleets {
		id := strings.TrimSpace(f.ID)
		if id == "" {
			continue
		}
		label := FormatFleetOptionLabel(f)
		if label == "" {
			label = id
		}
		desc := strings.TrimSpace(f.Provisioner)
		if desc != "" {
			desc = "Provisioner: " + desc
		}
		options = append(options, configuration.FieldOption{
			Label:       label,
			Value:       id,
			Description: desc,
		})
	}
	if len(options) == 0 {
		return fields
	}

	out := make([]configuration.Field, len(fields))
	copy(out, fields)
	for i := range out {
		if out[i].Name != configurationFieldMachineType {
			continue
		}
		if out[i].TypeOptions == nil {
			out[i].TypeOptions = &configuration.TypeOptions{}
		}
		out[i].TypeOptions.Select = &configuration.SelectTypeOptions{Options: options}
		break
	}
	return out
}
