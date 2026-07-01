package grafana

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	defaultSyntheticCheckFrequencySeconds = 60
	defaultSyntheticCheckTimeout          = 3000
	defaultSyntheticCheckIPVersion        = "V4"
)

var (
	allowedSyntheticHTTPMethods = []string{"GET", "POST", "PUT", "DELETE", "HEAD", "PATCH", "OPTIONS"}
	allowedSyntheticAlertTypes  = []string{
		"ProbeFailedExecutionsTooHigh",
		"TLSTargetCertificateCloseToExpiring",
		"HTTPRequestDurationTooHighAvg",
	}
	allowedSyntheticAlertPeriods = []string{"5m", "10m", "15m", "20m", "30m", "1h"}
	alertTypesRequiringPeriod    = []string{"ProbeFailedExecutionsTooHigh", "HTTPRequestDurationTooHighAvg"}
)

type SyntheticCheckHeaderInput struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

type SyntheticCheckLabelInput struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

type SyntheticCheckBasicAuthInput struct {
	Username string `json:"username,omitempty" mapstructure:"username"`
	Password string `json:"password,omitempty" mapstructure:"password"`
}

type SyntheticCheckHeaderMatchInput struct {
	Header       string `json:"header" mapstructure:"header"`
	Regexp       string `json:"regexp" mapstructure:"regexp"`
	AllowMissing *bool  `json:"allowMissing,omitempty" mapstructure:"allowMissing"`
}

type SyntheticCheckAlertInput struct {
	Name       string  `json:"name" mapstructure:"name"`
	Threshold  *int64  `json:"threshold,omitempty" mapstructure:"threshold"`
	Period     *string `json:"period,omitempty" mapstructure:"period"`
	RunbookURL *string `json:"runbookUrl,omitempty" mapstructure:"runbookUrl"`
}

type SyntheticCheckSpecBase struct {
	Job                          string                           `json:"job" mapstructure:"job"`
	Target                       string                           `json:"target" mapstructure:"target"`
	Enabled                      *bool                            `json:"enabled,omitempty" mapstructure:"enabled"`
	Frequency                    int64                            `json:"frequency" mapstructure:"frequency"`
	FrequencyMilliseconds        *int64                           `json:"-" mapstructure:"frequencyMilliseconds"`
	Timeout                      int64                            `json:"timeout" mapstructure:"timeout"`
	Probes                       []string                         `json:"probes" mapstructure:"probes"`
	Labels                       []SyntheticCheckLabelInput       `json:"labels,omitempty" mapstructure:"labels"`
	AlertSensitivity             string                           `json:"alertSensitivity,omitempty" mapstructure:"alertSensitivity"`
	BasicMetricsOnly             *bool                            `json:"basicMetricsOnly,omitempty" mapstructure:"basicMetricsOnly"`
	Method                       string                           `json:"method" mapstructure:"method"`
	Headers                      []SyntheticCheckHeaderInput      `json:"headers,omitempty" mapstructure:"headers"`
	Body                         *string                          `json:"body,omitempty" mapstructure:"body"`
	IPVersion                    string                           `json:"ipVersion,omitempty" mapstructure:"ipVersion"`
	Compression                  string                           `json:"compression,omitempty" mapstructure:"compression"`
	NoFollowRedirects            *bool                            `json:"noFollowRedirects,omitempty" mapstructure:"noFollowRedirects"`
	FailIfSSL                    *bool                            `json:"failIfSSL,omitempty" mapstructure:"failIfSSL"`
	FailIfNotSSL                 *bool                            `json:"failIfNotSSL,omitempty" mapstructure:"failIfNotSSL"`
	ValidStatusCodes             []int                            `json:"validStatusCodes,omitempty" mapstructure:"validStatusCodes"`
	FailIfBodyMatchesRegexp      []string                         `json:"failIfBodyMatchesRegexp,omitempty" mapstructure:"failIfBodyMatchesRegexp"`
	FailIfBodyNotMatchesRegexp   []string                         `json:"failIfBodyNotMatchesRegexp,omitempty" mapstructure:"failIfBodyNotMatchesRegexp"`
	FailIfHeaderMatchesRegexp    []SyntheticCheckHeaderMatchInput `json:"failIfHeaderMatchesRegexp,omitempty" mapstructure:"failIfHeaderMatchesRegexp"`
	FailIfHeaderNotMatchesRegexp []SyntheticCheckHeaderMatchInput `json:"failIfHeaderNotMatchesRegexp,omitempty" mapstructure:"failIfHeaderNotMatchesRegexp"`
	BasicAuth                    *SyntheticCheckBasicAuthInput    `json:"basicAuth,omitempty" mapstructure:"basicAuth"`
	BearerToken                  *string                          `json:"bearerToken,omitempty" mapstructure:"bearerToken"`
	Alerts                       []SyntheticCheckAlertInput       `json:"alerts,omitempty" mapstructure:"alerts"`
}

