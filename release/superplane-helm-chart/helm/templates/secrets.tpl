{{- define "secrets.jwt.name" }}
{{- if eq .Values.jwt.secretName "" }}
{{- printf "%s-jwt" .Release.Name }}
{{- else }}
{{- .Values.jwt.secretName }}
{{- end }}
{{- end }}

{{- define "secrets.oidc.name" }}
{{- if eq .Values.oidc.secretName "" }}
{{- printf "%s-oidc" .Release.Name }}
{{- else }}
{{- .Values.oidc.secretName }}
{{- end }}
{{- end }}

{{- define "secrets.authentication.name" }}
{{- if eq .Values.authentication.secretName "" }}
{{- printf "%s-authentication" .Release.Name }}
{{- else }}
{{- .Values.authentication.secretName }}
{{- end }}
{{- end }}

{{- define "secrets.telemetry.name" }}
{{- if eq .Values.telemetry.secretName "" }}
{{- printf "%s-telemetry" .Release.Name }}
{{- else }}
{{- .Values.telemetry.secretName }}
{{- end }}
{{- end }}


{{- define "secrets.sentry.name" }}
{{- if eq .Values.sentry.secretName "" }}
{{- printf "%s-sentry" .Release.Name }}
{{- else }}
{{- .Values.sentry.secretName }}
{{- end }}
{{- end }}

{{- define "secrets.dash0Web.name" }}
{{- if eq .Values.dash0Web.secretName "" }}
{{- printf "%s-dash0-web" .Release.Name }}
{{- else }}
{{- .Values.dash0Web.secretName }}
{{- end }}
{{- end }}

{{- define "secrets.posthog.name" }}
{{- if eq .Values.posthog.secretName "" }}
{{- printf "%s-posthog" .Release.Name }}
{{- else }}
{{- .Values.posthog.secretName }}
{{- end }}
{{- end }}

{{- define "secrets.polar.name" }}
{{- if eq .Values.polar.secretName "" }}
{{- printf "%s-polar" .Release.Name }}
{{- else }}
{{- .Values.polar.secretName }}
{{- end }}
{{- end }}

{{- define "secrets.linear.name" }}
{{- if eq .Values.linear.secretName "" }}
{{- printf "%s-linear" .Release.Name }}
{{- else }}
{{- .Values.linear.secretName }}
{{- end }}
{{- end }}

{{- define "superplane.serviceAccountName" -}}
{{- if .Values.serviceAccount.name }}
{{- .Values.serviceAccount.name }}
{{- else if .Values.serviceAccount.create }}
{{- .Release.Name }}
{{- else }}
{{- "default" }}
{{- end }}
{{- end }}

{{- define "runner.workers.enabled" -}}
{{- if kindIs "bool" .Values.runner.workers.enabled -}}
{{- .Values.runner.workers.enabled -}}
{{- else -}}
{{- .Values.runner.api.enabled -}}
{{- end -}}
{{- end -}}

{{- define "runner.activeLogs.claimName" -}}
{{- if .Values.runner.activeLogs.existingClaim -}}
{{- .Values.runner.activeLogs.existingClaim -}}
{{- else -}}
{{- printf "%s-runner-active-logs" .Release.Name -}}
{{- end -}}
{{- end -}}

{{- define "runner.processEnv" -}}
{{- $enabled := .enabled -}}
{{- range $flag := list
  "START_PUBLIC_API"
  "START_GRPC_GATEWAY"
  "START_EVENT_DISTRIBUTER"
  "START_WEBSOCKET_SERVER"
  "START_WEB_SERVER"
  "START_RBAC_POLICY_RELOAD_CONSUMER"
  "START_FILE_CLEANUP_WORKER"
  "START_RUNNER_API"
  "START_RUNNER_LOG_COMPACTOR"
  "START_RUNNER_CLEANUP_WORKER"
  "START_CONSUMERS"
  "START_EVENT_ROUTER"
  "START_RUN_FINALIZER"
  "START_RUN_INITIALIZER"
  "START_NODE_EXECUTOR"
  "START_EXECUTION_TERMINATOR"
  "START_NODE_QUEUE_WORKER"
  "START_NODE_REQUEST_WORKER"
  "START_APP_MESSAGE_WORKER"
  "START_INTEGRATION_REQUEST_WORKER"
  "START_WEBHOOK_PROVISIONER"
  "START_WEBHOOK_CLEANUP_WORKER"
  "START_INTEGRATION_CLEANUP_WORKER"
  "START_CANVAS_CLEANUP_WORKER"
  "START_FACTORY_CLEANUP_WORKER"
  "START_FACTORY_VELOCITY_SYNC_WORKER"
  "START_PRICE_BOOK_SYNC_WORKER"
  "START_PLANNING_SESSION_CLEANUP_WORKER"
  "START_EVENT_RETENTION_WORKER"
  "START_NODE_REQUEST_CLEANUP_WORKER"
}}
- name: {{ $flag }}
  value: {{ ternary "yes" "no" (has $flag $enabled) | quote }}
{{- end }}
{{- end -}}

