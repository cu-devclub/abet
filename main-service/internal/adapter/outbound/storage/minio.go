// Package storage implements port.Storage on a MinIO (S3 compatible) bucket.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

// Config is the connection setting of the object store.
type Config struct {
	Endpoint  string // host:port, no scheme
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

// MinIO stores objects in one bucket.
type MinIO struct {
	client *minio.Client
	bucket string
}

var _ port.Storage = (*MinIO)(nil)

// NewMinIO builds the client (no network access yet). Call EnsureBucket before serving.
func NewMinIO(cfg Config) (*MinIO, error) {
	if cfg.Endpoint == "" || cfg.Bucket == "" {
		return nil, errors.New("storage: MINIO_ENDPOINT and MINIO_BUCKET are required")
	}
	cl, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:     credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:    cfg.UseSSL,
		Transport: otelhttp.NewTransport(http.DefaultTransport), // one span per S3 request
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}
	return &MinIO{client: cl, bucket: cfg.Bucket}, nil
}

// EnsureBucket creates the bucket when it does not exist yet (idempotent; minio-init normally did it).
func (m *MinIO) EnsureBucket(ctx context.Context) error {
	ok, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		return fmt.Errorf("check bucket %q: %w", m.bucket, err)
	}
	if ok {
		return nil
	}
	if err := m.client.MakeBucket(ctx, m.bucket, minio.MakeBucketOptions{}); err != nil {
		if code := minio.ToErrorResponse(err).Code; code == "BucketAlreadyOwnedByYou" || code == "BucketAlreadyExists" {
			return nil
		}
		return fmt.Errorf("create bucket %q: %w", m.bucket, err)
	}
	slog.Info("storage bucket created", "bucket", m.bucket)
	return nil
}

// checkKey rejects keys that are empty, absolute or contain "." / ".." / empty segments.
func checkKey(key string) error {
	if key == "" || strings.HasPrefix(key, "/") || strings.ContainsRune(key, 0) || strings.Contains(key, "\\") {
		return fmt.Errorf("%w: invalid storage key %q", domain.ErrInvalidInput, key)
	}
	for _, seg := range strings.Split(strings.TrimSuffix(key, "/"), "/") {
		if seg == "" || seg == "." || seg == ".." {
			return fmt.Errorf("%w: invalid storage key %q", domain.ErrInvalidInput, key)
		}
	}
	return nil
}

func isNotFound(err error) bool {
	code := minio.ToErrorResponse(err).Code
	return code == "NoSuchKey" || code == "NotFound" || code == "NoSuchBucket"
}

func info(o minio.ObjectInfo) port.ObjectInfo {
	return port.ObjectInfo{Key: o.Key, Size: o.Size, LastModified: o.LastModified, ContentType: o.ContentType}
}

func (m *MinIO) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	if err := checkKey(key); err != nil {
		return err
	}
	_, err := m.client.PutObject(ctx, m.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("put %s: %w", key, err)
	}
	return nil
}

func (m *MinIO) PutFile(ctx context.Context, key, localPath, contentType string) error {
	if err := checkKey(key); err != nil {
		return err
	}
	if _, err := m.client.FPutObject(ctx, m.bucket, key, localPath, minio.PutObjectOptions{ContentType: contentType}); err != nil {
		return fmt.Errorf("put %s: %w", key, err)
	}
	return nil
}

func (m *MinIO) Get(ctx context.Context, key string) (io.ReadCloser, port.ObjectInfo, error) {
	if err := checkKey(key); err != nil {
		return nil, port.ObjectInfo{}, err
	}
	obj, err := m.client.GetObject(ctx, m.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, port.ObjectInfo{}, fmt.Errorf("get %s: %w", key, err)
	}
	st, err := obj.Stat() // GetObject is lazy: this is where a missing key shows up
	if err != nil {
		_ = obj.Close()
		if isNotFound(err) {
			return nil, port.ObjectInfo{}, fmt.Errorf("get %s: %w", key, domain.ErrNotFound)
		}
		return nil, port.ObjectInfo{}, fmt.Errorf("get %s: %w", key, err)
	}
	return obj, info(st), nil
}

