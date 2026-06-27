package database

import "time"

type Question struct {
	ID        uint      `gorm:"primaryKey"`
	GuildID   string    `gorm:"index;not null"`
	ChannelID string    `gorm:"index;not null"`
	MessageID string    `gorm:"index;not null;unique"`
	Title     string    `gorm:"type:varchar(255)"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type QuestionCheck struct {
	ID         uint       `gorm:"primaryKey"`
	QuestionID uint       `gorm:"index;not null"`
	Question   Question   `gorm:"constraint:OnDelete:CASCADE"`
	RoleID     string     `gorm:"index;not null"`
	MemberID   string     `gorm:"index;not null"`
	Date       *time.Time `gorm:"index"`
	CheckedAt  time.Time  `gorm:"autoCreateTime"`
	RemovedAt  *time.Time
}

type Ticket struct {
	ID            uint       `gorm:"primaryKey"`
	GuildID       string     `gorm:"index;not null"`
	ChannelID     string     `gorm:"index;not null;unique"`
	FirstRespID   *string    `gorm:"index"`
	FirstRespDate *time.Time `gorm:"index"`
	FirstRespAt   *time.Time
	ClosedAt      *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type DailyMessage struct {
	Date      time.Time `gorm:"type:date;primaryKey"`
	GuildID   string    `gorm:"primaryKey;not null"`
	MemberID  string    `gorm:"primaryKey;not null"`
	Count     int       `gorm:"default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type VoiceTime struct {
	ID        uint       `gorm:"primaryKey"`
	GuildID   string     `gorm:"index;not null"`
	MemberID  string     `gorm:"index;not null"`
	ChannelID string
	Date      *time.Time `gorm:"index"`
	JoinedAt  time.Time  `gorm:"index;not null"`
	LeftAt    *time.Time
	Duration  int64      `gorm:"default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type LOA struct {
	ID        uint       `gorm:"primaryKey"`
	GuildID   string     `gorm:"index;not null"`
	MemberID  string     `gorm:"index;not null"`
	StartDate *time.Time `gorm:"index"`
	EndDate   *time.Time `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Session struct {
	Token         string    `gorm:"primaryKey;type:varchar(64)"`
	DiscordUserID string    `gorm:"not null;index"`
	ExpiresAt     time.Time `gorm:"not null;index"`
	CreatedAt     time.Time
}
