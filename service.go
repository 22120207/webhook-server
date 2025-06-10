package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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
		client.Transport = transport
	}

	req, err := http.NewRequest("POST", telgramURL, body)
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
		return text, fmt.Errorf("telegram api returned non-OK status: %s, response: %s", resp.Status, string(text))
	}

	return text, nil
}
