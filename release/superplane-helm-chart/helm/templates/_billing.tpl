{{- define "superplane.billingEnabled" -}}
{{- if kindIs "bool" .Values.billing.enabled -}}
{{- if .Values.billing.enabled -}}
yes
{{- else -}}
no
{{- end -}}
{{- else -}}
{{- $raw := .Values.billing.enabled | default "" | toString | lower | trim -}}
{{- if or (eq $raw "yes") (eq $raw "true") -}}
yes
{{- else -}}
no
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "superplane.billingEnabledEnv" -}}
- name: SUPERPLANE_BILLING_ENABLED
  value: {{ include "superplane.billingEnabled" . | quote }}
{{- end -}}
