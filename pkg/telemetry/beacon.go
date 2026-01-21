package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
)

const defaultBeaconURL = "https://analytics.superplane.com/beacon"

type beaconConfig struct {
	url     string
	payload beaconPayload
}

type beaconPayload struct {
	InstallationType string `json:"installation_type"`
	InstallationID   string `json:"installation_id"`
}

func StartBeacon() {
	if !isBeaconEnabled() {
		return
	}

	go beaconSender()
}

func beaconSender() {
	sendBeacon()

	ticker := time.NewTicker(time.Hour)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			sendBeacon()
		}
	}()
}

func isBeaconEnabled() bool {
	return os.Getenv("SUPERPLANE_BEACON_ENABLED") == "yes"
}

func sendBeacon() {
	client := &http.Client{Timeout: 5 * time.Second}

	installationType := os.Getenv("SUPERPLANE_INSTALLATION_TYPE")
	if installationType == "" {
		log.Warn("Beacon not started - missing SUPERPLANE_INSTALLATION_TYPE")
		return
	}

	installationID, err := models.GetInstallationID()
	if err != nil {
		log.WithError(err).Warn("Failed to load installation ID")
		return
	}

	payload := beaconPayload{
		InstallationType: installationType,
		InstallationID:   installationID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.WithError(err).Warn("Failed to encode beacon payload")
		return
	}

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
