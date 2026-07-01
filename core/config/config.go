package config

import (
	"os"
	"strconv"
)

type Config struct {
	Environment      string
	ServerPort       string
	DiscordToken     string
	DBDriver         string
	DBHost           string
	DBUser           string
	DBPassword       string
	DBName           string
	DBPort           int
	QFSChannelID     string
	ModRoleID        string
	ModRoleName      string
	TicketCategoryID string
	QFSChannelName   string
	AdminCategoryID     string
	DiscordClientID     string
	DiscordClientSecret string
	DiscordRedirectURI  string
	DiscordGuildID      string
	ManagerRoleID       string
	DirectorRoleID      string
	FrontendURL         string
	ModLogChannelID     string
	GameLeadRoleID     string
	GameLeadCategoryID string
	UploadsDir string
}

func Load() *Config {
	return &Config{
		Environment:      getEnv("ENVIRONMENT", "development"),
		ServerPort:       getEnv("PORT", getEnv("SERVER_PORT", "8080")),
		DiscordToken:     getEnv("DISCORD_TOKEN", ""),
		DBDriver:         getEnv("DB_DRIVER", "postgres"),
		DBHost:           getEnv("DB_HOST", "127.0.0.1"),
		DBUser:           getEnv("DB_USER", "postgres"),
		DBPassword:       getEnv("DB_PASSWORD", ""),
		DBName:           getEnv("DB_NAME", "community_tracker2"),
		DBPort:           getEnvAsInt("DB_PORT", 5432),
		QFSChannelID:     getEnv("QFS_CHANNEL_ID", ""),
		ModRoleID:        getEnv("MOD_ROLE_ID", ""),
		ModRoleName:      getEnv("MOD_ROLE_NAME", ""),
		TicketCategoryID: getEnv("TICKET_CATEGORY_ID", ""),
		QFSChannelName:   getEnv("QFS_CHANNEL_NAME", ""),
		AdminCategoryID:     getEnv("ADMIN_CATEGORY_ID", ""),
		DiscordClientID:     getEnv("DISCORD_CLIENT_ID", ""),
		DiscordClientSecret: getEnv("DISCORD_CLIENT_SECRET", ""),
		DiscordRedirectURI:  getEnv("DISCORD_REDIRECT_URI", ""),
		DiscordGuildID:      getEnv("DISCORD_GUILD_ID", ""),
		ManagerRoleID:       getEnv("MANAGER_ROLE_ID", ""),
		DirectorRoleID:      getEnv("DIRECTOR_ROLE_ID", ""),
		FrontendURL:         getEnv("FRONTEND_URL", "http://localhost:3000"),
		ModLogChannelID:     getEnv("MODLOG_CHANNEL_ID", ""),
		GameLeadRoleID:     getEnv("GAME_LEAD_ROLE_ID", ""),
		GameLeadCategoryID: getEnv("GAME_LEAD_CATEGORY_ID", ""),
		UploadsDir: getEnv("UPLOADS_DIR", "uploads"),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvAsInt(key string, fallback int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return fallback
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return fallback
	}
	return value
}
