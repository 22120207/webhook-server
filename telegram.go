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
	"os"

	"github.com/joho/godotenv"
	"golang.org/x/net/proxy"
)

type ITelegramSender interface {
	SendTelegramMessage(message string) ([]byte, error)
}

type TelegramSender struct{}

func (t *TelegramSender) SendTelegramMessage(message string) ([]byte, error) {
	envErr := godotenv.Load()
	if envErr != nil {
		log.Fatal("Error loading .env file")
	}

	token := os.Getenv("BOT_TOKEN")
	chatId := os.Getenv("CHAT_ID")
	proxyURLStr := os.Getenv("PROXY_URL")
	proxyType := os.Getenv("PROXY_TYPE")
	proxyUser := os.Getenv("PROXY_USER")
	proxyPass := os.Getenv("PROXY_PASS")

	if token == "" {
		fmt.Println("Environment variable BOT_TOKEN is not set or is empty.")
	}
	if chatId == "" {
		fmt.Println("Environment variable CHAT_ID is not set or is empty.")
	}

	telegramURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?parse_mode=html", token)

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(TelegramMessage{
		ChatId:    chatId,
		Text:      message,
		ParseMode: "HTML",
	})
	if err != nil {
		return nil, err
	}

	client := &http.Client{}

	if proxyURLStr != "" {
		if proxyType == "socks5" {
			proxyURL, err := url.Parse(proxyURLStr)
			if err != nil {
				return nil, fmt.Errorf("error parsing proxy URL '%s': %w", proxyURLStr, err)
			}

			var auth *proxy.Auth
			if proxyUser != "" && proxyPass != "" {
				auth = &proxy.Auth{
					User:     proxyUser,
					Password: proxyPass,
				}
			}

			dialer, err := proxy.SOCKS5("tcp", proxyURL.Host, auth, proxy.Direct)
			if err != nil {
				return nil, fmt.Errorf("error creating SOCKS5 dialer: %w", err)
			}

			transport := &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return dialer.Dial(network, addr)
				},
			}
			client.Transport = transport
		} else {
			proxyURL, err := url.Parse(proxyURLStr)
			if err != nil {
				return nil, fmt.Errorf("error parsing proxy URL '%s': %w", proxyURLStr, err)
			}

			if proxyUser != "" && proxyPass != "" {
				proxyURL.User = url.UserPassword(proxyUser, proxyPass)
			}

			transport := &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
			client.Transport = transport
		}
	}

	req, err := http.NewRequest("POST", telegramURL, body)
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
