{{/*
Expand the name of the chart.
*/}}
{{- define "es-index-pruner.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "es-index-pruner.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end }}


{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "es-index-pruner.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "es-index-pruner.labels" -}}
helm.sh/chart: {{ include "es-index-pruner.chart" . }}
{{ include "es-index-pruner.selectorLabels" . }}
{{- end }}

{{- define "es-index-pruner.selectorLabels" -}}
app.kubernetes.io/name: {{ include "es-index-pruner.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
