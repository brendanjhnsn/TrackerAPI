package database

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/brendanjhnsn/TrackerAPI/core/config"
	_ "github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) (*gorm.DB, *sql.DB, error) {
	switch strings.ToLower(cfg.DBDriver) {
	case "postgres", "postgresql":
		return connectPostgres(cfg)
	case "mysql":
		return connectMySQL(cfg)
	default:
		return nil, nil, fmt.Errorf("unsupported DB driver: %s", cfg.DBDriver)
	}
}

func connectPostgres(cfg *config.Config) (*gorm.DB, *sql.DB, error) {
	if err := ensurePostgresDatabase(cfg); err != nil {
		return nil, nil, fmt.Errorf("failed to ensure postgres database: %w", err)
	}
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	logLevel := logger.Error
	if strings.ToLower(cfg.Environment) != "production" {
		logLevel = logger.Info
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}
	if err := db.AutoMigrate(&Question{}, &QuestionCheck{}, &Ticket{}, &DailyMessage{}, &VoiceTime{}, &LOA{}, &Session{}); err != nil {
		return nil, nil, fmt.Errorf("failed to migrate postgres schema: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get postgres sql db instance: %w", err)
	}
	return db, sqlDB, nil
}

func ensurePostgresDatabase(cfg *config.Config) error {
	adminDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword)
	adminDB, err := sql.Open("pgx", adminDSN)
	if err != nil {
		return err
	}
	defer adminDB.Close()
	if err := adminDB.Ping(); err != nil {
		return err
	}
	var exists bool
	if err := adminDB.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", cfg.DBName).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	quoted := fmt.Sprintf("\"%s\"", strings.ReplaceAll(cfg.DBName, "\"", "\"\""))
	_, err = adminDB.Exec("CREATE DATABASE " + quoted)
	return err
}

func connectMySQL(cfg *config.Config) (*gorm.DB, *sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)
	logLevel := logger.Error
	if strings.ToLower(cfg.Environment) != "production" {
		logLevel = logger.Info
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to mysql: %w", err)
	}
	if err := db.AutoMigrate(&Question{}, &QuestionCheck{}, &Ticket{}, &DailyMessage{}, &VoiceTime{}, &LOA{}, &Session{}); err != nil {
		return nil, nil, fmt.Errorf("failed to migrate mysql schema: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get mysql sql db instance: %w", err)
	}
	return db, sqlDB, nil
}

func Close(sqlDB *sql.DB) {
	if sqlDB != nil {
		_ = sqlDB.Close()
	}
}
