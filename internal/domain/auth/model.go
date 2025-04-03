package auth

import "time"

type SessionManagerKey struct{}

// UserSession represents the user session data stored in Redis

type User struct {
	ID           string
	Username     string
	Email        string
	PasswordHash string
	Role         string // "owner", "staff", "customer", etc.
	StudioID     string // Which studio they belong to (if applicable)
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}
