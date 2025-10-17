package storage

import (
	"io"

	"photosync-backend/internal/models"
)

type Connection interface{}

type StorageBackend interface {
	Connect(username, password string) (Connection, error)
	Upload(conn Connection, username, filename string, data io.Reader) error
	Download(conn Connection, username, filename string) ([]byte, error)
	List(conn Connection, username string) ([]models.FileInfo, error)
	Delete(conn Connection, username, filename string) error
	Close(conn Connection) error
	GetName() string
}
