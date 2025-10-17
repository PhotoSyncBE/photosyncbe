package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Auth        AuthConfig        `yaml:"auth"`
	Storage     StorageConfig     `yaml:"storage"`
	LDAP        LDAPConfig        `yaml:"ldap"`
	LDAPGeneric LDAPGenericConfig `yaml:"ldap_generic"`
	LocalAuth   LocalAuthConfig   `yaml:"local_auth"`
	OAuth2      OAuth2Config      `yaml:"oauth2"`
	SMB         SMBConfig         `yaml:"smb"`
	S3          S3Config          `yaml:"s3"`
	NFS         NFSConfig         `yaml:"nfs"`
	JWT         JWTConfig         `yaml:"jwt"`
	Pool        PoolConfig        `yaml:"pool"`
	Logging     LoggingConfig     `yaml:"logging"`
}

type AuthConfig struct {
	Type string `yaml:"type"`
}

type StorageConfig struct {
	Type string `yaml:"type"`
}

type ServerConfig struct {
	Host string    `yaml:"host"`
	Port string    `yaml:"port"`
	TLS  TLSConfig `yaml:"tls"`
}

type TLSConfig struct {
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type LDAPConfig struct {
	Server     string `yaml:"server"`
	BaseDN     string `yaml:"base_dn"`
	BindDN     string `yaml:"bind_dn"`
	BindPass   string `yaml:"bind_pass"`
	UserFilter string `yaml:"user_filter"`
}

type LDAPGenericConfig struct {
	Server        string `yaml:"server"`
	BaseDN        string `yaml:"base_dn"`
	BindDN        string `yaml:"bind_dn"`
	BindPass      string `yaml:"bind_pass"`
	UserFilter    string `yaml:"user_filter"`
	UserDNPattern string `yaml:"user_dn_pattern"`
}

type LocalAuthConfig struct {
	UsersFile string `yaml:"users_file"`
}

type OAuth2Config struct {
	Provider     string `yaml:"provider"`
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	RedirectURL  string `yaml:"redirect_url"`
}

type SMBConfig struct {
	Server string `yaml:"server"`
	Port   int    `yaml:"port"`
	Share  string `yaml:"share"`
	Path   string `yaml:"path"`
	Domain string `yaml:"domain"`
}

type S3Config struct {
	Endpoint   string `yaml:"endpoint"`
	Region     string `yaml:"region"`
	Bucket     string `yaml:"bucket"`
	AccessKey  string `yaml:"access_key"`
	SecretKey  string `yaml:"secret_key"`
	PathPrefix string `yaml:"path_prefix"`
	UseSSL     bool   `yaml:"use_ssl"`
}

type NFSConfig struct {
	Server string `yaml:"server"`
	Export string `yaml:"export"`
	Path   string `yaml:"path"`
}

type JWTConfig struct {
	SecretKey     string `yaml:"secret_key"`
	EncryptionKey string `yaml:"encryption_key"`
	Issuer        string `yaml:"issuer"`
	Expiry        string `yaml:"expiry"`
}

type PoolConfig struct {
	ConnectionTTL string `yaml:"connection_ttl"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.Server.Host == "" {
		return fmt.Errorf("server host is required")
	}
	if c.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}

	if err := c.validateJWT(); err != nil {
		return err
	}

	authType := c.Auth.Type
	if authType == "" {
		authType = "active_directory"
	}

	if err := c.validateAuth(authType); err != nil {
		return err
	}

	storageType := c.Storage.Type
	if storageType == "" {
		storageType = "smb"
	}

	if err := c.validateStorage(storageType); err != nil {
		return err
	}

	return nil
}

func (c *Config) validateJWT() error {
	if c.JWT.SecretKey == "" || containsPlaceholder(c.JWT.SecretKey) {
		return fmt.Errorf("jwt secret_key must be set (no placeholders allowed)")
	}
	if c.JWT.EncryptionKey == "" || containsPlaceholder(c.JWT.EncryptionKey) {
		return fmt.Errorf("jwt encryption_key must be set (no placeholders allowed)")
	}
	if len(c.JWT.EncryptionKey) != 32 {
		return fmt.Errorf("jwt encryption_key must be exactly 32 bytes")
	}
	return nil
}

func (c *Config) validateAuth(authType string) error {
	switch authType {
	case "active_directory":
		if c.LDAP.Server == "" {
			return fmt.Errorf("ldap server is required for active_directory auth")
		}
		if c.LDAP.BaseDN == "" {
			return fmt.Errorf("ldap base_dn is required for active_directory auth")
		}
	case "ldap":
		if c.LDAPGeneric.Server == "" {
			return fmt.Errorf("ldap_generic server is required for ldap auth")
		}
		if c.LDAPGeneric.BaseDN == "" {
			return fmt.Errorf("ldap_generic base_dn is required for ldap auth")
		}
	case "local":
		if c.LocalAuth.UsersFile == "" {
			return fmt.Errorf("local_auth users_file is required for local auth")
		}
	case "oauth2":
		if c.OAuth2.Provider == "" {
			return fmt.Errorf("oauth2 provider is required for oauth2 auth")
		}
		if c.OAuth2.ClientID == "" || containsPlaceholder(c.OAuth2.ClientID) {
			return fmt.Errorf("oauth2 client_id must be set (no placeholders)")
		}
		if c.OAuth2.ClientSecret == "" || containsPlaceholder(c.OAuth2.ClientSecret) {
			return fmt.Errorf("oauth2 client_secret must be set (no placeholders)")
		}
	}
	return nil
}

func (c *Config) validateStorage(storageType string) error {
	switch storageType {
	case "smb":
		if c.SMB.Server == "" {
			return fmt.Errorf("smb server is required for smb storage")
		}
		if c.SMB.Share == "" {
			return fmt.Errorf("smb share is required for smb storage")
		}
	case "s3":
		if c.S3.Bucket == "" {
			return fmt.Errorf("s3 bucket is required for s3 storage")
		}
		if containsPlaceholder(c.S3.AccessKey) || containsPlaceholder(c.S3.SecretKey) {
			return fmt.Errorf("s3 access_key and secret_key must be set (no placeholders)")
		}
	case "nfs":
		if c.NFS.Server == "" {
			return fmt.Errorf("nfs server is required for nfs storage")
		}
		if c.NFS.Export == "" {
			return fmt.Errorf("nfs export is required for nfs storage")
		}
	}
	return nil
}

func containsPlaceholder(s string) bool {
	placeholders := []string{"CHANGE_ME", "YOUR_VALUE_HERE", "REQUIRED", "PLACEHOLDER", "CHANGEME"}
	for _, p := range placeholders {
		if len(s) > 0 && (s == p || len(s) > len(p) && containsSubstring(s, p)) {
			return true
		}
	}
	return false
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (c *Config) GetJWTExpiry() (time.Duration, error) {
	return time.ParseDuration(c.JWT.Expiry)
}

func (c *Config) GetPoolTTL() (time.Duration, error) {
	return time.ParseDuration(c.Pool.ConnectionTTL)
}
