package gcpprovider

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/superplanehq/superplane/pkg/fleets/provider"
)

//go:embed userdata.sh.tmpl
var userDataTemplateSource string

var userDataTemplate = template.Must(template.New("userdata").Funcs(template.FuncMap{
	"shellQuote": shellQuote,
}).Parse(userDataTemplateSource))

type userData struct {
	provider.RunnerBootstrap
	TagsValue string
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func buildUserData(request provider.RunnerBootstrap) ([]byte, error) {
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

	tags, err := provider.FormatRunnerTags(request.Tags)
	if err != nil {
		return nil, fmt.Errorf("encode runner tags: %w", err)
	}
	var output bytes.Buffer
	if err := userDataTemplate.Execute(&output, userData{RunnerBootstrap: request, TagsValue: tags}); err != nil {
		return nil, fmt.Errorf("render GCP runner bootstrap: %w", err)
	}
	return output.Bytes(), nil
}
