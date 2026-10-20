package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	defaultBeaconURL    = "https://analytics.superplane.com/beacon"
	beaconInterval      = 7 * 24 * time.Hour
	developmentVersion  = "development"
	beaconVersionEnv    = "SUPERPLANE_VERSION"
	beaconEnabledEnv    = "SUPERPLANE_BEACON_ENABLED"
	installationTypeEnv = "SUPERPLANE_INSTALLATION_TYPE"
)

type beaconPayload struct {
	InstallationType string    `json:"installation_type"`
	InstallationID   string    `json:"installation_id"`
	InstalledAt      time.Time `json:"installed_at"`
	Version          string    `json:"version"`
	TasksCreated7d   int64     `json:"tasks_created_7d"`
	TasksStarted7d   int64     `json:"tasks_started_7d"`
	TasksFinished7d  int64     `json:"tasks_finished_7d"`
	OwnerEmail       string    `json:"owner_email,omitempty"`
	OrganizationName string    `json:"organization_name,omitempty"`
}

func StartBeacon() {
	if !BeaconEnabled() {
		return
	}

	go beaconSender()
}

func beaconSender() {
	sendBeacon()

	ticker := time.NewTicker(beaconInterval)
	defer ticker.Stop()
	for range ticker.C {
		sendBeacon()
	}
}

func BeaconEnabled() bool {
	return os.Getenv(beaconEnabledEnv) == "yes"
}

func installationVersion() string {
	version := strings.TrimSpace(os.Getenv(beaconVersionEnv))
	if version == "" {
		return developmentVersion
	}

	return version
}

func sendBeacon() {
	installationType := os.Getenv(installationTypeEnv)
	if installationType == "" {
		log.Warn("Beacon not started - missing SUPERPLANE_INSTALLATION_TYPE")
		return
	}

	payload, err := buildBeaconPayload(database.Conn(), installationType, time.Now().Add(-beaconInterval))
	if err != nil {
		log.WithError(err).Warn("Failed to build beacon payload")
		return
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.WithError(err).Warn("Failed to encode beacon payload")
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, defaultBeaconURL, bytes.NewReader(body))
	if err != nil {
		log.WithError(err).Warn("Failed to create beacon request")
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.WithError(err).Warn("Failed to send beacon")
		return
	}
	defer resp.Body.Close()
}

func buildBeaconPayload(tx *gorm.DB, installationType string, since time.Time) (beaconPayload, error) {
	metadata, err := models.GetInstallationMetadata(tx)
	if err != nil {
		return beaconPayload{}, err
	}

	counts, err := models.CountFactoryTaskActivitySince(tx, since)
	if err != nil {
		return beaconPayload{}, err
	}

	payload := beaconPayload{
		InstallationType: installationType,
		InstallationID:   metadata.InstallationID,
		InstalledAt:      metadata.CreatedAt,
		Version:          installationVersion(),
		TasksCreated7d:   counts.Created,
		TasksStarted7d:   counts.Started,
		TasksFinished7d:  counts.Finished,
	}
	if !metadata.ShareOwnerIdentity {
		return payload, nil
	}

	email, err := models.FirstInstallationAdminEmail(tx)
	if err != nil {
		return beaconPayload{}, err
	}
	organizationName, err := models.FirstOrganizationName(tx)
	if err != nil {
		return beaconPayload{}, err
	}

	payload.OwnerEmail = email
	payload.OrganizationName = organizationName
	return payload, nil
}
