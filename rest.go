package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type RestController struct {
	Telegram ITelegramSender
	Discord  IDiscordSender
}

func (rc *RestController) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", rc.HealthHandler)
	mux.HandleFunc("/telegram", rc.TelegramWebhookHandler)
	mux.HandleFunc("/discord", rc.DiscordWebhookHandler)
	return mux
}

func (rc *RestController) HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("UP"))
}

func (rc *RestController) TelegramWebhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var alertData GrafanaAlert
	if err := json.NewDecoder(r.Body).Decode(&alertData); err != nil {
		log.Printf("Invalid JSON body: %v", err)
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if len(alertData.Alerts) == 0 {
		http.Error(w, "No alerts found in request", http.StatusBadRequest)
		return
	}

	messages, err := RenderTelegramMessages(alertData.Alerts)
	if err != nil {
		log.Printf("Error rendering Telegram messages: %v", err)
		http.Error(w, "Error rendering messages", http.StatusInternalServerError)
		return
	}

	var responses []string
	for i, message := range messages {
		resp, err := rc.Telegram.SendTelegramMessage(message)
		if err != nil {
			log.Printf("Telegram message %d delivery failed: %v", i+1, err)
			http.Error(w, fmt.Sprintf("Message %d delivery failed", i+1), http.StatusInternalServerError)
			return
		}
		responses = append(responses, string(resp))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"messages_sent": %d, "responses": [%s]}`, len(messages), strings.Join(responses, ","))))
}

func (rc *RestController) DiscordWebhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var alertData GrafanaAlert
	if err := json.NewDecoder(r.Body).Decode(&alertData); err != nil {
		log.Printf("Invalid JSON body: %v", err)
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if len(alertData.Alerts) == 0 {
		http.Error(w, "No alerts found in request", http.StatusBadRequest)
		return
	}

	messages, err := RenderDiscordMessages(alertData.Alerts)
	if err != nil {
		log.Printf("Error rendering Discord messages: %v", err)
		http.Error(w, "Error rendering messages", http.StatusInternalServerError)
		return
	}

	var responses []string
	for i, message := range messages {
		resp, err := rc.Discord.SendDiscordMessage(message)
		if err != nil {
			log.Printf("Discord message %d delivery failed: %v", i+1, err)
			http.Error(w, fmt.Sprintf("Message %d delivery failed", i+1), http.StatusInternalServerError)
			return
		}
		responses = append(responses, string(resp))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"messages_sent": %d, "responses": [%s]}`, len(messages), strings.Join(responses, ","))))
}
