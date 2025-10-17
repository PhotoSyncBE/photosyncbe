package storage

import (
	"fmt"

	"photosync-backend/internal/config"
)

func NewStorageBackend(cfg *config.Config) (StorageBackend, error) {
	storageType := cfg.Storage.Type
	if storageType == "" {
		storageType = "smb"
	}

	switch storageType {
	case "smb":
		return NewSMBBackend(&cfg.SMB), nil
	case "s3":
		return NewS3Backend(&cfg.S3), nil
	case "nfs":
		return NewNFSBackend(&cfg.NFS), nil
	default:
		return nil, fmt.Errorf("unknown storage type: %s", storageType)
	}
}
