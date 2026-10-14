{{- define "fleet-manager.name" -}}
{{- printf "%s-fleet-manager" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "fleet-manager.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "fleet-manager.name" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- required "serviceAccount.name is required when serviceAccount.create is false" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "fleet-manager.labels" -}}
app.kubernetes.io/name: fleet-manager
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | quote }}
{{- end -}}

{{- define "fleet-manager.runtimeConfig" -}}
{{- $fleets := list -}}
{{- range .Values.fleets -}}
{{- $fleets = append $fleets (omit . "runnerVersion" "spec") -}}
{{- end -}}
{{- $config := dict
  "id" .Values.manager.id
  "superplaneUrl" (.Values.superplane.url | trimSuffix "/")
  "runnerReleaseBaseUrl" .Values.runner.releaseBaseUrl
  "reconciliation" .Values.reconciliation
  "http" .Values.http
  "fleets" $fleets
-}}
{{ toYaml $config }}
{{- end -}}
