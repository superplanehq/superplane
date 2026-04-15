package grafana

import (
	"encoding/base64"
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

func prepareSyntheticCheckUpdate(
	httpCtx core.HTTPContext,
	integration core.IntegrationContext,
	config any,
) (merged SyntheticCheckSpecBase, syntheticCheckID string, raw map[string]any, existing *SyntheticCheck, err error) {
	raw, err = decodeSyntheticCheckConfigMap(config)
	if err != nil {
		return SyntheticCheckSpecBase{}, "", nil, nil, err
	}

	id := strings.TrimSpace(fmt.Sprint(raw["syntheticCheck"]))
	if id == "" || id == "<nil>" {
		return SyntheticCheckSpecBase{}, "", raw, nil, errors.New("syntheticCheck is required")
	}
	if err := validateSyntheticCheckSelection(SyntheticCheckSelectionSpec{SyntheticCheck: id}); err != nil {
		return SyntheticCheckSpecBase{}, "", raw, nil, err
	}

	client, err := NewSyntheticsClient(httpCtx, integration)
	if err != nil {
		return SyntheticCheckSpecBase{}, "", raw, nil, err
	}

	existing, err = client.GetCheck(id)
	if err != nil {
		return SyntheticCheckSpecBase{}, "", raw, nil, fmt.Errorf("error loading synthetic check: %w", err)
	}

	base, err := syntheticCheckToSpecBase(existing)
	if err != nil {
		return SyntheticCheckSpecBase{}, "", raw, existing, err
	}

	merged, err = mergeSyntheticUpdatePatch(base, raw)
	if err != nil {
		return SyntheticCheckSpecBase{}, "", raw, existing, err
	}

	if err := validateSyntheticCheckBase(merged); err != nil {
		return SyntheticCheckSpecBase{}, "", raw, existing, err
	}

	return merged, id, raw, existing, nil
}

func syntheticCheckToSpecBase(check *SyntheticCheck) (SyntheticCheckSpecBase, error) {
	if check == nil {
		return SyntheticCheckSpecBase{}, errors.New("check is required")
	}
	http := check.Settings.HTTP
	if http == nil {
		return SyntheticCheckSpecBase{}, errors.New("check is missing HTTP settings")
	}

	freq := check.Frequency
	if freq >= 1000 && freq%1000 == 0 {
		freq /= 1000
	}

	probes := make([]string, 0, len(check.Probes))
	for _, p := range check.Probes {
		probes = append(probes, strconv.FormatInt(p, 10))
	}

	enabled := check.Enabled
	enabledPtr := &enabled

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
		Job:                        check.Job,
		Target:                     check.Target,
		Enabled:                    enabledPtr,
		Frequency:                  freq,
		Timeout:                    check.Timeout,
		Probes:                     probes,
		Labels:                     syntheticLabelsToInputs(check.Labels),
		Method:                     method,
		Headers:                    parseSyntheticHeaderStringsToInputs(http.Headers),
		NoFollowRedirects:          nfrPtr,
		FailIfSSL:                  failSSLPtr,
		FailIfNotSSL:               failNotSSLPtr,
		ValidStatusCodes:           append([]int(nil), http.ValidStatusCodes...),
		FailIfBodyMatchesRegexp:    append([]string(nil), http.FailIfBodyMatchesRegexp...),
		FailIfBodyNotMatchesRegexp: append([]string(nil), http.FailIfBodyNotMatchesRegexp...),
		FailIfHeaderMatchesRegexp:  syntheticHeaderMatchesToInputs(http.FailIfHeaderMatchesRegexp),
		TLS:                        syntheticTLSConfigToInput(http.TLSConfig),
		Alerts:                     syntheticAlertsToInputs(check.Alerts),
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

func syntheticTLSConfigToInput(cfg *SyntheticCheckTLSConfig) *SyntheticCheckTLSInput {
	if cfg == nil {
		return nil
	}
	insecure := cfg.InsecureSkipVerify
	input := &SyntheticCheckTLSInput{
		ServerName:         strings.TrimSpace(cfg.ServerName),
		InsecureSkipVerify: &insecure,
	}
	if cfg.CACert != "" {
		if s, err := decodeSyntheticPEMFromAPI(cfg.CACert); err == nil && strings.TrimSpace(s) != "" {
			input.CACert = &s
		}
	}
	if cfg.ClientCert != "" {
		if s, err := decodeSyntheticPEMFromAPI(cfg.ClientCert); err == nil && strings.TrimSpace(s) != "" {
			input.ClientCert = &s
		}
	}
	if cfg.ClientKey != "" {
		if s, err := decodeSyntheticPEMFromAPI(cfg.ClientKey); err == nil && strings.TrimSpace(s) != "" {
			input.ClientKey = &s
		}
	}
	hasPEM := input.CACert != nil || input.ClientCert != nil || input.ClientKey != nil
	if !insecure && input.ServerName == "" && !hasPEM {
		return nil
	}
	return input
}

func decodeSyntheticPEMFromAPI(encoded string) (string, error) {
	trimmed := strings.TrimSpace(encoded)
	if trimmed == "" {
		return "", errors.New("empty")
	}
	if strings.Contains(trimmed, "-----BEGIN") {
		return trimmed, nil
	}
	data, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		return "", err
	}
	return string(data), nil
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
			s := strings.TrimSpace(fmt.Sprint(req["body"]))
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
	if _, ok := req["tls"]; ok {
		if req["tls"] == nil {
			base.TLS = nil
		} else {
			var tlsInput SyntheticCheckTLSInput
			if err := mapstructure.Decode(req["tls"], &tlsInput); err != nil {
				return fmt.Errorf("request.tls: %w", err)
			}
			base.TLS = &tlsInput
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
