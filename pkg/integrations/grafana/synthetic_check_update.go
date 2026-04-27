package grafana

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

func decodeSyntheticCheckConfigMap(input any) (map[string]any, error) {
	var m map[string]any
	if err := mapstructure.Decode(input, &m); err != nil {
		return nil, fmt.Errorf("error decoding configuration: %v", err)
	}
	normalizeSyntheticCheckConfigMap(m)
	return m, nil
}

func syntheticCheckUpdateSectionPresent(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	v, ok := m[key]
	if !ok || v == nil {
		return false
	}
	return true
}

// prepareSyntheticCheckUpdate loads the existing check, merges the configuration patch, and validates the result.
// When forSetup is true and the synthetic check id is still a workflow expression, it skips the remote load so setup
// can succeed (same pattern as Get/Delete synthetic check components with resolveSyntheticCheckNodeMetadata). Execute
// must pass forSetup false so unresolved ids return a clear error instead of a failed GetCheck.
func prepareSyntheticCheckUpdate(
	httpCtx core.HTTPContext,
	integration core.IntegrationContext,
	config any,
	forSetup bool,
) (merged SyntheticCheckSpecBase, syntheticCheckID string, raw map[string]any, existing *SyntheticCheck, client *SyntheticsClient, err error) {
	raw, err = decodeSyntheticCheckConfigMap(config)
	if err != nil {
		return SyntheticCheckSpecBase{}, "", nil, nil, nil, err
	}

	id := strings.TrimSpace(fmt.Sprint(raw["syntheticCheck"]))
	if id == "" || id == "<nil>" {
		return SyntheticCheckSpecBase{}, "", raw, nil, nil, errors.New("syntheticCheck is required")
	}
	if err := validateSyntheticCheckSelection(SyntheticCheckSelectionSpec{SyntheticCheck: id}); err != nil {
		return SyntheticCheckSpecBase{}, "", raw, nil, nil, err
	}

	if isExpressionValue(id) {
		if forSetup {
			return SyntheticCheckSpecBase{}, id, raw, nil, nil, nil
		}
		return SyntheticCheckSpecBase{}, "", raw, nil, nil, errors.New("synthetic check id must be resolved before execution")
	}

	client, err = NewSyntheticsClient(httpCtx, integration)
	if err != nil {
		return SyntheticCheckSpecBase{}, "", raw, nil, nil, err
	}

	existing, err = client.GetCheck(id)
	if err != nil {
		return SyntheticCheckSpecBase{}, "", raw, nil, client, fmt.Errorf("error loading synthetic check: %w", err)
	}

	base, err := syntheticCheckToSpecBase(existing)
	if err != nil {
		return SyntheticCheckSpecBase{}, "", raw, existing, client, err
	}

	merged, err = mergeSyntheticUpdatePatch(base, raw)
	if err != nil {
		return SyntheticCheckSpecBase{}, "", raw, existing, client, err
	}

	if err := validateSyntheticCheckBase(merged); err != nil {
		return SyntheticCheckSpecBase{}, "", raw, existing, client, err
	}

	return merged, id, raw, existing, client, nil
}

