
{{/*
checks the .Values.containerRuntime for valid values
*/}}
{{- define "containerEngine.valid" -}}
{{- $valid := keys .Values.containerEngines | sortAlpha -}}
{{- if has .Values.container.runtime $valid -}}
{{- .Values.container.runtime  -}}
{{- else if has .Values.container.engine $valid -}}
{{- .Values.container.engine  -}}
{{- else -}}
{{- fail (printf "unknown container.engine: %v (must be one of %s)" .Values.container.engine (join ", " $valid)) -}}
{{- end -}}
{{- end -}}


{{- /*
containerEngine.get will select the attribute for the selected container engine
*/}}
{{- define "containerEngine.get" -}}
{{- $top := index . 0 -}}
{{- $field := index . 1 -}}
{{- $engine := (include "containerEngine.valid" $top )  -}}
{{- $engineValues := get $top.Values.containerEngines $engine  -}}
{{- index $engineValues $field -}}
{{- end -}}

{{- /*
ociRuntime.get will select the attribute for the selected container engine
*/}}
{{- define "ociRuntime.get" -}}
{{- $top := index . 0 -}}
{{- $field := index . 1 -}}
{{- $engine := (include "containerEngine.valid" $top )  -}}
{{- $engineValues := get $top.Values.containerEngines $engine  -}}
{{- index $engineValues.ociRuntime $field -}}
{{- end -}}

{{- /*
will omit attribute from the passed in object depending on the KubeVersion
*/}}
{{- define "omitForKuberVersion" -}}
{{- $top := index . 0 -}}
{{- $versionConstraint := index . 1 -}}
{{- $dict := index . 2 -}}
{{- $toOmit := index . 3 -}}
{{- if semverCompare $versionConstraint $top.Capabilities.KubeVersion.Version -}}
{{- $dict = omit $dict $toOmit -}}
{{- end -}}
{{- $dict | toYaml -}}
{{- end -}}

