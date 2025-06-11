package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

	message, err := RenderTelegramMessage(alertData.Alerts)
	if err != nil {
		log.Printf("Error rendering Telegram message: %v", err)
		http.Error(w, "Error rendering message", http.StatusInternalServerError)
		return
	}

	resp, err := rc.Telegram.SendTelegramMessage(message)
	if err != nil {
		log.Printf("Telegram message delivery failed: %v", err)
		http.Error(w, "Message delivery failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
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

	for _, alert := range alertData.Alerts {
		fmt.Println(alert)
	}

	message, err := RenderDiscordMessage(alertData.Alerts)
	if err != nil {
		log.Printf("Error rendering Discord message: %v", err)
		http.Error(w, "Error rendering message", http.StatusInternalServerError)
		return
	}

	resp, err := rc.Discord.SendDiscordMessage(message)
	if err != nil {
		log.Printf("Discord message delivery failed: %v", err)
		http.Error(w, "Message delivery failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}
