{{/*
Expand the name of the chart.
*/}}
{{- define "resource.name" -}}
{{- .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}


{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "resource.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "resource.labels" -}}
helm.sh/chart: {{ include "resource.chart" . }}
{{ include "resource.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "resource.selectorLabels" -}}
app.kubernetes.io/name: {{ include "resource.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "resource.resource" -}}
metadata:
  labels:
    {{- include "resource.labels" . | nindent 4 }}
    {{- with .Values.labels }}
      {{- toYaml . | nindent 4 }}
    {{- end }}
  {{- with .Values.annotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
    sha256sum: "{{ sha256sum (toYaml .) }}"
{{- end }}