type SyntheticCheckSelectionSpec struct {
	SyntheticCheck string `json:"syntheticCheck" mapstructure:"syntheticCheck"`
}

type SyntheticCheckNodeMetadata struct {
	CheckLabel     string `json:"checkLabel,omitempty" mapstructure:"checkLabel"`
	SyntheticCheck string `json:"syntheticCheck,omitempty" mapstructure:"syntheticCheck"`
	ProbeSummary   string `json:"probeSummary,omitempty" mapstructure:"probeSummary"`
}

func decodeSyntheticCheckSpec(input any, target any) error {
	var m map[string]any
	if err := mapstructure.Decode(input, &m); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	normalizeSyntheticCheckConfigMap(m)
	flat := flattenSyntheticCheckConfigMap(m)

	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           target,
		WeaklyTypedInput: true,
	})
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	if err := dec.Decode(flat); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	return nil
}

func validateSyntheticCheckBase(spec SyntheticCheckSpecBase) error {
	if strings.TrimSpace(spec.Job) == "" {
		return errors.New("job is required")
	}
	if strings.TrimSpace(spec.Target) == "" {
		return errors.New("target is required")
	}

	targetURL, err := url.Parse(strings.TrimSpace(spec.Target))
	if err != nil || targetURL.Scheme == "" || targetURL.Host == "" {
		return errors.New("target must be a valid absolute URL")
	}
	if targetURL.Scheme != "http" && targetURL.Scheme != "https" {
		return errors.New("target must use http or https")
	}

	if !slices.Contains(allowedSyntheticHTTPMethods, strings.ToUpper(strings.TrimSpace(spec.Method))) {
		return fmt.Errorf("method must be one of %s", strings.Join(allowedSyntheticHTTPMethods, ", "))
	}
	if spec.Frequency <= 0 {
		return errors.New("frequency must be greater than 0")
	}
	if spec.Timeout <= 0 {
		return errors.New("timeout must be greater than 0")
	}
	if len(spec.Probes) == 0 {
		return errors.New("at least one probe is required")
	}
	if spec.BasicAuth != nil {
		if strings.TrimSpace(spec.BasicAuth.Username) == "" || strings.TrimSpace(spec.BasicAuth.Password) == "" {
			return errors.New("basicAuth.username and basicAuth.password are required when basicAuth is set")
		}
	}
	for _, code := range spec.ValidStatusCodes {
		if code < 100 || code > 599 {
			return errors.New("validStatusCodes must contain valid HTTP status codes")
		}
	}
	for _, matcher := range spec.FailIfHeaderMatchesRegexp {
		if strings.TrimSpace(matcher.Header) == "" {
			return errors.New("failIfHeaderMatchesRegexp header is required")
		}
		if strings.TrimSpace(matcher.Regexp) == "" {
			return errors.New("failIfHeaderMatchesRegexp regex is required")
		}
	}
	for _, matcher := range spec.FailIfHeaderNotMatchesRegexp {
		if strings.TrimSpace(matcher.Header) == "" {
			return errors.New("failIfHeaderNotMatchesRegexp header is required")
		}
		if strings.TrimSpace(matcher.Regexp) == "" {
			return errors.New("failIfHeaderNotMatchesRegexp regex is required")
		}
	}
	return validateSyntheticCheckAlerts(spec.Alerts)
}

