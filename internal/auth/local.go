package auth

import (
	"encoding/json"
	"fmt"
	"os"

	"photosync-backend/internal/config"
	"photosync-backend/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type LocalAuth struct {
	config *config.LocalAuthConfig
	users  map[string]*LocalUser
}

type LocalUser struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	Email        string `json:"email"`
	FullName     string `json:"full_name"`
}

func NewLocalAuth(cfg *config.LocalAuthConfig) *LocalAuth {
	auth := &LocalAuth{
		config: cfg,
		users:  make(map[string]*LocalUser),
	}
	auth.loadUsers()
	return auth
}

func (a *LocalAuth) GetName() string {
	return "local"
}

func (a *LocalAuth) Authenticate(username, password string) (*models.UserInfo, error) {
	user, exists := a.users[username]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return &models.UserInfo{
		Username: user.Username,
		DN:       "local:" + user.Username,
		Email:    user.Email,
		FullName: user.FullName,
	}, nil
}

func (a *LocalAuth) loadUsers() error {
	data, err := os.ReadFile(a.config.UsersFile)
	if err != nil {
		return fmt.Errorf("failed to read users file: %w", err)
	}

	var users []LocalUser
	if err := json.Unmarshal(data, &users); err != nil {
		return fmt.Errorf("failed to parse users file: %w", err)
	}

	for i := range users {
		a.users[users[i].Username] = &users[i]
	}

	return nil
}