func syntheticCheckToSpecBase(check *SyntheticCheck) (SyntheticCheckSpecBase, error) {
	if check == nil {
		return SyntheticCheckSpecBase{}, errors.New("check is required")
	}
	http := check.Settings.HTTP
	if http == nil {
		return SyntheticCheckSpecBase{}, errors.New("check is missing HTTP settings")
	}

	freq := syntheticFrequencySecondsFromMilliseconds(check.Frequency)
	freqMilliseconds := check.Frequency

	probes := make([]string, 0, len(check.Probes))
	for _, p := range check.Probes {
		probes = append(probes, strconv.FormatInt(p, 10))
	}

	enabled := check.Enabled
	enabledPtr := &enabled
	basicMetricsOnly := check.BasicMetricsOnly
	basicMetricsOnlyPtr := &basicMetricsOnly

	nfr := http.NoFollowRedirects
	nfrPtr := &nfr
	failSSL := http.FailIfSSL
	failSSLPtr := &failSSL
	failNotSSL := http.FailIfNotSSL
	failNotSSLPtr := &failNotSSL

	method := strings.TrimSpace(http.Method)
	if method == "" {
		method = "GET"
	}

	base := SyntheticCheckSpecBase{
		Job:                          check.Job,
		Target:                       check.Target,
		Enabled:                      enabledPtr,
		Frequency:                    freq,
		FrequencyMilliseconds:        &freqMilliseconds,
		Timeout:                      check.Timeout,
		Probes:                       probes,
		Labels:                       syntheticLabelsToInputs(check.Labels),
		AlertSensitivity:             check.AlertSensitivity,
		BasicMetricsOnly:             basicMetricsOnlyPtr,
		Method:                       method,
		Headers:                      parseSyntheticHeaderStringsToInputs(http.Headers),
		IPVersion:                    http.IPVersion,
		Compression:                  http.Compression,
		NoFollowRedirects:            nfrPtr,
		FailIfSSL:                    failSSLPtr,
		FailIfNotSSL:                 failNotSSLPtr,
		ValidStatusCodes:             append([]int(nil), http.ValidStatusCodes...),
		FailIfBodyMatchesRegexp:      append([]string(nil), http.FailIfBodyMatchesRegexp...),
		FailIfBodyNotMatchesRegexp:   append([]string(nil), http.FailIfBodyNotMatchesRegexp...),
		FailIfHeaderMatchesRegexp:    syntheticHeaderMatchesToInputs(http.FailIfHeaderMatchesRegexp),
		FailIfHeaderNotMatchesRegexp: syntheticHeaderMatchesToInputs(http.FailIfHeaderNotMatchesRegexp),
		Alerts:                       syntheticAlertsToInputs(check.Alerts),
	}

	if http.Body != "" {
		b := http.Body
		base.Body = &b
	}
	if http.BearerToken != "" {
		t := http.BearerToken
		base.BearerToken = &t
	}
	if http.BasicAuth != nil {
		base.BasicAuth = &SyntheticCheckBasicAuthInput{
			Username: http.BasicAuth.Username,
			Password: http.BasicAuth.Password,
		}
	}

	return base, nil
}

func syntheticLabelsToInputs(labels []SyntheticCheckLabel) []SyntheticCheckLabelInput {
	out := make([]SyntheticCheckLabelInput, 0, len(labels))
	for _, l := range labels {
		out = append(out, SyntheticCheckLabelInput{Name: l.Name, Value: l.Value})
	}
	return out
}

func syntheticAlertsToInputs(alerts []SyntheticCheckAlert) []SyntheticCheckAlertInput {
	out := make([]SyntheticCheckAlertInput, 0, len(alerts))
	for _, a := range alerts {
		th := a.Threshold
		var period *string
		if strings.TrimSpace(a.Period) != "" {
			p := strings.TrimSpace(a.Period)
			period = &p
		}
		var runbook *string
		if strings.TrimSpace(a.RunbookURL) != "" {
			r := strings.TrimSpace(a.RunbookURL)
			runbook = &r
		}
		out = append(out, SyntheticCheckAlertInput{
			Name:       a.Name,
			Threshold:  &th,
			Period:     period,
			RunbookURL: runbook,
		})
	}
	return out
}

func syntheticHeaderMatchesToInputs(matches []SyntheticCheckHeaderMatch) []SyntheticCheckHeaderMatchInput {
	out := make([]SyntheticCheckHeaderMatchInput, 0, len(matches))
	for _, m := range matches {
		am := m.AllowMissing
		out = append(out, SyntheticCheckHeaderMatchInput{
			Header:       m.Header,
			Regexp:       m.Regexp,
			AllowMissing: &am,
		})
	}
	return out
}

