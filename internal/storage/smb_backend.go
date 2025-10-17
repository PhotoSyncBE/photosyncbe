package storage

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/hirochachacha/go-smb2"
	"photosync-backend/internal/config"
	"photosync-backend/internal/models"
)

type SMBBackend struct {
	config *config.SMBConfig
}

type SMBConnection struct {
	session  *smb2.Session
	share    *smb2.Share
	username string
}

func NewSMBBackend(cfg *config.SMBConfig) *SMBBackend {
	return &SMBBackend{config: cfg}
}

func (b *SMBBackend) GetName() string {
	return "smb"
}

func (b *SMBBackend) Connect(username, password string) (Connection, error) {
	log.Printf("SMB: Connecting to %s:%d as user %s, share %s", b.config.Server, b.config.Port, username, b.config.Share)

	address := net.JoinHostPort(b.config.Server, fmt.Sprintf("%d", b.config.Port))
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Printf("SMB: TCP connection failed to %s: %v", address, err)
		return nil, fmt.Errorf("failed to connect to SMB server: %w", err)
	}

	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     username,
			Password: password,
			Domain:   b.config.Domain,
		},
	}

	session, err := d.Dial(conn)
	if err != nil {
		conn.Close()
		log.Printf("SMB: Session establishment failed for user %s: %v", username, err)
		return nil, fmt.Errorf("failed to establish SMB session: %w", err)
	}

	share, err := session.Mount(b.config.Share)
	if err != nil {
		session.Logoff()
		log.Printf("SMB: Failed to mount share '%s' for user %s: %v", b.config.Share, username, err)
		return nil, fmt.Errorf("failed to mount share '%s': %w", b.config.Share, err)
	}

	log.Printf("SMB: Successfully connected user %s to share %s", username, b.config.Share)
	return &SMBConnection{
		session:  session,
		share:    share,
		username: username,
	}, nil
}

func (b *SMBBackend) Upload(conn Connection, username, filename string, data io.Reader) error {
	smbConn := conn.(*SMBConnection)

	userDir := username
	if b.config.Path != "" {
		userDir = b.config.Path + "/" + username
	}

	if err := b.ensureDirectory(smbConn.share, userDir); err != nil {
		return fmt.Errorf("failed to create user directory: %w", err)
	}

	fullPath := userDir + "/" + filename

	file, err := smbConn.share.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, data)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (b *SMBBackend) Download(conn Connection, username, filename string) ([]byte, error) {
	smbConn := conn.(*SMBConnection)

	userDir := username
	if b.config.Path != "" {
		userDir = b.config.Path + "/" + username
	}
	fullPath := userDir + "/" + filename

	file, err := smbConn.share.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

func (b *SMBBackend) List(conn Connection, username string) ([]models.FileInfo, error) {
	smbConn := conn.(*SMBConnection)

	userDir := username
	if b.config.Path != "" {
		userDir = b.config.Path + "/" + username
	}

	log.Printf("SMB: Listing files for user %s in directory: %s", username, userDir)

	if err := b.ensureDirectory(smbConn.share, userDir); err != nil {
		log.Printf("SMB: Could not ensure directory %s exists: %v - will try to list anyway", userDir, err)
	}

	entries, err := smbConn.share.ReadDir(userDir)
	if err != nil {
		log.Printf("SMB: ReadDir failed for %s: %v - returning empty array", userDir, err)
		return []models.FileInfo{}, nil
	}

	files := []models.FileInfo{}
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, models.FileInfo{
				Name:    entry.Name(),
				Size:    entry.Size(),
				ModTime: entry.ModTime(),
			})
		}
	}

	log.Printf("SMB: Found %d files in %s", len(files), userDir)
	return files, nil
}

func (b *SMBBackend) Delete(conn Connection, username, filename string) error {
	smbConn := conn.(*SMBConnection)

	userDir := username
	if b.config.Path != "" {
		userDir = b.config.Path + "/" + username
	}
	fullPath := userDir + "/" + filename

	err := smbConn.share.Remove(fullPath)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

func (b *SMBBackend) Close(conn Connection) error {
	if conn == nil {
		return nil
	}
	smbConn := conn.(*SMBConnection)
	return smbConn.session.Logoff()
}

func (b *SMBBackend) ensureDirectory(share *smb2.Share, path string) error {
	log.Printf("SMB: Checking if directory exists: %s", path)

	file, err := share.Open(path)
	if err == nil {
		file.Close()
		log.Printf("SMB: Directory %s already exists and is accessible", path)
		return nil
	}

	log.Printf("SMB: Directory %s open failed (%v), attempting to create...", path, err)

	err = share.Mkdir(path, 0755)
	if err != nil {
		log.Printf("SMB: Mkdir failed for %s: %v", path, err)
		if file, openErr := share.Open(path); openErr == nil {
			file.Close()
			log.Printf("SMB: Directory %s exists and is now accessible", path)
			return nil
		}
		return fmt.Errorf("failed to access or create directory %s: mkdir error: %w", path, err)
	}

	log.Printf("SMB: Directory %s created successfully", path)
	return nil
}
