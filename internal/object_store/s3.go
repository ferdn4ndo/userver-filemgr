package object_store

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Backend struct {
	bucket  string
	root    string
	client  *s3.Client
	presign *s3.PresignClient
}

func (s *s3Backend) key(realPath string) string {
	k := strings.TrimLeft(path.Clean("/"+realPath), "/")
	if s.root != "" {
		return s.root + "/" + k
	}
	return k
}

func (s *s3Backend) Put(ctx context.Context, realPath string, r io.Reader, size int64, contentType string) error {
	in := &s3.PutObjectInput{
		Bucket: &s.bucket,
		Key:    stringPtr(s.key(realPath)),
		Body:   r,
	}
	if contentType != "" {
		in.ContentType = &contentType
	}
	_, err := s.client.PutObject(ctx, in)
	return err
}

func (s *s3Backend) Delete(ctx context.Context, realPath string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    stringPtr(s.key(realPath)),
	})
	return err
}

func (s *s3Backend) Open(ctx context.Context, realPath string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    stringPtr(s.key(realPath)),
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

func (s *s3Backend) DownloadURL(ctx context.Context, realPath, virtualPath string, contentType string, expiry time.Duration, attachment bool) (string, error) {
	filename := path.Base(strings.ReplaceAll(virtualPath, "\\", "/"))
	if filename == "" || filename == "." {
		filename = "download"
	}
	disposition := "inline"
	if attachment {
		disposition = "attachment"
	}
	disposition = fmt.Sprintf(`%s; filename="%s"`, disposition, filename)

	in := &s3.GetObjectInput{
		Bucket:                     &s.bucket,
		Key:                        stringPtr(s.key(realPath)),
		ResponseContentDisposition: &disposition,
	}
	if contentType != "" {
		in.ResponseContentType = &contentType
	}
	out, err := s.presign.PresignGetObject(ctx, in, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", err
	}
	return out.URL, nil
}

func stringPtr(s string) *string { return &s }
