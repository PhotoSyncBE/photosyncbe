package auth

import (
	"context"
	"fmt"

	"photosync-backend/internal/config"
	"photosync-backend/internal/models"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

type OAuth2Auth struct {
	config *config.OAuth2Config
	oauth  *oauth2.Config
}

func NewOAuth2Auth(cfg *config.OAuth2Config) *OAuth2Auth {
	var endpoint oauth2.Endpoint

	switch cfg.Provider {
	case "google":
		endpoint = google.Endpoint
	case "github":
		endpoint = github.Endpoint
	default:
		endpoint = oauth2.Endpoint{}
	}

	return &OAuth2Auth{
		config: cfg,
		oauth: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint:     endpoint,
			Scopes:       []string{"openid", "profile", "email"},
		},
	}
}

func (a *OAuth2Auth) GetName() string {
	return "oauth2"
}

func (a *OAuth2Auth) Authenticate(username, password string) (*models.UserInfo, error) {
	token, err := a.oauth.Exchange(context.Background(), password)
	if err != nil {
		return nil, fmt.Errorf("oauth2 token exchange failed: %w", err)
	}

	return &models.UserInfo{
		Username: username,
		DN:       "oauth2:" + username,
		Email:    username,
		FullName: token.Extra("name").(string),
	}, nil
}
