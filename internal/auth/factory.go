package auth

import (
	"fmt"

	"photosync-backend/internal/config"
)

func NewAuthenticator(cfg *config.Config) (Authenticator, error) {
	authType := cfg.Auth.Type
	if authType == "" {
		authType = "active_directory"
	}

	switch authType {
	case "active_directory":
		return NewActiveDirectoryAuth(&cfg.LDAP), nil
	case "ldap":
		return NewLDAPAuth(&cfg.LDAPGeneric), nil
	case "local":
		return NewLocalAuth(&cfg.LocalAuth), nil
	case "oauth2":
		return NewOAuth2Auth(&cfg.OAuth2), nil
	default:
		return nil, fmt.Errorf("unknown auth type: %s", authType)
	}
}
