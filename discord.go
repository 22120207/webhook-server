package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type IDiscordSender interface {
	SendDiscordMessage(message string) ([]byte, error)
}

type DiscordSender struct{}

func (t *DiscordSender) SendDiscordMessage(message string) ([]byte, error) {
	envErr := godotenv.Load()
	if envErr != nil {
		log.Fatal("Error loading .env file")
	}

	discordURL := os.Getenv("DISCORD_URL")

	if discordURL == "" {
		fmt.Println("Environment variable DISCORD_URL is not set or is empty.")
	}

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(DiscordMessage{
		Content: message,
	})
	if err != nil {
		return nil, err
	}

	client := &http.Client{}

	req, err := http.NewRequest("POST", discordURL, body)
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending HTTP request: %w", err)
	}
	defer resp.Body.Close()

	text, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return text, fmt.Errorf("Discord returned non-OK: %s - %s", resp.Status, text)
	}

	return text, nil
}

var discordTemplate = `
{{ define "discord_harddrive" }}
  {{ range . }}
    {{ if eq .Status "firing" }}{{ template "discord_alert_firing" . }}{{ end }}
    {{ if eq .Status "resolved" }}{{ template "discord_alert_resolved" . }}{{ end }}
  {{ end }}
{{ end }}

{{ define "discord_alert_firing" }}
❗️❗️❗️❗️❗️ CẢNH BÁO ❗️❗️❗️❗️❗️

🚨 Vấn đề: {{ .Annotations.summary }} 🚨
**Thời gian hoạt động =** {{ printf "%.2f" (div .Values.B 31536000) }} năm

**Thông tin node:**
{{ if index .Labels "instance" }}- Node = {{ index .Labels "instance" }}{{ end }}
{{ if index .Labels "device" }}- Device = {{ index .Labels "device" }}{{ end }}
{{ end }}

{{ define "discord_alert_resolved" }}
🤟🤟🤟 Đã giải quyết xong 🤘🤘🤘

🔧🛠️✨ Vấn đề: {{ .Annotations.summary }} 🔩⚙️🔨

**Thông tin nodes:**
{{ if index .Labels "instance" }}- Node = {{ index .Labels "instance" }}{{ end }}
{{ if index .Labels "device" }}- Device = {{ index .Labels "device" }}{{ end }}
{{ end }}
`

func RenderDiscordMessage(alerts []Alert) (string, error) {
	funcMap := template.FuncMap{
		"div": func(a interface{}, b float64) float64 {
			var af float64
			switch v := a.(type) {
			case float64:
				af = v
			case int:
				af = float64(v)
			case int64:
				af = float64(v)
			case json.Number:
				f, _ := v.Float64()
				af = f
			default:
				af = 0
			}
			if b == 0 {
				return 0
			}
			return af / b
		},
	}

	tmpl, err := template.New("discord").Funcs(funcMap).Parse(discordTemplate)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "discord_harddrive", alerts)
	if err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}
