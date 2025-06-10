package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("grafana-telegram-proxy")

type RestController struct {
	Service ITelegramSender
}

func (this *RestController) Start() {
	http.HandleFunc("/health", this.HealthHandler)
	if useAuth() {
		http.HandleFunc("/", basicAuth(this.WebhookHandler))
	} else {
		http.HandleFunc("/", this.WebhookHandler)
	}
	fmt.Println("Starting server on port:", strings.Split(getPort(), ":")[1])
	log.Fatal(http.ListenAndServe(getPort(), nil))
}

func useAuth() bool {
	_, usernameOk := os.LookupEnv("USERNAME")
	_, passwordOk := os.LookupEnv("PASSWORD")
	return usernameOk && passwordOk
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

	resp, err := this.Service.SendTelegramMessage(message)
	if err != nil {
		http.Error(w, "Message delivery failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}
