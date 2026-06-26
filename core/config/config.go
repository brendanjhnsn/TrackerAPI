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
	AdminCategoryID  string
}

func Load() *Config {
	return &Config{
		Environment:      getEnv("ENVIRONMENT", "development"),
		ServerPort:       getEnv("SERVER_PORT", "8080"),
		DiscordToken:     getEnv("DISCORD_TOKEN", ""),
		DBDriver:         getEnv("DB_DRIVER", "postgres"),
		DBHost:           getEnv("DB_HOST", "127.0.0.1"),
		DBUser:           getEnv("DB_USER", "postgres"),
		DBPassword:       getEnv("DB_PASSWORD", ""),
		DBName:           getEnv("DB_NAME", "community_tracker"),
		DBPort:           getEnvAsInt("DB_PORT", 5432),
		QFSChannelID:     getEnv("QFS_CHANNEL_ID", ""),
		ModRoleID:        getEnv("MOD_ROLE_ID", ""),
		ModRoleName:      getEnv("MOD_ROLE_NAME", ""),
		TicketCategoryID: getEnv("TICKET_CATEGORY_ID", ""),
		QFSChannelName:   getEnv("QFS_CHANNEL_NAME", ""),
		AdminCategoryID:  getEnv("ADMIN_CATEGORY_ID", ""),
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
