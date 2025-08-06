{{/* vim: set filetype=mustache: */}}

{{- define "unit-operator.fullname" -}}
{{- include "common.names.fullname" . -}}
{{- end -}}

{{/*
Return the proper image name
*/}}
{{- define "unit-operator.image" -}}
{{ include "common.images.image" (dict "imageRoot" .Values.image "global" .Values.global) }}
{{- end -}}

{{/*
Return the proper Docker Image Registry Secret Names
*/}}
{{- define "unit-operator.imagePullSecrets" -}}
{{- include "common.images.pullSecrets" (dict "images" (list .Values.image) "global" .Values.global) }}
{{- end -}}

{{/*
Create the name of the cluster role to use
*/}}
{{- define "unit-operator.clusterRoleName" -}}
{{- if .Values.rbac.create }}
{{- default (include "common.names.fullname" .) .Values.rbac.name }}
{{- else }}
{{- default "default" .Values.rbac.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the cluster role binding to use
*/}}
{{- define "unit-operator.clusterRoleBindingName" -}}
{{- if .Values.rbac.create }}
{{- default (include "common.names.fullname" .) .Values.rbac.name }}
{{- else }}
{{- default "default" .Values.rbac.name }}
{{- end }}
{{- end }}

{{/*
 Create the name of the service account to use
 */}}
{{- define "unit-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{ default (include "common.names.fullname" .) .Values.serviceAccount.name | trunc 63 | trimSuffix "-" }}
{{- else -}}
{{ default "default" .Values.serviceAccount.name | trunc 63 | trimSuffix "-" }}
{{- end -}}
{{- end -}}

{{/*
Create the name of the role to use
*/}}
{{- define "unit-operator.roleName" -}}
{{- if .Values.rbac.create }}
{{- default (include "common.names.fullname" .) .Values.rbac.name }}
{{- else }}
{{- default "default" .Values.rbac.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the role binding to use
*/}}
{{- define "unit-operator.roleBindingName" -}}
{{- if .Values.rbac.create }}
{{- default (include "common.names.fullname" .) .Values.rbac.name }}
{{- else }}
{{- default "default" .Values.rbac.name }}
{{- end }}
{{- end }}