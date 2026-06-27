package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/brendanjhnsn/TrackerAPI/core/config"
	"github.com/brendanjhnsn/TrackerAPI/core/database"
	"github.com/brendanjhnsn/TrackerAPI/modules/auth"
	"github.com/brendanjhnsn/TrackerAPI/modules/dailymessages"
	"github.com/brendanjhnsn/TrackerAPI/modules/loa"
	"github.com/brendanjhnsn/TrackerAPI/modules/qfs"
	"github.com/brendanjhnsn/TrackerAPI/modules/tickets"
	"github.com/brendanjhnsn/TrackerAPI/modules/voicetime"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatalf("Error loading .env file: %v", err)
	}

	cfg := config.Load()
	log.Printf("Environment: %s", cfg.Environment)

	if cfg.DiscordGuildID == "" {
		log.Fatal("DISCORD_GUILD_ID is required")
	}
	if cfg.DiscordClientID == "" {
		log.Fatal("DISCORD_CLIENT_ID is required")
	}
	if cfg.DiscordClientSecret == "" {
		log.Fatal("DISCORD_CLIENT_SECRET is required")
	}
	if cfg.DiscordToken == "" {
		log.Fatal("DISCORD_TOKEN is required")
	}

	db, sqlDB, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer database.Close(sqlDB)

	loaMod     := loa.New(db, cfg)
	ticketsMod := tickets.New(db, cfg)
	voiceMod   := voicetime.New(db, cfg)
	qfsMod     := qfs.New(db, cfg)
	dailyMod   := dailymessages.New(db, cfg)
	authMod    := auth.New(db, cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	loaMod.RegisterRoutes(mux)
	ticketsMod.RegisterRoutes(mux)
	voiceMod.RegisterRoutes(mux)
	qfsMod.RegisterRoutes(mux)
	dailyMod.RegisterRoutes(mux)
	authMod.RegisterRoutes(mux)

	go func() {
		log.Printf("Starting API server on port %s", cfg.ServerPort)
		if err := http.ListenAndServe(":"+cfg.ServerPort, corsMiddleware(cfg.FrontendURL, cfg.Environment, authMod.Middleware(mux))); err != nil {
			log.Fatalf("Error starting API server: %v", err)
		}
	}()

	discord, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
	}
	discord.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsGuildMessageReactions |
		discordgo.IntentsGuildVoiceStates |
		discordgo.IntentsGuildMembers
	defer discord.Close()

	ticketsMod.Register(discord)
	voiceMod.Register(discord)
	qfsMod.Register(discord)
	dailyMod.Register(discord)

	if err := discord.Open(); err != nil {
		log.Fatalf("Error opening Discord connection: %v", err)
	}

	log.Println("API is now running. Press Ctrl+C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
}
