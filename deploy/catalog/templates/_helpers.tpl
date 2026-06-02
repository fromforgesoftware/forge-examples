{{- define "catalog.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "catalog.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name (include "catalog.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "catalog.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{ include "catalog.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: petstore
{{- end -}}

{{- define "catalog.selectorLabels" -}}
app.kubernetes.io/name: {{ include "catalog.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "catalog.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "catalog.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "catalog.image" -}}
{{- printf "%s:%s" .Values.image.repository (.Values.image.tag | default .Chart.AppVersion) -}}
{{- end -}}

{{- define "catalog.dbSecretName" -}}
{{- if .Values.database.existingSecret -}}
{{- .Values.database.existingSecret -}}
{{- else -}}
{{- printf "%s-db" (include "catalog.fullname" .) -}}
{{- end -}}
{{- end -}}

{{/* Shared env block for server + migrator. */}}
{{- define "catalog.env" -}}
- name: SVC_NAME
  value: {{ include "catalog.name" . | quote }}
- name: REST_ADDRESS
  value: ":{{ .Values.ports.http }}"
- name: HTTP_ADDRESS
  value: ":{{ .Values.ports.http }}"
- name: DB_HOST
  value: {{ .Values.database.host | quote }}
- name: DB_PORT
  value: {{ .Values.database.port | quote }}
- name: DB_NAME
  value: {{ .Values.database.name | quote }}
- name: DB_SCHEMA
  value: {{ .Values.database.schema | quote }}
- name: DB_SSL
  value: {{ .Values.database.ssl | quote }}
- name: DB_LOG_LEVEL
  value: {{ .Values.database.logLevel | default "warn" | quote }}
- name: DB_USER
  valueFrom:
    secretKeyRef:
      name: {{ include "catalog.dbSecretName" . }}
      key: DB_USER
- name: DB_PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{ include "catalog.dbSecretName" . }}
      key: DB_PASSWORD
- name: AEGIS_ISSUER
  value: {{ .Values.aegisIssuer | quote }}
{{- end -}}
