package studio

type SessionManagerKey struct{}

// UserSession represents the user session data stored in Redis
type UserSession struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Tenant   string `json:"tenant"` // Store which tenant this session belongs to
}

type User struct {
	ID       string
	Username string
	Email    string
	Password string
}
