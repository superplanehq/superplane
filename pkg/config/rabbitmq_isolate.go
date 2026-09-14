package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	testProcessIsolateEnv = "TEST_PROCESS_ISOLATE"
	defaultManagementPort = "15672"
)

var (
	isolatedRabbitURL     string
	isolatedRabbitURLErr  error
	isolatedRabbitURLOnce sync.Once
)

func processTestVhost(pid int) string {
	return "superplane_" + strconv.Itoa(pid) + "_test"
}

func rewriteRabbitMQURLForProcess(raw string, pid int) (string, string, bool, error) {
	if !shouldIsolateRabbitMQ() {
		return raw, "", false, nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", "", false, fmt.Errorf("parse RABBITMQ_URL: %w", err)
	}

	if !isDefaultRabbitMQVhost(parsed.Path) {
		return raw, "", false, nil
	}

	vhost := processTestVhost(pid)
	parsed.Path = "/" + vhost
	parsed.RawPath = ""
	return parsed.String(), vhost, true, nil
}

func shouldIsolateRabbitMQ() bool {
	if os.Getenv(testProcessIsolateEnv) == "0" {
		return false
	}
	return strings.HasSuffix(os.Getenv("DB_NAME"), "_test")
}

func isDefaultRabbitMQVhost(path string) bool {
	return path == "" || path == "/"
}

func managementBaseURL(rawAMQP string) (string, error) {
	if override := os.Getenv("RABBITMQ_MANAGEMENT_URL"); override != "" {
		return strings.TrimRight(override, "/"), nil
	}

	parsed, err := url.Parse(rawAMQP)
	if err != nil {
		return "", fmt.Errorf("parse RABBITMQ_URL: %w", err)
	}

	host := parsed.Hostname()
	if host == "" {
		return "", fmt.Errorf("RABBITMQ_URL has no host")
	}

	return "http://" + net.JoinHostPort(host, defaultManagementPort), nil
}

func isolateRabbitMQURL(raw string) (string, error) {
	rewritten, vhost, isolate, err := rewriteRabbitMQURLForProcess(raw, os.Getpid())
	if err != nil {
		return "", err
	}
	if !isolate {
		return rewritten, nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse RABBITMQ_URL: %w", err)
	}

	user := "guest"
	pass := "guest"
	if parsed.User != nil {
		user = parsed.User.Username()
		if p, ok := parsed.User.Password(); ok {
			pass = p
		}
	}

	managementURL, err := managementBaseURL(raw)
	if err != nil {
		return "", err
	}
	if err := ensureRabbitMQVhost(managementURL, user, pass, vhost); err != nil {
		return "", err
	}

	return rewritten, nil
}

func ensureRabbitMQVhost(managementURL, user, pass, vhost string) error {
	client := &http.Client{Timeout: 5 * time.Second}

	vhostPath := url.PathEscape(vhost)
	if err := rabbitManagementPut(client, managementURL+"/api/vhosts/"+vhostPath, user, pass, []byte("{}")); err != nil {
		return fmt.Errorf("create RabbitMQ vhost %s: %w", vhost, err)
	}

	body, err := json.Marshal(map[string]string{
		"configure": ".*",
		"write":     ".*",
		"read":      ".*",
	})
	if err != nil {
		return err
	}

	permissionsPath := url.PathEscape(user) + "/" + vhostPath
	if err := rabbitManagementPut(client, managementURL+"/api/permissions/"+permissionsPath, user, pass, body); err != nil {
		return fmt.Errorf("grant RabbitMQ permissions on vhost %s: %w", vhost, err)
	}

	return nil
}

func rabbitManagementPut(client *http.Client, endpoint, user, pass string, body []byte) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(http.MethodPut, endpoint, reader)
	if err != nil {
		return err
	}
	req.SetBasicAuth(user, pass)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return nil
	}

	limited, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
	return fmt.Errorf("HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(limited)))
}
