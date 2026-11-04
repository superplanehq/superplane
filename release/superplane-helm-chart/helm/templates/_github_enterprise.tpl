{{- define "superplane.githubEnterpriseEnv" -}}
- name: SUPERPLANE_GITHUB_ENTERPRISE_ALLOWED_HOSTS
  value: {{ .Values.githubEnterprise.allowedHosts | quote }}
{{- if .Values.githubEnterprise.extraCACertsSecret }}
- name: SSL_CERT_FILE
  value: /etc/ssl/superplane/ca-certificates.crt
{{- end }}
{{- end }}

{{- define "superplane.githubEnterpriseVolumeMount" -}}
{{- if .Values.githubEnterprise.extraCACertsSecret }}
- name: github-enterprise-ca
  mountPath: /etc/ssl/superplane
  readOnly: true
{{- end }}
{{- end }}

{{- define "superplane.githubEnterpriseVolume" -}}
{{- if .Values.githubEnterprise.extraCACertsSecret }}
- name: github-enterprise-ca
  secret:
    secretName: {{ .Values.githubEnterprise.extraCACertsSecret | quote }}
    items:
      - key: {{ .Values.githubEnterprise.extraCACertsKey | default "ca-certificates.crt" | quote }}
        path: ca-certificates.crt
{{- end }}
{{- end }}
