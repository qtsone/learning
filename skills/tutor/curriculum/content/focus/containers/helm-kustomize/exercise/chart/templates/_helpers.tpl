{{/*
Helpers are this chart's private functions: a name is defined once here and
included everywhere it is needed, so a release named "staging" cannot end up
with a Deployment called "timesvc" and a Service called "staging-timesvc".

Files starting with an underscore render nothing themselves — Helm skips them
when it collects manifests, and only the definitions below survive.

Nothing here is a TODO. Use these from your templates:

    {{ include "timesvc.fullname" . }}
    {{- include "timesvc.labels" . | nindent 4 }}
*/}}

{{- define "timesvc.name" -}}
{{- .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Release name plus chart name: unique per release inside a namespace, and
without the "timesvc-timesvc" you get when the release is named after the
chart. 63 characters is the Kubernetes limit for a name.
*/}}
{{- define "timesvc.fullname" -}}
{{- if contains .Chart.Name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name (include "timesvc.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{/*
Selector labels are the ones a Service selects on and a Deployment matches on.
They are kept separate from the full label set because a Deployment's
spec.selector is immutable: put a label that changes (a version, a chart
revision) in here and the next upgrade fails with "field is immutable".
*/}}
{{- define "timesvc.selectorLabels" -}}
app.kubernetes.io/name: {{ include "timesvc.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "timesvc.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version }}
{{ include "timesvc.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}
