package object_store

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/ferdn4ndo/userver-filemgr/lib"
)

// Backend reads/writes blobs for a storage row.
type Backend interface {
	Put(ctx context.Context, realPath string, r io.Reader, size int64, contentType string) error
	Open(ctx context.Context, realPath string) (io.ReadCloser, error)
	Delete(ctx context.Context, realPath string) error
	// DownloadURL returns a client-fetchable URL (presigned or file path for local).
	DownloadURL(ctx context.Context, realPath, virtualPath string, contentType string, expiry time.Duration, attachment bool) (string, error)
}

type Factory struct {
	env lib.Env
}

func NewFactory(env lib.Env) *Factory {
	return &Factory{env: env}
}

func (f *Factory) ForStorage(storageType string, credsJSON []byte) (Backend, error) {
	switch storageType {
	case "LOCAL", "":
		var m map[string]any
		_ = json.Unmarshal(credsJSON, &m)
		root := f.env.LocalRoot
		if v, ok := m["LOCAL_ROOT"].(string); ok && v != "" {
			root = v
		}
		return &localBackend{root: root}, nil
	case "AMAZON_S3":
		var c struct {
			AWSID      string `json:"AWS_S3_ID"`
			AWSKey     string `json:"AWS_S3_KEY"`
			Bucket     string `json:"AWS_S3_BUCKET"`
			Region     string `json:"AWS_S3_REGION"`
			RootFolder string `json:"AWS_S3_ROOT_FOLDER"`
		}
		if err := json.Unmarshal(credsJSON, &c); err != nil {
			return nil, err
		}
		if c.AWSID == "" || c.AWSKey == "" || c.Bucket == "" || c.Region == "" {
			return nil, fmt.Errorf("S3 credentials incomplete")
		}
		cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
			awsconfig.WithRegion(c.Region),
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(c.AWSID, c.AWSKey, "")),
		)
		if err != nil {
			return nil, err
		}
		client := s3.NewFromConfig(cfg)
		presign := s3.NewPresignClient(client)
		up := manager.NewUploader(client)
		return &s3Backend{bucket: c.Bucket, root: strings.Trim(c.RootFolder, "/"), client: client, presign: presign, uploader: up}, nil
	default:
		return nil, fmt.Errorf("unsupported storage type %q", storageType)
	}
}

type localBackend struct {
	root string
}

func (l *localBackend) abs(key string) string {
	key = strings.TrimLeft(filepath.ToSlash(key), "/")
	return filepath.Join(l.root, filepath.FromSlash(key))
}

func (l *localBackend) Put(ctx context.Context, realPath string, r io.Reader, size int64, _ string) error {
	p := l.abs(realPath)
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, r)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(p)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(p)
		return closeErr
	}
	return nil
}

func (l *localBackend) Delete(ctx context.Context, realPath string) error {
	return os.Remove(l.abs(realPath))
}

func (l *localBackend) Open(_ context.Context, realPath string) (io.ReadCloser, error) {
	return os.Open(l.abs(realPath))
}

func (l *localBackend) DownloadURL(_ context.Context, realPath, _ string, _ string, _ time.Duration, _ bool) (string, error) {
	return l.abs(realPath), nil
}
