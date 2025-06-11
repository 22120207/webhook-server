package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

type ITelegramSender interface {
	SendTelegramMessage(message string) ([]byte, error)
}

type TelegramSender struct{}

func (t *TelegramSender) SendTelegramMessage(message string) ([]byte, error) {
	config, err := GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	telegramURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", config.BotToken)

	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(TelegramMessage{
		ChatID:    config.ChatID,
		Text:      message,
		ParseMode: "HTML",
	}); err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	if config.ProxyURL != "" {
		transport, err := createProxyTransport(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create proxy transport: %w", err)
		}
		client.Transport = transport
	}

	req, err := http.NewRequest("POST", telegramURL, body)
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
		return text, fmt.Errorf("telegram API returned %s: %s", resp.Status, text)
	}
	fmt.Println(resp.Status, resp.Body)

	return text, nil
}

func createProxyTransport(config *Config) (*http.Transport, error) {
	proxyURL, err := url.Parse(config.ProxyURL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL '%s': %w", config.ProxyURL, err)
	}

	if config.ProxyType == "socks5" {
		var auth *proxy.Auth
		if config.ProxyUser != "" && config.ProxyPass != "" {
			auth = &proxy.Auth{
				User:     config.ProxyUser,
				Password: config.ProxyPass,
			}
		}

		dialer, err := proxy.SOCKS5("tcp", proxyURL.Host, auth, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
		}

		return &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			},
		}, nil
	}

	if config.ProxyUser != "" && config.ProxyPass != "" {
		proxyURL.User = url.UserPassword(config.ProxyUser, config.ProxyPass)
	}

	return &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}, nil
}

const telegramTemplate = `
{{- define "telegram_harddrive" -}}
{{- range . -}}
{{- if eq .Status "firing" }}{{ template "telegram_alert_firing" . }}{{ end -}}
{{- if eq .Status "resolved" }}{{ template "telegram_alert_resolved" . }}{{ end -}}
{{- end -}}
{{- end -}}

{{- define "telegram_alert_firing" -}}
❗️❗️❗️❗️❗️ CẢNH BÁO ❗️❗️❗️❗️❗️

🚨 Vấn đề: {{ .Annotations.summary }} 🚨
<b>Thời gian hoạt động:</b> {{ printf "%.2f" (div .Values.B 31536000) }} năm

<b>Thông tin node:</b>
{{- if index .Labels "instance" }}
- Node: {{ index .Labels "instance" }}
{{- end }}
{{- if index .Labels "device" }}
- Device: {{ index .Labels "device" }}
{{- end }}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
{{- end -}}

{{- define "telegram_alert_resolved" -}}
🤟🤟🤟 Đã giải quyết xong 🤘🤘🤘

🔧🛠️✨ Vấn đề: {{ .Annotations.summary }} 🔩⚙️🔨

<b>Thông tin node:</b>
{{- if index .Labels "instance" }}
- Node: {{ index .Labels "instance" }}
{{- end }}
{{- if index .Labels "device" }}
- Device: {{ index .Labels "device" }}
{{- end }}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
{{- end -}}
`

func RenderTelegramMessages(alerts []Alert) ([]string, error) {
	const maxTelegramLength = 4096

	funcMap := template.FuncMap{
		"div": safeDivide,
	}

	tmpl, err := template.New("telegram").Funcs(funcMap).Parse(telegramTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var messages []string
	var currentBatch []Alert

	for _, alert := range alerts {
		testBatch := append(currentBatch, alert)

		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, "telegram_harddrive", testBatch); err != nil {
			return nil, fmt.Errorf("failed to execute template: %w", err)
		}

		if buf.Len() > maxTelegramLength {
			if len(currentBatch) > 0 {
				var batchBuf bytes.Buffer
				if err := tmpl.ExecuteTemplate(&batchBuf, "telegram_harddrive", currentBatch); err != nil {
					return nil, fmt.Errorf("failed to execute template for batch: %w", err)
				}
				messages = append(messages, batchBuf.String())
				currentBatch = []Alert{alert}
			} else {
				var singleBuf bytes.Buffer
				if err := tmpl.ExecuteTemplate(&singleBuf, "telegram_harddrive", []Alert{alert}); err != nil {
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
		if err := tmpl.ExecuteTemplate(&buf, "telegram_harddrive", currentBatch); err != nil {
			return nil, fmt.Errorf("failed to execute template for final batch: %w", err)
		}
		messages = append(messages, buf.String())
	}

	return messages, nil
}

func RenderTelegramMessage(alerts []Alert) (string, error) {
	messages, err := RenderTelegramMessages(alerts)
	if err != nil {
		return "", err
	}
	if len(messages) == 0 {
		return "", fmt.Errorf("no messages generated")
	}
	return messages[0], nil
}
