package auth

import "photosync-backend/internal/models"

type Authenticator interface {
	Authenticate(username, password string) (*models.UserInfo, error)
	GetName() string
}
