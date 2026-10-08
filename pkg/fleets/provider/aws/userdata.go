package awsprovider

import (
	"bytes"
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/superplanehq/superplane/pkg/fleets/provider"
)

//go:embed userdata.sh.tmpl
var userDataTemplateSource string

var userDataTemplate = template.Must(template.New("userdata").Funcs(template.FuncMap{
	"shellQuote": strconv.Quote,
}).Parse(userDataTemplateSource))

type userData struct {
	provider.RunnerBootstrap
	CloudWatchAgentConfig string
	Tags                  string
}

type cloudWatchAgentConfig struct {
	Agent cloudWatchAgentSettings `json:"agent"`
	Logs  cloudWatchLogs          `json:"logs"`
}

type cloudWatchAgentSettings struct {
	Region string `json:"region"`
}

type cloudWatchLogs struct {
	LogsCollected cloudWatchLogsCollected `json:"logs_collected"`
}

type cloudWatchLogsCollected struct {
	Files cloudWatchLogFiles `json:"files"`
}

type cloudWatchLogFiles struct {
	CollectList []cloudWatchLogFile `json:"collect_list"`
}

type cloudWatchLogFile struct {
	FilePath      string `json:"file_path"`
	LogGroupName  string `json:"log_group_name"`
	LogStreamName string `json:"log_stream_name"`
}

func buildUserData(
	request provider.RunnerBootstrap,
	cloudWatchRegion string,
	cloudWatchLogGroup string,
) ([]byte, error) {
	switch {
	case strings.TrimSpace(request.RunnerID) == "":
		return nil, fmt.Errorf("runner ID is required")
	case strings.TrimSpace(request.FleetID) == "":
		return nil, fmt.Errorf("fleet ID is required")
	case strings.TrimSpace(request.RunnerAPIURL) == "":
		return nil, fmt.Errorf("runner API URL is required")
	case strings.TrimSpace(request.RegistrationToken) == "":
		return nil, fmt.Errorf("runner registration token is required")
	case strings.TrimSpace(request.Artifact.URL) == "":
		return nil, fmt.Errorf("runner artifact URL is required")
	case len(strings.TrimSpace(request.Artifact.SHA256)) != 64:
		return nil, fmt.Errorf("runner artifact SHA-256 is invalid")
	}

	templateData := userData{RunnerBootstrap: request}
	tags, err := formatRunnerTags(request.Tags)
	if err != nil {
		return nil, fmt.Errorf("encode runner tags: %w", err)
	}
	templateData.Tags = tags
	if cloudWatchLogGroup != "" {
		config := cloudWatchAgentConfig{
			Agent: cloudWatchAgentSettings{Region: cloudWatchRegion},
			Logs: cloudWatchLogs{
				LogsCollected: cloudWatchLogsCollected{
					Files: cloudWatchLogFiles{
						CollectList: []cloudWatchLogFile{{
							FilePath:      "/var/log/superplane-runner.log",
							LogGroupName:  cloudWatchLogGroup,
							LogStreamName: "{instance_id}",
						}},
					},
				},
			},
		}
		encoded, err := json.Marshal(config)
		if err != nil {
			return nil, fmt.Errorf("encode CloudWatch Agent config: %w", err)
		}
		templateData.CloudWatchAgentConfig = string(encoded)
	}

	var output bytes.Buffer
	if err := userDataTemplate.Execute(&output, templateData); err != nil {
		return nil, fmt.Errorf("render AWS runner bootstrap: %w", err)
	}
	return output.Bytes(), nil
}

func formatRunnerTags(tags map[string]string) (string, error) {
	if len(tags) == 0 {
		return "", nil
	}
	keys := make([]string, 0, len(tags))
	for key := range tags {
		if strings.TrimSpace(key) == "" || strings.Contains(key, "=") {
			return "", fmt.Errorf("runner tag key %q is invalid", key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fields := make([]string, 0, len(keys))
	for _, key := range keys {
		fields = append(fields, key+"="+tags[key])
	}
	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	if err := writer.Write(fields); err != nil {
		return "", err
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}
	return strings.TrimSuffix(output.String(), "\n"), nil
}