func validateSyntheticCheckSelection(spec SyntheticCheckSelectionSpec) error {
	if strings.TrimSpace(spec.SyntheticCheck) == "" {
		return errors.New("syntheticCheck is required")
	}
	return nil
}

func buildSyntheticCheckPayload(spec SyntheticCheckSpecBase) (SyntheticCheck, error) {
	probes, err := parseSyntheticProbeIDs(spec.Probes)
	if err != nil {
		return SyntheticCheck{}, err
	}

	enabled := true
	if spec.Enabled != nil {
		enabled = *spec.Enabled
	}

	ipVersion := strings.TrimSpace(spec.IPVersion)
	if ipVersion == "" {
		ipVersion = defaultSyntheticCheckIPVersion
	}

	alertSensitivity := strings.TrimSpace(spec.AlertSensitivity)
	if alertSensitivity == "" {
		alertSensitivity = "none"
	}

	basicMetricsOnly := true
	if spec.BasicMetricsOnly != nil {
		basicMetricsOnly = *spec.BasicMetricsOnly
	}

	check := SyntheticCheck{
		Job:              strings.TrimSpace(spec.Job),
		Target:           strings.TrimSpace(spec.Target),
		Frequency:        syntheticFrequencyMilliseconds(spec),
		Timeout:          spec.Timeout,
		Enabled:          enabled,
		AlertSensitivity: alertSensitivity,
		BasicMetricsOnly: basicMetricsOnly,
		Labels:           make([]SyntheticCheckLabel, 0, len(spec.Labels)),
		Probes:           probes,
		Settings: SyntheticCheckSettings{
			HTTP: &SyntheticCheckHTTPSettings{
				Method:                       strings.ToUpper(strings.TrimSpace(spec.Method)),
				Headers:                      buildSyntheticHeaderStrings(spec.Headers),
				IPVersion:                    ipVersion,
				Compression:                  strings.TrimSpace(spec.Compression),
				NoFollowRedirects:            spec.NoFollowRedirects != nil && *spec.NoFollowRedirects,
				FailIfSSL:                    spec.FailIfSSL != nil && *spec.FailIfSSL,
				FailIfNotSSL:                 spec.FailIfNotSSL != nil && *spec.FailIfNotSSL,
				ValidStatusCodes:             append([]int(nil), spec.ValidStatusCodes...),
				FailIfBodyMatchesRegexp:      append([]string(nil), spec.FailIfBodyMatchesRegexp...),
				FailIfBodyNotMatchesRegexp:   append([]string(nil), spec.FailIfBodyNotMatchesRegexp...),
				FailIfHeaderMatchesRegexp:    buildSyntheticHeaderMatches(spec.FailIfHeaderMatchesRegexp),
				FailIfHeaderNotMatchesRegexp: buildSyntheticHeaderMatches(spec.FailIfHeaderNotMatchesRegexp),
			},
		},
	}

	if spec.Body != nil {
		check.Settings.HTTP.Body = *spec.Body
	}
	if spec.BearerToken != nil && strings.TrimSpace(*spec.BearerToken) != "" {
		check.Settings.HTTP.BearerToken = strings.TrimSpace(*spec.BearerToken)
	}
	if spec.BasicAuth != nil {
		check.Settings.HTTP.BasicAuth = &SyntheticCheckBasicAuth{
			Username: strings.TrimSpace(spec.BasicAuth.Username),
			Password: strings.TrimSpace(spec.BasicAuth.Password),
		}
	}
	for _, label := range spec.Labels {
		name := strings.TrimSpace(label.Name)
		value := strings.TrimSpace(label.Value)
		if name == "" || value == "" {
			continue
		}
		check.Labels = append(check.Labels, SyntheticCheckLabel{Name: name, Value: value})
	}

	return check, nil
}

