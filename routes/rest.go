package routes

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"webhook-server/config"
	"webhook-server/helper"
	"webhook-server/model"
	"webhook-server/service"
)

type RestController struct {
	Telegram    service.ITelegramSender
	Discord     service.IDiscordSender
	MongoClient *mongo.Client
}

type SuppressedAlert struct {
	NodeInstance    string    `bson:"node_instance"`
	Device          string    `bson:"device"`
	SuppressedUntil time.Time `bson:"suppressed_until"`
	AlertSummary    string    `bson:"alert_summary"`
}

func (rc *RestController) SetUpRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", rc.HealthHandler)
	mux.HandleFunc("/telegram", rc.TelegramWebhookHandler)
	mux.HandleFunc("/discord", rc.DiscordWebhookHandler)
	mux.HandleFunc("/discord/interactions", rc.DiscordInteractionHandler)
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

	var alertData model.GrafanaAlert
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
		singleAlerts := []model.Alert{alert}
		message, err := service.RenderTelegramMessage(singleAlerts)
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
}

func (rc *RestController) DiscordWebhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var alertData model.GrafanaAlert
	if err := json.NewDecoder(r.Body).Decode(&alertData); err != nil {
		log.Printf("Invalid JSON body: %v", err)
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if len(alertData.Alerts) == 0 {
		http.Error(w, "No alerts found in request", http.StatusBadRequest)
		return
	}

	config, err := config.GetConfig()
	if err != nil {
		log.Printf("Error loading config: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	collection := rc.MongoClient.Database(config.MongoDBDatabase).Collection("suppressed_alerts")

	for _, alert := range alertData.Alerts {
		nodeInstance := alert.Labels["instance"]
		device := alert.Labels["device"]

		if alert.Status == "firing" {
			// Check if alert is suppressed
			var result SuppressedAlert
			err := collection.FindOne(context.TODO(), bson.M{
				"node_instance":    nodeInstance,
				"device":           device,
				"suppressed_until": bson.M{"$gt": time.Now()},
			}).Decode(&result)
			if err == nil {
				log.Printf("Alert suppressed for %s %s until %v", nodeInstance, device, result.SuppressedUntil)
				continue
			} else if err != mongo.ErrNoDocuments {
				log.Printf("Error checking suppression: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Build and send firing message with button
			message := buildFiringMessage(alert)
			components := []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Resolve for 72h",
							Style:    discordgo.PrimaryButton,
							CustomID: fmt.Sprintf("resolve:%s:%s", nodeInstance, device),
						},
					},
				},
			}
			resp, err := rc.Discord.SendDiscordMessageWithComponents(message, components)
			if err != nil {
				log.Printf("Error sending Discord message: %v", err)
				http.Error(w, "Error sending message", http.StatusInternalServerError)
				return
			}
			log.Printf("Sent firing alert to Discord: %s", resp)
		} else if alert.Status == "resolved" {
			// Build and send resolved message
			message := buildResolvedMessage(alert)
			resp, err := rc.Discord.SendDiscordMessage(message)
			if err != nil {
				log.Printf("Error sending Discord message: %v", err)
				http.Error(w, "Error sending message", http.StatusInternalServerError)
				return
			}
			log.Printf("Sent resolved alert to Discord: %s", resp)

			// Remove suppression entry if it exists
			_, err = collection.DeleteOne(context.TODO(), bson.M{
				"node_instance": nodeInstance,
				"device":        device,
			})
			if err != nil {
				log.Printf("Error removing suppression: %v", err)
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (rc *RestController) DiscordInteractionHandler(w http.ResponseWriter, r *http.Request) {
	config, err := config.GetConfig()
	if err != nil {
		log.Printf("Error loading config: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Verify Discord signature
	signature := r.Header.Get("X-Signature-Ed25519")
	timestamp := r.Header.Get("X-Signature-Timestamp")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if !verifyDiscordSignature(signature, timestamp, string(body), config.DiscordPublicKey) {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	var interaction discordgo.Interaction
	if err := json.Unmarshal(body, &interaction); err != nil {
		log.Printf("Invalid interaction JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if interaction.Type == discordgo.InteractionPing {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"type": discordgo.InteractionResponsePong,
		})
		return
	}

	if interaction.Type == discordgo.InteractionMessageComponent {
		customID := interaction.MessageComponentData().CustomID
		if strings.HasPrefix(customID, "resolve:") {
			parts := strings.Split(customID, ":")
			if len(parts) != 3 {
				log.Printf("Invalid custom ID: %s", customID)
				return
			}
			nodeInstance := parts[1]
			device := parts[2]

			// Suppress alert in MongoDB
			collection := rc.MongoClient.Database(config.MongoDBDatabase).Collection("suppressed_alerts")
			suppressedUntil := time.Now().Add(72 * time.Hour)
			_, err := collection.UpdateOne(
				context.TODO(),
				bson.M{"node_instance": nodeInstance, "device": device},
				bson.M{"$set": bson.M{
					"suppressed_until": suppressedUntil,
					"alert_summary":    "User resolved via Discord",
				}},
				options.Update().SetUpsert(true),
			)
			if err != nil {
				log.Printf("Error suppressing alert: %v", err)
				return
			}

			// Update original message
			updatedMessage := fmt.Sprintf("**Alert suppressed for 72 hours by %s**", interaction.Member.User.Username)
			components := []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Resolve for 72h",
							Style:    discordgo.PrimaryButton,
							CustomID: customID,
							Disabled: true,
						},
					},
				},
			}
			if err := rc.Discord.UpdateMessage(interaction.ChannelID, interaction.Message.ID, updatedMessage, components); err != nil {
				log.Printf("Error updating message: %v", err)
			}

			// Send ephemeral response
			response := &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("Alert for %s %s has been suppressed for 72 hours.", nodeInstance, device),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			}
			json.NewEncoder(w).Encode(response)
		}
	}
}

func buildFiringMessage(alert model.Alert) string {
	summary := alert.Annotations["summary"]
	nodeInstance := alert.Labels["instance"]
	device := alert.Labels["device"]
	uptimeYears := fmt.Sprintf("%.2f", helper.SafeDivide(alert.Values["B"], 31536000))

	return fmt.Sprintf("# ❗️❗️� Hodgson CẢNH BÁO HỆ THỐNG ❗️❗️❗️\n\n"+
		"> 🚨 **Vấn đề:** %s\n"+
		"> ⏳ **Thời gian hoạt động:** %s năm\n"+
		"### 🖥️ Thông tin node:\n"+
		"> 🔹 **Node:** %s\n"+
		"> 🔸 **Device:** %s\n"+
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━",
		summary, uptimeYears, nodeInstance, device)
}

func buildResolvedMessage(alert model.Alert) string {
	summary := alert.Annotations["summary"]
	nodeInstance := alert.Labels["instance"]
	device := alert.Labels["device"]

	return fmt.Sprintf("# 🤟 ĐÃ GIẢI QUYẾT 🤘\n\n"+
		"> 🔧🛠️✨ **Vấn đề:** %s\n"+
		"### 🖥️ Thông tin node:\n"+
		"> 🔹 **Node:** %s\n"+
		"> 🔸 **Device:** %s\n"+
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━",
		summary, nodeInstance, device)
}

func verifyDiscordSignature(signature, timestamp, body, publicKey string) bool {
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	pubKeyBytes, err := hex.DecodeString(publicKey)
	if err != nil {
		return false
	}
	message := timestamp + body
	return ed25519.Verify(pubKeyBytes, []byte(message), sigBytes)
}