func (m *MinIO) Stat(ctx context.Context, key string) (port.ObjectInfo, error) {
	if err := checkKey(key); err != nil {
		return port.ObjectInfo{}, err
	}
	st, err := m.client.StatObject(ctx, m.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if isNotFound(err) {
			return port.ObjectInfo{}, fmt.Errorf("stat %s: %w", key, domain.ErrNotFound)
		}
		return port.ObjectInfo{}, fmt.Errorf("stat %s: %w", key, err)
	}
	return info(st), nil
}

func (m *MinIO) Copy(ctx context.Context, srcKey, dstKey string) error {
	if err := checkKey(srcKey); err != nil {
		return err
	}
	if err := checkKey(dstKey); err != nil {
		return err
	}
	_, err := m.client.CopyObject(ctx,
		minio.CopyDestOptions{Bucket: m.bucket, Object: dstKey},
		minio.CopySrcOptions{Bucket: m.bucket, Object: srcKey})
	if err != nil {
		if isNotFound(err) {
			return fmt.Errorf("copy %s: %w", srcKey, domain.ErrNotFound)
		}
		return fmt.Errorf("copy %s to %s: %w", srcKey, dstKey, err)
	}
	return nil
}

func (m *MinIO) Delete(ctx context.Context, key string) error {
	if err := checkKey(key); err != nil {
		return err
	}
	if err := m.client.RemoveObject(ctx, m.bucket, key, minio.RemoveObjectOptions{}); err != nil && !isNotFound(err) {
		return fmt.Errorf("delete %s: %w", key, err)
	}
	return nil
}

// DeletePrefix removes everything below prefix ("outputs/<id>/"); a missing prefix is not an error.
// A prefix without trailing slash gets one, so "outputs/ab" never touches "outputs/abc/".
func (m *MinIO) DeletePrefix(ctx context.Context, prefix string) error {
	if err := checkKey(prefix); err != nil {
		return err
	}
	prefix = strings.TrimSuffix(prefix, "/") + "/"
	keys := make(chan minio.ObjectInfo)
	var listErr error
	go func() {
		defer close(keys)
		for o := range m.client.ListObjects(ctx, m.bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
			if o.Err != nil {
				listErr = o.Err
				return
			}
			select {
			case keys <- o:
			case <-ctx.Done():
				listErr = ctx.Err()
				return
			}
		}
	}()
	var firstErr error
	for e := range m.client.RemoveObjects(ctx, m.bucket, keys, minio.RemoveObjectsOptions{}) {
		if firstErr == nil {
			firstErr = fmt.Errorf("delete %s: %w", e.ObjectName, e.Err)
		}
	}
	if firstErr != nil {
		return firstErr
	}
	if listErr != nil {
		return fmt.Errorf("list %s: %w", prefix, listErr)
	}
	return nil
}

func (m *MinIO) List(ctx context.Context, prefix string, recursive bool) ([]port.ObjectInfo, error) {
	if strings.HasPrefix(prefix, "/") || strings.Contains(prefix, "..") {
		return nil, fmt.Errorf("%w: invalid storage prefix %q", domain.ErrInvalidInput, prefix)
	}
	out := []port.ObjectInfo{}
	for o := range m.client.ListObjects(ctx, m.bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: recursive}) {
		if o.Err != nil {
			return nil, fmt.Errorf("list %s: %w", prefix, o.Err)
		}
		out = append(out, info(o))
	}
	return out, nil
}

// Ping succeeds when the endpoint answers and the bucket exists.
func (m *MinIO) Ping(ctx context.Context) error {
	ok, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		return fmt.Errorf("storage unreachable: %w", err)
	}
	if !ok {
		return fmt.Errorf("storage bucket %q does not exist", m.bucket)
	}
	return nil
}