func parseSyntheticHeaderStringsToInputs(headers []string) []SyntheticCheckHeaderInput {
	out := make([]SyntheticCheckHeaderInput, 0, len(headers))
	for _, h := range headers {
		idx := strings.IndexByte(h, ':')
		if idx <= 0 {
			continue
		}
		out = append(out, SyntheticCheckHeaderInput{
			Name:  strings.TrimSpace(h[:idx]),
			Value: strings.TrimSpace(h[idx+1:]),
		})
	}
	return out
}

func mergeSyntheticUpdatePatch(base SyntheticCheckSpecBase, m map[string]any) (SyntheticCheckSpecBase, error) {
	if syntheticCheckUpdateSectionPresent(m, "job") {
		if v, ok := m["job"]; ok {
			base.Job = strings.TrimSpace(fmt.Sprint(v))
		}
	}
	if syntheticCheckUpdateSectionPresent(m, "labels") {
		var labels []SyntheticCheckLabelInput
		if err := mapstructure.Decode(m["labels"], &labels); err != nil {
			return SyntheticCheckSpecBase{}, fmt.Errorf("labels: %w", err)
		}
		base.Labels = labels
	}
	if syntheticCheckUpdateSectionPresent(m, "request") {
		req := toStringMap(m["request"])
		if err := overlaySyntheticRequest(&base, req); err != nil {
			return SyntheticCheckSpecBase{}, err
		}
	}
	if syntheticCheckUpdateSectionPresent(m, "schedule") {
		sch := toStringMap(m["schedule"])
		if err := overlaySyntheticSchedule(&base, sch); err != nil {
			return SyntheticCheckSpecBase{}, err
		}
	}
	if syntheticCheckUpdateSectionPresent(m, "validation") {
		val := toStringMap(m["validation"])
		if err := overlaySyntheticValidation(&base, val); err != nil {
			return SyntheticCheckSpecBase{}, err
		}
	}
	if syntheticCheckUpdateSectionPresent(m, "alerts") {
		var alerts []SyntheticCheckAlertInput
		if err := mapstructure.Decode(m["alerts"], &alerts); err != nil {
			return SyntheticCheckSpecBase{}, fmt.Errorf("alerts: %w", err)
		}
		base.Alerts = alerts
	}
	return base, nil
}

func overlaySyntheticRequest(base *SyntheticCheckSpecBase, req map[string]any) error {
	if _, ok := req["target"]; ok {
		base.Target = strings.TrimSpace(fmt.Sprint(req["target"]))
	}
	if _, ok := req["method"]; ok {
		base.Method = strings.TrimSpace(fmt.Sprint(req["method"]))
	}
	if _, ok := req["ipVersion"]; ok {
		base.IPVersion = strings.TrimSpace(fmt.Sprint(req["ipVersion"]))
	}
	if _, ok := req["compression"]; ok {
		base.Compression = strings.TrimSpace(fmt.Sprint(req["compression"]))
	}
	if _, ok := req["headers"]; ok {
		var headers []SyntheticCheckHeaderInput
		if err := mapstructure.Decode(req["headers"], &headers); err != nil {
			return fmt.Errorf("request.headers: %w", err)
		}
		base.Headers = headers
	}
	if _, ok := req["body"]; ok {
		if req["body"] == nil {
			base.Body = nil
		} else {
			s := fmt.Sprint(req["body"])
			base.Body = &s
		}
	}
	if _, ok := req["noFollowRedirects"]; ok {
		if req["noFollowRedirects"] == nil {
			base.NoFollowRedirects = nil
		} else {
			v := castToBool(req["noFollowRedirects"])
			base.NoFollowRedirects = &v
		}
	}
	if _, ok := req["basicAuth"]; ok {
		if req["basicAuth"] == nil {
			base.BasicAuth = nil
		} else {
			var auth SyntheticCheckBasicAuthInput
			if err := mapstructure.Decode(req["basicAuth"], &auth); err != nil {
				return fmt.Errorf("request.basicAuth: %w", err)
			}
			base.BasicAuth = &auth
		}
	}
	if _, ok := req["bearerToken"]; ok {
		if req["bearerToken"] == nil {
			base.BearerToken = nil
		} else {
			s := strings.TrimSpace(fmt.Sprint(req["bearerToken"]))
			base.BearerToken = &s
		}
	}
	return nil
}

