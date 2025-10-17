package storage

import (
	"fmt"
	"io"

	"photosync-backend/internal/config"
	"photosync-backend/internal/models"
	nfs "github.com/vmware/go-nfs-client/nfs"
	"github.com/vmware/go-nfs-client/nfs/rpc"
)

type NFSBackend struct {
	config *config.NFSConfig
}

type NFSConnection struct {
	mount    *nfs.Target
	username string
}

func NewNFSBackend(cfg *config.NFSConfig) *NFSBackend {
	return &NFSBackend{config: cfg}
}

func (b *NFSBackend) GetName() string {
	return "nfs"
}

func (b *NFSBackend) Connect(username, password string) (Connection, error) {
	mount, err := nfs.DialMount(b.config.Server)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NFS server: %w", err)
	}

	auth := rpc.NewAuthUnix("photosync", 1000, 1000)

	target, err := mount.Mount(b.config.Export, auth.Auth())
	if err != nil {
		return nil, fmt.Errorf("failed to mount NFS export: %w", err)
	}

	return &NFSConnection{
		mount:    target,
		username: username,
	}, nil
}

func (b *NFSBackend) Upload(conn Connection, username, filename string, data io.Reader) error {
	nfsConn := conn.(*NFSConnection)

	userDir := b.getUserPath(username)

	if err := b.ensureDirectory(nfsConn.mount, userDir); err != nil {
		return fmt.Errorf("failed to create user directory: %w", err)
	}

	fullPath := userDir + "/" + filename

	body, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	_, err = nfsConn.mount.OpenFile(fullPath, 0644)
	if err == nil {
		return fmt.Errorf("file already exists")
	}

	file, err := nfsConn.mount.OpenFile(fullPath, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = file.Write(body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (b *NFSBackend) Download(conn Connection, username, filename string) ([]byte, error) {
	nfsConn := conn.(*NFSConnection)

	fullPath := b.getUserPath(username) + "/" + filename

	file, err := nfsConn.mount.Open(fullPath)
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

func (b *NFSBackend) List(conn Connection, username string) ([]models.FileInfo, error) {
	nfsConn := conn.(*NFSConnection)

	userDir := b.getUserPath(username)

	if err := b.ensureDirectory(nfsConn.mount, userDir); err != nil {
		return []models.FileInfo{}, nil
	}

	entries, err := nfsConn.mount.ReadDirPlus(userDir)
	if err != nil {
		return []models.FileInfo{}, nil
	}

	files := []models.FileInfo{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, models.FileInfo{
			Name:    entry.Name(),
			Size:    int64(entry.Size()),
			ModTime: entry.ModTime(),
		})
	}

	return files, nil
}

func (b *NFSBackend) Delete(conn Connection, username, filename string) error {
	nfsConn := conn.(*NFSConnection)

	fullPath := b.getUserPath(username) + "/" + filename

	err := nfsConn.mount.Remove(fullPath)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

func (b *NFSBackend) Close(conn Connection) error {
	if conn == nil {
		return nil
	}
	nfsConn := conn.(*NFSConnection)
	nfsConn.mount.Close()
	return nil
}

func (b *NFSBackend) getUserPath(username string) string {
	if b.config.Path != "" {
		return b.config.Path + "/" + username
	}
	return username
}

func (b *NFSBackend) ensureDirectory(mount *nfs.Target, path string) error {
	_, err := mount.Mkdir(path, 0755)
	if err != nil {
		return nil
	}
	return nil
}
