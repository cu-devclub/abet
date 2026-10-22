// Package memstore is an in-memory port.Storage for tests (behaves like the MinIO adapter, plus inspection helpers).
package memstore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

type object struct {
	data []byte
	info port.ObjectInfo
}

// Store keeps objects in a map.
type Store struct {
	mu   sync.Mutex
	objs map[string]object
	// PutErr, when set, makes Put and PutFile fail (to test upload failures).
	PutErr error
}

var _ port.Storage = (*Store)(nil)

func New() *Store { return &Store{objs: map[string]object{}} }

func (s *Store) put(key string, data []byte, ct string, at time.Time) {
	s.objs[key] = object{data: data, info: port.ObjectInfo{Key: key, Size: int64(len(data)), LastModified: at, ContentType: ct}}
}

func (s *Store) Put(_ context.Context, key string, r io.Reader, _ int64, ct string) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.put(key, b, ct, time.Now())
	return nil
}

func (s *Store) PutFile(ctx context.Context, key, localPath, ct string) error {
	b, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}
	return s.Put(ctx, key, bytes.NewReader(b), int64(len(b)), ct)
}

func (s *Store) Get(_ context.Context, key string) (io.ReadCloser, port.ObjectInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.objs[key]
	if !ok {
		return nil, port.ObjectInfo{}, fmt.Errorf("get %s: %w", key, domain.ErrNotFound)
	}
	return io.NopCloser(bytes.NewReader(o.data)), o.info, nil
}

func (s *Store) Stat(_ context.Context, key string) (port.ObjectInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.objs[key]
	if !ok {
		return port.ObjectInfo{}, fmt.Errorf("stat %s: %w", key, domain.ErrNotFound)
	}
	return o.info, nil
}

func (s *Store) Copy(_ context.Context, src, dst string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.objs[src]
	if !ok {
		return fmt.Errorf("copy %s: %w", src, domain.ErrNotFound)
	}
	s.put(dst, append([]byte(nil), o.data...), o.info.ContentType, time.Now())
	return nil
}

func (s *Store) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objs, key)
	return nil
}

func (s *Store) DeletePrefix(_ context.Context, prefix string) error {
	prefix = strings.TrimSuffix(prefix, "/") + "/"
	s.mu.Lock()
	defer s.mu.Unlock()
	for k := range s.objs {
		if strings.HasPrefix(k, prefix) {
			delete(s.objs, k)
		}
	}
	return nil
}

func (s *Store) List(_ context.Context, prefix string, recursive bool) ([]port.ObjectInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []port.ObjectInfo{}
	for k, o := range s.objs {
		if !strings.HasPrefix(k, prefix) || (!recursive && strings.Contains(k[len(prefix):], "/")) {
			continue
		}
		out = append(out, o.info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

func (s *Store) Ping(context.Context) error { return nil }

// ---- test helpers ----

// Seed stores data under key (now).
func (s *Store) Seed(key, data string) { s.SeedAged(key, data, 0) }

// SeedAged stores data with a LastModified of age ago.
func (s *Store) SeedAged(key, data string, age time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.put(key, []byte(data), "", time.Now().Add(-age))
}

func (s *Store) Has(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.objs[key]
	return ok
}

func (s *Store) Bytes(key string) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.objs[key].data
}

// Keys lists all keys below prefix, sorted.
func (s *Store) Keys(prefix string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []string{}
	for k := range s.objs {
		if strings.HasPrefix(k, prefix) {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
