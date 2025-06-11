package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"time"
)

type IDiscordSender interface {
	SendDiscordMessage(message string) ([]byte, error)
}

type DiscordSender struct{}

func (d *DiscordSender) SendDiscordMessage(message string) ([]byte, error) {
	config, err := GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(DiscordMessage{
		Content: message,
	}); err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("POST", config.DiscordURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	text, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		return text, fmt.Errorf("discord API returned %s: %s", resp.Status, text)
	}

	return text, nil
}

const discordTemplate = `
{{- define "discord_harddrive" -}}
{{- range . -}}
{{- if eq .Status "firing" }}{{ template "discord_alert_firing" . }}{{ end -}}
{{- if eq .Status "resolved" }}{{ template "discord_alert_resolved" . }}{{ end -}}
{{- end -}}
{{- end -}}

{{- define "discord_alert_firing" -}}
# ❗️❗️❗️ CẢNH BÁO HỆ THỐNG ❗️❗️❗️

> 🚨 **Vấn đề:** {{ .Annotations.summary }}

> ⏳ **Thời gian hoạt động:** {{ printf "%.2f" (div .Values.B 31536000) }} năm

### 🖥️ Thông tin node:
{{- if index .Labels "instance" }}
> 🔹 **Node:** {{ index .Labels "instance" }}
{{- end }}
{{- if index .Labels "device" }}
> 🔸 **Device:** {{ index .Labels "device" }}
{{- end }}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
{{- end -}}

{{- define "discord_alert_resolved" -}}
# 🤟 ĐÃ GIẢI QUYẾT 🤘

> 🔧🛠️✨ **Vấn đề:** {{ .Annotations.summary }}

### 🖥️ Thông tin node:
{{- if index .Labels "instance" }}
> 🔹 **Node:** {{ index .Labels "instance" }}
{{- end }}
{{- if index .Labels "device" }}
> 🔸 **Device:** {{ index .Labels "device" }}
{{- end }}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
{{- end -}}
`

func RenderDiscordMessage(alerts []Alert) (string, error) {
	funcMap := template.FuncMap{
		"div": safeDivide,
	}

	tmpl, err := template.New("discord").Funcs(funcMap).Parse(discordTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "discord_harddrive", alerts); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
