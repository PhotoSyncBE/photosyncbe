package auth

import (
	"fmt"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"photosync-backend/internal/config"
	"photosync-backend/internal/models"
)

type LDAPAuth struct {
	config *config.LDAPGenericConfig
}

func NewLDAPAuth(cfg *config.LDAPGenericConfig) *LDAPAuth {
	return &LDAPAuth{config: cfg}
}

func (a *LDAPAuth) GetName() string {
	return "ldap"
}

func (a *LDAPAuth) Authenticate(username, password string) (*models.UserInfo, error) {
	conn, err := ldap.DialURL(a.config.Server)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP: %w", err)
	}
	defer conn.Close()

	if a.config.BindDN != "" {
		err = conn.Bind(a.config.BindDN, a.config.BindPass)
		if err != nil {
			return nil, fmt.Errorf("failed to bind service account: %w", err)
		}
	}

	filter := strings.ReplaceAll(a.config.UserFilter, "{username}", ldap.EscapeFilter(username))

	searchRequest := ldap.NewSearchRequest(
		a.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		filter,
		[]string{"dn", "cn", "mail", "uid"},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to search user: %w", err)
	}

	if len(sr.Entries) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	if len(sr.Entries) > 1 {
		return nil, fmt.Errorf("multiple users found")
	}

	userDN := sr.Entries[0].DN

	err = conn.Bind(userDN, password)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials: %w", err)
	}

	userInfo := &models.UserInfo{
		Username: username,
		DN:       userDN,
		Email:    sr.Entries[0].GetAttributeValue("mail"),
		FullName: sr.Entries[0].GetAttributeValue("cn"),
	}

	return userInfo, nil
}
