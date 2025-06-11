package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("grafana-telegram-proxy")

type RestController struct {
	Telegram ITelegramSender
	Discord  IDiscordSender
}

func (this *RestController) Start() {
	http.HandleFunc("/health", this.HealthHandler)

	http.HandleFunc("/", this.WebhookHandler)

	fmt.Println("Starting server on port:", strings.Split("0.0.0.0:8080", ":")[1])
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
}

func (_ *RestController) HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET requests are allowed", 400)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("UP"))
}

func (this *RestController) WebhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusBadRequest)
		return
	}

	var alertData GrafanaAlert
	err := json.NewDecoder(r.Body).Decode(&alertData)
	if err != nil {
		http.Error(w, "Invalid JSON body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(alertData.Alerts) == 0 {
		http.Error(w, "No alerts found in request", http.StatusBadRequest)
		return
	}

	message, err := RenderTelegramMessage(alertData.Alerts)
	if err != nil {
		http.Error(w, "Error rendering message: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Println(message)

	resp, err := this.Telegram.SendTelegramMessage(message)
	if err != nil {
		http.Error(w, "Message delivery failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}