{{- define "superplane.fleetManager.serviceAccountName" -}}
{{- if .Values.fleetManager.serviceAccount.name }}
{{- .Values.fleetManager.serviceAccount.name }}
{{- else if .Values.fleetManager.serviceAccount.create }}
{{- printf "%s-fleet-manager" .Release.Name }}
{{- else }}
{{- "default" }}
{{- end }}
{{- end }}

{{- define "superplane.podLabels" -}}
{{- range $key, $value := . }}
{{ $key }}: {{ $value | toString | quote }}
{{- end }}
{{- end }}

{{- define "secrets.encryption.name" }}
{{- if eq .Values.encryption.secretName "" }}
{{- printf "%s-encryption" .Release.Name }}
{{- else }}
{{- .Values.encryption.secretName }}
{{- end }}
{{- end }}

{{- define "secrets.session.name" }}
{{- if eq .Values.session.secretName "" }}
{{- printf "%s-session" .Release.Name }}
{{- else }}
{{- .Values.session.secretName }}
{{- end }}
{{- end }}

{{- define "secrets.rabbitmq.name" }}
{{- if eq .Values.rabbitmq.secretName "" }}
{{- printf "%s-rabbitmq" .Release.Name }}
{{- else }}
{{- .Values.rabbitmq.secretName }}
{{- end }}
{{- end }}

{{- define "secrets.database.name" }}
{{- if eq .Values.database.secretName "" }}
{{- printf "%s-database" .Release.Name }}
{{- else }}
{{- .Values.database.secretName }}
{{- end }}
{{- end }}

{{- define "secrets.email.name" }}
{{- if eq .Values.email.secretName "" }}
{{- printf "%s-email" .Release.Name }}
{{- else }}
{{- .Values.email.secretName }}
{{- end }}
{{- end }}

{{- define "secrets.feedback.name" }}
{{- printf "%s-feedback" .Release.Name }}
{{- end }}

{{- define "superplane.migrateWaitContainer" -}}
- name: wait-for-migrate
  image: "{{ .Values.image.registry }}/{{ .Values.image.name }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
  imagePullPolicy: {{ .Values.image.pullPolicy }}
  command: ["bash", "-c"]
  args:
    - |
      set -euo pipefail
      if [ "${POSTGRES_DB_SSL:-}" = "true" ]; then
        export PGSSLMODE=require
      else
        export PGSSLMODE=disable
      fi
      latest_schema=$(ls /app/db/migrations/*.up.sql | sed 's#.*/##' | cut -d_ -f1 | sort -n | tail -1)
      latest_data=""
      if ls /app/db/data_migrations/*.up.sql >/dev/null 2>&1; then
        latest_data=$(ls /app/db/data_migrations/*.up.sql | sed 's#.*/##' | cut -d_ -f1 | sort -n | tail -1)
      fi
      echo "Waiting for schema ${latest_schema}"
      for _ in $(seq 1 90); do
        schema=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d "$DB_NAME" -tAc "SELECT version FROM schema_migrations WHERE dirty = false ORDER BY version DESC LIMIT 1" 2>/dev/null || true)
        schema=$(printf '%s' "$schema" | tr -d '[:space:]')
        if [ "$schema" != "$latest_schema" ]; then
          sleep 5
          continue
        fi
        if [ -z "$latest_data" ]; then
          exit 0
        fi
        data=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d "$DB_NAME" -tAc "SELECT version FROM data_migrations WHERE dirty = false ORDER BY version DESC LIMIT 1" 2>/dev/null || true)
        data=$(printf '%s' "$data" | tr -d '[:space:]')
        if [ "$data" = "$latest_data" ]; then
          exit 0
        fi
        sleep 5
      done
      echo "Timed out waiting for migrations"
      exit 1
  envFrom:
    - secretRef:
        name: {{ include "secrets.database.name" . }}
  securityContext:
    allowPrivilegeEscalation: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
    capabilities:
      drop: ["ALL"]
{{- end }}
