{{/*
Security context blocks.

toYaml sorts keys and writes lists as blocks. The previous templates used a
fixed order and a flow list, for example drop: [ALL]. When the values still
match those defaults, emit the previous text so a default render stays the
same. Emit toYaml when a caller overrides the values.
*/}}

{{- define "superplane.writeSecurityContext" -}}
{{- if .text -}}
{{- $rendered := .text | trim | nindent (int .indent) | trimSuffix "\n" -}}
{{- if .skipLeadingNewline -}}
{{- trimPrefix "\n" $rendered -}}
{{- else -}}
{{- $rendered -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "superplane.securityContextText" -}}
{{- if eq (toJson .context) (toJson .default) -}}
{{- .legacy -}}
{{- else -}}
{{- printf "securityContext:%s" (.context | toYaml | nindent 2) -}}
{{- end -}}
{{- end -}}

{{- define "superplane.appContainerSecurityContext" -}}
{{- $default := dict "privileged" false "readOnlyRootFilesystem" true "capabilities" (dict "drop" (list "ALL")) -}}
{{- $legacy := "securityContext:\n  privileged: false\n  readOnlyRootFilesystem: true\n  capabilities:\n    drop: [ALL]" -}}
{{- $text := include "superplane.securityContextText" (dict "context" .securityContext "default" $default "legacy" $legacy) -}}
{{- include "superplane.writeSecurityContext" (dict "text" $text "indent" .indent "skipLeadingNewline" .skipLeadingNewline) -}}
{{- end -}}

{{- define "superplane.appPodSecurityContext" -}}
{{- $ctx := dict -}}
{{- range $key, $value := (.podSecurityContext | default dict) -}}
{{- $_ := set $ctx $key $value -}}
{{- end -}}
{{- if .mountLogs -}}
{{- $_ := set $ctx "fsGroup" .fsGroup -}}
{{- end -}}
{{- $text := "" -}}
{{- if not $ctx -}}
{{- else if and .mountLogs (eq (len (keys $ctx)) 1) -}}
{{- $text = printf "securityContext:\n  fsGroup: %v" .fsGroup -}}
{{- else -}}
{{- $text = printf "securityContext:%s" ($ctx | toYaml | nindent 2) -}}
{{- end -}}
{{- include "superplane.writeSecurityContext" (dict "text" $text "indent" .indent) -}}
{{- end -}}

{{- define "superplane.migratePodSecurityContext" -}}
{{- $ctx := deepCopy (.migratePodSecurityContext | default dict) -}}
{{- if .podSecurityContext -}}
{{- $ctx = mergeOverwrite $ctx (deepCopy .podSecurityContext) -}}
{{- end -}}
{{- $default := dict "runAsNonRoot" true "runAsUser" 65534 "runAsGroup" 65534 "fsGroup" 65534 "seccompProfile" (dict "type" "RuntimeDefault") -}}
{{- $legacy := "securityContext:\n  runAsNonRoot: true\n  runAsUser: 65534\n  runAsGroup: 65534\n  fsGroup: 65534\n  seccompProfile:\n    type: RuntimeDefault" -}}
{{- $text := include "superplane.securityContextText" (dict "context" $ctx "default" $default "legacy" $legacy) -}}
{{- include "superplane.writeSecurityContext" (dict "text" $text "indent" .indent) -}}
{{- end -}}

{{- define "superplane.migrateContainerSecurityContext" -}}
{{- $ctx := deepCopy (.migrateSecurityContext | default dict) -}}
{{- $appDefault := dict "privileged" false "readOnlyRootFilesystem" true "capabilities" (dict "drop" (list "ALL")) -}}
{{- if and .securityContext (ne (toJson .securityContext) (toJson $appDefault)) -}}
{{- $ctx = mergeOverwrite $ctx (deepCopy .securityContext) -}}
{{- end -}}
{{- $default := dict "allowPrivilegeEscalation" false "readOnlyRootFilesystem" true "runAsNonRoot" true "capabilities" (dict "drop" (list "ALL")) -}}
{{- $legacy := "securityContext:\n  allowPrivilegeEscalation: false\n  readOnlyRootFilesystem: true\n  runAsNonRoot: true\n  capabilities:\n    drop: [ALL]" -}}
{{- if .quoteDrop -}}
{{- $legacy = "securityContext:\n  allowPrivilegeEscalation: false\n  readOnlyRootFilesystem: true\n  runAsNonRoot: true\n  capabilities:\n    drop: [\"ALL\"]" -}}
{{- end -}}
{{- $text := include "superplane.securityContextText" (dict "context" $ctx "default" $default "legacy" $legacy) -}}
{{- include "superplane.writeSecurityContext" (dict "text" $text "indent" .indent) -}}
{{- end -}}
