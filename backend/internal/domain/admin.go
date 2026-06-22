package domain

type Admin struct {
	Username       string `json:"username"`
	HashedPassword string `json:"hashed_password"`
	CreatedAt      string `json:"created_at"`
	LastLogin      string `json:"last_login,omitempty"`
}
