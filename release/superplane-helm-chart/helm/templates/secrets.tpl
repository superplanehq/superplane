{{- define "secrets.jwt.name" }}
{{- if eq .Values.jwt.secretName "" }}
{{- printf "%s-jwt" .Release.Name }}
{{- else }}
{{- .Values.jwt.secretName }}
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

{{- define "secrets.installation.name" }}
{{- if eq .Values.installation.secretName "" }}
{{- printf "%s-installation" .Release.Name }}
{{- else }}
{{- .Values.installation.secretName }}
{{- end }}
{{- end }}

{{- define "secrets.sentry.name" }}
{{- if eq .Values.sentry.secretName "" }}
{{- printf "%s-sentry" .Release.Name }}
{{- else }}
{{- .Values.sentry.secretName }}
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
