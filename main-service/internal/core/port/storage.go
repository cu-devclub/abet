package port

import (
	"context"
	"io"
	"time"
)

// ObjectInfo describes one stored object.
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	ContentType  string
}

// Storage is the object store (MinIO bucket) shared with the portfolio worker. Keys look like "covers/<id>.pdf".
type Storage interface {
	// Put uploads r; size -1 is allowed (multipart upload of unknown length).
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	// PutFile uploads a local file.
	PutFile(ctx context.Context, key, localPath, contentType string) error
	// Get opens an object for streaming; domain.ErrNotFound when it does not exist. The caller must Close it.
	Get(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error)
	// Stat returns the object's metadata; domain.ErrNotFound when it does not exist.
	Stat(ctx context.Context, key string) (ObjectInfo, error)
	// Copy copies an object server-side.
	Copy(ctx context.Context, srcKey, dstKey string) error
	// Delete removes an object; a missing object is not an error.
	Delete(ctx context.Context, key string) error
	// DeletePrefix removes every object below prefix.
	DeletePrefix(ctx context.Context, prefix string) error
	// List returns the objects below prefix (recursive = false stops at the first "/").
	List(ctx context.Context, prefix string, recursive bool) ([]ObjectInfo, error)
	// Ping checks that the store is reachable and the bucket exists.
	Ping(ctx context.Context) error
}
