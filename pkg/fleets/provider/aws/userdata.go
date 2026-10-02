package awsprovider

import (
	"bytes"
	_ "embed"
	"fmt"
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

	var output bytes.Buffer
	if err := userDataTemplate.Execute(&output, request); err != nil {
		return nil, fmt.Errorf("render AWS runner bootstrap: %w", err)
	}
	return output.Bytes(), nil
}
