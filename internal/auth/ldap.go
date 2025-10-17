package auth

import (
	"fmt"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"photosync-backend/internal/config"
	"photosync-backend/internal/models"
)

type ActiveDirectoryAuth struct {
	config *config.LDAPConfig
}

func NewActiveDirectoryAuth(cfg *config.LDAPConfig) *ActiveDirectoryAuth {
	return &ActiveDirectoryAuth{config: cfg}
}

func (c *ActiveDirectoryAuth) GetName() string {
	return "active_directory"
}

func (c *ActiveDirectoryAuth) Authenticate(username, password string) (*models.UserInfo, error) {
	conn, err := ldap.DialURL(c.config.Server)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP: %w", err)
	}
	defer conn.Close()

	if c.config.BindDN != "" {
		err = conn.Bind(c.config.BindDN, c.config.BindPass)
		if err != nil {
			return nil, fmt.Errorf("failed to bind service account: %w", err)
		}
	}

	filter := strings.ReplaceAll(c.config.UserFilter, "{username}", ldap.EscapeFilter(username))

	searchRequest := ldap.NewSearchRequest(
		c.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		filter,
		[]string{"dn", "cn", "mail", "sAMAccountName"},
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
		Username: sr.Entries[0].GetAttributeValue("sAMAccountName"),
		DN:       userDN,
		Email:    sr.Entries[0].GetAttributeValue("mail"),
		FullName: sr.Entries[0].GetAttributeValue("cn"),
	}

	return userInfo, nil
}