func buildSyntheticHeaderStrings(headers []SyntheticCheckHeaderInput) []string {
	formatted := make([]string, 0, len(headers))
	for _, header := range headers {
		name := strings.TrimSpace(header.Name)
		value := strings.TrimSpace(header.Value)
		if name == "" || value == "" {
			continue
		}
		formatted = append(formatted, name+":"+value)
	}

	return formatted
}

func buildSyntheticHeaderMatches(values []SyntheticCheckHeaderMatchInput) []SyntheticCheckHeaderMatch {
	matches := make([]SyntheticCheckHeaderMatch, 0, len(values))
	for _, value := range values {
		header := strings.TrimSpace(value.Header)
		regexp := strings.TrimSpace(value.Regexp)
		if header == "" || regexp == "" {
			continue
		}

		matches = append(matches, SyntheticCheckHeaderMatch{
			Header:       header,
			Regexp:       regexp,
			AllowMissing: value.AllowMissing != nil && *value.AllowMissing,
		})
	}

	if len(matches) == 0 {
		return nil
	}

	return matches
}

func buildSyntheticAlertDrafts(alerts []SyntheticCheckAlertInput) []SyntheticCheckAlert {
	drafts := make([]SyntheticCheckAlert, 0, len(alerts))
	for _, alert := range alerts {
		name := strings.TrimSpace(alert.Name)
		if name == "" || alert.Threshold == nil || *alert.Threshold <= 0 {
			continue
		}

		draft := SyntheticCheckAlert{
			Name:      name,
			Threshold: *alert.Threshold,
		}
		if alert.Period != nil {
			draft.Period = strings.TrimSpace(*alert.Period)
		}
		if alert.RunbookURL != nil {
			draft.RunbookURL = strings.TrimSpace(*alert.RunbookURL)
		}

		drafts = append(drafts, draft)
	}

	if len(drafts) == 0 {
		return nil
	}

	return drafts
}

func normalizeSyntheticFrequency(value int64) int64 {
	return value * 1000
}

func syntheticFrequencyMilliseconds(spec SyntheticCheckSpecBase) int64 {
	if spec.FrequencyMilliseconds != nil {
		return *spec.FrequencyMilliseconds
	}

	return normalizeSyntheticFrequency(spec.Frequency)
}

func syntheticFrequencySecondsFromMilliseconds(value int64) int64 {
	if value <= 0 {
		return 0
	}

	return (value + 999) / 1000
}

func parseSyntheticProbeIDs(probes []string) ([]int64, error) {
	parsed := make([]int64, 0, len(probes))
	for _, probe := range probes {
		value := strings.TrimSpace(probe)
		if value == "" {
			continue
		}

		probeID, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("probe %q is not a valid probe id", value)
		}
		parsed = append(parsed, probeID)
	}
	if len(parsed) == 0 {
		return nil, errors.New("at least one probe is required")
	}

	return parsed, nil
}

func validateSyntheticCheckAlerts(alerts []SyntheticCheckAlertInput) error {
	for _, alert := range alerts {
		name := strings.TrimSpace(alert.Name)
		if name == "" {
			return errors.New("alert name is required")
		}
		if !slices.Contains(allowedSyntheticAlertTypes, name) {
			return fmt.Errorf("unsupported alert type %q", alert.Name)
		}
		if alert.Threshold == nil || *alert.Threshold <= 0 {
			return fmt.Errorf("alert threshold is required for %s", name)
		}
		if slices.Contains(alertTypesRequiringPeriod, name) {
			if alert.Period == nil || strings.TrimSpace(*alert.Period) == "" {
				return fmt.Errorf("alert period is required for %s", name)
			}
			if !slices.Contains(allowedSyntheticAlertPeriods, strings.TrimSpace(*alert.Period)) {
				return fmt.Errorf("unsupported alert period %q", strings.TrimSpace(*alert.Period))
			}
		}
	}

	return nil
}

