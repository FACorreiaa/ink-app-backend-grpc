package studio

type SessionManagerKey struct{}

// UserSession represents the user session data stored in Redis

type User struct {
	ID       string
	Username string
	Email    string
	Password string
}
