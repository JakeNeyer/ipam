{{/*
Expand the name of the chart.
*/}}
{{- define "ipam.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "ipam.fullname" -}}
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

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "ipam.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "ipam.labels" -}}
helm.sh/chart: {{ include "ipam.chart" . }}
{{ include "ipam.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "ipam.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ipam.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "ipam.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "ipam.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
OAuth provider id -> env prefix (e.g. keycloak -> OAUTH_KEYCLOAK_).
Must match server/config oauthEnvPrefix (uppercase, hyphens to underscores).
*/}}
{{- define "ipam.oauthEnvPrefix" -}}
{{- $id := .id | replace "-" "_" | upper -}}
{{- printf "OAUTH_%s_" $id -}}
{{- end }}

{{/*
True when the provider has the minimum config to be enabled.
*/}}
{{- define "ipam.oauthProviderConfigured" -}}
{{- $p := .provider -}}
{{- if and $p.clientId $p.authUrl $p.tokenUrl $p.userInfoUrl -}}
true
{{- end -}}
{{- end }}

{{/*
Comma-separated OAUTH_PROVIDERS value.
*/}}
{{- define "ipam.oauthProvidersList" -}}
{{- $ids := list -}}
{{- range $id, $p := .Values.oauth.providers -}}
{{- if include "ipam.oauthProviderConfigured" (dict "provider" $p) -}}
{{- $ids = append $ids $id -}}
{{- end -}}
{{- end -}}
{{- join "," $ids -}}
{{- end }}

{{/*
Secret key in existingSecret for a provider client secret.
*/}}
{{- define "ipam.oauthClientSecretKey" -}}
{{- default (printf "oauth-%s-client-secret" .id) .provider.existingSecretKey -}}
{{- end }}

{{/*
OAuth-related container env vars (OAUTH_PROVIDERS and OAUTH_<ID>_*).
*/}}
{{- define "ipam.oauthEnv" }}
{{- $root := . }}
{{- $providers := .Values.oauth.providers | default dict }}
{{- if $providers }}
{{- $list := include "ipam.oauthProvidersList" $root }}
{{- if $list }}

- name: OAUTH_PROVIDERS
  value: {{ $list | quote }}
{{- range $id, $p := $providers }}
{{- if include "ipam.oauthProviderConfigured" (dict "provider" $p) }}
{{- $prefix := include "ipam.oauthEnvPrefix" (dict "id" $id) }}

- name: {{ $prefix }}CLIENT_ID
  value: {{ $p.clientId | quote }}
{{- if $p.clientSecret }}
- name: {{ $prefix }}CLIENT_SECRET
  value: {{ $p.clientSecret | quote }}
{{- else if $root.Values.existingSecret }}
- name: {{ $prefix }}CLIENT_SECRET
  valueFrom:
    secretKeyRef:
      name: {{ $root.Values.existingSecret }}
      key: {{ include "ipam.oauthClientSecretKey" (dict "id" $id "provider" $p) }}
{{- end }}
- name: {{ $prefix }}AUTH_URL
  value: {{ $p.authUrl | quote }}
- name: {{ $prefix }}TOKEN_URL
  value: {{ $p.tokenUrl | quote }}
- name: {{ $prefix }}USERINFO_URL
  value: {{ $p.userInfoUrl | quote }}
{{- if $p.scopes }}
- name: {{ $prefix }}SCOPES
  value: {{ join "," $p.scopes | quote }}
{{- end }}
{{- if $p.displayName }}
- name: {{ $prefix }}DISPLAY_NAME
  value: {{ $p.displayName | quote }}
{{- end }}
{{- if $p.userIdClaim }}
- name: {{ $prefix }}USER_ID_CLAIM
  value: {{ $p.userIdClaim | quote }}
{{- end }}
{{- if $p.emailClaim }}
- name: {{ $prefix }}EMAIL_CLAIM
  value: {{ $p.emailClaim | quote }}
{{- end }}
{{- if $p.emailsUrl }}
- name: {{ $prefix }}EMAILS_URL
  value: {{ $p.emailsUrl | quote }}
{{- end }}
{{- if $p.emailsPrimaryClaim }}
- name: {{ $prefix }}EMAILS_PRIMARY_CLAIM
  value: {{ $p.emailsPrimaryClaim | quote }}
{{- end }}
{{- if $p.emailVerifiedClaim }}
- name: {{ $prefix }}EMAIL_VERIFIED_CLAIM
  value: {{ $p.emailVerifiedClaim | quote }}
{{- end }}
{{- if $p.emailsVerifiedClaim }}
- name: {{ $prefix }}EMAILS_VERIFIED_CLAIM
  value: {{ $p.emailsVerifiedClaim | quote }}
{{- end }}
{{- if $p.allowEmailMatch }}
- name: {{ $prefix }}ALLOW_EMAIL_MATCH
  value: {{ $p.allowEmailMatch | quote }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