// resolveSyntheticCheckNodeMetadata resolves workflow node metadata for a synthetic check.
// When loadedCheck is non-nil (e.g. after prepareSyntheticCheckUpdate), it is used and no extra HTTP calls are made.
func resolveSyntheticCheckNodeMetadata(ctx core.SetupContext, syntheticCheck string, loadedCheck *SyntheticCheck) error {
	if isExpressionValue(syntheticCheck) || ctx.Metadata == nil || ctx.HTTP == nil {
		return nil
	}

	var nodeMeta SyntheticCheckNodeMetadata
	_ = mapstructure.Decode(ctx.Metadata.Get(), &nodeMeta)
	if nodeMeta.CheckLabel != "" && nodeMeta.SyntheticCheck == syntheticCheck {
		return nil
	}

	var check *SyntheticCheck
	if loadedCheck != nil {
		check = loadedCheck
	} else {
		client, err := NewSyntheticsClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil
		}
		var err2 error
		check, err2 = client.GetCheck(syntheticCheck)
		if err2 != nil {
			return nil
		}
	}

	nodeMeta.CheckLabel = syntheticCheckLabel(check)
	nodeMeta.SyntheticCheck = syntheticCheck
	return ctx.Metadata.Set(nodeMeta)
}

// resolveSyntheticProbeSummaryMetadata resolves configured probe IDs to human-readable labels
// (probe name and region) for workflow node metadata. It is best-effort: failures are ignored.
// When client is non-nil (e.g. from prepareSyntheticCheckUpdate), it is reused instead of building a new client.
func resolveSyntheticProbeSummaryMetadata(ctx core.SetupContext, probeIDStrings []string, client *SyntheticsClient) error {
	if ctx.Metadata == nil || ctx.HTTP == nil || len(probeIDStrings) == 0 {
		return nil
	}

	for _, probeID := range probeIDStrings {
		if isExpressionValue(probeID) {
			return nil
		}
	}

	if client == nil {
		var err error
		client, err = NewSyntheticsClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil
		}
	}

	probes, err := client.ListProbes()
	if err != nil {
		return nil
	}

	probeByID := make(map[int64]SyntheticProbe, len(probes))
	for i := range probes {
		probeByID[probes[i].ID] = probes[i]
	}

	labels := make([]string, 0, len(probeIDStrings))
	for _, idStr := range probeIDStrings {
		idStr = strings.TrimSpace(idStr)
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			labels = append(labels, idStr)
			continue
		}
		if p, ok := probeByID[id]; ok {
			labels = append(labels, syntheticProbeDisplayLabel(p))
		} else {
			labels = append(labels, idStr)
		}
	}

	summary := formatProbeSummaryLine(labels)
	if summary == "" {
		return nil
	}

	var existing SyntheticCheckNodeMetadata
	_ = mapstructure.Decode(ctx.Metadata.Get(), &existing)
	existing.ProbeSummary = summary
	return ctx.Metadata.Set(existing)
}

func syntheticProbeDisplayLabel(p SyntheticProbe) string {
	name := strings.TrimSpace(p.Name)
	region := strings.TrimSpace(p.Region)
	switch {
	case name != "" && region != "":
		return fmt.Sprintf("%s (%s)", name, region)
	case name != "":
		return name
	case region != "":
		return region
	default:
		return strconv.FormatInt(p.ID, 10)
	}
}

func formatProbeSummaryLine(labels []string) string {
	if len(labels) == 0 {
		return ""
	}
	if len(labels) == 1 {
		return labels[0]
	}
	if len(labels) <= 3 {
		return strings.Join(labels, ", ")
	}
	return fmt.Sprintf("%s +%d", strings.Join(labels[:3], ", "), len(labels)-3)
}

func syntheticCheckLabel(check *SyntheticCheck) string {
	if check == nil {
		return ""
	}

	job := strings.TrimSpace(check.Job)
	target := strings.TrimSpace(check.Target)

	switch {
	case job != "" && target != "":
		return fmt.Sprintf("%s (%s)", job, target)
	case job != "":
		return job
	default:
		return target
	}
}
