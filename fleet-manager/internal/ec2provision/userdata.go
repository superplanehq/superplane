package ec2provision

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"strconv"
	"strings"
	"text/template"
)

//go:embed userdata.sh.tmpl
var userDataTemplateSrc string

var userDataTemplate = template.Must(template.New("userdata").Funcs(template.FuncMap{
	"shellQuote": func(s string) string { return strconv.Quote(s) },
}).Parse(userDataTemplateSrc))

type userDataVars struct {
	RunnerS3URI                     string
	RunnerInstallAWSRegion          string
	FleetManagerURL                 string
	RunnersAuthToken                string
	RunnerTerminateAfterEachTask    bool
	RunnerCloudWatchLogGroup        string
	RunnerCloudWatchLogStreamPrefix string
	CloudWatchAgentJSON             string
	RestartPolicy                   string
}

func userDataVarsFromConfig(c Config) userDataVars {
	restartPolicy := "always"
	if c.RunnerTerminateAfterEachTask {
		restartPolicy = "no"
	}

	procRegion := strings.TrimSpace(c.RunnerProcessLogRegion)
	if procRegion == "" {
		procRegion = strings.TrimSpace(c.RunnerInstallAWSRegion)
	}
	if procRegion == "" {
		procRegion = "us-east-1"
	}

	return userDataVars{
		RunnerS3URI:                     strings.TrimSpace(c.RunnerS3URI),
		RunnerInstallAWSRegion:          strings.TrimSpace(c.RunnerInstallAWSRegion),
		FleetManagerURL:                 c.FleetManagerURL,
		RunnersAuthToken:                strings.TrimSpace(c.RunnersAuthToken),
		RunnerTerminateAfterEachTask:    c.RunnerTerminateAfterEachTask,
		RunnerCloudWatchLogGroup:        strings.TrimSpace(c.RunnerCloudWatchLogGroup),
		RunnerCloudWatchLogStreamPrefix: strings.TrimSpace(c.RunnerCloudWatchLogStreamPrefix),
		CloudWatchAgentJSON:             mustCloudWatchAgentJSON(strings.TrimSpace(c.RunnerProcessLogGroup), procRegion),
		RestartPolicy:                   restartPolicy,
	}
}

func mustCloudWatchAgentJSON(logGroup, region string) string {
	cfg := map[string]any{
		"logs": map[string]any{
			"logs_collected": map[string]any{
				"files": map[string]any{
					"collect_list": []map[string]string{{
						"file_path":        "/var/log/superplane-runner.log",
						"log_group_name":   logGroup,
						"log_stream_name":  "{instance_id}/runner-process",
						"timestamp_format": "%Y-%m-%dT%H:%M:%S",
					}},
				},
			},
			"log_stream_name": "{instance_id}/runner-process",
		},
		"agent": map[string]string{
			"region": region,
		},
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		panic("cloudwatch agent config: " + err.Error())
	}
	return string(b)
}

func userDataScript(c Config) (string, error) {
	var buf bytes.Buffer
	if err := userDataTemplate.Execute(&buf, userDataVarsFromConfig(c)); err != nil {
		return "", err
	}
	return buf.String(), nil
}
