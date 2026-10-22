package storage_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"abet/main-service/internal/adapter/outbound/storage"
	"abet/main-service/internal/core/domain"
)

// These specs need a real MinIO: set MINIO_TEST_ENDPOINT (host:port) and MINIO_TEST_ACCESS_KEY / MINIO_TEST_SECRET_KEY
// (root credentials are fine). Every run uses its own bucket.
var _ = Describe("MinIO adapter", func() {
	var (
		s   *storage.MinIO
		ctx = context.Background()
	)
	BeforeEach(func() {
		endpoint := os.Getenv("MINIO_TEST_ENDPOINT")
		if endpoint == "" {
			Skip("MINIO_TEST_ENDPOINT is not set")
		}
		var err error
		s, err = storage.NewMinIO(storage.Config{
			Endpoint: endpoint, AccessKey: os.Getenv("MINIO_TEST_ACCESS_KEY"), SecretKey: os.Getenv("MINIO_TEST_SECRET_KEY"),
			Bucket: "t09b-" + strings.ToLower(uuid.NewString()[:13]),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(s.EnsureBucket(ctx)).To(Succeed())
	})

	read := func(key string) string {
		rc, _, err := s.Get(ctx, key)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		defer rc.Close()
		b, err := io.ReadAll(rc)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		return string(b)
	}
	put := func(key, body string) {
		ExpectWithOffset(1, s.Put(ctx, key, strings.NewReader(body), int64(len(body)), "text/plain")).To(Succeed())
	}

	It("creates the bucket idempotently and pings", func() {
		Expect(s.EnsureBucket(ctx)).To(Succeed())
		Expect(s.Ping(ctx)).To(Succeed())
	})

	It("fails to ping when the bucket does not exist", func() {
		other, err := storage.NewMinIO(storage.Config{
			Endpoint: os.Getenv("MINIO_TEST_ENDPOINT"), AccessKey: os.Getenv("MINIO_TEST_ACCESS_KEY"),
			SecretKey: os.Getenv("MINIO_TEST_SECRET_KEY"), Bucket: "t09b-missing-bucket",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(other.Ping(ctx)).NotTo(Succeed())
	})

	It("puts and gets with size, content type and modification time", func() {
		Expect(s.Put(ctx, "covers/a.pdf", strings.NewReader("%PDF-1.4 hello"), 14, "application/pdf")).To(Succeed())
		rc, info, err := s.Get(ctx, "covers/a.pdf")
		Expect(err).NotTo(HaveOccurred())
		defer rc.Close()
		b, _ := io.ReadAll(rc)
		Expect(string(b)).To(Equal("%PDF-1.4 hello"))
		Expect(info.Size).To(Equal(int64(14)))
		Expect(info.ContentType).To(Equal("application/pdf"))
		Expect(info.Key).To(Equal("covers/a.pdf"))
		Expect(info.LastModified).To(BeTemporally("~", time.Now(), time.Minute))
	})

	It("accepts an unknown size (-1, multipart)", func() {
		body := strings.Repeat("x", 6<<20)
		Expect(s.Put(ctx, "uploads/big.zip", strings.NewReader(body), -1, "application/zip")).To(Succeed())
		st, err := s.Stat(ctx, "uploads/big.zip")
		Expect(err).NotTo(HaveOccurred())
		Expect(st.Size).To(Equal(int64(len(body))))
	})

	It("uploads a local file", func() {
		p := filepath.Join(GinkgoT().TempDir(), "u.zip")
		Expect(os.WriteFile(p, []byte("zipdata"), 0o600)).To(Succeed())
		Expect(s.PutFile(ctx, "uploads/u.zip", p, "application/zip")).To(Succeed())
		Expect(read("uploads/u.zip")).To(Equal("zipdata"))
	})

	It("reports a missing object as domain.ErrNotFound (Get, Stat, Copy)", func() {
		_, _, err := s.Get(ctx, "nope/x.pdf")
		Expect(err).To(MatchError(domain.ErrNotFound))
		_, err = s.Stat(ctx, "nope/x.pdf")
		Expect(err).To(MatchError(domain.ErrNotFound))
		Expect(s.Copy(ctx, "nope/x.pdf", "nope/y.pdf")).To(MatchError(domain.ErrNotFound))
	})

	It("copies server-side and keeps the source", func() {
		put("outputs/s1/x-main.pdf", "pdf-bytes")
		Expect(s.Copy(ctx, "outputs/s1/x-main.pdf", "portfolios/s1.pdf")).To(Succeed())
		Expect(read("portfolios/s1.pdf")).To(Equal("pdf-bytes"))
		Expect(read("outputs/s1/x-main.pdf")).To(Equal("pdf-bytes"))
	})

	It("deletes one object and treats a missing one as success", func() {
		put("uploads/d.zip", "z")
		Expect(s.Delete(ctx, "uploads/d.zip")).To(Succeed())
		_, err := s.Stat(ctx, "uploads/d.zip")
		Expect(err).To(MatchError(domain.ErrNotFound))
		Expect(s.Delete(ctx, "uploads/d.zip")).To(Succeed())
	})

	It("deletes a whole prefix but not its siblings", func() {
		put("outputs/aa/1.pdf", "1")
		put("outputs/aa/sub/2.md", "2")
		put("outputs/aab/3.pdf", "3")
		Expect(s.DeletePrefix(ctx, "outputs/aa/")).To(Succeed())
		objs, err := s.List(ctx, "outputs/", true)
		Expect(err).NotTo(HaveOccurred())
		Expect(objs).To(HaveLen(1))
		Expect(objs[0].Key).To(Equal("outputs/aab/3.pdf"))
		Expect(s.DeletePrefix(ctx, "outputs/aa")).To(Succeed()) // no trailing slash: still only that directory
		Expect(s.DeletePrefix(ctx, "outputs/missing/")).To(Succeed())
		objs, _ = s.List(ctx, "outputs/", true)
		Expect(objs).To(HaveLen(1))
	})

	It("lists recursively or one level, with LastModified, and an empty prefix yields an empty slice", func() {
		put("uploads/a.zip", "a")
		put("uploads/b.zip", "b")
		put("outputs/x/1.pdf", "1")
		rec, err := s.List(ctx, "", true)
		Expect(err).NotTo(HaveOccurred())
		Expect(rec).To(HaveLen(3))
		for _, o := range rec {
			Expect(o.LastModified).NotTo(BeZero())
		}
		flat, err := s.List(ctx, "uploads/", false)
		Expect(err).NotTo(HaveOccurred())
		Expect(flat).To(HaveLen(2))
		none, err := s.List(ctx, "covers/", true)
		Expect(err).NotTo(HaveOccurred())
		Expect(none).NotTo(BeNil())
		Expect(none).To(BeEmpty())
	})

	It("rejects unsafe keys", func() {
		for _, k := range []string{"", "/abs", "a/../b", "a//b", "..", "a\b"} {
			Expect(s.Put(ctx, k, strings.NewReader("x"), 1, "")).To(MatchError(domain.ErrInvalidInput), "key %q", k)
		}
		Expect(s.DeletePrefix(ctx, "")).To(MatchError(domain.ErrInvalidInput))
	})
})

var _ = Describe("MinIO adapter without a server", func() {
	It("requires an endpoint and a bucket", func() {
		_, err := storage.NewMinIO(storage.Config{})
		Expect(err).To(HaveOccurred())
	})
	It("reports an unreachable endpoint from Ping", func() {
		s, err := storage.NewMinIO(storage.Config{Endpoint: "127.0.0.1:1", Bucket: "b", AccessKey: "a", SecretKey: "s"})
		Expect(err).NotTo(HaveOccurred())
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		Expect(s.Ping(ctx)).NotTo(Succeed())
	})
})
