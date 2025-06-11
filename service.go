package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
)

type ITelegramSender interface {
	SendTelegramMessage(message string) ([]byte, error)
}

type TelegramSender struct{}

func (t *TelegramSender) SendTelegramMessage(message string) ([]byte, error) {
	token := getEnv("BOT_TOKEN", "")
	chatId := getEnv("CHAT_ID", "")
	proxyURLStr := getEnv("PROXY_URL", "")

	if token == "" {
		fmt.Println("Environment variable BOT_TOKEN is not set or is empty.")
	}

	if chatId == "" {
		fmt.Println("Environment variable CHAT_ID is not set or is empty.")
	}

	if proxyURLStr == "" {
		fmt.Println("Environment variable PROXY_URL is not set or is empty.")
	} else {
		fmt.Println(proxyURLStr)
	}

	telgramURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?parse_mode=html", token)

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(Message{chatId, message})
	if err != nil {
		return nil, err
	}

	client := &http.Client{}

	if proxyURLStr != "" {
		proxyURL, err := url.Parse(proxyURLStr)
		if err != nil {
			return nil, fmt.Errorf("error parsing proxy URL '%s': %w", proxyURLStr, err)
		}

		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}

		if proxyURL.User != nil {
			username := proxyURL.User.Username()
			password, _ := proxyURL.User.Password()

			auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))

			transport.ProxyConnectHeader = http.Header{}
			transport.ProxyConnectHeader.Add("Proxy-Authorization", "Basic "+auth)
		}

		client.Transport = transport
	}

	req, err := http.NewRequest("POST", telgramURL, body)
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending HTTP request: %w", err)
	}
	defer resp.Body.Close()

	text, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	fmt.Printf("Telegram API Response Status: %s\n", resp.Status)
	fmt.Printf("Telegram API Response Body: %s\n", text)

	if resp.StatusCode != http.StatusOK {
		return text, fmt.Errorf("Telegram returned non-OK: %s - %s", resp.Status, text)
	}

	return text, nil
}

var telegramTemplate = `
{{ define "telegram_harddrive" }}
  {{ range . }}
    {{ if eq .Status "firing" }}{{ template "telegram_alert_firing" . }}{{ end }}
    {{ if eq .Status "resolved" }}{{ template "telegram_alert_resolved" . }}{{ end }}
  {{ end }}
{{ end }}

{{ define "telegram_alert_firing" }}
❗️❗️❗️❗️❗️ CẢNH BÁO ❗️❗️❗️❗️❗️

🚨 Vấn đề: {{ .Annotations.summary }} 🚨
<b>Thời gian hoạt động = </b>{{ printf "%.2f" (div .Values.B 31536000) }} năm

<b>Thông tin node:</b>
{{ if index .Labels "instance" }}- Node = {{ index .Labels "instance" }}{{ end }}
{{ if index .Labels "device" }}- Device = {{ index .Labels "device" }}{{ end }}

{{ end }}

{{ define "telegram_alert_resolved" }}
🤟🤟🤟 Đã giải quyết xong 🤘🤘🤘

🔧🛠️✨ Vấn đề: {{ .Annotations.summary }} 🔩⚙️🔨

<b>Thông tin nodes:</b>
{{ if index .Labels "instance" }}- Node = {{ index .Labels "instance" }}{{ end }}
{{ if index .Labels "device" }}- Device = {{ index .Labels "device" }}{{ end }}

{{ end }}
 `

func RenderTelegramMessage(alerts []Alert) (string, error) {
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

	tmpl, err := template.New("telegram").Funcs(funcMap).Parse(telegramTemplate)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "telegram_harddrive", alerts)
	if err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}
