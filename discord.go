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

	if resp.StatusCode != http.StatusOK {
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
❗️❗️❗️❗️❗️ CẢNH BÁO ❗️❗️❗️❗️❗️

🚨 Vấn đề: {{ .Annotations.summary }} 🚨
**Thời gian hoạt động:** {{ printf "%.2f" (div .Values.B 31536000) }} năm

**Thông tin node:**
{{- if index .Labels "instance" }}
- Node: {{ index .Labels "instance" }}
{{- end }}
{{- if index .Labels "device" }}
- Device: {{ index .Labels "device" }}
{{- end }}
{{- end -}}

{{- define "discord_alert_resolved" -}}
🤟🤟🤟 Đã giải quyết xong 🤘🤘🤘

🔧🛠️✨ Vấn đề: {{ .Annotations.summary }} 🔩⚙️🔨

**Thông tin node:**
{{- if index .Labels "instance" }}
- Node: {{ index .Labels "instance" }}
{{- end }}
{{- if index .Labels "device" }}
- Device: {{ index .Labels "device" }}
{{- end }}
{{- end -}}
`

func RenderDiscordMessages(alerts []Alert) ([]string, error) {
	const maxDiscordLength = 2000

	funcMap := template.FuncMap{
		"div": safeDivide,
	}

	tmpl, err := template.New("discord").Funcs(funcMap).Parse(discordTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var messages []string
	var currentBatch []Alert

	for _, alert := range alerts {
		testBatch := append(currentBatch, alert)

		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, "discord_harddrive", testBatch); err != nil {
			return nil, fmt.Errorf("failed to execute template: %w", err)
		}

		if buf.Len() > maxDiscordLength {
			if len(currentBatch) > 0 {
				var batchBuf bytes.Buffer
				if err := tmpl.ExecuteTemplate(&batchBuf, "discord_harddrive", currentBatch); err != nil {
					return nil, fmt.Errorf("failed to execute template for batch: %w", err)
				}
				messages = append(messages, batchBuf.String())
				currentBatch = []Alert{alert}
			} else {
				var singleBuf bytes.Buffer
				if err := tmpl.ExecuteTemplate(&singleBuf, "discord_harddrive", []Alert{alert}); err != nil {
					return nil, fmt.Errorf("failed to execute template for single alert: %w", err)
				}
				messages = append(messages, singleBuf.String())
			}
		} else {
			currentBatch = testBatch
		}
	}

	if len(currentBatch) > 0 {
		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, "discord_harddrive", currentBatch); err != nil {
			return nil, fmt.Errorf("failed to execute template for final batch: %w", err)
		}
		messages = append(messages, buf.String())
	}

	return messages, nil
}

func RenderDiscordMessage(alerts []Alert) (string, error) {
	messages, err := RenderDiscordMessages(alerts)
	if err != nil {
		return "", err
	}
	if len(messages) == 0 {
		return "", fmt.Errorf("no messages generated")
	}
	return messages[0], nil
}
