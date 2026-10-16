{{- define "superplane.billingEnabled" -}}
{{- $raw := .Values.billing.enabled | default "" | toString | lower | trim -}}
{{- if or (eq $raw "yes") (eq $raw "true") -}}
yes
{{- else if or (eq $raw "no") (eq $raw "false") -}}
no
{{- else if .Values.polar.enabled -}}
yes
{{- else -}}
no
{{- end -}}
{{- end -}}

{{- define "superplane.billingEnabledEnv" -}}
- name: SUPERPLANE_BILLING_ENABLED
  value: {{ include "superplane.billingEnabled" . | quote }}
{{- end -}}
