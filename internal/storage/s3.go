package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"photosync-backend/internal/config"
	"photosync-backend/internal/models"
)

type S3Backend struct {
	config *config.S3Config
	client *s3.Client
}

type S3Connection struct {
	username string
}

func NewS3Backend(cfg *config.S3Config) *S3Backend {
	awsCfg := aws.Config{
		Region: cfg.Region,
		Credentials: credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		),
	}

	if cfg.Endpoint != "" {
		awsCfg.BaseEndpoint = aws.String(cfg.Endpoint)
	}

	return &S3Backend{
		config: cfg,
		client: s3.NewFromConfig(awsCfg),
	}
}

func (b *S3Backend) GetName() string {
	return "s3"
}

func (b *S3Backend) Connect(username, password string) (Connection, error) {
	return &S3Connection{username: username}, nil
}

func (b *S3Backend) Upload(conn Connection, username, filename string, data io.Reader) error {
	key := b.getObjectKey(username, filename)

	body, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	_, err = b.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(b.config.Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(body),
	})

	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

func (b *S3Backend) Download(conn Connection, username, filename string) ([]byte, error) {
	key := b.getObjectKey(username, filename)

	result, err := b.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(b.config.Bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to download from S3: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read S3 object: %w", err)
	}

	return data, nil
}

func (b *S3Backend) List(conn Connection, username string) ([]models.FileInfo, error) {
	prefix := b.getUserPrefix(username)

	result, err := b.client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket: aws.String(b.config.Bucket),
		Prefix: aws.String(prefix),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list S3 objects: %w", err)
	}

	files := []models.FileInfo{}
	for _, obj := range result.Contents {
		filename := (*obj.Key)[len(prefix):]
		if filename != "" {
			files = append(files, models.FileInfo{
				Name:    filename,
				Size:    *obj.Size,
				ModTime: *obj.LastModified,
			})
		}
	}

	return files, nil
}

func (b *S3Backend) Delete(conn Connection, username, filename string) error {
	key := b.getObjectKey(username, filename)

	_, err := b.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(b.config.Bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	return nil
}

func (b *S3Backend) Close(conn Connection) error {
	return nil
}

func (b *S3Backend) getUserPrefix(username string) string {
	if b.config.PathPrefix != "" {
		return b.config.PathPrefix + "/" + username + "/"
	}
	return username + "/"
}

func (b *S3Backend) getObjectKey(username, filename string) string {
	return b.getUserPrefix(username) + filename
}
