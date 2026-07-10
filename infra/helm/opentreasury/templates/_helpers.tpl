{{- define "opentreasury.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "opentreasury.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define "opentreasury.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
app.kubernetes.io/name: {{ include "opentreasury.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/* Selector labels for one component; call with (dict "root" . "component" "core-api") */}}
{{- define "opentreasury.selectorLabels" -}}
app.kubernetes.io/name: {{ include "opentreasury.name" .root }}
app.kubernetes.io/instance: {{ .root.Release.Name }}
app.kubernetes.io/component: {{ .component }}
{{- end }}

{{/* Image reference for one component; call with (dict "root" . "component" "core-api") */}}
{{- define "opentreasury.image" -}}
{{- $tag := default .root.Chart.AppVersion .root.Values.image.tag -}}
{{- printf "%s/%s:%s" .root.Values.image.registry .component $tag }}
{{- end }}

{{- define "opentreasury.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "opentreasury.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/* Name of the Secret holding the database DSN. */}}
{{- define "opentreasury.databaseSecretName" -}}
{{- if .Values.database.existingSecret }}
{{- .Values.database.existingSecret }}
{{- else }}
{{- printf "%s-database" (include "opentreasury.fullname" .) }}
{{- end }}
{{- end }}

{{/* Restrictive pod security context shared by the Go services. */}}
{{- define "opentreasury.securityContext" -}}
allowPrivilegeEscalation: false
readOnlyRootFilesystem: true
runAsNonRoot: true
runAsUser: 65532
capabilities:
  drop: ["ALL"]
seccompProfile:
  type: RuntimeDefault
{{- end }}

{{/* Prometheus scrape annotations; call with (dict "root" . "port" 8080) */}}
{{- define "opentreasury.metricsAnnotations" -}}
{{- if .root.Values.metrics.annotations }}
prometheus.io/scrape: "true"
prometheus.io/port: {{ .port | quote }}
prometheus.io/path: /metrics
{{- end }}
{{- end }}
