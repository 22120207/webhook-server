package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"webhook-server/service/config"
	"webhook-server/service/contact"
	"webhook-server/service/routes"
)

var mongoClient *mongo.Client

func main() {
	config, err := config.GetConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	mongoClient, err = mongo.Connect(context.TODO(), options.Client().ApplyURI(config.MongoDBURI))
	if err != nil {
		log.Fatalf("Error connecting to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(context.TODO())

	discord, err := discordgo.New("Bot " + config.DiscordBotToken)
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
	}
	defer discord.Close()

	discord.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Println("Discord bot is ready")
	})
	if err := discord.Open(); err != nil {
		log.Fatalf("Error opening Discord connection: %v", err)
	}

	server := &routes.RestController{
		Telegram: &contact.TelegramSender{},
		Discord: &contact.DiscordSender{
			Discord:   discord,
			ChannelID: config.DiscordChannelID,
		},
		MongoClient: mongoClient,
	}

	httpServer := &http.Server{
		Addr:         ":8080",
		Handler:      server.SetUpRoutes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Starting server on port 8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited")
}