func overlaySyntheticSchedule(base *SyntheticCheckSpecBase, sch map[string]any) error {
	if _, ok := sch["enabled"]; ok {
		if sch["enabled"] == nil {
			base.Enabled = nil
		} else {
			v := castToBool(sch["enabled"])
			base.Enabled = &v
		}
	}
	if _, ok := sch["frequency"]; ok {
		base.Frequency = castToInt64(sch["frequency"])
		base.FrequencyMilliseconds = nil
	}
	if _, ok := sch["frequencyMilliseconds"]; ok {
		v := castToInt64(sch["frequencyMilliseconds"])
		if v > 0 {
			base.FrequencyMilliseconds = &v
		}
	}
	if _, ok := sch["timeout"]; ok {
		base.Timeout = castToInt64(sch["timeout"])
	}
	if _, ok := sch["probes"]; ok {
		var probes []string
		if err := mapstructure.Decode(sch["probes"], &probes); err != nil {
			return fmt.Errorf("schedule.probes: %w", err)
		}
		base.Probes = probes
	}
	return nil
}

func overlaySyntheticValidation(base *SyntheticCheckSpecBase, val map[string]any) error {
	if _, ok := val["failIfSSL"]; ok {
		if val["failIfSSL"] == nil {
			base.FailIfSSL = nil
		} else {
			v := castToBool(val["failIfSSL"])
			base.FailIfSSL = &v
		}
	}
	if _, ok := val["failIfNotSSL"]; ok {
		if val["failIfNotSSL"] == nil {
			base.FailIfNotSSL = nil
		} else {
			v := castToBool(val["failIfNotSSL"])
			base.FailIfNotSSL = &v
		}
	}
	if _, ok := val["validStatusCodes"]; ok {
		var codes []int
		if err := mapstructure.Decode(val["validStatusCodes"], &codes); err != nil {
			return fmt.Errorf("validation.validStatusCodes: %w", err)
		}
		base.ValidStatusCodes = codes
	}
	if _, ok := val["failIfBodyMatchesRegexp"]; ok {
		var xs []string
		if err := mapstructure.Decode(val["failIfBodyMatchesRegexp"], &xs); err != nil {
			return fmt.Errorf("validation.failIfBodyMatchesRegexp: %w", err)
		}
		base.FailIfBodyMatchesRegexp = xs
	}
	if _, ok := val["failIfBodyNotMatchesRegexp"]; ok {
		var xs []string
		if err := mapstructure.Decode(val["failIfBodyNotMatchesRegexp"], &xs); err != nil {
			return fmt.Errorf("validation.failIfBodyNotMatchesRegexp: %w", err)
		}
		base.FailIfBodyNotMatchesRegexp = xs
	}
	if _, ok := val["failIfHeaderMatchesRegexp"]; ok {
		var xs []SyntheticCheckHeaderMatchInput
		if err := mapstructure.Decode(val["failIfHeaderMatchesRegexp"], &xs); err != nil {
			return fmt.Errorf("validation.failIfHeaderMatchesRegexp: %w", err)
		}
		base.FailIfHeaderMatchesRegexp = xs
	}
	if _, ok := val["failIfHeaderNotMatchesRegexp"]; ok {
		var xs []SyntheticCheckHeaderMatchInput
		if err := mapstructure.Decode(val["failIfHeaderNotMatchesRegexp"], &xs); err != nil {
			return fmt.Errorf("validation.failIfHeaderNotMatchesRegexp: %w", err)
		}
		base.FailIfHeaderNotMatchesRegexp = xs
	}
	return nil
}

func castToBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return strings.EqualFold(t, "true") || t == "1"
	default:
		return castToInt64(v) != 0
	}
}

func castToInt64(v any) int64 {
	switch t := v.(type) {
	case int:
		return int64(t)
	case int64:
		return t
	case float64:
		return int64(t)
	case float32:
		return int64(t)
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0
		}
		return i
	}
}
