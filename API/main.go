package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/brendanjhnsn/TrackerAPI/core/config"
	"github.com/brendanjhnsn/TrackerAPI/core/database"
	"github.com/brendanjhnsn/TrackerAPI/modules/attachments"
	"github.com/brendanjhnsn/TrackerAPI/modules/auth"
	"github.com/brendanjhnsn/TrackerAPI/modules/dailymessages"
	"github.com/brendanjhnsn/TrackerAPI/modules/dailystats"
	"github.com/brendanjhnsn/TrackerAPI/modules/gameleads"
	"github.com/brendanjhnsn/TrackerAPI/modules/loa"
	"github.com/brendanjhnsn/TrackerAPI/modules/modnotes"
	"github.com/brendanjhnsn/TrackerAPI/modules/permissions"
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

	loaMod          := loa.New(db, cfg)
	ticketsMod      := tickets.New(db, cfg)
	voiceMod        := voicetime.New(db, cfg)
	qfsMod          := qfs.New(db, cfg)
	dailyMod        := dailymessages.New(db, cfg)
	authMod         := auth.New(db, cfg)
	modNotesMod     := modnotes.New(db, cfg)
	permissionsMod  := permissions.New(db, cfg)
	dailyStatsMod   := dailystats.New(db, cfg)
	attachmentsMod  := attachments.New(db, cfg)

	mux := http.NewServeMux()
	// SPA static file server — serves the built React app from ./dist.
	// Falls back to index.html for any path that doesn't match a real file,
	// enabling client-side routing. Must be registered before API routes so
	// the more-specific API patterns take priority in Go's mux.
	mux.Handle("/", http.FileServer(spaFS{http.Dir("./dist")}))
	loaMod.RegisterRoutes(mux)
	ticketsMod.RegisterRoutes(mux)
	voiceMod.RegisterRoutes(mux)
	qfsMod.RegisterRoutes(mux)
	dailyMod.RegisterRoutes(mux)
	authMod.RegisterRoutes(mux)
	modNotesMod.RegisterRoutes(mux)
	permissionsMod.RegisterRoutes(mux)
	dailyStatsMod.RegisterRoutes(mux)
	attachmentsMod.RegisterRoutes(mux)

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
		discordgo.IntentsGuildMembers |
		discordgo.IntentMessageContent |
		discordgo.Intent(1 << 18) // GUILD_MODERATION — needed for AuditLogEntryCreate events
	defer discord.Close()

	glMod := gameleads.New(db, cfg)
	glMod.RegisterRoutes(mux)

	ticketsMod.Register(discord)
	voiceMod.Register(discord)
	qfsMod.Register(discord)
	dailyMod.Register(discord)
	modNotesMod.Register(discord)
	glMod.Register(discord)

	if err := discord.Open(); err != nil {
		log.Fatalf("Error opening Discord connection: %v", err)
	}

	log.Println("API is now running. Press Ctrl+C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
}
